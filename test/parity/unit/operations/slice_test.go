package operations

// Ported from py-polars/tests/unit/operations/test_slice.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func sliceDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "b", Values: []any{"a", "b", "c"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// test_python_slicing_data_frame: slice(1, 10) and slice(1, 2) yield rows 2,3.
func TestSliceWithinAndBeyond(t *testing.T) {
	t.Parallel()
	for _, length := range []int{10, 2} {
		out := sliceDF(t).Slice(1, length)
		if out.Height() != 2 {
			t.Fatalf("slice(1,%d) height: got %d, want 2", length, out.Height())
		}
		a, _ := out.GetColumn("a")
		if v, _ := a.Value(0).(int64); v != 2 {
			t.Fatalf("slice first: got %v, want 2", a.Value(0))
		}
	}
}

// Negative starting index before the start: the window [offset, offset+length)
// is clamped to [0, len), so slice(-5, 4) on a 3-row frame keeps only rows {1,2}
// (matching Polars).
func TestSliceNegativeOffset(t *testing.T) {
	t.Parallel()
	out := sliceDF(t).Slice(-5, 4)
	if out.Height() != 2 {
		t.Fatalf("slice(-5,4) height: got %d, want 2 (window clamped to [0,len))", out.Height())
	}
	a, _ := out.GetColumn("a")
	for i, w := range []int64{1, 2} {
		if v, _ := a.Value(i).(int64); v != w {
			t.Fatalf("slice[%d]: got %v, want %d", i, a.Value(i), w)
		}
	}
}

// Series slice.
func TestSliceSeries(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(0), int64(1), int64(2), int64(3), int64(4), int64(5)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Slice(2, 3)
	for i, w := range []int64{2, 3, 4} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("slice[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}
