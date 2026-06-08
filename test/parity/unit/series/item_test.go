package series

// Ported from py-polars/tests/unit/series/test_item.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestItemInt(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(42)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	v, err := s.Item(0)
	if err != nil {
		t.Fatalf("item failed: %v", err)
	}
	if vi, ok := v.(int64); !ok || vi != 42 {
		t.Fatalf("item(0): got %v, want 42", v)
	}
}

func TestItemString(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.String, Values: []any{"hello"}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	v, err := s.Item(0)
	if err != nil {
		t.Fatalf("item failed: %v", err)
	}
	if vs, ok := v.(string); !ok || vs != "hello" {
		t.Fatalf("item(0): got %v, want hello", v)
	}
}

func TestItemBool(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Boolean, Values: []any{true}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	v, err := s.Item(0)
	if err != nil {
		t.Fatalf("item failed: %v", err)
	}
	if vb, ok := v.(bool); !ok || vb != true {
		t.Fatalf("item(0): got %v, want true", v)
	}
}

func TestItemFloat(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "f", DType: polars.Float64, Values: []any{float64(3.14)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	v, err := s.Item(0)
	if err != nil {
		t.Fatalf("item failed: %v", err)
	}
	if vf, ok := v.(float64); !ok || vf != 3.14 {
		t.Fatalf("item(0): got %v, want 3.14", v)
	}
}

func TestItemNull(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{nil}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	v, err := s.Item(0)
	if err != nil {
		t.Fatalf("item failed: %v", err)
	}
	if v != nil {
		t.Fatalf("item(0) on null: got %v, want nil", v)
	}
}
