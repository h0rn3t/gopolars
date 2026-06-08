package operations

// Ported from py-polars/tests/unit/operations/aggregation/test_implode.py (py-1.28.1).
// Series.implode collapses a Series into a single List-typed row.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestImplode(t *testing.T) {
	t.Parallel()
	s, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	out := s.Implode()
	if out.Len() != 1 {
		t.Fatalf("implode len: got %d, want 1", out.Len())
	}
	if out.DataType() != polars.List {
		t.Fatalf("implode dtype: got %v, want List", out.DataType())
	}
	list, ok := out.Value(0).([]any)
	if !ok || len(list) != 3 {
		t.Fatalf("implode value: got %v (%T), want a 3-element list", out.Value(0), out.Value(0))
	}
	for i, w := range []int64{1, 2, 3} {
		if v, _ := list[i].(int64); v != w {
			t.Fatalf("implode[0][%d]: got %v, want %d", i, list[i], w)
		}
	}
}

func TestImplodeEmpty(t *testing.T) {
	t.Parallel()
	s, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{}})
	out := s.Implode()
	if out.Len() != 1 {
		t.Fatalf("implode empty len: got %d, want 1 (single empty list)", out.Len())
	}
	list, ok := out.Value(0).([]any)
	if !ok || len(list) != 0 {
		t.Fatalf("implode empty value: got %v, want empty list", out.Value(0))
	}
}
