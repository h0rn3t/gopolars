package operations

// Ported from py-polars/tests/unit/operations/test_over.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// DISCREPANCY: Python supports window aggregations via sum().over("g") in
// with_columns. gopolars rejects aggregation expressions outside GroupBy.Agg with
// "unsupported expr kind agg", so .Over(...) on an aggregation errors here. We
// assert the error and document the gap.
func TestOverAggregationGap(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"a", "a", "b", "b"}},
		{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	if _, err := df.WithColumns(polars.Sum(polars.Col("v")).Over("g").Alias("group_sum")); err == nil {
		t.Fatal("expected error for aggregation .over() in with_columns, got nil")
	}
}
