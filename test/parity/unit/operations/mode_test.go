package operations

// Ported from py-polars/tests/unit/operations/test_mode.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Mode returns the most frequent value.
func TestModeSingle(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(2), int64(3), int64(2)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	got := s.Mode()
	switch v := got.(type) {
	case int64:
		if v != 2 {
			t.Fatalf("mode: got %d, want 2", v)
		}
	case float64:
		if v != 2 {
			t.Fatalf("mode: got %v, want 2", v)
		}
	default:
		t.Fatalf("mode: unexpected type %T(%v)", got, got)
	}
}

func TestModeString(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.String, Values: []any{"x", "y", "x", "x"}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if v, ok := s.Mode().(string); !ok || v != "x" {
		t.Fatalf("mode: got %v, want x", s.Mode())
	}
}
