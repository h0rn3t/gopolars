package series

// Ported from py-polars/tests/unit/series/test_getitem.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestGetItemInt(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(10), int64(20), int64(30), int64(40)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	if v, ok := s.Value(0).(int64); !ok || v != 10 {
		t.Fatalf("value[0]: got %v, want 10", s.Value(0))
	}
	if v, ok := s.Value(1).(int64); !ok || v != 20 {
		t.Fatalf("value[1]: got %v, want 20", s.Value(1))
	}
	if v, ok := s.Value(3).(int64); !ok || v != 40 {
		t.Fatalf("value[3]: got %v, want 40", s.Value(3))
	}
}

func TestGetItemString(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.String, Values: []any{"foo", "bar", "baz"}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	if v, ok := s.Value(0).(string); !ok || v != "foo" {
		t.Fatalf("value[0]: got %v, want foo", s.Value(0))
	}
	if v, ok := s.Value(2).(string); !ok || v != "baz" {
		t.Fatalf("value[2]: got %v, want baz", s.Value(2))
	}
}

func TestGetItemNull(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, int64(3)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	if s.Value(0) == nil {
		t.Fatalf("value[0]: should not be nil")
	}
	if s.Value(1) != nil {
		t.Fatalf("value[1]: should be nil, got %v", s.Value(1))
	}
	if s.Value(2) == nil {
		t.Fatalf("value[2]: should not be nil")
	}
}

func TestGetItemFloat(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "f", DType: polars.Float64, Values: []any{float64(1.5), float64(2.5)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	if v, ok := s.Value(0).(float64); !ok || v != 1.5 {
		t.Fatalf("value[0]: got %v, want 1.5", s.Value(0))
	}
	if v, ok := s.Value(1).(float64); !ok || v != 2.5 {
		t.Fatalf("value[1]: got %v, want 2.5", s.Value(1))
	}
}
