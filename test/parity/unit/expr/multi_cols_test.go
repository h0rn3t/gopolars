package expr

// Ported from py-polars/tests/unit/expr/test_expr_multi_cols.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestExprMultiCols(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{int64(4), int64(5), int64(6)}},
			{Name: "c", Values: []any{int64(7), int64(8), int64(9)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").Add(polars.Col("b")).Add(polars.Col("c")).Alias("sum_all"),
	)
	if err != nil {
		t.Fatalf("select multi cols: %v", err)
	}
	s, _ := result.GetColumn("sum_all")
	if v, ok := s.Value(0).(int64); !ok || v != 12 {
		t.Fatalf("sum_all[0]: got %v, want 12", s.Value(0))
	}
}

func TestExprNestedExpr(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{int64(4), int64(5), int64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").Mul(polars.Col("b")).Add(polars.Lit(int64(10))).Alias("computed"),
	)
	if err != nil {
		t.Fatalf("select nested: %v", err)
	}
	s, _ := result.GetColumn("computed")
	if v, ok := s.Value(0).(int64); !ok || v != 14 {
		t.Fatalf("computed[0]: got %v, want 14", s.Value(0))
	}
}

func TestExprMultiColsInFilter(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "b", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Filter(polars.Col("a").Gt(polars.Lit(int64(2))).And(polars.Col("b").Lt(polars.Lit(int64(40)))))
	if err != nil {
		t.Fatalf("filter multi cols: %v", err)
	}
	if result.Height() != 1 {
		t.Fatalf("filter height: got %d, want 1", result.Height())
	}
}

func TestExprChainedSelect(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{int64(4), int64(5), int64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a"),
		polars.Col("b"),
		polars.Col("a").Add(polars.Col("b")).Alias("c"),
	)
	if err != nil {
		t.Fatalf("select chained: %v", err)
	}
	if result.Width() != 3 {
		t.Fatalf("width: got %d, want 3", result.Width())
	}
}
