package chunk

import (
	"fmt"
	"maps"
	"math"
	"math/rand"
	"runtime"
	"slices"
	"sort"
	"sync"
	"testing"
)

// radixTestFloats builds a float64 slice that exercises the paths the parallel
// argsort must get right: many equal keys (stability), negatives and positives
// (the order-preserving transform's sign handling), and both -0.0 and 0.0.
func radixTestFloats(n int) []float64 {
	r := rand.New(rand.NewSource(7))
	vals := make([]float64, n)
	negZeroVal := 0.0
	negZeroVal = -negZeroVal
	for i := range vals {
		switch i % 8 {
		case 0:
			vals[i] = 0.0
		case 1:
			vals[i] = negZeroVal
		case 2:
			vals[i] = -float64(r.Intn(50)) // repeated negatives
		case 3:
			vals[i] = float64(r.Intn(50)) // repeated positives
		default:
			vals[i] = r.Float64()*200 - 100
		}
	}
	return vals
}

func radixTestInts(n int) []int64 {
	r := rand.New(rand.NewSource(11))
	vals := make([]int64, n)
	for i := range vals {
		switch i % 5 {
		case 0:
			vals[i] = 0
		case 1:
			vals[i] = -int64(r.Intn(100)) // repeats
		case 2:
			vals[i] = math.MinInt64
		case 3:
			vals[i] = math.MaxInt64
		default:
			vals[i] = r.Int63() - (1 << 62)
		}
	}
	return vals
}

// stableRefFloat returns the permutation a stable comparison sort produces —
// the contract both radix paths must reproduce exactly, ties included.
func stableRefFloat(vals []float64) []int {
	idx := make([]int, len(vals))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return vals[idx[a]] < vals[idx[b]] })
	return idx
}

func stableRefInt(vals []int64) []int {
	idx := make([]int, len(vals))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return vals[idx[a]] < vals[idx[b]] })
	return idx
}

// radixDists are the value distributions the parallel argsort must handle
// identically to the stable reference. The degenerate ones carry the weight:
// the parallel path splits the input into runs and merges them, so a run
// boundary can fall inside a block of equal keys ("all equal"), and a run can
// lie entirely below ("ascending") or entirely above ("descending") the run it
// is merged with — the cases where a merge drains one side without ever
// comparing against the other.
var radixDists = []struct {
	name   string
	floats func(n int) []float64
	ints   func(n int) []int64
}{
	{
		name:   "mixed",
		floats: radixTestFloats,
		ints:   radixTestInts,
	},
	{
		name:   "all equal",
		floats: func(n int) []float64 { return make([]float64, n) },
		ints:   func(n int) []int64 { return make([]int64, n) },
	},
	{
		name:   "ascending",
		floats: func(n int) []float64 { return genFloats(n, func(i int) float64 { return float64(i) }) },
		ints:   func(n int) []int64 { return genInts(n, func(i int) int64 { return int64(i) }) },
	},
	{
		name:   "descending",
		floats: func(n int) []float64 { return genFloats(n, func(i int) float64 { return float64(-i) }) },
		ints:   func(n int) []int64 { return genInts(n, func(i int) int64 { return int64(-i) }) },
	},
	{
		// Ties inside each half plus a strict split between them: every key in
		// the first half compares less than every key in the second.
		name:   "two blocks",
		floats: func(n int) []float64 { return genFloats(n, func(i int) float64 { return float64(2 * i / max(n, 1)) }) },
		ints:   func(n int) []int64 { return genInts(n, func(i int) int64 { return int64(2 * i / max(n, 1)) }) },
	},
}

func genFloats(n int, f func(int) float64) []float64 {
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = f(i)
	}
	return vals
}

func genInts(n int, f func(int) int64) []int64 {
	vals := make([]int64, n)
	for i := range vals {
		vals[i] = f(i)
	}
	return vals
}

