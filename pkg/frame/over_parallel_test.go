package frame

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

// buildOverFrame builds an n-row frame with a string partition key "g" of the
// given cardinality, a null-free float "v", and every nullEvery-th "v" null when
// nullEvery > 0.
func buildOverFrame(t testing.TB, n, cardinality, nullEvery int) DataFrame {
	t.Helper()
	g := make([]any, n)
	v := make([]any, n)
	for i := 0; i < n; i++ {
		g[i] = fmt.Sprintf("g%d", i%cardinality)
		if nullEvery > 0 && i%nullEvery == 0 {
			v[i] = nil
		} else {
			v[i] = float64(i%97) - 20
		}
	}
	df, err := FromAnyColumns(FromAnyColumnsInput{Columns: []SeriesInput{
		{Name: "g", Values: g},
		{Name: "v", Values: v},
	}})
	if err != nil {
		t.Fatalf("FromAnyColumns: %v", err)
	}
	return df
}

// overCumSum evaluates cum_sum().over("g") and returns the resulting column's
// values, so results computed under different worker counts can be compared.
func overCumSum(t testing.TB, df DataFrame) []any {
	t.Helper()
	out, err := df.Select(expr.Col("v").CumSum().Over("g").Alias("cs"))
	if err != nil {
		t.Fatalf("over cum_sum: %v", err)
	}
	col, ok := out.Series("cs")
	if !ok {
		t.Fatalf("missing cs column")
	}
	vals := make([]any, col.Len())
	for i := range vals {
		vals[i] = col.Value(i)
	}
	return vals
}

// TestOverResultIndependentOfWorkerCount checks the windowed aggregation is
// unaffected by how the sharded group build split the rows.
func TestOverResultIndependentOfWorkerCount(t *testing.T) {
	original := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(original)

	for _, tc := range []struct {
		name        string
		n           int
		cardinality int
		nullEvery   int
	}{
		{"small/low-cardinality", 500, 3, 0},
		{"large/low-cardinality", 40000, 5, 0},
		{"large/with-nulls", 40000, 5, 11},
		{"large/high-cardinality", 40000, 20000, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			df := buildOverFrame(t, tc.n, tc.cardinality, tc.nullEvery)
			var reference []any
			for _, procs := range []int{1, 2, 3, 8} {
				runtime.GOMAXPROCS(procs)
				got := overCumSum(t, df)
				if reference == nil {
					reference = got
					continue
				}
				if len(got) != len(reference) {
					t.Fatalf("GOMAXPROCS=%d: length %d, want %d", procs, len(got), len(reference))
				}
				for i := range got {
					if got[i] != reference[i] {
						t.Fatalf("GOMAXPROCS=%d: row %d = %v, want %v", procs, i, got[i], reference[i])
					}
				}
			}
		})
	}
}

// TestOverDoesNotLeakAcrossPartitions checks the running sum of a partition
// counts only that partition's earlier rows, with the partitions interleaved so
// that any cross-partition carry would show up.
func TestOverDoesNotLeakAcrossPartitions(t *testing.T) {
	const n = 40000
	const cardinality = 4
	df := buildOverFrame(t, n, cardinality, 0)
	got := overCumSum(t, df)

	// Reference: a plain per-partition running sum over the base column, which is
	// itself the global cum_sum the Over target produces.
	base, err := df.Select(expr.Col("v").CumSum().Alias("b"))
	if err != nil {
		t.Fatalf("base cum_sum: %v", err)
	}
	baseCol, ok := base.Series("b")
	if !ok {
		t.Fatalf("missing base column")
	}
	sums := make([]float64, cardinality)
	for i := 0; i < n; i++ {
		part := i % cardinality
		sums[part] += baseCol.Value(i).(float64)
		if got[i].(float64) != sums[part] {
			t.Fatalf("row %d (partition %d): got %v, want %v", i, part, got[i], sums[part])
		}
	}
}
