package operations

// Ported from py-polars/tests/unit/operations/aggregation/test_aggregations.py (py-1.28.1, representative subset)

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func aggSeries(t *testing.T) polars.Series {
	t.Helper()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	return s
}

// Basic reductions on a Series.
func TestSeriesReductions(t *testing.T) {
	t.Parallel()
	s := aggSeries(t)
	if got := s.Sum(); got != 10 {
		t.Fatalf("sum: got %v, want 10", got)
	}
	if got := s.Min(); got != 1 {
		t.Fatalf("min: got %v, want 1", got)
	}
	if got := s.Max(); got != 4 {
		t.Fatalf("max: got %v, want 4", got)
	}
	if got := s.Mean(); math.Abs(got-2.5) > 1e-9 {
		t.Fatalf("mean: got %v, want 2.5", got)
	}
}

// Reductions ignore nulls.
func TestReductionsIgnoreNulls(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if got := s.Sum(); got != 4 {
		t.Fatalf("sum: got %v, want 4 (null ignored)", got)
	}
	if got := s.Mean(); math.Abs(got-2.0) > 1e-9 {
		t.Fatalf("mean: got %v, want 2.0 (null ignored)", got)
	}
}

// Count and n_unique.
func TestCountNUnique(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if got := s.NUnique(); got != 3 {
		t.Fatalf("n_unique: got %d, want 3", got)
	}
}
