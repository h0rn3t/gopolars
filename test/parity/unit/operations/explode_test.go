package operations

// Ported from py-polars/tests/unit/operations/test_explode.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Series.explode flattens a List series.
func TestExplodeSeries(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "l", DType: polars.List, Values: []any{
		[]any{int64(1), int64(2)}, []any{int64(3), int64(4), int64(5)},
	}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Explode()
	if out.Len() != 5 {
		t.Fatalf("explode len: got %d, want 5", out.Len())
	}
	for i, w := range []int64{1, 2, 3, 4, 5} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("explode[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}

// DataFrame.explode expands a list column, repeating the scalar columns.
func TestExplodeDataFrame(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{"a", "b"}},
		{Name: "l", Values: []any{[]any{int64(1), int64(2)}, []any{int64(3)}}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Explode("l")
	if err != nil {
		t.Fatalf("DataFrame.Explode failed: %v", err)
	}
	if out.Height() != 3 {
		t.Fatalf("explode height: got %d, want 3", out.Height())
	}
	k, _ := out.GetColumn("k")
	for i, w := range []string{"a", "a", "b"} {
		if v, _ := k.Value(i).(string); v != w {
			t.Fatalf("k[%d]: got %v, want %s", i, k.Value(i), w)
		}
	}
}
