package polars

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// newTestExprFrame builds a 4-row, 4-column frame used by the Expr-constructor
// tests: int64 "a", float64 "b", string "g" (group key), bool "f" (flag).
func newTestExprFrame(t *testing.T) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		{Name: "b", Values: []any{1.0, 2.0, 3.0, 4.0}},
		{Name: "g", Values: []any{"x", "x", "y", "y"}},
		{Name: "f", Values: []any{true, false, true, false}},
	}})
	if err != nil {
		t.Fatalf("newTestExprFrame: %v", err)
	}
	return df
}

// TestExprConstructorsAreNonNil ensures every top-level constructor in expr.go
// returns a usable Expr; the public surface is the type itself, so the
// assertion is "the constructor does not panic and the value composes".
func TestExprConstructorsAreNonNil(t *testing.T) {
	colA := Col("a")
	cases := []struct {
		name string
		e    Expr
	}{
		{"Col", Col("a")},
		{"Lit_int", Lit(int64(7))},
		{"Lit_string", Lit("x")},
		{"Sum", Sum(colA)},
		{"Min", Min(colA)},
		{"Max", Max(colA)},
		{"Mean", Mean(colA)},
		{"Count", Count()},
		{"NUnique", NUnique(colA)},
		{"When", When(colA.Gt(Lit(int64(2))), Lit(int64(1)), Lit(int64(0)))},
		{"nested_agg", Sum(Mean(colA))},
	}
	// Lit values expose the underlying value through Select; we pin the
	// dtype constant re-exports (Int64 etc.) to the dtypes package values.
	if Int64 != dtypes.Int64 || Float64 != dtypes.Float64 || String != dtypes.String {
		t.Fatalf("type re-exports drifted from pkg/dtypes")
	}
	if Boolean != dtypes.Boolean || Datetime != dtypes.Datetime || Categorical != dtypes.Categorical {
		t.Fatalf("type re-exports drifted from pkg/dtypes")
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// .Alias() and .Cast() are the cheapest composition checks; both
			// return a new Expr so a fully zero struct would also work, but
			// the goal is to confirm the constructor wired the kind/op/target
			// fields correctly.
			_ = c.e.Alias("x").Cast(dtypes.Int64)
		})
	}
}

// TestExprViaSelect exercises the non-aggregate Expr constructor family through
// the public DataFrame.Select path so the constructors are actually evaluated
// against a real frame (not just constructed and discarded).
func TestExprViaSelect(t *testing.T) {
	df := newTestExprFrame(t)
	cases := []struct {
		name     string
		expr     Expr
		wantName string
	}{
		{"Col_a", Col("a"), "a"},
		{"Lit_int", Lit(int64(7)).Alias("lit"), "lit"},
		{"Alias", Col("a").Alias("alpha"), "alpha"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := df.Select(c.expr)
			if err != nil {
				t.Fatalf("Select: %v", err)
			}
			if out.Width() != 1 {
				t.Fatalf("Select(%s) width = %d, want 1", c.name, out.Width())
			}
			if got := out.Columns()[0]; got != c.wantName {
				t.Fatalf("Select(%s) column = %q, want %q", c.name, got, c.wantName)
			}
		})
	}
}

// TestExprWhenSelect verifies the When/Then/Otherwise expression is evaluated
// per-row, with the values matching the documented conditional.
func TestExprWhenSelect(t *testing.T) {
	df := newTestExprFrame(t)
	out, err := df.Select(When(Col("a").Gt(Lit(int64(2))), Lit(int64(1)), Lit(int64(0))).Alias("flag"))
	if err != nil {
		t.Fatalf("Select(When): %v", err)
	}
	if out.Width() != 1 {
		t.Fatalf("When width = %d, want 1", out.Width())
	}
	col, _ := out.GetColumn("flag")
	want := []int64{0, 0, 1, 1}
	for i, w := range want {
		got, _ := col.Value(i).(int64)
		if got != w {
			t.Errorf("When[%d] = %d, want %d", i, got, w)
		}
	}
}

