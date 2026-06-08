package operations

// Ported from py-polars/tests/unit/operations/test_value_counts.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// value_counts returns a DataFrame with the distinct values and their counts.
func TestValueCounts(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.String, Values: []any{"x", "y", "x", "z", "x", "y"}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.ValueCounts()
	if err != nil {
		t.Fatalf("value_counts: %v", err)
	}
	// 3 distinct values -> 3 rows
	if out.Height() != 3 {
		t.Fatalf("value_counts rows: got %d, want 3", out.Height())
	}
	if out.Width() < 2 {
		t.Fatalf("value_counts width: got %d, want >=2 (value + count)", out.Width())
	}
}

func TestValueCountsAllUnique(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.ValueCounts()
	if err != nil {
		t.Fatalf("value_counts: %v", err)
	}
	if out.Height() != 3 {
		t.Fatalf("rows: got %d, want 3", out.Height())
	}
}
