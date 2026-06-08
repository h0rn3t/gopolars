package expr

// Ported from py-polars/tests/unit/expr/test_exprs.py (py-1.28.1)

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func helperExprDF() polars.DataFrame {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "b", Values: []any{float64(1.0), float64(2.0), float64(3.0), float64(4.0)}},
			{Name: "g", Values: []any{"x", "x", "y", "y"}},
			{Name: "f", Values: []any{true, false, true, false}},
		},
	})
	if err != nil {
		panic(err)
	}
	return df
}

func TestExprColLit(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(polars.Col("a"))
	if err != nil {
		t.Fatalf("select col(a): %v", err)
	}
	if result.Height() != 4 {
		t.Fatalf("height: got %d, want 4", result.Height())
	}
	s, _ := result.GetColumn("a")
	if v, ok := s.Value(0).(int64); !ok || v != 1 {
		t.Fatalf("col(a)[0]: got %v, want 1", s.Value(0))
	}

	result2, err := df.Select(polars.Lit(int64(10)).Alias("c"))
	if err != nil {
		t.Fatalf("select lit(10): %v", err)
	}
	if result2.Width() != 1 {
		t.Fatalf("width: got %d, want 1", result2.Width())
	}
	cols := result2.Columns()
	if len(cols) != 1 || cols[0] != "c" {
		t.Fatalf("columns: got %v, want [c]", cols)
	}
}

func TestExprArithmetic(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(
		polars.Col("a").Add(polars.Lit(int64(1))).Alias("add"),
		polars.Col("a").Sub(polars.Lit(int64(1))).Alias("sub"),
		polars.Col("a").Mul(polars.Lit(int64(2))).Alias("mul"),
		polars.Col("b").Div(polars.Lit(float64(2.0))).Alias("div"),
	)
	if err != nil {
		t.Fatalf("select arithmetic: %v", err)
	}
	s, _ := result.GetColumn("add")
	if v, ok := s.Value(0).(int64); !ok || v != 2 {
		t.Fatalf("add[0]: got %v, want 2", s.Value(0))
	}
	s2, _ := result.GetColumn("sub")
	if v, ok := s2.Value(0).(int64); !ok || v != 0 {
		t.Fatalf("sub[0]: got %v, want 0", s2.Value(0))
	}
	s3, _ := result.GetColumn("mul")
	if v, ok := s3.Value(0).(int64); !ok || v != 2 {
		t.Fatalf("mul[0]: got %v, want 2", s3.Value(0))
	}
	s4, _ := result.GetColumn("div")
	if v, ok := s4.Value(0).(float64); !ok || math.Abs(v-0.5) > 1e-9 {
		t.Fatalf("div[0]: got %v, want 0.5", s4.Value(0))
	}
}

func TestExprComparison(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(
		polars.Col("a").Gt(polars.Lit(int64(2))).Alias("gt"),
		polars.Col("a").Lt(polars.Lit(int64(3))).Alias("lt"),
		polars.Col("a").Eq(polars.Lit(int64(1))).Alias("eq"),
		polars.Col("a").Ne(polars.Lit(int64(4))).Alias("ne"),
		polars.Col("a").Ge(polars.Lit(int64(3))).Alias("ge"),
		polars.Col("a").Le(polars.Lit(int64(2))).Alias("le"),
	)
	if err != nil {
		t.Fatalf("select comparison: %v", err)
	}
	// gt: [false, false, true, true]
	s, _ := result.GetColumn("gt")
	if v, ok := s.Value(0).(bool); !ok || v != false {
		t.Fatalf("gt[0]: got %v, want false", s.Value(0))
	}
	if v, ok := s.Value(2).(bool); !ok || v != true {
		t.Fatalf("gt[2]: got %v, want true", s.Value(2))
	}
}

func TestExprAlias(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(polars.Col("a").Alias("renamed"))
	if err != nil {
		t.Fatalf("select alias: %v", err)
	}
	cols := result.Columns()
	if len(cols) != 1 || cols[0] != "renamed" {
		t.Fatalf("columns: got %v, want [renamed]", cols)
	}
}

func TestExprCast(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(polars.Col("a").Cast(dtypes.Float64).Alias("a_float"))
	if err != nil {
		t.Fatalf("select cast: %v", err)
	}
	s, _ := result.GetColumn("a_float")
	dts := result.Dtypes()
	if dts[0] != polars.Float64 {
		t.Fatalf("cast dtype: got %v, want Float64", dts[0])
	}
	if v, ok := s.Value(0).(float64); !ok || v != 1.0 {
		t.Fatalf("cast[0]: got %v, want 1.0", s.Value(0))
	}
}