// assertPermutation reports whether idx holds every index in [0,n) exactly once.
// Full equality against the stable reference already implies this; the separate
// check exists to name the failure mode, because a merge that drops or
// duplicates an index still yields a plausibly sorted result.
func assertPermutation(t *testing.T, idx []int, n int, what string) {
	t.Helper()
	if len(idx) != n {
		t.Errorf("%s: len = %d, want %d", what, len(idx), n)
		return
	}
	seen := make([]bool, n)
	for _, ix := range idx {
		if ix < 0 || ix >= n {
			t.Errorf("%s: index %d out of range [0,%d)", what, ix, n)
			return
		}
		if seen[ix] {
			t.Errorf("%s: index %d appears more than once", what, ix)
			return
		}
		seen[ix] = true
	}
}

// TestArgsortParallelMatchesSequentialAndStable checks the parallel argsort
// equals both the sequential radix and a stable comparison sort, at sizes below
// and above the parallel threshold. This path backs DataFrame/sort and Expr/rank
// but had no test before.
func TestArgsortParallelMatchesSequentialAndStable(t *testing.T) {
	t.Parallel()
	sizes := []int{0, 1, 2, 1000, parallelMergeThreshold - 1, parallelMergeThreshold + 1, 300000}
	for _, dist := range radixDists {
		for _, n := range sizes {
			floats := dist.floats(n)
			wantF := stableRefFloat(floats)
			got := ArgsortFloat64Parallel(floats)
			assertPermutation(t, got, n, fmt.Sprintf("%s/n=%d: float parallel", dist.name, n))
			if !equalInts(got, wantF) {
				t.Fatalf("%s/n=%d: float parallel argsort != stable sort", dist.name, n)
			}
			if got := ArgsortFloat64(floats); !equalInts(got, wantF) {
				t.Fatalf("%s/n=%d: float sequential argsort != stable sort", dist.name, n)
			}

			ints := dist.ints(n)
			wantI := stableRefInt(ints)
			gotI := ArgsortInt64Parallel(ints)
			assertPermutation(t, gotI, n, fmt.Sprintf("%s/n=%d: int parallel", dist.name, n))
			if !equalInts(gotI, wantI) {
				t.Fatalf("%s/n=%d: int parallel argsort != stable sort", dist.name, n)
			}
			if got := ArgsortInt64(ints); !equalInts(got, wantI) {
				t.Fatalf("%s/n=%d: int sequential argsort != stable sort", dist.name, n)
			}
		}
	}
}

// assertAscendingWithStableTies checks in one O(n) pass that idx orders the keys
// ascending and that equal keys keep their original index order. Used where
// building a sort.SliceStable reference would cost more than the property check.
func assertAscendingWithStableTies[T float64 | int64](t *testing.T, idx []int, key func(int) T, what string) {
	t.Helper()
	for i := 1; i < len(idx); i++ {
		prev, cur := key(idx[i-1]), key(idx[i])
		switch {
		case prev > cur:
			t.Fatalf("%s: not ascending at %d: key(%d) = %v > key(%d) = %v", what, i, idx[i-1], prev, idx[i], cur)
		case prev == cur && idx[i-1] > idx[i]:
			t.Fatalf("%s: unstable tie at %d: equal keys %v but original index %d precedes %d",
				what, i, prev, idx[i-1], idx[i])
		}
	}
}

// TestArgsortParallelStableWithinTieGroups checks tie order at a size and
// cardinality the reference-equality test cannot afford: 1M rows over 5 distinct
// keys, where nearly every comparison is a tie. This is the failure mode a
// range-divided merge introduces — choosing a boundary with the wrong tie
// strictness reorders equal keys while still producing a sorted result, so
// "is it sorted" never catches it.
func TestArgsortParallelStableWithinTieGroups(t *testing.T) {
	t.Parallel()
	const n = 1 << 20
	r := rand.New(rand.NewSource(13))
	floats := make([]float64, n)
	ints := make([]int64, n)
	for i := range floats {
		floats[i] = float64(r.Intn(5))
		ints[i] = int64(r.Intn(5))
	}

	gotF := ArgsortFloat64Parallel(floats)
	assertPermutation(t, gotF, n, "ArgsortFloat64Parallel 5-key")
	assertAscendingWithStableTies(t, gotF, func(i int) float64 { return floats[i] }, "ArgsortFloat64Parallel")

	gotI := ArgsortInt64Parallel(ints)
	assertPermutation(t, gotI, n, "ArgsortInt64Parallel 5-key")
	assertAscendingWithStableTies(t, gotI, func(i int) int64 { return ints[i] }, "ArgsortInt64Parallel")
}

