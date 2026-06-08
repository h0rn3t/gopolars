package operations

// Ported from py-polars/tests/unit/operations/test_drop.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func dropDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(2), int64(1), int64(3)}},
		{Name: "b", Values: []any{"a", "b", "c"}},
		{Name: "c", Values: []any{int64(1), int64(2), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// test_drop: dropping one column changes shape to (3, 2).
func TestDropColumn(t *testing.T) {
	t.Parallel()
	out, err := dropDF(t).Drop("a")
	if err != nil {
		t.Fatalf("drop: %v", err)
	}
	if out.Height() != 3 || out.Width() != 2 {
		t.Fatalf("shape: got %dx%d, want 3x2", out.Height(), out.Width())
	}
	for _, c := range out.Columns() {
		if c == "a" {
			t.Fatalf("column a should be dropped, columns=%v", out.Columns())
		}
	}
}

// Dropping multiple columns.
func TestDropMultiple(t *testing.T) {
	t.Parallel()
	out, err := dropDF(t).Drop("a", "c")
	if err != nil {
		t.Fatalf("drop multi: %v", err)
	}
	if cols := out.Columns(); len(cols) != 1 || cols[0] != "b" {
		t.Fatalf("columns: got %v, want [b]", out.Columns())
	}
}

// drop_in_place returns the removed series.
func TestDropInPlace(t *testing.T) {
	t.Parallel()
	_, err := dropDF(t).DropInPlace("a")
	if err != nil {
		t.Fatalf("drop_in_place: %v", err)
	}
}
