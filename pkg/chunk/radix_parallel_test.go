package chunk

import (
	"math"
	"math/rand"
	"runtime"
	"sort"
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

// TestArgsortParallelMatchesSequentialAndStable checks the parallel argsort
// equals both the sequential radix and a stable comparison sort, at sizes below
// and above the parallel threshold. This path backs DataFrame/sort and Expr/rank
// but had no test before.
func TestArgsortParallelMatchesSequentialAndStable(t *testing.T) {
	for _, n := range []int{1, 2, 1000, parallelMergeThreshold - 1, parallelMergeThreshold + 1, 300000} {
		floats := radixTestFloats(n)
		wantF := stableRefFloat(floats)
		if got := ArgsortFloat64Parallel(floats); !equalInts(got, wantF) {
			t.Fatalf("n=%d: float parallel argsort != stable sort", n)
		}
		if got := ArgsortFloat64(floats); !equalInts(got, wantF) {
			t.Fatalf("n=%d: float sequential argsort != stable sort", n)
		}

		ints := radixTestInts(n)
		wantI := stableRefInt(ints)
		if got := ArgsortInt64Parallel(ints); !equalInts(got, wantI) {
			t.Fatalf("n=%d: int parallel argsort != stable sort", n)
		}
		if got := ArgsortInt64(ints); !equalInts(got, wantI) {
			t.Fatalf("n=%d: int sequential argsort != stable sort", n)
		}
	}
}

// TestArgsortParallelIndependentOfWorkerCount checks the permutation does not
// depend on how many ranges the input was split into — the property the
// worker cap (maxRadixMergeWorkers) must not be able to change.
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

// TestRadixWorkersRespectsCap checks the range count stays within the measured
// cap however many cores are available, and never drops below one.
func TestRadixWorkersRespectsCap(t *testing.T) {
	original := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(original)

	for _, procs := range []int{1, 2, 6, 12, 64} {
		runtime.GOMAXPROCS(procs)
		got := radixWorkers()
		if got < 1 {
			t.Fatalf("GOMAXPROCS=%d: radixWorkers()=%d, want >=1", procs, got)
		}
		if got > maxRadixMergeWorkers {
			t.Fatalf("GOMAXPROCS=%d: radixWorkers()=%d, want <=%d", procs, got, maxRadixMergeWorkers)
		}
		if want := min(procs, maxRadixMergeWorkers); got != want {
			t.Fatalf("GOMAXPROCS=%d: radixWorkers()=%d, want %d", procs, got, want)
		}
	}
}