// TestArgsortParallelIndependentOfWorkerCount checks the permutation does not
// depend on how many ranges the input was split into, nor on how many output
// ranges its merges were divided into — both follow from GOMAXPROCS, so neither
// may be observable in the result.
func TestArgsortParallelIndependentOfWorkerCount(t *testing.T) {
	const n = 300000
	floats := radixTestFloats(n)
	ints := radixTestInts(n)
	wantF := stableRefFloat(floats)
	wantI := stableRefInt(ints)

	original := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(original)

	for _, procs := range []int{1, 2, 3, 5, 8, 16} {
		runtime.GOMAXPROCS(procs)
		if got := ArgsortFloat64Parallel(floats); !equalInts(got, wantF) {
			t.Fatalf("GOMAXPROCS=%d: float permutation differs from the stable reference", procs)
		}
		if got := ArgsortInt64Parallel(ints); !equalInts(got, wantI) {
			t.Fatalf("GOMAXPROCS=%d: int permutation differs from the stable reference", procs)
		}
	}
}

// mergeFixture builds two sorted runs placed at src[off:off+la] and
// src[off+la:off+la+lb] inside a larger buffer, so tests exercise a non-zero lo
// and catch index arithmetic that only works at offset 0. Keys are drawn from a
// small set so ties, not ordering, dominate the boundary decisions.
func mergeFixture(r *rand.Rand, sh mergeShape) (keys []uint64, src []int, lo, mid, hi int) {
	lo, mid, hi = sh.off, sh.off+sh.la, sh.off+sh.la+sh.lb
	keys = make([]uint64, hi+sh.off)
	src = make([]int, hi+sh.off)
	for i := range src {
		src[i] = i
	}
	left := make([]uint64, sh.la)
	right := make([]uint64, sh.lb)
	for i := range left {
		left[i] = uint64(r.Intn(10))
	}
	for i := range right {
		// disjoint puts every right key strictly above every left key, so an
		// output range can fall entirely inside one run — the case where a merge
		// drains one side without ever comparing against the other.
		if sh.disjoint {
			left[i%max(sh.la, 1)] = uint64(r.Intn(5))
			right[i] = uint64(5 + r.Intn(5))
		} else {
			right[i] = uint64(r.Intn(10))
		}
	}
	slices.Sort(left)
	slices.Sort(right)
	copy(keys[lo:mid], left)
	copy(keys[mid:hi], right)
	return keys, src, lo, mid, hi
}

type mergeShape struct {
	off, la, lb int
	disjoint    bool
}

var mergeShapes = []mergeShape{
	{off: 0, la: 0, lb: 0}, {off: 0, la: 0, lb: 5}, {off: 0, la: 5, lb: 0},
	{off: 0, la: 1, lb: 1}, {off: 0, la: 7, lb: 3}, {off: 0, la: 3, lb: 7},
	{off: 3, la: 0, lb: 4}, {off: 3, la: 4, lb: 0}, {off: 3, la: 64, lb: 64},
	{off: 5, la: 129, lb: 17}, {off: 7, la: 17, lb: 129}, {off: 2, la: 1000, lb: 1000},
	{off: 3, la: 64, lb: 64, disjoint: true}, {off: 5, la: 129, lb: 17, disjoint: true},
	{off: 1, la: 1000, lb: 1000, disjoint: true},
}

