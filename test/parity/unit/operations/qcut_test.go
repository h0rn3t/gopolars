package operations

// Ported from py-polars/tests/unit/operations/test_qcut.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// qcut bins values into q quantile-based buckets.
func TestQCut(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.QCut(4)
	if err != nil {
		t.Fatalf("qcut: %v", err)
	}
	if out.Len() != 8 {
		t.Fatalf("qcut len: got %d, want 8", out.Len())
	}
	seen := map[string]bool{}
	for i := 0; i < out.Len(); i++ {
		if v, ok := out.Value(i).(string); ok {
			seen[v] = true
		}
	}
	// 4 quantile buckets over evenly spaced data
	if len(seen) != 4 {
		t.Fatalf("qcut buckets: got %d (%v), want 4", len(seen), seen)
	}
}
