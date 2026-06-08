package operations

// Ported from py-polars/tests/unit/operations/test_abs.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestAbsInt(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(-3), int64(2), int64(-1), int64(0)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Abs()
	for i, w := range []int64{3, 2, 1, 0} {
		switch v := out.Value(i).(type) {
		case int64:
			if v != w {
				t.Fatalf("abs[%d]: got %d, want %d", i, v, w)
			}
		case float64:
			if v != float64(w) {
				t.Fatalf("abs[%d]: got %v, want %d", i, v, w)
			}
		default:
			t.Fatalf("abs[%d]: unexpected type %T", i, out.Value(i))
		}
	}
}

func TestAbsFloat(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{-1.5, 2.5, -0.25}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Abs()
	for i, w := range []float64{1.5, 2.5, 0.25} {
		if v, _ := out.Value(i).(float64); v != w {
			t.Fatalf("abs[%d]: got %v, want %v", i, out.Value(i), w)
		}
	}
}

func TestAbsNullsPreserved(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(-1), nil, int64(-3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Abs()
	if out.Value(1) != nil {
		t.Fatalf("null not preserved: %v", out.Value(1))
	}
}