// TestMergeSplitAtMatchesMergedPrefix checks the co-ranked boundary against the
// sequential merge at every offset k: the elements left of the split must be
// exactly the first k of the merged run. Comparing as sets is the right check —
// the values are indices, so they are unique — and it does not presuppose the
// ranged merge this boundary exists to enable.
func TestMergeSplitAtMatchesMergedPrefix(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(5))
	for _, sh := range mergeShapes {
		keys, src, lo, mid, hi := mergeFixture(r, sh)
		dst := make([]int, len(src))
		mergeRuns(keys, src, dst, lo, mid, mid, hi, lo)

		for k := 0; k <= hi-lo; k++ {
			i, j := mergeSplitAt(keys, src, lo, mid, hi, k)
			if got := (i - lo) + (j - mid); got != k {
				t.Fatalf("mergeSplitAt(la=%d,lb=%d,k=%d) = (%d,%d), covers %d elements, want %d",
					sh.la, sh.lb, k, i, j, got, k)
			}
			if i < lo || i > mid || j < mid || j > hi {
				t.Fatalf("mergeSplitAt(la=%d,lb=%d,k=%d) = (%d,%d) out of range [%d,%d]/[%d,%d]",
					sh.la, sh.lb, k, i, j, lo, mid, mid, hi)
			}
			want := make(map[int]bool, k)
			for _, ix := range dst[lo : lo+k] {
				want[ix] = true
			}
			got := make(map[int]bool, k)
			for _, ix := range src[lo:i] {
				got[ix] = true
			}
			for _, ix := range src[mid:j] {
				got[ix] = true
			}
			if !maps.Equal(got, want) {
				t.Fatalf("mergeSplitAt(la=%d,lb=%d,k=%d) = (%d,%d): prefix set differs from the merged prefix",
					sh.la, sh.lb, k, i, j)
			}
		}
	}
}

// TestMergePairIntoIdenticalForEveryPartCount checks that dividing one merge into
// P output ranges gives exactly the result of merging it in one piece, for every
// P including P larger than the element count. This is the spec scenario "result
// is independent of how the combining work is divided", checked at the level of a
// single merge rather than a whole sort, and it is the test that would catch a
// boundary whose tie strictness disagrees with mergeRuns.
func TestMergePairIntoIdenticalForEveryPartCount(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(9))
	for _, sh := range mergeShapes {
		keys, src, lo, mid, hi := mergeFixture(r, sh)
		want := make([]int, len(src))
		mergeRuns(keys, src, want, lo, mid, mid, hi, lo)

		for _, parts := range []int{1, 2, 3, 4, 8, 16, sh.la + sh.lb + 5} {
			got := make([]int, len(src))
			var wg sync.WaitGroup
			mergePairInto(&wg, keys, src, got, lo, mid, hi, parts)
			wg.Wait()
			if !equalInts(got[lo:hi], want[lo:hi]) {
				t.Errorf("mergePairInto(off=%d,la=%d,lb=%d,disjoint=%t,parts=%d) differs from the single-range merge:\n got %v\nwant %v",
					sh.off, sh.la, sh.lb, sh.disjoint, parts, got[lo:hi], want[lo:hi])
			}
		}
	}
}

// TestMergeRunsParallelAnyRunCountAndBudget merges pre-sorted runs for run counts
// that are not powers of two, at several parts budgets. parallelRadixArgsort only
// ever asks for a power of two, so these are the levels that carry an odd
// trailing run — the branch that would silently drop a run's worth of indices if
// it were removed as unreachable. It also pins the spec property that the result
// does not depend on the budget.
func TestMergeRunsParallelAnyRunCountAndBudget(t *testing.T) {
	t.Parallel()
	const n = 5000
	r := rand.New(rand.NewSource(21))
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = float64(r.Intn(50)) // heavy duplication, so ties drive the merges
	}
	keys := make([]uint64, n)
	for i, v := range vals {
		keys[i] = orderPreservingFloat(v)
	}
	want := stableRefFloat(vals)

	for _, runCount := range []int{1, 2, 3, 5, 6, 7, 9, 12, 13} {
		for _, budget := range []int{1, 2, 3, 12} {
			bounds := rangeBounds(n, runCount)
			idx := make([]int, n)
			for i := range idx {
				idx[i] = i
			}
			tmp := make([]int, n)
			sortRangesParallel(keys, idx, tmp, bounds)
			got := mergeRunsParallel(keys, idx, tmp, bounds, budget)
			assertPermutation(t, got, n, fmt.Sprintf("runs=%d budget=%d", runCount, budget))
			if !equalInts(got, want) {
				t.Errorf("mergeRunsParallel(runs=%d, budget=%d) != stable reference", runCount, budget)
			}
		}
	}
}

