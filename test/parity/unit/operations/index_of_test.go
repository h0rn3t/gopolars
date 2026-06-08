package operations

// Ported from py-polars/tests/unit/operations/test_index_of.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// index_of returns the position of the first matching value, or -1 (Python: None)
// when the value is absent.
func TestIndexOf(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(10), int64(20), int64(30), int64(40)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if idx := s.IndexOf(int64(30)); idx != 2 {
		t.Fatalf("index_of(30): got %d, want 2", idx)
	}
	if idx := s.IndexOf(int64(10)); idx != 0 {
		t.Fatalf("index_of(10): got %d, want 0", idx)
	}
	if idx := s.IndexOf(int64(99)); idx != -1 {
		t.Fatalf("index_of(99): got %d, want -1 (absent)", idx)
	}
}

func TestIndexOfStrings(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.String, Values: []any{"x", "y", "z"}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if idx := s.IndexOf("y"); idx != 1 {
		t.Fatalf("index_of(y): got %d, want 1", idx)
	}
}
