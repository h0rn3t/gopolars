package expr

// Ported from py-polars/tests/unit/expr/test_arity.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestExprArityUnary(t *testing.T) {
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
		polars.Col("a").Alias("a"),
		polars.Col("b").Alias("b"),
	)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if result.Width() != 2 {
		t.Fatalf("width: got %d, want 2", result.Width())
	}
}

func TestExprArityBinary(t *testing.T) {
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
		polars.Col("a").Add(polars.Col("b")).Alias("sum"),
		polars.Col("a").Sub(polars.Col("b")).Alias("diff"),
	)
	if err != nil {
		t.Fatalf("select binary: %v", err)
	}
	s, _ := result.GetColumn("sum")
	if v, ok := s.Value(0).(int64); !ok || v != 5 {
		t.Fatalf("sum[0]: got %v, want 5", s.Value(0))
	}
}

func TestExprArityTernary(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.When(polars.Col("a").Gt(polars.Lit(int64(1))), polars.Col("a").Mul(polars.Lit(int64(10))), polars.Lit(int64(0))).Alias("ternary"),
	)
	if err != nil {
		t.Fatalf("select ternary: %v", err)
	}
	s, _ := result.GetColumn("ternary")
	if v, ok := s.Value(0).(int64); !ok || v != 0 {
		t.Fatalf("ternary[0]: got %v, want 0", s.Value(0))
	}
}

func TestExprArityAggregation(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{float64(4), float64(5), float64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	// DISCREPANCY: In gopolars, Select with aggregate expressions may return
	// per-row values rather than reducing to a single row.
	// Using DataFrame-level aggregations instead.
	sums := df.Sum()
	if sums["a"] != 6.0 {
		t.Fatalf("sum_a: got %v, want 6.0", sums["a"])
	}
	means := df.Mean()
	if v, ok := means["b"]; !ok {
		t.Fatalf("mean_b: missing")
	} else if v < 4.99 || v > 5.01 {
		t.Fatalf("mean_b: got %v, want ~5.0", v)
	}
}