// TestMergePartsSharesBudgetByLength checks the budget split: proportional to
// pair length, never below one, and never handing a level more ranges than the
// budget allows.
func TestMergePartsSharesBudgetByLength(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pairLen, totalLen, budget, want int
	}{
		// Proportional split, at lengths long enough that minMergePartLen does not
		// bind (a 1M level: the cap allows 64 parts, so the budget decides).
		{pairLen: 1 << 20, totalLen: 1 << 20, budget: 12, want: 12}, // single pair takes the whole budget
		{pairLen: 1 << 19, totalLen: 1 << 20, budget: 12, want: 6},  // two equal pairs split it
		{pairLen: 1 << 18, totalLen: 1 << 20, budget: 12, want: 3},

		{pairLen: 1, totalLen: 1 << 20, budget: 12, want: 1},      // rounds to zero, floored at one
		{pairLen: 1 << 20, totalLen: 1 << 20, budget: 1, want: 1}, // no budget to divide
		{pairLen: 1 << 20, totalLen: 0, budget: 12, want: 1},      // empty level
		{pairLen: 0, totalLen: 1 << 20, budget: 12, want: 1},      // empty pair

		// minMergePartLen caps the share: at the smallest input that reaches the
		// parallel path, the lowest level's pairs are one part each, and only the
		// upper levels — where the pairs are long enough — get the whole budget.
		{pairLen: 1 << 14, totalLen: 1 << 16, budget: 12, want: 1},
		{pairLen: 1 << 15, totalLen: 1 << 16, budget: 12, want: 2},
		{pairLen: 1 << 16, totalLen: 1 << 16, budget: 12, want: 4},
		{pairLen: 1 << 20, totalLen: 1 << 20, budget: 12, want: 12}, // 1M: cap 64, budget wins
	}
	for _, c := range cases {
		if got := mergeParts(c.pairLen, c.totalLen, c.budget); got != c.want {
			t.Errorf("mergeParts(%d, %d, %d) = %d, want %d", c.pairLen, c.totalLen, c.budget, got, c.want)
		}
	}
}

// radixBenchN and radixBenchKeys describe the reference sort workload: 1M
// uniform float64 in [-50,50), the distribution bench/top30 sorts its "v" column
// on, already run through the order-preserving transform.
const radixBenchN = 1 << 20

func radixBenchKeys(n int) []uint64 {
	r := rand.New(rand.NewSource(7))
	keys := make([]uint64, n)
	for i := range keys {
		keys[i] = orderPreservingFloat(r.Float64()*100 - 50)
	}
	return keys
}

// The four benchmarks below exist to attribute the parallel argsort's time to a
// phase rather than to the whole. They are what showed the merge, not the radix
// passes, to be the dominant cost, and they are how a change to either phase is
// held to account: compare Phase1Ranges + Phase2Merge against Parallel, and
// Parallel against Sequential.

// BenchmarkRadixPhase1Ranges measures phase 1 alone — the concurrent radix sort
// of each range, with no merging.
func BenchmarkRadixPhase1Ranges(b *testing.B) {
	keys := radixBenchKeys(radixBenchN)
	runs, _ := radixWorkers()
	bounds := rangeBounds(len(keys), runs)
	b.ReportAllocs()
	for b.Loop() {
		idx := make([]int, len(keys))
		for j := range idx {
			idx[j] = j
		}
		tmp := make([]int, len(keys))
		sortRangesParallel(keys, idx, tmp, bounds)
	}
}

// BenchmarkRadixPhase2Merge measures phase 2 alone — merging runs that are
// already sorted, which is the phase this package's merge strategy governs.
func BenchmarkRadixPhase2Merge(b *testing.B) {
	keys := radixBenchKeys(radixBenchN)
	n := len(keys)
	runs, _ := radixWorkers()
	bounds := rangeBounds(n, runs)
	base := make([]int, n)
	for j := range base {
		base[j] = j
	}
	sortRangesParallel(keys, base, make([]int, n), bounds)

	idx := make([]int, n)
	tmp := make([]int, n)
	b.ReportAllocs()
	for b.Loop() {
		copy(idx, base)
		mergeRunsParallel(keys, idx, tmp, bounds, runtime.GOMAXPROCS(0))
	}
}

// BenchmarkRadixArgsortParallel measures the whole parallel argsort.
func BenchmarkRadixArgsortParallel(b *testing.B) {
	keys := radixBenchKeys(radixBenchN)
	b.ReportAllocs()
	for b.Loop() {
		parallelRadixArgsort(keys)
	}
}

