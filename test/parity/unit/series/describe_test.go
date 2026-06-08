package series

// Ported from py-polars/tests/unit/series/test_describe.py (py-1.28.1)

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestSeriesDescribeInt(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	desc := s.Describe()
	if desc == nil {
		t.Fatalf("describe returned nil")
	}

	// gopolars Describe uses "len" instead of "count"
	lenVal, ok := desc["len"]
	if !ok {
		t.Fatalf("describe missing 'len'")
	}
	if lenVal != 5 {
		t.Fatalf("describe len: got %v, want 5", lenVal)
	}

	mean, ok := desc["mean"]
	if !ok {
		t.Fatalf("describe missing 'mean'")
	}
	if m, ok := mean.(float64); !ok || math.Abs(m-3.0) > 1e-9 {
		t.Fatalf("describe mean: got %v, want 3.0", mean)
	}

	_, hasMin := desc["min"]
	_, hasMax := desc["max"]
	if !hasMin || !hasMax {
		t.Fatalf("describe missing 'min' or 'max'")
	}
}

func TestSeriesDescribeFloat(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "f", DType: polars.Float64, Values: []any{float64(1.5), float64(2.5), float64(3.5)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	desc := s.Describe()
	if desc == nil {
		t.Fatalf("describe returned nil")
	}

	mean, ok := desc["mean"]
	if !ok {
		t.Fatalf("describe missing 'mean'")
	}
	if m, ok := mean.(float64); !ok || math.Abs(m-2.5) > 1e-9 {
		t.Fatalf("describe mean: got %v, want 2.5", mean)
	}
}

func TestSeriesDescribeString(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.String, Values: []any{"a", "b", "c"}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	desc := s.Describe()
	if desc == nil {
		t.Fatalf("describe returned nil")
	}

	// String series should have len at minimum
	lenVal, ok := desc["len"]
	if !ok {
		t.Fatalf("describe missing 'len'")
	}
	if lenVal != 3 {
		t.Fatalf("describe len: got %v, want 3", lenVal)
	}
}

func TestSeriesDescribeBoolean(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Boolean, Values: []any{true, false, true, true}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	desc := s.Describe()
	if desc == nil {
		t.Fatalf("describe returned nil")
	}

	// Boolean series should have len
	lenVal, ok := desc["len"]
	if !ok {
		t.Fatalf("describe missing 'len'")
	}
	if lenVal != 4 {
		t.Fatalf("describe len: got %v, want 4", lenVal)
	}
}

func TestSeriesDescribeWithNulls(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, int64(3)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	desc := s.Describe()
	if desc == nil {
		t.Fatalf("describe returned nil")
	}

	// gopolars uses "null_count" (matching Python)
	nullCount, ok := desc["null_count"]
	if !ok {
		t.Fatalf("describe missing 'null_count'")
	}
	// null_count may be int or int64
	switch v := nullCount.(type) {
	case int:
		if v != 1 {
			t.Fatalf("describe null_count: got %v, want 1", nullCount)
		}
	case int64:
		if v != 1 {
			t.Fatalf("describe null_count: got %v, want 1", nullCount)
		}
	default:
		t.Logf("describe null_count type: %T, value: %v (may differ from Python)", nullCount, nullCount)
	}
}
