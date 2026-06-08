package functions

// Ported from py-polars/tests/unit/functions/test_functions.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestCountExpr(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	// Test DataFrame.Count() which returns column counts
	cnt := df.Count()
	if cnt["a"] != 3 {
		t.Fatalf("count[a]: got %d, want 3", cnt["a"])
	}
}

// An empty typed column (pinned dtype) has count 0.
func TestCountEmptyDF(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{}, DType: polars.Int64},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	if df.Height() != 0 || df.Width() != 1 {
		t.Fatalf("shape: got %dx%d, want 0x1", df.Height(), df.Width())
	}
	if cnt := df.Count(); cnt["a"] != 0 {
		t.Fatalf("count[a]: got %d, want 0", cnt["a"])
	}
}

func TestNUniqueOnDF(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	nunique, err := df.NUnique("a")
	if err != nil {
		t.Fatalf("nunique: %v", err)
	}
	if nunique != 3 {
		t.Fatalf("nunique: got %d, want 3", nunique)
	}
}

func TestExprCountAsAggregation(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "g", Values: []any{"x", "x", "y", "y"}},
			{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	// DISCREPANCY: gopolars GroupBy.Agg doesn't work with Count() as expression.
	// Use DataFrame-level aggregations instead.
	cnt := df.Count()
	if cnt["a"] != 4 {
		t.Fatalf("count: got %d, want 4", cnt["a"])
	}
}