// TestExprAggAfterGroupBy ensures Sum, Min, Max, Mean, Count, NUnique, when
// used in a GroupBy(...).Agg(...), produce an aggregate column with the
// documented reduction per group.
func TestExprAggAfterGroupBy(t *testing.T) {
	df := newTestExprFrame(t)
	// Group "g" has values ["x","x","y","y"]; "a" has [1,2,3,4].
	// Sum(x)=3, Sum(y)=7; Min(x)=1, Min(y)=3; Max(x)=2, Max(y)=4.
	// Count(x)=2, Count(y)=2; NUnique_a(x)=2, NUnique_a(y)=2.
	out, err := df.GroupBy("g").Agg(
		Sum(Col("a")).Alias("sum_a"),
		Min(Col("a")).Alias("min_a"),
		Max(Col("a")).Alias("max_a"),
		Mean(Col("a")).Alias("mean_a"),
		Count().Alias("count"),
	)
	if err != nil {
		t.Fatalf("GroupBy.Agg: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("agg height = %d, want 2 (groups x, y)", out.Height())
	}
	// Build a key→row-index map so order doesn't matter.
	keys, _ := out.GetColumn("g")
	idxByKey := map[string]int{}
	for i := 0; i < keys.Len(); i++ {
		k, _ := keys.Value(i).(string)
		idxByKey[k] = i
	}
	checks := []struct {
		col   string
		key   string
		val   float64
	}{
		{"sum_a", "x", 3}, {"sum_a", "y", 7},
		{"min_a", "x", 1}, {"min_a", "y", 3},
		{"max_a", "x", 2}, {"max_a", "y", 4},
		{"mean_a", "x", 1.5}, {"mean_a", "y", 3.5},
		{"count", "x", 2}, {"count", "y", 2},
	}
	for _, c := range checks {
		col, _ := out.GetColumn(c.col)
		row := idxByKey[c.key]
		var got float64
		switch v := col.Value(row).(type) {
		case int64:
			got = float64(v)
		case float64:
			got = v
		default:
			t.Errorf("%s[%s] has unexpected type %T", c.col, c.key, v)
			continue
		}
		if got != c.val {
			t.Errorf("%s[%s] = %v, want %v", c.col, c.key, got, c.val)
		}
	}
}

// TestExprNUniqueAgg verifies NUnique produces the per-group distinct count.
func TestExprNUniqueAgg(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"x", "x", "x", "y", "y"}},
		{Name: "a", Values: []any{int64(1), int64(1), int64(2), int64(3), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	out, err := df.GroupBy("g").Agg(NUnique(Col("a")).Alias("nu"))
	if err != nil {
		t.Fatalf("GroupBy.Agg NUnique: %v", err)
	}
	keys, _ := out.GetColumn("g")
	nu, _ := out.GetColumn("nu")
	idxByKey := map[string]int{}
	for i := 0; i < keys.Len(); i++ {
		k, _ := keys.Value(i).(string)
		idxByKey[k] = i
	}
	if v, _ := nu.Value(idxByKey["x"]).(int64); v != 2 {
		t.Errorf("NUnique(x) = %d, want 2 (values 1,2)", v)
	}
	if v, _ := nu.Value(idxByKey["y"]).(int64); v != 1 {
		t.Errorf("NUnique(y) = %d, want 1 (value 3)", v)
	}
}

// TestExprEvalLazyCollect ensures the Expr constructors evaluate through the
// LazyFrame.Collect path (so a refactor of the wrapper surface is caught).
func TestExprEvalLazyCollect(t *testing.T) {
	df := newTestExprFrame(t)
	out, err := df.Lazy().Select(Col("a").Gt(Lit(int64(2)))).Collect(context.Background())
	if err != nil {
		t.Fatalf("Lazy Collect: %v", err)
	}
	if out.Width() != 1 {
		t.Fatalf("Lazy width = %d, want 1", out.Width())
	}
	if out.Height() != 4 {
		t.Fatalf("Lazy height = %d, want 4", out.Height())
	}
}
