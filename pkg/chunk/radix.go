package chunk

import (
	"math"
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
// library: the kernel is small, has no external dependency or supply-chain
// risk, and a sequential LSD radix pass is already memory-bandwidth bound, so a
// parallel merge would add complexity for limited extra gain at these sizes.

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

// maxRadixMergeWorkers caps how many ranges the parallel-merge argsort splits
// into, independently of GOMAXPROCS.
//
// Phase 1 (sorting the ranges) scales with the worker count, but the pairwise
// merge that follows costs log2(workers) waves of O(n) merging plus a
// fork-join barrier each — so past a point every extra worker buys less sorting
// than it adds in merging and coordination. Measured on a 12-core M4 Pro at 1M
// rows (`bench/top30`, GOMAXPROCS swept), where a profile attributed 37% of
// DataFrame/sort to pthread_cond wait/signal and 11% to mergeInto:
//
//	workers:  1      2      4      6      12
//	sort:  44.8ms 26.2ms 18.4ms 16.7ms 20.1ms
//	rank:  24.0ms 18.7ms 15.7ms 14.6ms 15.5ms
//
// Both curves bottom out at 6 and regress by 12, so the cap keeps the sort off
// the far side of that curve while GOMAXPROCS still drives the surrounding
// parallel gather.
//
// ponytail: a cap on the existing merge-based radix, not a new algorithm. An
// MSD-partition radix would remove the merge waves outright, but it needs an
// adaptive choice of partition byte to avoid collapsing to one bucket on the
// narrow value ranges this benchmark uses (int64 in [0,1000), float64 in
// [-50,50)) — real work with real stability risk, deferred until this cap stops
// being enough.
const maxRadixMergeWorkers = 6

// radixWorkers returns the range count for the parallel-merge argsort: every
// available core, capped by maxRadixMergeWorkers.
func radixWorkers() int {
	workers := runtime.GOMAXPROCS(0)
	if workers > maxRadixMergeWorkers {
		return maxRadixMergeWorkers
	}
	return workers
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
// `workers` contiguous ranges in place within a shared idx buffer (each worker
// writes only its own disjoint span), then pairwise-merges the sorted runs,
// ping-ponging between idx and tmp so the whole sort uses only the same three
// buffers (keys, idx, tmp) as the sequential radix — no per-run allocation. The
// merge always drains the earlier (lower-index) run first on equal keys, so the
// result is the same stable ascending permutation as radixArgsort.
func parallelRadixArgsort(keys []uint64) []int {
	n := len(keys)
	workers := radixWorkers()
	if n < parallelMergeThreshold || workers <= 1 {
		return radixArgsort(keys)
	}
	if workers > n {
		workers = n
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	tmp := make([]int, n)

	bounds := make([]int, workers+1)
	for w := 0; w <= workers; w++ {
		bounds[w] = w * n / workers
	}
	// Phase 1: sort each range in place (idx[lo:hi]) using tmp[lo:hi] as scratch.
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			radixSortRangeInto(keys, idx, tmp, lo, hi)
		}(bounds[w], bounds[w+1])
	}
	wg.Wait()

	// Phase 2: pairwise-merge the sorted runs, ping-ponging idx<->tmp.
	src, dst := idx, tmp
	runs := bounds
	for len(runs) > 2 { // more than one run remaining
		runs = mergePairsParallel(keys, src, dst, runs)
		src, dst = dst, src
	}
	return src
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

// mergePairsParallel merges adjacent pairs of the sorted runs in src into dst at
// the same (disjoint) index spans — so the per-pair merges run concurrently
// without sharing output — and returns the new run boundaries. An odd trailing
// run is copied across unchanged so dst holds the full level result.
func mergePairsParallel(keys []uint64, src, dst []int, runs []int) []int {
	numRuns := len(runs) - 1
	var wg sync.WaitGroup
	for i := 0; i+1 < numRuns; i += 2 {
		lo, mid, hi := runs[i], runs[i+1], runs[i+2]
		wg.Add(1)
		go func(lo, mid, hi int) {
			defer wg.Done()
			mergeInto(keys, src, dst, lo, mid, hi)
		}(lo, mid, hi)
	}
	if numRuns%2 == 1 { // odd trailing run: carry it up unchanged
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

// mergeInto stably merges the two sorted runs src[lo:mid] and src[mid:hi] into
// dst[lo:hi] by key. On equal keys it takes from the left run first, which (since
// the left run holds the lower original indices) preserves ascending-index order
// across ties — matching the sequential radix's stability.
func mergeInto(keys []uint64, src, dst []int, lo, mid, hi int) {
	i, j, k := lo, mid, lo
	for i < mid && j < hi {
		if keys[src[j]] < keys[src[i]] {
			dst[k] = src[j]
			j++
		} else {
			dst[k] = src[i]
			i++
		}
		k++
	}
	for i < mid {
		dst[k] = src[i]
		i++
		k++
	}
	for j < hi {
		dst[k] = src[j]
		j++
		k++
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