// BenchmarkRadixArgsortSequential is the single-threaded reference the parallel
// path has to beat by enough to justify its coordination cost.
func BenchmarkRadixArgsortSequential(b *testing.B) {
	keys := radixBenchKeys(radixBenchN)
	b.ReportAllocs()
	for b.Loop() {
		radixArgsort(keys)
	}
}

// BenchmarkRadixArgsortRunsSweep sweeps the run count at the budget radixWorkers
// would pick, so the choice of run count rests on a measurement rather than on
// the argument for it. Run with -cpu to sweep GOMAXPROCS at the same time.
func BenchmarkRadixArgsortRunsSweep(b *testing.B) {
	keys := radixBenchKeys(radixBenchN)
	budget := runtime.GOMAXPROCS(0)
	defaultRuns, _ := radixWorkers()
	for _, runs := range []int{1, 2, 4, 6, 8, 12, 16} {
		if runs > budget*2 {
			continue
		}
		name := fmt.Sprintf("runs=%d", runs)
		if runs == defaultRuns {
			name += "(default)"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				parallelRadixArgsortWith(keys, runs, budget)
			}
		})
	}
}

// BenchmarkRadixArgsortBudgetSweep sweeps the parts budget at the default run
// count. budget=1 gives every pair exactly one part (see mergeParts), which is
// the one-goroutine-per-pair strategy this package used before merges were
// divided by output range — so budget=1 against budget=GOMAXPROCS measures what
// that change is worth, on whatever machine and -cpu it is run.
func BenchmarkRadixArgsortBudgetSweep(b *testing.B) {
	keys := radixBenchKeys(radixBenchN)
	runs, dflt := radixWorkers()
	for _, budget := range []int{1, 2, 4, 8, 12} {
		if budget > dflt*2 {
			continue
		}
		name := fmt.Sprintf("budget=%d", budget)
		switch budget {
		case 1:
			name += "(pre-change)"
		case dflt:
			name += "(default)"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				parallelRadixArgsortWith(keys, runs, budget)
			}
		})
	}
}

// BenchmarkRadixArgsortNearThreshold covers the sizes where minMergePartLen
// actually binds. At 1M it never does — the shortest pair at any level is far
// above it — so the constant can only be judged just above parallelMergeThreshold.
func BenchmarkRadixArgsortNearThreshold(b *testing.B) {
	runs, budget := radixWorkers()
	for _, n := range []int{parallelMergeThreshold + 1, 4 * parallelMergeThreshold, 16 * parallelMergeThreshold} {
		keys := radixBenchKeys(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				parallelRadixArgsortWith(keys, runs, budget)
			}
		})
	}
}

// TestRadixWorkersDerivedFromAvailableParallelism checks both numbers against
// GOMAXPROCS across the whole range, not just a few points: the budget tracks
// available parallelism with no hidden ceiling, and the run count is always a
// power of two in (GOMAXPROCS/2, GOMAXPROCS].
//
// The power-of-two invariant is load-bearing, not cosmetic — it is what makes
// every merge level a pair count with no odd run left over, which is why
// mergePairsParallel has no carry branch to get wrong.
func TestRadixWorkersDerivedFromAvailableParallelism(t *testing.T) {
	original := runtime.GOMAXPROCS(0)
	t.Cleanup(func() { runtime.GOMAXPROCS(original) })

	for procs := 1; procs <= 32; procs++ {
		runtime.GOMAXPROCS(procs)
		runs, budget := radixWorkers()
		if budget != procs {
			t.Errorf("GOMAXPROCS=%d: budget = %d, want %d", procs, budget, procs)
		}
		if runs < 1 || runs > procs {
			t.Errorf("GOMAXPROCS=%d: runs = %d, want within [1,%d]", procs, runs, procs)
			continue
		}
		if runs&(runs-1) != 0 {
			t.Errorf("GOMAXPROCS=%d: runs = %d, want a power of two", procs, runs)
		}
		if runs*2 <= procs {
			t.Errorf("GOMAXPROCS=%d: runs = %d, want the largest power of two <= %d", procs, runs, procs)
		}
	}
}
