package series

// Ported from py-polars/tests/unit/series/test_extend.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestExtendBasic(t *testing.T) {
	t.Parallel()

	s1, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	s2, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(3), int64(4)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result, err := s1.Extend(s2)
	if err != nil {
		t.Fatalf("extend failed: %v", err)
	}
	if result.Len() != 4 {
		t.Fatalf("extend len: got %d, want 4", result.Len())
	}
	expected := []int64{1, 2, 3, 4}
	for i, exp := range expected {
		v, ok := result.Value(i).(int64)
		if !ok || v != exp {
			t.Fatalf("extend[%d]: got %v, want %d", i, result.Value(i), exp)
		}
	}
}

func TestExtendWithNulls(t *testing.T) {
	t.Parallel()

	s1, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	s2, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{nil, int64(4)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result, err := s1.Extend(s2)
	if err != nil {
		t.Fatalf("extend with nulls failed: %v", err)
	}
	if result.Len() != 4 {
		t.Fatalf("extend len: got %d, want 4", result.Len())
	}
	if result.Value(0).(int64) != 1 {
		t.Fatalf("extend[0]: got %v, want 1", result.Value(0))
	}
	if result.Value(2) != nil {
		t.Fatalf("extend[2]: should be null, got %v", result.Value(2))
	}
}

func TestExtendDtypeMismatch(t *testing.T) {
	t.Parallel()

	s1, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	s2, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Float64, Values: []any{float64(3.0), float64(4.0)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	_, err = s1.Extend(s2)
	if err == nil {
		t.Fatalf("expected error for dtype mismatch, got nil")
	}
}
