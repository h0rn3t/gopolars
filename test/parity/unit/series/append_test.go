package series

// Ported from py-polars/tests/unit/series/test_append.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestAppend(t *testing.T) {
	t.Parallel()

	s1, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	s2, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(4), int64(5)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result, err := s1.Append(s2)
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if result.Len() != 5 {
		t.Fatalf("append len: got %d, want 5", result.Len())
	}
	expected := []int64{1, 2, 3, 4, 5}
	for i, exp := range expected {
		v, ok := result.Value(i).(int64)
		if !ok || v != exp {
			t.Fatalf("append[%d]: got %v, want %d", i, result.Value(i), exp)
		}
	}
	// Append should keep the name of the first series
	if result.Name() != "a" {
		t.Fatalf("append name: got %q, want %q", result.Name(), "a")
	}
}

func TestAppendSelf(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result, err := s.Append(s)
	if err != nil {
		t.Fatalf("append self failed: %v", err)
	}
	if result.Len() != 4 {
		t.Fatalf("append self len: got %d, want 4", result.Len())
	}
	expected := []int64{1, 2, 1, 2}
	for i, exp := range expected {
		v, ok := result.Value(i).(int64)
		if !ok || v != exp {
			t.Fatalf("append self[%d]: got %v, want %d", i, result.Value(i), exp)
		}
	}
}

func TestAppendDtypeMismatch(t *testing.T) {
	t.Parallel()

	s1, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	s2, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Float64, Values: []any{float64(3.0), float64(4.0)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	_, err = s1.Append(s2)
	if err == nil {
		t.Fatalf("expected error for dtype mismatch, got nil")
	}
}

func TestAppendNullSeries(t *testing.T) {
	t.Parallel()

	// Python: appending a null series to an Int64 series converts nulls
	s1, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	s2, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{nil, nil}})
	if err != nil {
		t.Fatalf("new series with nulls failed: %v", err)
	}

	result, err := s1.Append(s2)
	if err != nil {
		t.Fatalf("append null series failed: %v", err)
	}
	if result.Len() != 4 {
		t.Fatalf("append null series len: got %d, want 4", result.Len())
	}
}

func TestExtend(t *testing.T) {
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
}
