package datatypes

// Ported from py-polars/tests/unit/datatypes/test_enum.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Enum construction preserves the string category values.
func TestEnumConstruction(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "e", DType: polars.Enum, Values: []any{"x", "y", "x"}})
	if err != nil {
		t.Fatalf("new enum: %v", err)
	}
	if s.DataType() != polars.Enum {
		t.Fatalf("dtype: got %v, want Enum", s.DataType())
	}
	if v, ok := s.Value(1).(string); !ok || v != "y" {
		t.Fatalf("value[1]: got %v, want y", s.Value(1))
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
}

// Enum with nulls keeps null slots.
func TestEnumNulls(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "e", DType: polars.Enum, Values: []any{"a", nil, "b"}})
	if err != nil {
		t.Fatalf("new enum: %v", err)
	}
	if s.Value(1) != nil {
		t.Fatalf("value[1]: got %v, want nil", s.Value(1))
	}
	if s.NullCount() != 1 {
		t.Fatalf("null count: got %d, want 1", s.NullCount())
	}
}
