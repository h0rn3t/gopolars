package chunk

import (
	"math"
	"math/bits"
	"runtime"
	"sync"
)

// LSD radix argsort for numeric columns. These return an index permutation that
// orders the input ascending, in O(n) time (8 passes over 8-bit digits) rather
// than the O(n log n) of a comparison sort. They use the Lemire/Herf
// order-preserving key transform so unsigned-integer digit order equals the
// signed/float numeric order.
//
// The decision (design D7) was to implement this inline rather than vendor a
// library: the kernel is small and has no external dependency or supply-chain
// risk.
//
// That decision also held that "a parallel merge would add complexity for
// limited extra gain at these sizes". Measurement disproved it. Attributing the
// 1M argsort by phase showed the merge costing 10.9ms against 3.6ms for all the
// radix passes together, because the merge tree ran one goroutine per pair: its
// last level merged the whole input on a single goroutine while every other core
// waited. Dividing each merge by output range instead (mergeSplitAt,
// mergePairInto) took the merge to 2.5ms and the whole argsort from 17.5ms to
// 5.3ms — see BenchmarkRadixPhase1Ranges / BenchmarkRadixPhase2Merge, which exist
// to keep that attribution honest.

// orderPreservingFloat maps a float64 to a uint64 whose unsigned order matches
// ascending float order: flip all bits for negatives (sign set), flip just the
// sign bit for non-negatives. Callers must exclude NaN beforehand.
//
// -0.0 is collapsed onto +0.0 first. Comparing with `<` (what the comparison
// sort this kernel must match uses, and what Polars' Rust f64 comparison does)
// treats -0.0 and 0.0 as equal, so a stable sort has to leave them in their
// original relative order. Their raw bit patterns differ only in the sign bit,
// which the transform would otherwise turn into -0.0 sorting strictly before
// 0.0 — reordering equal values and breaking stability.
func orderPreservingFloat(v float64) uint64 {
	if v == 0 { // true for both -0.0 and +0.0
		v = 0
	}
	bits := math.Float64bits(v)
	mask := uint64(int64(bits)>>63) | (uint64(1) << 63)
	return bits ^ mask
}

// orderPreservingInt maps an int64 to a uint64 whose unsigned order matches
// ascending signed order (flip the sign bit).
func orderPreservingInt(v int64) uint64 {
	return uint64(v) ^ (uint64(1) << 63)
}

// ArgsortFloat64 returns indices that sort vals ascending. Behavior is undefined
// for NaN inputs (callers gate the radix path on a NaN-free, null-free column).
func ArgsortFloat64(vals []float64) []int {
	keys := make([]uint64, len(vals))
	for i, v := range vals {
		keys[i] = orderPreservingFloat(v)
	}
	return radixArgsort(keys)
}

// ArgsortInt64 returns indices that sort vals ascending.
func ArgsortInt64(vals []int64) []int {
	keys := make([]uint64, len(vals))
	for i, v := range vals {
		keys[i] = orderPreservingInt(v)
	}
	return radixArgsort(keys)
}

// parallelMergeThreshold is the length at or above which the parallel-merge
// argsort splits the input across workers and merges the sorted runs. Below it
// (and with a single worker) the sequential LSD radix wins on constant factors —
// the merge adds an O(n) pass that only pays off on large inputs.
const parallelMergeThreshold = 1 << 16 // 65536

// minMergePartLen is the smallest output range worth its own goroutine. Dividing
// a merge finer than this trades O(n/parts) of merging for a worker wakeup that
// costs more than it saves — the same reason parallelMergeThreshold exists, one
// level down. It bounds parts from above; the budget bounds it from below.
//
// Swept on a 12-core M4 Pro with BenchmarkRadixArgsortNearThreshold, the sizes
// where it binds at all (at 1M the shortest pair at every level is already far
// above any of these, so the constant is inert there):
//
//	minMergePartLen:  1<<12         1<<14         1<<16
//	n=65537:       496µs ±42%    509µs ±13%    774µs ±38%
//	n=262144:      1.499m ±12%   1.487m ±10%   1.899m ±31%
//	goroutines at n=65537:  90            50            40
//
// 1<<12 and 1<<14 tie on time; 1<<14 reaches it with half the goroutines and far
// tighter variance, and 1<<16 starves the merge of parts. Hence 1<<14.
const minMergePartLen = 1 << 14 // 16384

