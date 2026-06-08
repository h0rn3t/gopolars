package operations

// Ported from py-polars/tests/unit/operations/test_reverse.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestReverseSeries(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Reverse()
	for i, w := range []int64{3, 2, 1} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("reverse[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}

func TestReverseDataFrame(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out := df.Reverse()
	a, _ := out.GetColumn("a")
	if v, _ := a.Value(0).(int64); v != 3 {
		t.Fatalf("reversed first: got %v, want 3", a.Value(0))
	}
}
