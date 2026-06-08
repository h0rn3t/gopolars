package operations

// Ported from py-polars/tests/unit/operations/test_drop_nulls.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDropNullsSeries(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, int64(3), nil, int64(5)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.DropNulls()
	if out.Len() != 3 {
		t.Fatalf("len: got %d, want 3", out.Len())
	}
	for i, w := range []int64{1, 3, 5} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("drop_nulls[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}

func TestDropNullsDataFrame(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), nil, int64(3)}},
		{Name: "b", Values: []any{int64(10), int64(20), nil}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out := df.DropNulls()
	if out.Height() != 1 {
		t.Fatalf("height: got %d, want 1 (only first row has no nulls)", out.Height())
	}
}