// radixWorkers returns how the parallel argsort spends the available cores: runs
// is how many ranges phase 1 sorts concurrently, budget is how many concurrent
// output ranges each merge level may divide itself into.
//
// These used to be one number. While a merge ran on one goroutine per pair, the
// parallelism of a level was its pair count, so raising the run count added
// merge waves faster than it added sorting — the reason a hard cap of 6 beat
// GOMAXPROCS=12 on a 12-core M4 Pro (sort 16.7ms at 6 vs 20.1ms at 12). Dividing
// a merge by output range instead (mergePairInto) unties them: the budget alone
// sets the parallelism of every level, and the run count only sets how many
// levels there are.
//
// runs is therefore the largest power of two not exceeding GOMAXPROCS. A power
// of two gives exactly log2(runs) levels with no odd run to carry, and trading a
// few idle cores in phase 1 for one fewer level is worth it: a level costs a
// fork-join barrier, measured at ~200µs of parked-thread wakeup on darwin, while
// phase 1 is only a few ms in total.
func radixWorkers() (runs, budget int) {
	budget = runtime.GOMAXPROCS(0)
	if budget < 1 {
		budget = 1
	}
	// Largest power of two <= budget.
	runs = 1 << (bits.Len(uint(budget)) - 1)
	return runs, budget
}

// ArgsortFloat64Parallel is ArgsortFloat64 that radix-sorts disjoint ranges
// concurrently and stable-merges the runs above parallelMergeThreshold, falling
// back to the sequential radix below it. The result is identical to
// ArgsortFloat64 (same stable ascending permutation). Callers gate on a NaN-free,
// null-free column as for ArgsortFloat64.
func ArgsortFloat64Parallel(vals []float64) []int {
	keys := make([]uint64, len(vals))
	for i, v := range vals {
		keys[i] = orderPreservingFloat(v)
	}
	return parallelRadixArgsort(keys)
}

// ArgsortInt64Parallel is ArgsortInt64 with the same parallel-merge strategy as
// ArgsortFloat64Parallel; identical result to ArgsortInt64.
func ArgsortInt64Parallel(vals []int64) []int {
	keys := make([]uint64, len(vals))
	for i, v := range vals {
		keys[i] = orderPreservingInt(v)
	}
	return parallelRadixArgsort(keys)
}

// parallelRadixArgsort sorts indices [0,n) by their uint64 key. It radix-sorts
// `runs` contiguous ranges in place within a shared idx buffer (each worker
// writes only its own disjoint span), then merges the sorted runs level by
// level, ping-ponging between idx and tmp so the whole sort uses only the same
// three buffers (keys, idx, tmp) as the sequential radix — no per-run
// allocation. Every merge drains the earlier (lower-index) run first on equal
// keys, so the result is the same stable ascending permutation as radixArgsort,
// whatever the run count or the parts budget.
//
// runs never exceeds n here: the branch below leaves n >= parallelMergeThreshold
// (65536), well above any real GOMAXPROCS.
func parallelRadixArgsort(keys []uint64) []int {
	runs, budget := radixWorkers()
	return parallelRadixArgsortWith(keys, runs, budget)
}