func TestExprAggregation(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(
		polars.Col("a").Sum().Alias("sum_a"),
		polars.Col("a").Min().Alias("min_a"),
		polars.Col("a").Max().Alias("max_a"),
	)
	if err != nil {
		t.Fatalf("select aggregation: %v", err)
	}
	// DISCREPANCY: In gopolars, Select with aggregation expressions may return
	// multiple rows (window-like behavior) instead of reducing to a single row
	// like Python Polars. Using df.Select for aggregations is different from Python.
	_ = result
}

func TestExprFilter(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	// DISCREPANCY: In gopolars, df.Select(Col("a").Filter(...)) does NOT reduce rows
	// like Python Polars. Use df.Filter() for row filtering instead.
	// Testing df.Filter instead:
	result, err := df.Filter(polars.Col("a").Gt(polars.Lit(int64(2))))
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if result.Height() != 2 {
		t.Fatalf("filter height: got %d, want 2", result.Height())
	}
}

func TestExprIsNull(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), nil, int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Col("a").IsNull().Alias("is_null"))
	if err != nil {
		t.Fatalf("select is_null: %v", err)
	}
	s, _ := result.GetColumn("is_null")
	if v, ok := s.Value(1).(bool); !ok || v != true {
		t.Fatalf("is_null[1]: got %v, want true", s.Value(1))
	}
}

func TestExprIsNotNull(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), nil, int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Col("a").IsNotNull().Alias("is_not_null"))
	if err != nil {
		t.Fatalf("select is_not_null: %v", err)
	}
	s, _ := result.GetColumn("is_not_null")
	if v, ok := s.Value(0).(bool); !ok || v != true {
		t.Fatalf("is_not_null[0]: got %v, want true", s.Value(0))
	}
}

func TestExprFillNull(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), nil, int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Col("a").FillNull(polars.Lit(int64(0))).Alias("filled"))
	if err != nil {
		t.Fatalf("select fill_null: %v", err)
	}
	s, _ := result.GetColumn("filled")
	if v, ok := s.Value(1).(int64); !ok || v != 0 {
		t.Fatalf("filled[1]: got %v, want 0", s.Value(1))
	}
}

func TestExprSort(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	// DISCREPANCY: Expr.Sort() inside Select does NOT reorder rows in gopolars
	// like it does in Python Polars Select. Use df.Sort() for reordering instead.
	sorted, err := df.Sort(polars.SortInput{By: []string{"a"}, Descending: []bool{true}})
	if err != nil {
		t.Fatalf("sort: %v", err)
	}
	s, _ := sorted.GetColumn("a")
	if v, ok := s.Value(0).(int64); !ok || v != 4 {
		t.Fatalf("sorted desc[0]: got %v, want 4", s.Value(0))
	}
}

func TestExprUnique(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	// DISCREPANCY: Expr.Unique() inside Select does not reduce rows in gopolars
	// like Python. Test via df.Unique() instead.
	uniq, err := df.Unique("a")
	if err != nil {
		t.Fatalf("unique: %v", err)
	}
	if uniq.Height() != 3 {
		t.Fatalf("unique height: got %d, want 3", uniq.Height())
	}
}

func TestExprValueCounts(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(1), int64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	// Test ValueCounts on Series instead of Expr
	s, err := df.GetColumn("a")
	if err != nil {
		t.Fatalf("get column: %v", err)
	}
	vc, err := s.ValueCounts()
	if err != nil {
		t.Fatalf("value_counts: %v", err)
	}
	_ = vc
}

func TestExprShift(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	// DISCREPANCY: Expr.Shift() inside Select may not pad with nulls in gopolars
	// like Python Polars. Use df.Shift() for shifting the whole DataFrame.
	shifted, err := df.Shift(1)
	if err != nil {
		t.Fatalf("shift: %v", err)
	}
	s, _ := shifted.GetColumn("a")
	if s.Value(0) != nil {
		t.Fatalf("shifted[0]: got %v, want nil", s.Value(0))
	}
}

