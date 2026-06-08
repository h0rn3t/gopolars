package constructors

// Ported from py-polars/tests/unit/constructors/test_convert.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Tests for type conversion during construction

func TestCastInt64ToInt32(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), int64(2), int64(3)},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	// Go: only Int64/Float64/String/Boolean dtypes available. Test cast to Float64.
	casted, err := s.Cast(polars.Float64)
	if err != nil {
		t.Fatalf("cast to Float64 failed: %v", err)
	}
	if casted.Len() != 3 {
		t.Fatalf("casted len: got %d, want 3", casted.Len())
	}
}

func TestCastInt64ToFloat64(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), int64(2), int64(3)},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	casted, err := s.Cast(polars.Float64)
	if err != nil {
		t.Fatalf("cast to Float64 failed: %v", err)
	}
	if v, ok := casted.Value(0).(float64); !ok || v != 1.0 {
		t.Fatalf("casted value[0]: got %v, want 1.0", casted.Value(0))
	}
}

func TestCastFloat64ToString(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "f",
		DType:  polars.Float64,
		Values: []any{float64(1.5), float64(2.5), float64(3.5)},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	casted, err := s.Cast(polars.String)
	if err != nil {
		t.Fatalf("cast to String failed: %v", err)
	}
	if casted.Len() != 3 {
		t.Fatalf("casted len: got %d, want 3", casted.Len())
	}
}

func TestCastStringToInt64(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "s",
		DType:  polars.String,
		Values: []any{"1", "2", "3"},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	casted, err := s.Cast(polars.Int64)
	if err != nil {
		t.Fatalf("cast to Int64 failed: %v", err)
	}
	if v, ok := casted.Value(0).(int64); !ok || v != 1 {
		t.Fatalf("casted value[0]: got %v, want 1", casted.Value(0))
	}
}

func TestCastBoolToInt64(t *testing.T) {
	t.Parallel()

	// Python Polars casts Bool to Int64 (True->1, False->0); gopolars matches.
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "b",
		DType:  polars.Boolean,
		Values: []any{true, false, true},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	casted, err := s.Cast(polars.Int64)
	if err != nil {
		t.Fatalf("Bool->Int64 cast failed: %v", err)
	}
	if casted.DataType() != polars.Int64 {
		t.Fatalf("casted dtype: got %v, want Int64", casted.DataType())
	}
	for i, w := range []int64{1, 0, 1} {
		if v, ok := casted.Value(i).(int64); !ok || v != w {
			t.Fatalf("casted[%d]: got %T(%v), want %d", i, casted.Value(i), casted.Value(i), w)
		}
	}
}

func TestCastPreservesNulls(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), nil, int64(3)},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	casted, err := s.Cast(polars.Float64)
	if err != nil {
		t.Fatalf("cast to Float64 failed: %v", err)
	}
	if casted.Value(0) == nil {
		t.Fatalf("casted value[0]: should not be nil")
	}
	if casted.Value(1) != nil {
		t.Fatalf("casted value[1]: should be nil, got %v", casted.Value(1))
	}
	if casted.Value(2) == nil {
		t.Fatalf("casted value[2]: should not be nil")
	}
}
