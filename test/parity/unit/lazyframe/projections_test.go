package lazyframe

// Ported from py-polars/tests/unit/lazyframe/test_projections.py (py-1.28.1, representative subset)
//
// Projection pushdown is an internal optimization; we verify the observable
// result (only requested columns survive).

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestProjectionNarrowsColumns(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{int64(3), int64(4)}},
		{Name: "c", Values: []any{int64(5), int64(6)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Lazy().Select(polars.Col("a"), polars.Col("c")).Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	cols := out.Columns()
	if len(cols) != 2 || cols[0] != "a" || cols[1] != "c" {
		t.Fatalf("projection columns: got %v, want [a c]", cols)
	}
}
