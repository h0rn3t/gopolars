package dataframe

// Ported from py-polars/tests/unit/dataframe/test_window.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFWindowSingleRowLiteral(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "store_id", Values: []any{int64(1)}},
			{Name: "cost_price", Values: []any{float64(2.0)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}

	costPrice := polars.Col("cost_price")
	inverseCostPrice := polars.Lit(float64(1.0)).Div(costPrice)
	result, err := df.WithColumns(
		inverseCostPrice.Div(inverseCostPrice.Sum().Over("store_id")).Alias("result"),
	)
	if err != nil {
		// DISCREPANCY: gopolars may not support this window expression pattern
		t.Fatalf("window expression with literal ambiguity: %v", err)
	}
	if result.Width() != 3 {
		t.Fatalf("result width: got %d, want 3", result.Width())
	}
}
