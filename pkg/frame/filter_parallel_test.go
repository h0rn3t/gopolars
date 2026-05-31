package frame

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// buildFilterFixture builds a two-column float64 DataFrame of height n with
// deterministic values. Column "a" (the predicate column) and the payload
// column "b" both carry interspersed nulls so the equivalence check exercises
// null handling in both the predicate and the gathered output.
func buildFilterFixture(t *testing.T, n int) DataFrame {
	t.Helper()
	av := make([]any, n)
	bv := make([]any, n)
	for i := range n {
		if i%7 == 0 {
			av[i] = nil // null predicate operand -> row dropped
		} else {
			av[i] = float64((i % 101) - 50) // spans roughly [-50, 50]
		}
		if i%5 == 0 {
			bv[i] = nil // null payload -> must be preserved on surviving rows
		} else {
			bv[i] = float64(i)
		}
	}
	a, err := series.New("a", dtypes.Float64, av)
	if err != nil {
		t.Fatalf("series a: %v", err)
	}
	b, err := series.New("b", dtypes.Float64, bv)
	if err != nil {
		t.Fatalf("series b: %v", err)
	}
	df, err := New(NewInput{Series: []series.Series{a, b}})
	if err != nil {
		t.Fatalf("new df: %v", err)
	}
	return df
}

// TestFilterParallelMatchesSequential asserts the parallel batch filter (at or
// above parallelFilterThreshold) produces a result element-wise identical to
// the single-threaded path — same surviving rows, same order, same null
// handling in every column — across sizes that straddle the threshold. It also
// checks both results against a raw sequential oracle for absolute correctness.
func TestFilterParallelMatchesSequential(t *testing.T) {
	sizes := []int{
		1000,
		parallelFilterThreshold - 1,
		parallelFilterThreshold,
		parallelFilterThreshold + 1,
		100_000,
	}
	orig := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(orig)

	pred := expr.Col("a").Gt(expr.Lit(0.0))

	for _, n := range sizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			df := buildFilterFixture(t, n)

			// Force the sequential path (workers<=1 always stays single-threaded).
			runtime.GOMAXPROCS(1)
			seqDF, err := df.Filter(pred)
			if err != nil {
				t.Fatalf("sequential filter: %v", err)
			}

			// Force the parallel path (>= threshold fans out across workers).
			runtime.GOMAXPROCS(max(2, orig))
			parDF, err := df.Filter(pred)
			if err != nil {
				t.Fatalf("parallel filter: %v", err)
			}

			assertFramesEqual(t, seqDF, parDF)
			assertMatchesFilterOracle(t, df, parDF, pred)
		})
	}
}

// assertMatchesFilterOracle recomputes the expected survivors of `Col("a") > 0`
// with a trivial sequential loop and verifies got matches it exactly, including
// the preserved nulls of the payload column and the surviving row order.
func assertMatchesFilterOracle(t *testing.T, src, got DataFrame, _ expr.Expr) {
	t.Helper()
	srcA, _ := src.Series("a")
	srcB, _ := src.Series("b")
	keep := make([]int, 0, src.Height())
	for i := range src.Height() {
		if srcA.IsNull(i) {
			continue
		}
		if srcA.Value(i).(float64) > 0 {
			keep = append(keep, i)
		}
	}
	if got.Height() != len(keep) {
		t.Fatalf("oracle height: got=%d want=%d", got.Height(), len(keep))
	}
	gotA, _ := got.Series("a")
	gotB, _ := got.Series("b")
	for j, srcIdx := range keep {
		if gotA.Value(j) != srcA.Value(srcIdx) {
			t.Fatalf("oracle row %d: a got=%v want=%v", j, gotA.Value(j), srcA.Value(srcIdx))
		}
		if srcB.IsNull(srcIdx) != gotB.IsNull(j) {
			t.Fatalf("oracle row %d: b null got=%v want=%v", j, gotB.IsNull(j), srcB.IsNull(srcIdx))
		}
		if !srcB.IsNull(srcIdx) && gotB.Value(j) != srcB.Value(srcIdx) {
			t.Fatalf("oracle row %d: b got=%v want=%v", j, gotB.Value(j), srcB.Value(srcIdx))
		}
	}
}
