package frame

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// buildNullFixture builds a 3-column frame of height n. Column "b" is null on
// every nullEvery-th row (so nullEvery=10 is sparse ~10%, nullEvery=2 is dense
// ~50%); "a" and "c" are null-free payload columns. drop_nulls drops exactly the
// rows where "b" is null, gathering every column.
func buildNullFixture(t *testing.T, n, nullEvery int) DataFrame {
	t.Helper()
	a := make([]float64, n)
	b := make([]float64, n)
	c := make([]int64, n)
	bn := make([]bool, n)
	for i := range n {
		a[i] = float64(i)
		b[i] = float64(i) * 2
		c[i] = int64(i)
		if i%nullEvery == 0 {
			bn[i] = true
		}
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromFloat64("a", a, nil),
		series.FromFloat64("b", b, bn),
		series.FromInt64("c", c, nil),
	}})
	if err != nil {
		t.Fatalf("buildNullFixture New: %v", err)
	}
	return df
}

// seqVsParDF runs op with GOMAXPROCS forced to 1 (sequential gather) and to >= 2
// (parallel gather), returning both results.
func seqVsParDF(t *testing.T, op func() DataFrame) (seq, par DataFrame) {
	t.Helper()
	orig := runtime.GOMAXPROCS(1)
	seq = op()
	runtime.GOMAXPROCS(max(2, orig))
	par = op()
	runtime.GOMAXPROCS(orig)
	return seq, par
}

// TestDropNullsParallelMatchesSequential verifies drop_nulls' parallel column
// gather equals the sequential gather for both sparse and dense null densities
// at a size above the gather threshold, under -race. The sparse case keeps far
// more than parallelFilterThreshold rows, so the parallel gather branch engages.
func TestDropNullsParallelMatchesSequential(t *testing.T) {
	const n = 1 << 16                        // 65536: above parallelFilterThreshold (32768)
	for _, nullEvery := range []int{10, 2} { // ~10% sparse, ~50% dense
		t.Run(fmt.Sprintf("nullEvery=%d", nullEvery), func(t *testing.T) {
			df := buildNullFixture(t, n, nullEvery)
			seq, par := seqVsParDF(t, func() DataFrame { return df.DropNulls() })
			assertFramesEqual(t, seq, par)

			// The kept count must clear the parallel-gather gate so the sparse case
			// exercises the parallel path (not silently fall back to sequential).
			if nullEvery == 10 && par.Height() < parallelFilterThreshold {
				t.Fatalf("sparse drop_nulls kept %d rows, below the parallel gather threshold %d — parallel path would not engage",
					par.Height(), parallelFilterThreshold)
			}
		})
	}
}

// TestDropNullsNoNullsReturnsSharedFrame verifies the no-null fast path returns a
// frame that SHARES the input columns (same backing pointers) rather than
// materializing a copy.
func TestDropNullsNoNullsReturnsSharedFrame(t *testing.T) {
	df := buildNullFixture(t, 1000, 1) // built with nulls...
	// ...but drop_nulls on a column with no nulls in scope hits the fast path:
	// scope only "a" and "c", which are null-free.
	out := df.DropNulls("a", "c")
	if out.Height() != df.Height() {
		t.Fatalf("no-null drop_nulls changed height: got %d, want %d", out.Height(), df.Height())
	}
	for _, name := range df.order {
		if out.cols[name].Column() != df.cols[name].Column() {
			t.Fatalf("no-null drop_nulls did not share column %q (expected shared backing, got a copy)", name)
		}
	}
}

// TestFilterSelectivityExtremes verifies the parallel filter gather equals the
// sequential one at the empty (zero survivors) and full (all survivors)
// selectivity extremes, where the gather sizes differ most from the half case.
func TestFilterSelectivityExtremes(t *testing.T) {
	const n = 1 << 16
	a := make([]float64, n)
	b := make([]float64, n)
	for i := range n {
		a[i] = float64(i)
		b[i] = float64(i) * 3
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromFloat64("a", a, nil),
		series.FromFloat64("b", b, nil),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := []struct {
		name string
		pred expr.Expr
		want int
	}{
		{"empty", expr.Col("a").Gt(expr.Lit(1e18)), 0},
		{"full", expr.Col("a").Gt(expr.Lit(-1.0)), n},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seq, par := seqVsParDF(t, func() DataFrame {
				out, err := df.Filter(tc.pred)
				if err != nil {
					t.Fatalf("filter: %v", err)
				}
				return out
			})
			assertFramesEqual(t, seq, par)
			if par.Height() != tc.want {
				t.Fatalf("%s selectivity: height %d, want %d", tc.name, par.Height(), tc.want)
			}
		})
	}
}

// TestUniqueParallelMatchesSequentialLarge verifies unique() over a large frame
// with heavily repeated keys produces the same distinct rows and encounter order
// through the parallel gather as through the sequential one, under -race.
func TestUniqueParallelMatchesSequentialLarge(t *testing.T) {
	const n = 1 << 16
	const distinct = 4096 // many repeats per key
	k := make([]int64, n)
	tag := make([]int64, n)
	for i := range n {
		k[i] = int64(i % distinct)
		tag[i] = int64(i)
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromInt64("k", k, nil),
		series.FromInt64("tag", tag, nil),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	orig := runtime.GOMAXPROCS(1)
	seq, err := df.Unique("k")
	if err != nil {
		runtime.GOMAXPROCS(orig)
		t.Fatalf("sequential unique: %v", err)
	}
	runtime.GOMAXPROCS(max(2, orig))
	par, err := df.Unique("k")
	runtime.GOMAXPROCS(orig)
	if err != nil {
		t.Fatalf("parallel unique: %v", err)
	}

	assertFramesEqual(t, seq, par)
	if par.Height() != distinct {
		t.Fatalf("unique height %d, want %d distinct keys", par.Height(), distinct)
	}
	// First-seen encounter order: distinct key g first appears at row g, carrying
	// tag == g.
	tagCol := par.cols["tag"]
	for g := range distinct {
		if got := tagCol.Value(g); got != int64(g) {
			t.Fatalf("unique row %d tag = %v, want %d (encounter order broken)", g, got, g)
		}
	}
}
