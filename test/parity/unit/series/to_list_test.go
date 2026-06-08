package series

// Ported from py-polars/tests/unit/series/test_to_list.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestToListInt(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	list := s.ToList()
	if len(list) != 3 {
		t.Fatalf("tolist len: got %d, want 3", len(list))
	}
	if v, ok := list[0].(int64); !ok || v != 1 {
		t.Fatalf("tolist[0]: got %v, want 1", list[0])
	}
	if v, ok := list[1].(int64); !ok || v != 2 {
		t.Fatalf("tolist[1]: got %v, want 2", list[1])
	}
	if v, ok := list[2].(int64); !ok || v != 3 {
		t.Fatalf("tolist[2]: got %v, want 3", list[2])
	}
}

func TestToListFloat(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "f", DType: polars.Float64, Values: []any{float64(1.5), float64(2.5)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	list := s.ToList()
	if len(list) != 2 {
		t.Fatalf("tolist len: got %d, want 2", len(list))
	}
	if v, ok := list[0].(float64); !ok || v != 1.5 {
		t.Fatalf("tolist[0]: got %v, want 1.5", list[0])
	}
}

func TestToListString(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.String, Values: []any{"a", "b", "c"}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	list := s.ToList()
	if len(list) != 3 {
		t.Fatalf("tolist len: got %d, want 3", len(list))
	}
	if v, ok := list[0].(string); !ok || v != "a" {
		t.Fatalf("tolist[0]: got %v, want a", list[0])
	}
	if v, ok := list[2].(string); !ok || v != "c" {
		t.Fatalf("tolist[2]: got %v, want c", list[2])
	}
}

func TestToListBool(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Boolean, Values: []any{true, false, true}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	list := s.ToList()
	if len(list) != 3 {
		t.Fatalf("tolist len: got %d, want 3", len(list))
	}
	if v, ok := list[0].(bool); !ok || v != true {
		t.Fatalf("tolist[0]: got %v, want true", list[0])
	}
	if v, ok := list[1].(bool); !ok || v != false {
		t.Fatalf("tolist[1]: got %v, want false", list[1])
	}
}

func TestToListWithNulls(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, int64(3)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	list := s.ToList()
	if len(list) != 3 {
		t.Fatalf("tolist len: got %d, want 3", len(list))
	}
	if list[0] == nil {
		t.Fatalf("tolist[0]: should not be nil")
	}
	if list[1] != nil {
		t.Fatalf("tolist[1]: should be nil, got %v", list[1])
	}
	if list[2] == nil {
		t.Fatalf("tolist[2]: should not be nil")
	}
}

func TestToListEmpty(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "e", DType: polars.Int64, Values: []any{}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	list := s.ToList()
	if len(list) != 0 {
		t.Fatalf("tolist len: got %d, want 0", len(list))
	}
}
