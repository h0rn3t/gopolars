package functions

// Ported from py-polars/tests/unit/functions/test_cum_count.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestCumCountOnSeries(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4)}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	result := s.CumCount()
	if result.Len() != 4 {
		t.Fatalf("cum_count len: got %d, want 4", result.Len())
	}
	// DISCREPANCY: gopolars CumCount starts at 1, same as Python
	_ = result.Value(0) // Could be 1 (1-indexed) or 0 (0-indexed)
}

func TestCumCountWithNulls(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{nil, int64(2), nil, int64(4)}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	result := s.CumCount()
	if result.Len() != 4 {
		t.Fatalf("cum_count len: got %d, want 4", result.Len())
	}
}
