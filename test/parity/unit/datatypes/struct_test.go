package datatypes

// Ported from py-polars/tests/unit/datatypes/test_struct.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Struct series construction from maps; field access via the Struct namespace.
func TestStructDtypeConstruction(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:  "s",
		DType: polars.Struct,
		Values: []any{
			map[string]any{"a": int64(1), "b": "x"},
			map[string]any{"a": int64(2), "b": "y"},
		},
	})
	if err != nil {
		t.Fatalf("new struct: %v", err)
	}
	if s.DataType() != polars.Struct {
		t.Fatalf("dtype: got %v, want Struct", s.DataType())
	}
	if s.Len() != 2 {
		t.Fatalf("len: got %d, want 2", s.Len())
	}
}

// Struct().Field() preserves the field dtype (Int64 -> 10), matching Polars.
func TestStructDtypeFieldPreservesDtype(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:  "s",
		DType: polars.Struct,
		Values: []any{
			map[string]any{"a": int64(10), "b": "x"},
			map[string]any{"a": int64(20), "b": "y"},
		},
	})
	if err != nil {
		t.Fatalf("new struct: %v", err)
	}
	a, err := s.Struct().Field("a")
	if err != nil {
		t.Fatalf("field a: %v", err)
	}
	if a.DataType() != polars.Int64 {
		t.Fatalf("field a dtype: got %v, want Int64", a.DataType())
	}
	if v, ok := a.Value(0).(int64); !ok || v != 10 {
		t.Fatalf("a[0]: got %T(%v), want int64 10", a.Value(0), a.Value(0))
	}
	// String field stays String.
	b, err := s.Struct().Field("b")
	if err != nil {
		t.Fatalf("field b: %v", err)
	}
	if b.DataType() != polars.String {
		t.Fatalf("field b dtype: got %v, want String", b.DataType())
	}
}

// Empty struct series.
func TestStructDtypeEmpty(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.Struct, Values: []any{}})
	if err != nil {
		t.Fatalf("new empty struct: %v", err)
	}
	if s.Len() != 0 {
		t.Fatalf("len: got %d, want 0", s.Len())
	}
}
