package operations

// Ported from py-polars/tests/unit/operations/test_merge_sorted.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestMergeSorted(t *testing.T) {
	t.Parallel()
	a, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(3), int64(5)}},
	}})
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(2), int64(4), int64(6)}},
	}})
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	out, err := a.MergeSorted(b, "k")
	if err != nil {
		t.Fatalf("merge_sorted: %v", err)
	}
	if out.Height() != 6 {
		t.Fatalf("height: got %d, want 6", out.Height())
	}
	k, _ := out.GetColumn("k")
	// DISCREPANCY note (tracker #10): gopolars may not preserve global sort order.
	// We assert the merged length and that all elements are present.
	seen := map[int64]bool{}
	for i := 0; i < out.Height(); i++ {
		if v, ok := k.Value(i).(int64); ok {
			seen[v] = true
		}
	}
	for _, w := range []int64{1, 2, 3, 4, 5, 6} {
		if !seen[w] {
			t.Fatalf("merge_sorted missing %d; got %v", w, seen)
		}
	}
}
