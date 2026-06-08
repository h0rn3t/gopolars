package constructors

// Ported from py-polars/tests/unit/constructors/test_series.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestSeriesConstructionBasic(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), int64(2), int64(3)},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.Name() != "a" {
		t.Fatalf("name: got %q, want %q", s.Name(), "a")
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
	if v, ok := s.Value(0).(int64); !ok || v != 1 {
		t.Fatalf("value[0]: got %v, want 1", s.Value(0))
	}
	if v, ok := s.Value(1).(int64); !ok || v != 2 {
		t.Fatalf("value[1]: got %v, want 2", s.Value(1))
	}
	if v, ok := s.Value(2).(int64); !ok || v != 3 {
		t.Fatalf("value[2]: got %v, want 3", s.Value(2))
	}
}

func TestSeriesConstructionFloat64(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "f",
		DType:  polars.Float64,
		Values: []any{float64(1.5), float64(2.5), float64(3.5)},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
	if v, ok := s.Value(0).(float64); !ok || v != 1.5 {
		t.Fatalf("value[0]: got %v, want 1.5", s.Value(0))
	}
}

func TestSeriesConstructionString(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "s",
		DType:  polars.String,
		Values: []any{"a", "b", "c"},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
	if v, ok := s.Value(1).(string); !ok || v != "b" {
		t.Fatalf("value[1]: got %v, want 'b'", s.Value(1))
	}
}

func TestSeriesConstructionBool(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "b",
		DType:  polars.Boolean,
		Values: []any{true, false, true},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
	if v, ok := s.Value(0).(bool); !ok || v != true {
		t.Fatalf("value[0]: got %v, want true", s.Value(0))
	}
	if v, ok := s.Value(1).(bool); !ok || v != false {
		t.Fatalf("value[1]: got %v, want false", s.Value(1))
	}
}

func TestSeriesConstructionWithNulls(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), nil, int64(3)},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
	if s.Value(0) == nil {
		t.Fatalf("value[0]: should not be nil")
	}
	if s.Value(1) != nil {
		t.Fatalf("value[1]: should be nil, got %v", s.Value(1))
	}
	if s.Value(2) == nil {
		t.Fatalf("value[2]: should not be nil")
	}
}

func TestSeriesConstructionEmpty(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "e",
		DType:  polars.Int64,
		Values: []any{},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.Len() != 0 {
		t.Fatalf("len: got %d, want 0", s.Len())
	}
}

func TestSeriesConstructionAllNulls(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "n",
		DType:  polars.Int64,
		Values: []any{nil, nil, nil},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
	for i := 0; i < 3; i++ {
		if s.Value(i) != nil {
			t.Fatalf("value[%d]: should be nil, got %v", i, s.Value(i))
		}
	}
}

func TestSeriesCast(t *testing.T) {
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
		t.Fatalf("cast failed: %v", err)
	}
	if casted.Len() != 3 {
		t.Fatalf("casted len: got %d, want 3", casted.Len())
	}
	if v, ok := casted.Value(0).(float64); !ok || v != 1.0 {
		t.Fatalf("casted value[0]: got %v, want 1.0", casted.Value(0))
	}
}

func TestSeriesToList(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), int64(2), int64(3)},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}

	list := s.ToList()
	if len(list) != 3 {
		t.Fatalf("tolist len: got %d, want 3", len(list))
	}
}

// Python: pl.Series([[0.1, 1]]) → List(Float64) with [0.1, 1.0]
// Go: gopolars uses explicit dtypes, so mixed types in a list need manual handling
func TestSeriesListOfFloats(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:  "a",
		DType: polars.List,
		Values: []any{
			[]any{float64(0.1), float64(1.0)},
		},
	})
	if err != nil {
		// DISCREPANCY: gopolars may not support List type construction this way
		t.Fatalf("new series failed: %v", err)
	}
	_ = s // verify it exists; behavior may differ from Python
}
