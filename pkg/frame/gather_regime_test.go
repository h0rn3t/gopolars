package frame

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// buildRegimeFixture builds a four-column frame: "a" float64 null-free (the
// predicate column — a null predicate result makes the batch path decline, which
// would bypass the regime under test), "b" string with nulls, "c" int64
// null-free, "d" float64 with nulls. Four columns is what makes
// len(columns) == GOMAXPROCS(4) reachable in the tests below.
func buildRegimeFixture(t *testing.T, n int) DataFrame {
	t.Helper()
	av := make([]any, n)
	bv := make([]any, n)
	cv := make([]any, n)
	dv := make([]any, n)
	labels := []string{"kyiv", "lviv", "odesa", "kharkiv", "dnipro"}
	for i := range n {
		av[i] = float64(i % 100)
		if i%5 == 0 {
			bv[i] = nil
		} else {
			bv[i] = labels[i%len(labels)]
		}
		cv[i] = int64(i)
		if i%7 == 0 {
			dv[i] = nil
		} else {
			dv[i] = float64(i) / 2
		}
	}
	cols := make([]series.Series, 0, 4)
	for _, spec := range []struct {
		name   string
		dtype  dtypes.DataType
		values []any
	}{
		{"a", dtypes.Float64, av},
		{"b", dtypes.String, bv},
		{"c", dtypes.Int64, cv},
		{"d", dtypes.Float64, dv},
	} {
		s, err := series.New(spec.name, spec.dtype, spec.values)
		if err != nil {
			t.Fatalf("series %s: %v", spec.name, err)
		}
		cols = append(cols, s)
	}
	df, err := New(NewInput{Series: cols})
	if err != nil {
		t.Fatalf("new df: %v", err)
	}
	return df
}

// TestFilterEqualsSequentialAtColumnsEqualWorkers pins the regime boundary: with
// exactly as many columns as workers the fused row-range wave now runs (before,
// this fell into the column-per-worker gather), and its result must stay
// element-wise identical to the sequential path across selectivities and a
// height that is not a multiple of 64.
func TestFilterEqualsSequentialAtColumnsEqualWorkers(t *testing.T) {
	orig := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(orig)

	sizes := []int{parallelFilterThreshold + 1, 65_573}
	// Column "a" spans [0,100), so these bounds give ~0%, ~1%, ~50%, ~99%, 100%.
	bounds := []float64{100, 98, 49, 0, -1}

	for _, n := range sizes {
		df := buildRegimeFixture(t, n)
		if len(df.order) != 4 {
			t.Fatalf("fixture must have 4 columns, got %d", len(df.order))
		}
		for _, bound := range bounds {
			t.Run(fmt.Sprintf("n=%d/a_gt_%g", n, bound), func(t *testing.T) {
				pred := expr.Col("a").Gt(expr.Lit(bound))

				runtime.GOMAXPROCS(1)
				seqDF, err := df.Filter(pred)
				if err != nil {
					t.Fatalf("sequential filter: %v", err)
				}

				// len(columns) == workers: the case this change routes to the wave.
				runtime.GOMAXPROCS(len(df.order))
				parDF, err := df.Filter(pred)
				if err != nil {
					t.Fatalf("parallel filter: %v", err)
				}

				assertFramesEqual(t, parDF, seqDF)
				assertRegimeFilterOracle(t, df, parDF, bound)
			})
		}
	}
}

// assertRegimeFilterOracle recomputes the survivors of `a > bound` with a
// trivial sequential loop and checks every column of got against them, so each
// selectivity is verified absolutely and not only against the sequential path.
func assertRegimeFilterOracle(t *testing.T, src, got DataFrame, bound float64) {
	t.Helper()
	keep := make([]int, 0, src.Height())
	srcA := src.cols["a"]
	for i := range src.Height() {
		if srcA.Value(i).(float64) > bound {
			keep = append(keep, i)
		}
	}
	if got.Height() != len(keep) {
		t.Fatalf("oracle height: got=%d want=%d", got.Height(), len(keep))
	}
	for _, name := range src.order {
		srcCol, gotCol := src.cols[name], got.cols[name]
		for j, srcIdx := range keep {
			if srcCol.IsNull(srcIdx) != gotCol.IsNull(j) {
				t.Fatalf("col=%s row=%d: null=%v want=%v", name, j, gotCol.IsNull(j), srcCol.IsNull(srcIdx))
			}
			if !srcCol.IsNull(srcIdx) && gotCol.Value(j) != srcCol.Value(srcIdx) {
				t.Fatalf("col=%s row=%d: %v want=%v", name, j, gotCol.Value(j), srcCol.Value(srcIdx))
			}
		}
	}
}

// TestDropNullsEqualsRefAtColumnsEqualWorkers is the DropNulls half of the same
// boundary: four columns, GOMAXPROCS 4, checked against the sequential
// keep-set oracle for sparse, dense and total null coverage.
func TestDropNullsEqualsRefAtColumnsEqualWorkers(t *testing.T) {
	orig := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(orig)

	for _, n := range []int{parallelFilterThreshold + 1, 65_573} {
		df := buildRegimeFixture(t, n)
		runtime.GOMAXPROCS(len(df.order))
		for _, targets := range [][]string{{"b"}, {"d"}, {"b", "d"}, {"c"}, nil} {
			t.Run(fmt.Sprintf("n=%d/targets=%v", n, targets), func(t *testing.T) {
				assertDropNullsMatchesRef(t, df, targets)
			})
		}
	}

	// A target that is null in every row drops every row: the wave must produce
	// an empty frame, not a partially written one. Still four columns, so the
	// regime is unchanged.
	n := parallelFilterThreshold + 1
	base := buildRegimeFixture(t, n)
	allNull, err := series.New("d", dtypes.Float64, make([]any, n))
	if err != nil {
		t.Fatalf("series d: %v", err)
	}
	withAllNull, err := New(NewInput{Series: []series.Series{
		base.cols["a"], base.cols["b"], base.cols["c"], allNull,
	}})
	if err != nil {
		t.Fatalf("new df: %v", err)
	}
	runtime.GOMAXPROCS(len(withAllNull.order))
	if got := withAllNull.DropNulls("d"); got.Height() != 0 {
		t.Fatalf("all-null target: height=%d, want 0", got.Height())
	}
}
