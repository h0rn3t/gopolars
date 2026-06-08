package datatypes

// Ported from py-polars/tests/unit/datatypes/test_list.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func newListSeries(t *testing.T, name string, vals []any) polars.Series {
	t.Helper()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.List, Values: vals})
	if err != nil {
		t.Fatalf("new list series: %v", err)
	}
	return s
}

// List construction: each row holds a slice; dtype is List.
func TestListConstruction(t *testing.T) {
	t.Parallel()
	s := newListSeries(t, "l", []any{[]any{int64(1), int64(2)}, []any{int64(3)}})
	if s.Len() != 2 {
		t.Fatalf("len: got %d, want 2", s.Len())
	}
	if s.DataType() != polars.List {
		t.Fatalf("dtype: got %v, want List", s.DataType())
	}
	if _, ok := s.Value(0).([]any); !ok {
		t.Fatalf("row 0 is not a slice: %T", s.Value(0))
	}
}

// explode flattens the inner lists into a single column.
func TestListExplode(t *testing.T) {
	t.Parallel()
	s := newListSeries(t, "l", []any{[]any{int64(1), int64(2)}, []any{int64(3)}})
	ex := s.Explode()
	if ex.Len() != 3 {
		t.Fatalf("explode len: got %d, want 3", ex.Len())
	}
	for i, w := range []int64{1, 2, 3} {
		if v, ok := ex.Value(i).(int64); !ok || v != w {
			t.Fatalf("explode[%d]: got %v, want %d", i, ex.Value(i), w)
		}
	}
}

// implode collapses a flat series into one List-valued row (inverse of explode).
func TestListImplode(t *testing.T) {
	t.Parallel()
	flat, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("flat: %v", err)
	}
	out := flat.Implode()
	if out.Len() != 1 {
		t.Fatalf("implode len: got %d, want 1", out.Len())
	}
}
