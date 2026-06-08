package operations

// Ported from py-polars/tests/unit/operations/unique/test_unique.py (py-1.28.1, representative subset)

import (
	"sort"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Series.unique removes duplicates.
func TestUniqueSeries(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(2), int64(3), int64(1)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Unique()
	got := make([]int64, 0, out.Len())
	for i := 0; i < out.Len(); i++ {
		if v, ok := out.Value(i).(int64); ok {
			got = append(got, v)
		}
	}
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	want := []int64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("unique count: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unique: got %v, want %v", got, want)
		}
	}
}

// DataFrame.unique drops duplicate rows.
func TestUniqueDataFrame(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(1), int64(2)}},
		{Name: "b", Values: []any{"x", "x", "y"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Unique()
	if err != nil {
		t.Fatalf("unique: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("unique height: got %d, want 2", out.Height())
	}
}

// n_unique via NUnique on a series.
func TestNUnique(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if n := s.NUnique(); n != 3 {
		t.Fatalf("n_unique: got %d, want 3", n)
	}
}
