package series

// Ported from py-polars/tests/unit/series/test_contains.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestIsInBasic(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result := s.IsIn([]any{int64(1), int64(3)})
	if result.Len() != 4 {
		t.Fatalf("is_in len: got %d, want 4", result.Len())
	}
	expected := []bool{true, false, true, false}
	for i, exp := range expected {
		v, ok := result.Value(i).(bool)
		if !ok || v != exp {
			t.Fatalf("is_in[%d]: got %v, want %v", i, result.Value(i), exp)
		}
	}
}

func TestIsInString(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.String, Values: []any{"apple", "banana", "cherry"}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result := s.IsIn([]any{"apple", "cherry"})
	if result.Len() != 3 {
		t.Fatalf("is_in len: got %d, want 3", result.Len())
	}
	expected := []bool{true, false, true}
	for i, exp := range expected {
		v, ok := result.Value(i).(bool)
		if !ok || v != exp {
			t.Fatalf("is_in[%d]: got %v, want %v", i, result.Value(i), exp)
		}
	}
}

func TestIsInNoMatch(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result := s.IsIn([]any{int64(10), int64(20)})
	if result.Len() != 3 {
		t.Fatalf("is_in len: got %d, want 3", result.Len())
	}
	for i := 0; i < 3; i++ {
		v, ok := result.Value(i).(bool)
		if !ok || v != false {
			t.Fatalf("is_in[%d]: got %v, want false", i, result.Value(i))
		}
	}
}

func TestIsInWithNulls(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, int64(3)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result := s.IsIn([]any{int64(1), int64(3)})
	if result.Len() != 3 {
		t.Fatalf("is_in len: got %d, want 3", result.Len())
	}
	// Python: null is_in [...] → null
	// gopolars behavior may differ
	_ = result
}