// parallelRadixArgsortWith is parallelRadixArgsort with the run count and parts
// budget supplied rather than derived, so a benchmark can sweep them without
// restating the phase structure.
func parallelRadixArgsortWith(keys []uint64, runs, budget int) []int {
	n := len(keys)
	if n < parallelMergeThreshold || runs <= 1 {
		return radixArgsort(keys)
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	tmp := make([]int, n)

	bounds := rangeBounds(n, runs)
	// Phase 1: sort each range in place (idx[lo:hi]) using tmp[lo:hi] as scratch.
	sortRangesParallel(keys, idx, tmp, bounds)
	// Phase 2: merge the sorted runs into one.
	return mergeRunsParallel(keys, idx, tmp, bounds, budget)
}

// mergeRunsParallel merges the sorted runs delimited by bounds into a single
// ordered run, ping-ponging between idx and tmp so no buffer beyond these two is
// needed, and returns whichever of them holds the finished permutation. budget is
// the number of concurrent output ranges each level may divide itself into.
func mergeRunsParallel(keys []uint64, idx, tmp []int, bounds []int, budget int) []int {
	src, dst := idx, tmp
	runs := bounds
	for len(runs) > 2 { // more than one run remaining
		runs = mergePairsParallel(keys, src, dst, runs, budget)
		src, dst = dst, src
	}
	return src
}

// rangeBounds splits [0,n) into runs contiguous ranges of near-equal length,
// returned as runs+1 boundaries.
func rangeBounds(n, runs int) []int {
	bounds := make([]int, runs+1)
	for w := range bounds {
		bounds[w] = w * n / runs
	}
	return bounds
}

// sortRangesParallel radix-sorts each range idx[bounds[w]:bounds[w+1]] in place
// concurrently, using the matching span of tmp as scratch. Every worker writes
// only within its own span, so the ranges need no synchronization beyond the
// wait here; the goroutines cannot outlive the call.
func sortRangesParallel(keys []uint64, idx, tmp []int, bounds []int) {
	var wg sync.WaitGroup
	for w := 0; w+1 < len(bounds); w++ {
		lo, hi := bounds[w], bounds[w+1]
		wg.Go(func() { radixSortRangeInto(keys, idx, tmp, lo, hi) })
	}
	wg.Wait()
}

// radixSortRangeInto sorts idx[lo:hi] ascending by keys using tmp[lo:hi] as
// scratch (stable LSD radix, eight 8-bit passes). The eight passes are even, so
// the sorted result lands back in idx[lo:hi].
func radixSortRangeInto(keys []uint64, idx, tmp []int, lo, hi int) {
	if hi-lo <= 1 {
		return
	}
	a := idx[lo:hi]
	b := tmp[lo:hi]
	var count [256]int
	for shift := uint(0); shift < 64; shift += 8 {
		for i := range count {
			count[i] = 0
		}
		for _, ix := range a {
			count[(keys[ix]>>shift)&0xff]++
		}
		sum := 0
		for i := 0; i < 256; i++ {
			c := count[i]
			count[i] = sum
			sum += c
		}
		for _, ix := range a {
			d := (keys[ix] >> shift) & 0xff
			b[count[d]] = ix
			count[d]++
		}
		a, b = b, a
	}
}

// mergeParts is the share of a level's parts budget that a pair of length
// pairLen receives, where the level's pairs total totalLen: proportional to
// length, capped so no part falls below minMergePartLen, and never less than one.
//
// Proportional rather than equal because an odd carry leaves pairs of unequal
// length, and a level takes as long as its slowest part. The cap is what keeps
// the lowest levels — many small pairs — from spawning a goroutine per pair for
// a few thousand elements each.
func mergeParts(pairLen, totalLen, budget int) int {
	if budget <= 1 || totalLen <= 0 {
		return 1
	}
	parts := budget * pairLen / totalLen
	if byLen := pairLen / minMergePartLen; parts > byLen {
		parts = byLen
	}
	return max(parts, 1)
}

// mergePairsParallel merges adjacent pairs of the sorted runs in src into dst at
// the same (disjoint) index spans and returns the new run boundaries. An odd
// trailing run is copied across unchanged so dst holds the full level result.
//
// The whole level is one wave: budget is shared across its pairs, and each pair
// divides its share into output ranges (mergePairInto). Sizing the parallelism
// by output range rather than by pair count is what keeps the last level — a
// single pair covering the entire input — off one goroutine.
func mergePairsParallel(keys []uint64, src, dst []int, runs []int, budget int) []int {
	numRuns := len(runs) - 1
	totalLen := runs[numRuns] - runs[0]
	var wg sync.WaitGroup
	for i := 0; i+1 < numRuns; i += 2 {
		lo, mid, hi := runs[i], runs[i+1], runs[i+2]
		mergePairInto(&wg, keys, src, dst, lo, mid, hi, mergeParts(hi-lo, totalLen, budget))
	}
	if numRuns%2 == 1 { // odd trailing run: carry it up unchanged
		// parallelRadixArgsort never reaches this: radixWorkers returns a power
		// of two, so every level has an even run count. It is kept because the
		// carry is what makes this function correct for any bounds, and dropping
		// it would silently leave a run's span in dst holding stale indices.
		// Costs nothing on the sort path, where it never executes.
		lo, hi := runs[numRuns-1], runs[numRuns]
		copy(dst[lo:hi], src[lo:hi])
	}
	wg.Wait()
	next := make([]int, 0, numRuns/2+2)
	for i := 0; i < numRuns; i += 2 {
		next = append(next, runs[i])
	}
	next = append(next, runs[numRuns])
	return next
}

// mergeSplitAt returns the split point (i, j) of the two sorted runs src[lo:mid]
// and src[mid:hi] at output offset k: merging src[lo:i] with src[mid:j] yields
// exactly the first k elements of the merged run, so (i-lo) + (j-mid) == k.
// This is what lets one merge be divided into disjoint output ranges.
//
// The boundary must reproduce mergeInto's tie rule (the left run wins an equal
// key; the right run wins only when strictly smaller), which as a property of
// (i, j) reads:
//
//	i > lo && j < hi  =>  keys[src[i-1]] <= keys[src[j]]
//	j > mid && i < mid  =>  keys[src[j-1]] < keys[src[i]]
//
// Searching on the first condition alone is enough. It is monotone in i, and the
// largest i satisfying it satisfies the second too: were the second violated
// there, i+1 would satisfy the first as well, contradicting maximality.
func mergeSplitAt(keys []uint64, src []int, lo, mid, hi, k int) (int, int) {
	// a is how many of the k come from the left run, k-a from the right.
	alo := max(0, k-(hi-mid))
	ahi := min(k, mid-lo)
	for alo < ahi {
		// Rounding up keeps a >= alo+1 >= 1, so src[lo+a-1] is always in range.
		a := (alo + ahi + 1) / 2
		if b := k - a; b < hi-mid && keys[src[lo+a-1]] > keys[src[mid+b]] {
			ahi = a - 1 // took too many from the left
		} else {
			alo = a
		}
	}
	return lo + alo, mid + (k - alo)
}

// mergeRuns stably merges the sorted runs src[i0:i1] and src[j0:j1] into dst
// starting at out. On equal keys it takes from the left run first, which (since
// the left run holds the lower original indices) preserves ascending-index order
// across ties — matching the sequential radix's stability.
//
// This is the only place that tie rule is written down; mergeSplitAt restates it
// as a property of a boundary, so the two have to stay in step.
//
// The runs need not be adjacent and out need not equal i0 — that is exactly what
// lets one merge be divided into disjoint output ranges.
func mergeRuns(keys []uint64, src, dst []int, i0, i1, j0, j1, out int) {
	i, j, k := i0, j0, out
	for i < i1 && j < j1 {
		if keys[src[j]] < keys[src[i]] {
			dst[k] = src[j]
			j++
		} else {
			dst[k] = src[i]
			i++
		}
		k++
	}
	for i < i1 {
		dst[k] = src[i]
		i++
		k++
	}
	for j < j1 {
		dst[k] = src[j]
		j++
		k++
	}
}

// mergePairInto merges the sorted runs src[lo:mid] and src[mid:hi] into
// dst[lo:hi], dividing the output into parts disjoint ranges, one goroutine per
// range added to wg. The caller owns wg and must Wait before reading dst; the
// goroutines cannot outlive that Wait. Ranges never overlap, so they share no
// output and need no further synchronization.
//
// Dividing by output range rather than by pair is the point: it keeps a single
// large merge from running on one goroutine while the other cores idle.
//
// The result is identical for every parts value, because the boundaries come
// from mergeSplitAt, which reproduces mergeRuns' tie rule.
func mergePairInto(wg *sync.WaitGroup, keys []uint64, src, dst []int, lo, mid, hi, parts int) {
	total := hi - lo
	parts = min(max(parts, 1), max(total, 1))
	i0, j0, prev := lo, mid, 0
	for p := 1; p <= parts; p++ {
		k := p * total / parts
		i1, j1 := mergeSplitAt(keys, src, lo, mid, hi, k)
		li0, lj0, out := i0, j0, lo+prev
		wg.Go(func() { mergeRuns(keys, src, dst, li0, i1, lj0, j1, out) })
		i0, j0, prev = i1, j1, k
	}
}

// radixArgsort sorts indices [0,n) by their uint64 key via stable LSD radix
// (eight 8-bit passes).
func radixArgsort(keys []uint64) []int {
	n := len(keys)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	tmp := make([]int, n)
	var count [256]int
	for shift := uint(0); shift < 64; shift += 8 {
		for i := range count {
			count[i] = 0
		}
		for i := 0; i < n; i++ {
			count[(keys[idx[i]]>>shift)&0xff]++
		}
		sum := 0
		for i := 0; i < 256; i++ {
			c := count[i]
			count[i] = sum
			sum += c
		}
		for i := 0; i < n; i++ {
			d := (keys[idx[i]] >> shift) & 0xff
			tmp[count[d]] = idx[i]
			count[d]++
		}
		idx, tmp = tmp, idx
	}
	return idx
}
