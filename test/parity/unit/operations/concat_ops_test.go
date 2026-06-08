package operations

// Ported from py-polars/tests/unit/operations/test_concat.py (py-1.28.1),
// representative subset: vertical (row-stacking) and horizontal (column-joining)
// DataFrame concatenation.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestConcatVertical(t *testing.T) {
	t.Parallel()
	a, _ := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "x", Values: []any{int64(1), int64(2)}},
	}})
	b, _ := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "x", Values: []any{int64(3)}},
	}})
	out, err := a.Concat(polars.ConcatInput{Others: []polars.DataFrame{b}})
	if err != nil {
		t.Fatalf("concat vertical: %v", err)
	}
	if out.Height() != 3 || out.Width() != 1 {
		t.Fatalf("concat vertical shape: got %dx%d, want 3x1", out.Height(), out.Width())
	}
	col := firstColValues(t, out, "x")
	if got := collectInts(col); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("concat vertical values: got %v, want [1 2 3]", got)
	}
}

func TestConcatHorizontal(t *testing.T) {
	t.Parallel()
	a, _ := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "x", Values: []any{int64(1), int64(2)}},
	}})
	b, _ := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "y", Values: []any{int64(3), int64(4)}},
	}})
	out, err := a.Concat(polars.ConcatInput{Others: []polars.DataFrame{b}, How: "horizontal"})
	if err != nil {
		t.Fatalf("concat horizontal: %v", err)
	}
	if out.Height() != 2 || out.Width() != 2 {
		t.Fatalf("concat horizontal shape: got %dx%d, want 2x2", out.Height(), out.Width())
	}
}
