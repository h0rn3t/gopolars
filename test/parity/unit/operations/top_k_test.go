package operations

// Ported from py-polars/tests/unit/operations/test_top_k.py (py-1.28.1, representative subset)

import (
	"sort"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func topKSeries(t *testing.T) polars.Series {
	t.Helper()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(5), int64(1), int64(4), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	return s
}

func collectInts(s polars.Series) []int64 {
	out := make([]int64, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if v, ok := s.Value(i).(int64); ok {
			out = append(out, v)
		}
	}
	return out
}

// top_k returns the k largest values.
func TestTopK(t *testing.T) {
	t.Parallel()
	out := topKSeries(t).TopK(3)
	got := collectInts(out)
	sort.Slice(got, func(i, j int) bool { return got[i] > got[j] })
	want := []int64{5, 4, 3}
	if len(got) != 3 {
		t.Fatalf("top_k count: got %v, want 3 values", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("top_k: got %v, want %v", got, want)
		}
	}
}

// bottom_k returns the k smallest values.
func TestBottomK(t *testing.T) {
	t.Parallel()
	out := topKSeries(t).BottomK(2)
	got := collectInts(out)
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	want := []int64{1, 2}
	if len(got) != 2 {
		t.Fatalf("bottom_k count: got %v, want 2 values", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("bottom_k: got %v, want %v", got, want)
		}
	}
}
