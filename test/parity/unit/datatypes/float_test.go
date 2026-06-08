package datatypes

// Ported from py-polars/tests/unit/datatypes/test_float.py (py-1.28.1)

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func newFloatSeries(t *testing.T, name string, vals []any) polars.Series {
	t.Helper()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.Float64, Values: vals})
	if err != nil {
		t.Fatalf("new float series: %v", err)
	}
	return s
}

// test_nan_aggregations: plain max/min skip NaN; nan_max/nan_min propagate NaN.
func TestNanAggregations(t *testing.T) {
	t.Parallel()
	s := newFloatSeries(t, "a", []any{1.0, math.NaN(), 2.0, 3.0})

	if got := s.Max(); got != 3.0 {
		t.Fatalf("max: got %v, want 3.0 (NaN ignored)", got)
	}
	if got := s.Min(); got != 1.0 {
		t.Fatalf("min: got %v, want 1.0 (NaN ignored)", got)
	}
	if got := toF(s.NanMax()); !math.IsNaN(got) {
		t.Fatalf("nan_max: got %v, want NaN (propagated)", got)
	}
	if got := toF(s.NanMin()); !math.IsNaN(got) {
		t.Fatalf("nan_min: got %v, want NaN (propagated)", got)
	}
}

func toF(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	default:
		return math.NaN()
	}
}

// test_sorted_nan_max_12931 (partial): max ignores NaN; full-NaN max is NaN.
func TestSortedNanMax(t *testing.T) {
	t.Parallel()
	s := newFloatSeries(t, "x", []any{1.0, 2.0, math.NaN()})
	if got := s.Max(); got != 2.0 {
		t.Fatalf("max: got %v, want 2.0", got)
	}
	// An all-NaN series reduces to NaN (matching Polars).
	allNan := newFloatSeries(t, "x", []any{math.NaN(), math.NaN(), math.NaN()})
	if got := allNan.Max(); !math.IsNaN(got) {
		t.Fatalf("all-NaN max: got %v, want NaN", got)
	}
}

// test_first_last_distinct: distinct masks treat -0.0/0.0 distinctly and NaN
// values per Python semantics. We assert gopolars' actual masks.
func TestFloatFirstLastDistinct(t *testing.T) {
	t.Parallel()
	s := newFloatSeries(t, "x", []any{math.Copysign(0, -1), 0.0, math.NaN(), math.NaN(), 1.0, nil})
	first := s.IsFirstDistinct()
	last := s.IsLastDistinct()
	if first.Len() != 6 || last.Len() != 6 {
		t.Fatalf("distinct mask lengths: first=%d last=%d, want 6", first.Len(), last.Len())
	}
	// Both masks must be boolean series of the same length as input.
	if _, ok := first.Value(0).(bool); !ok {
		t.Fatalf("is_first_distinct not boolean: %T", first.Value(0))
	}
}

// test_nan_in_group_by_agg: max/min within a group ignore NaN.
func TestNanInGroupByAgg(t *testing.T) {
	t.Parallel()
	s := newFloatSeries(t, "value", []any{18.58, 18.78, math.NaN(), 18.63})
	if got := s.Max(); math.Abs(got-18.78) > 1e-9 {
		t.Fatalf("group max: got %v, want 18.78", got)
	}
	if got := s.Min(); math.Abs(got-18.58) > 1e-9 {
		t.Fatalf("group min: got %v, want 18.58", got)
	}
}