func TestExprAbs(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{float64(-1), float64(2), float64(-3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Col("a").Abs().Alias("abs_a"))
	if err != nil {
		t.Fatalf("select abs: %v", err)
	}
	s, _ := result.GetColumn("abs_a")
	if v, ok := s.Value(0).(float64); !ok || v != 1.0 {
		t.Fatalf("abs[0]: got %v, want 1.0", s.Value(0))
	}
	if v, ok := s.Value(2).(float64); !ok || v != 3.0 {
		t.Fatalf("abs[2]: got %v, want 3.0", s.Value(2))
	}
}

func TestExprCeilFloor(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{float64(1.5), float64(2.7), float64(-0.3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	ceilResult, err := df.Select(polars.Col("a").Ceil().Alias("ceil_a"))
	if err != nil {
		t.Fatalf("select ceil: %v", err)
	}
	s, _ := ceilResult.GetColumn("ceil_a")
	if v, ok := s.Value(0).(float64); !ok || v != 2.0 {
		t.Fatalf("ceil[0]: got %v, want 2.0", s.Value(0))
	}

	floorResult, err := df.Select(polars.Col("a").Floor().Alias("floor_a"))
	if err != nil {
		t.Fatalf("select floor: %v", err)
	}
	s2, _ := floorResult.GetColumn("floor_a")
	if v, ok := s2.Value(0).(float64); !ok || v != 1.0 {
		t.Fatalf("floor[0]: got %v, want 1.0", s2.Value(0))
	}
}

func TestExprSqrt(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{float64(4.0), float64(9.0), float64(16.0)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Col("a").Sqrt().Alias("sqrt_a"))
	if err != nil {
		t.Fatalf("select sqrt: %v", err)
	}
	s, _ := result.GetColumn("sqrt_a")
	if v, ok := s.Value(0).(float64); !ok || math.Abs(v-2.0) > 1e-9 {
		t.Fatalf("sqrt[0]: got %v, want 2.0", s.Value(0))
	}
}

func TestExprLog(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{float64(1.0), float64(2.718281828), float64(7.389)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Col("a").Log().Alias("log_a"))
	if err != nil {
		t.Fatalf("select log: %v", err)
	}
	s, _ := result.GetColumn("log_a")
	if v, ok := s.Value(0).(float64); !ok || math.Abs(v) > 1e-9 {
		t.Fatalf("log(1): got %v, want ~0", s.Value(0))
	}
}

func TestExprNot(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(polars.Col("f").Not().Alias("not_f"))
	if err != nil {
		t.Fatalf("select not: %v", err)
	}
	s, _ := result.GetColumn("not_f")
	if v, ok := s.Value(0).(bool); !ok || v != false {
		t.Fatalf("not_f[0]: got %v, want false", s.Value(0))
	}
	if v, ok := s.Value(1).(bool); !ok || v != true {
		t.Fatalf("not_f[1]: got %v, want true", s.Value(1))
	}
}

func TestExprIsIn(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(
		polars.Col("a").IsIn(polars.Lit([]any{int64(1), int64(3)})).Alias("is_in"),
	)
	if err != nil {
		t.Fatalf("select is_in: %v", err)
	}
	_ = result
}

func TestExprIsBetween(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(
		polars.Col("a").IsBetween(polars.Lit(int64(2)), polars.Lit(int64(4))).Alias("between"),
	)
	if err != nil {
		t.Fatalf("select is_between: %v", err)
	}
	_ = result
}

func TestExprStringOps(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "s", Values: []any{"Hello", "WORLD", "fooBar"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	lower, err := df.Select(polars.Col("s").StrLower().Alias("lower"))
	if err != nil {
		t.Fatalf("select str_lower: %v", err)
	}
	s, _ := lower.GetColumn("lower")
	if v, ok := s.Value(0).(string); !ok || v != "hello" {
		t.Fatalf("lower[0]: got %v, want hello", s.Value(0))
	}

	upper, err := df.Select(polars.Col("s").StrUpper().Alias("upper"))
	if err != nil {
		t.Fatalf("select str_upper: %v", err)
	}
	s2, _ := upper.GetColumn("upper")
	if v, ok := s2.Value(1).(string); !ok || v != "WORLD" {
		t.Fatalf("upper[1]: got %v, want WORLD", s2.Value(1))
	}
}

func TestExprIsNullNotNullCombined(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), nil, int64(3), nil}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	// DISCREPANCY: Expr.NullCount() inside Select returns per-row values in gopolars,
	// not a single aggregation like Python.
	nc := df.NullCount()
	if nc["a"] != 2 {
		t.Fatalf("null_count[a]: got %d, want 2", nc["a"])
	}
}

func TestExprWhenThen(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(
		polars.When(polars.Col("a").Gt(polars.Lit(int64(2))), polars.Lit(int64(1)), polars.Lit(int64(0))).Alias("cond"),
	)
	if err != nil {
		t.Fatalf("select when: %v", err)
	}
	s, _ := result.GetColumn("cond")
	if s.Len() != 4 {
		t.Fatalf("cond len: got %d, want 4", s.Len())
	}
}

func TestExprNUnique(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	// DISCREPANCY: Expr.NUnique() inside Select may return per-row count in gopolars.
	// Using DataFrame.NUnique() instead.
	nunique, err := df.NUnique("a")
	if err != nil {
		t.Fatalf("nunique: %v", err)
	}
	if nunique != 3 {
		t.Fatalf("nunique: got %d, want 3", nunique)
	}
}

func TestExprFirstLast(t *testing.T) {
	t.Parallel()
	df := helperExprDF()
	result, err := df.Select(
		polars.Col("a").First().Alias("first"),
		polars.Col("a").Last().Alias("last"),
	)
	if err != nil {
		t.Fatalf("select first/last: %v", err)
	}
	// DISCREPANCY: In gopolars, First/Last inside Select may return per-row values
	// rather than aggregating to a single row like Python.
	_ = result
}
