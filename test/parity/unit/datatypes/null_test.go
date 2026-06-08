package datatypes

// Ported from py-polars/tests/unit/datatypes/test_null.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_null_comp_14118 (core): eq_missing treats null==null as true, ne_missing
// as false; both are non-null booleans.
func TestNullEqMissing(t *testing.T) {
	t.Parallel()
	a, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, int64(3)}})
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Int64, Values: []any{int64(1), nil, int64(4)}})
	if err != nil {
		t.Fatalf("b: %v", err)
	}

	eqm, err := a.EqMissing(b)
	if err != nil {
		t.Fatalf("eq_missing: %v", err)
	}
	for i, w := range []bool{true, true, false} {
		if v, ok := eqm.Value(i).(bool); !ok || v != w {
			t.Fatalf("eq_missing[%d]: got %v, want %v", i, eqm.Value(i), w)
		}
	}

	nem, err := a.NeMissing(b)
	if err != nil {
		t.Fatalf("ne_missing: %v", err)
	}
	for i, w := range []bool{false, false, true} {
		if v, ok := nem.Value(i).(bool); !ok || v != w {
			t.Fatalf("ne_missing[%d]: got %v, want %v", i, nem.Value(i), w)
		}
	}
}

// An all-null column is constructed by pinning its dtype; gopolars represents it
// as that dtype with every value null (it has no dedicated Null dtype).
func TestNullColumnConstruction(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{nil, nil, nil}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
	if s.NullCount() != 3 {
		t.Fatalf("null count: got %d, want 3", s.NullCount())
	}
	if !s.IsNull().Value(0).(bool) {
		t.Fatalf("is_null[0]: want true")
	}
}
