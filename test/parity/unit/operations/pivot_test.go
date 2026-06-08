package operations

// Ported from py-polars/tests/unit/operations/test_pivot.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// pivot reshapes long data to wide using index/columns/values.
func TestPivot(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"a", "a", "b", "b"}},
		{Name: "k", Values: []any{"x", "y", "x", "y"}},
		{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := polars.Pivot(df, polars.PivotInput{Index: "g", Columns: "k", Values: "v", Agg: "sum"})
	if err != nil {
		t.Fatalf("pivot: %v", err)
	}
	// one row per distinct g (a, b)
	if out.Height() != 2 {
		t.Fatalf("pivot height: got %d, want 2", out.Height())
	}
	// columns: g, x, y
	if out.Width() != 3 {
		t.Fatalf("pivot width: got %d, want 3 (g,x,y)", out.Width())
	}
}
