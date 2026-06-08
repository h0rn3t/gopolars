package operations

// Ported from py-polars/tests/unit/operations/test_rle.py (py-1.28.1), rle_id
// portion (test_rle covers the value/count form in misc_ops_test.go).
//
// DISCREPANCY (cosmetic): Python `rle_id` is UInt32; gopolars returns Int64.
// The run-id sequence (0-based, incremented on each change) matches.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestRleId(t *testing.T) {
	t.Parallel()
	s, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64,
		Values: []any{int64(12), int64(3345), int64(12), int64(3), int64(4), int64(4), int64(1), int64(12)}})
	out := s.RleId()
	want := []int64{0, 1, 2, 3, 4, 4, 5, 6}
	if got := collectInts(out); len(got) != len(want) {
		t.Fatalf("rle_id len: got %v, want %v", got, want)
	}
	for i, w := range want {
		if got, _ := out.Value(i).(int64); got != w {
			t.Fatalf("rle_id[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}

func TestRleIdAllEqual(t *testing.T) {
	t.Parallel()
	s, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(7), int64(7), int64(7)}})
	out := s.RleId()
	for i := 0; i < out.Len(); i++ {
		if got, _ := out.Value(i).(int64); got != 0 {
			t.Fatalf("rle_id[%d]: got %v, want 0 (single run)", i, out.Value(i))
		}
	}
}
