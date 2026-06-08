package operations

// Ported from py-polars/tests/unit/operations/aggregation/test_folds.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Fold across columns with an "add" accumulator equals the horizontal sum.
func TestFoldAdd(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "b", Values: []any{int64(10), int64(20), int64(30)}},
		{Name: "c", Values: []any{int64(100), int64(200), int64(300)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Fold("add", []string{"a", "b", "c"}, "total")
	if err != nil {
		t.Fatalf("Fold(add) failed: %v", err)
	}
	s, err := out.GetColumn("total")
	if err != nil {
		t.Fatalf("get total: %v", err)
	}
	for i, w := range []float64{111, 222, 333} {
		if toFloatAny(s.Value(i)) != w {
			t.Fatalf("fold[%d]: got %v, want %v", i, s.Value(i), w)
		}
	}
}
