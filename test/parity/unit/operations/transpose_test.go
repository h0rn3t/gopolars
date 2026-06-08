package operations

// Ported from py-polars/tests/unit/operations/test_transpose.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// transpose of a 3-row x 2-col frame is a 2-row x 3-col frame with default
// column names column_0..column_2 (matching Polars).
func TestTranspose(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "b", Values: []any{int64(4), int64(5), int64(6)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Transpose()
	if err != nil {
		t.Fatalf("transpose: %v", err)
	}
	if out.Height() != 2 || out.Width() != 3 {
		t.Fatalf("transpose shape: got %dx%d, want 2x3", out.Height(), out.Width())
	}
	cols := out.Columns()
	for i, want := range []string{"column_0", "column_1", "column_2"} {
		if cols[i] != want {
			t.Fatalf("transpose column %d: got %q, want %q", i, cols[i], want)
		}
	}
	// column_0 holds the first row of each original column: [a[0], b[0]] = [1, 4].
	c0, _ := out.GetColumn("column_0")
	if v, _ := c0.Value(0).(int64); v != 1 {
		t.Fatalf("column_0[0]: got %v, want 1", c0.Value(0))
	}
	if v, _ := c0.Value(1).(int64); v != 4 {
		t.Fatalf("column_0[1]: got %v, want 4", c0.Value(1))
	}
}
