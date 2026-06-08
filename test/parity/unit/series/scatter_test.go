package series

// Ported from py-polars/tests/unit/series/test_scatter.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestScatterBasic(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result, err := s.Scatter([]int{0, 2}, []any{int64(10), int64(30)})
	if err != nil {
		t.Fatalf("scatter failed: %v", err)
	}
	if result.Len() != 4 {
		t.Fatalf("scatter len: got %d, want 4", result.Len())
	}
	if v, ok := result.Value(0).(int64); !ok || v != 10 {
		t.Fatalf("scatter[0]: got %v, want 10", result.Value(0))
	}
	if v, ok := result.Value(1).(int64); !ok || v != 2 {
		t.Fatalf("scatter[1]: got %v, want 2", result.Value(1))
	}
	if v, ok := result.Value(2).(int64); !ok || v != 30 {
		t.Fatalf("scatter[2]: got %v, want 30", result.Value(2))
	}
	if v, ok := result.Value(3).(int64); !ok || v != 4 {
		t.Fatalf("scatter[3]: got %v, want 4", result.Value(3))
	}
}

func TestScatterSingleIndex(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result, err := s.Scatter([]int{1}, []any{int64(20)})
	if err != nil {
		t.Fatalf("scatter single failed: %v", err)
	}
	if v, ok := result.Value(0).(int64); !ok || v != 1 {
		t.Fatalf("scatter[0]: got %v, want 1", result.Value(0))
	}
	if v, ok := result.Value(1).(int64); !ok || v != 20 {
		t.Fatalf("scatter[1]: got %v, want 20", result.Value(1))
	}
	if v, ok := result.Value(2).(int64); !ok || v != 3 {
		t.Fatalf("scatter[2]: got %v, want 3", result.Value(2))
	}
}

func TestScatterWithNull(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	result, err := s.Scatter([]int{0}, []any{nil})
	if err != nil {
		// DISCREPANCY: gopolars may not support replacing with nil via Scatter
		t.Fatalf("scatter with nil — %v", err)
	}
	if result.Value(0) != nil {
		t.Fatalf("scatter[0]: should be nil, got %v", result.Value(0))
	}
}
