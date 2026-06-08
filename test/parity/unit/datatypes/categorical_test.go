package datatypes

// Ported from py-polars/tests/unit/datatypes/test_categorical.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Categorical construction preserves the string values; Cat().Codes() assigns
// integer codes in first-seen order.
func TestCategoricalCodes(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "c", DType: polars.Categorical, Values: []any{"a", "b", "a", "c", "b"}})
	if err != nil {
		t.Fatalf("new categorical: %v", err)
	}
	if s.DataType() != polars.Categorical {
		t.Fatalf("dtype: got %v, want Categorical", s.DataType())
	}
	if v, ok := s.Value(0).(string); !ok || v != "a" {
		t.Fatalf("value[0]: got %v, want a", s.Value(0))
	}
	codes, err := s.Cat().Codes()
	if err != nil {
		t.Fatalf("codes: %v", err)
	}
	// first-seen order: a=0, b=1, c=2
	for i, w := range []int64{0, 1, 0, 2, 1} {
		if v, ok := codes.Value(i).(int64); !ok || v != w {
			t.Fatalf("code[%d]: got %v, want %d", i, codes.Value(i), w)
		}
	}
}

// Categorical with nulls: null codes stay null.
func TestCategoricalNullCodes(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "c", DType: polars.Categorical, Values: []any{"x", nil, "x"}})
	if err != nil {
		t.Fatalf("new categorical: %v", err)
	}
	codes, err := s.Cat().Codes()
	if err != nil {
		t.Fatalf("codes: %v", err)
	}
	if codes.Value(1) != nil {
		t.Fatalf("code[1]: got %v, want nil", codes.Value(1))
	}
	if v, ok := codes.Value(0).(int64); !ok || v != 0 {
		t.Fatalf("code[0]: got %v, want 0", codes.Value(0))
	}
}
