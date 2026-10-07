package frame

import (
	"math"
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/expr/evalbatch"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// nullablePredicateFrame has n rows of Float64 columns "a" (null on every 5th
// row) and "b" (null on every 7th row) and a value column "v" holding the row
// number; withFlag adds a Boolean "flag" null on every 3rd row.
func nullablePredicateFrame(t *testing.T, n int, withFlag bool) DataFrame {
	t.Helper()
	a, b, v := make([]float64, n), make([]float64, n), make([]float64, n)
	aNulls, bNulls := make([]bool, n), make([]bool, n)
	flag, flagNulls := make([]bool, n), make([]bool, n)
	for i := range n {
		a[i] = float64(i % 4)
		b[i] = float64(i%10) / 10
		v[i] = float64(i)
		aNulls[i] = i%5 == 0
		bNulls[i] = i%7 == 0
		flag[i] = i%2 == 0
		flagNulls[i] = i%3 == 0
	}
	cols := []series.Series{
		series.FromFloat64("a", a, aNulls),
		series.FromFloat64("b", b, bNulls),
		series.FromFloat64("v", v, nil),
	}
	if withFlag {
		cols = append(cols, series.FromColumn("flag", chunk.NewBool(flag, flagNulls)))
	}
	df, err := New(NewInput{Series: cols})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return df
}

// Each predicate is null on some rows: eq on a null operand is null, and
// null OR false, null AND true and NOT null stay null.
var nullablePredicates = map[string]expr.Expr{
	"eq or gt":          expr.Col("a").Eq(expr.Lit(1.0)).Or(expr.Col("b").Gt(expr.Lit(0.5))),
	"cast eq and gt":    expr.Col("a").Cast(dtypes.Int64).Eq(expr.Lit(int64(2))).And(expr.Col("b").Lt(expr.Lit(0.9))),
	"is_in or eq":       expr.Col("a").IsIn(expr.Lit([]float64{3})).Or(expr.Col("b").Eq(expr.Lit(0.2))),
	"gt and bool":       expr.Col("b").Gt(expr.Lit(0.1)).And(expr.Col("flag")),
	"not nullable bool": expr.Col("flag").Not(),
	"nullable bool":     expr.Col("flag"),
}

func TestFilterNullPredicateStaysOnBatchPath(t *testing.T) {
	for _, n := range []int{1000, 100_000} {
		df := nullablePredicateFrame(t, n, true)
		for name, pred := range nullablePredicates {
			plan, ok := evalbatch.Compile(pred)
			if !ok {
				t.Fatalf("Compile(%s) = not supported", name)
			}
			t.Setenv("GOPOLARS_TYPED_STORAGE", "1")
			if _, ok := df.filterMask(plan, df.chunkColumns()); !ok {
				t.Errorf("filterMask(%s) on %d rows fell back to the row-wise path", name, n)
			}
			batch, err := df.Filter(pred)
			if err != nil {
				t.Fatalf("Filter(%s): %v", name, err)
			}
			t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
			rowWise, err := df.Filter(pred)
			if err != nil {
				t.Fatalf("row-wise Filter(%s): %v", name, err)
			}
			got, _ := batch.Series("v")
			want, _ := rowWise.Series("v")
			gotV, _ := got.Column().Float64s()
			wantV, _ := want.Column().Float64s()
			if !slices.Equal(gotV, wantV) {
				t.Errorf("Filter(%s) on %d rows kept %d rows, want %d (row-wise)", name, n, len(gotV), len(wantV))
			}
		}
	}
}

func TestFilterAggregateNullPredicate(t *testing.T) {
	for _, n := range []int{1000, 100_000} {
		df := nullablePredicateFrame(t, n, false)
		for _, name := range []string{"eq or gt", "cast eq and gt", "is_in or eq"} {
			pred := nullablePredicates[name]
			t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
			rowWise, err := df.Filter(pred)
			if err != nil {
				t.Fatalf("row-wise Filter(%s): %v", name, err)
			}
			kept, _ := rowWise.Series("v")
			keptV, _ := kept.Column().Float64s()
			var want float64
			for _, x := range keptV {
				want += x
			}
			t.Setenv("GOPOLARS_TYPED_STORAGE", "1")
			agg, ok, err := df.FilterAggregate(pred, "sum", nil)
			if err != nil || !ok {
				t.Errorf("FilterAggregate(%s, sum) on %d rows ok = %v, err = %v, want the fused path", name, n, ok, err)
			} else if s, _ := agg.Series("v"); math.Abs(s.Value(0).(float64)-want) > 1e-6*want {
				t.Errorf("FilterAggregate(%s, sum v) on %d rows = %v, want %v", name, n, s.Value(0), want)
			}
			direct, err := df.FilterAggregateDirect(pred, "sum", []string{"v"})
			if err != nil {
				t.Errorf("FilterAggregateDirect(%s, sum) on %d rows error = %v, want nil", name, n, err)
			} else if math.Abs(direct["v"]-want) > 1e-6*want {
				t.Errorf("FilterAggregateDirect(%s, sum v) on %d rows = %v, want %v", name, n, direct["v"], want)
			}
		}
	}
}
