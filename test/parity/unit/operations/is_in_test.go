package operations

// Ported from py-polars/tests/unit/operations/test_is_in.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestIsInInts(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.IsIn([]any{int64(2), int64(4)})
	for i, w := range []bool{false, true, false, true} {
		if v, ok := out.Value(i).(bool); !ok || v != w {
			t.Fatalf("is_in[%d]: got %v, want %v", i, out.Value(i), w)
		}
	}
}

func TestIsInStrings(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.String, Values: []any{"x", "y", "z"}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.IsIn([]any{"y", "z"})
	for i, w := range []bool{false, true, true} {
		if v, ok := out.Value(i).(bool); !ok || v != w {
			t.Fatalf("is_in[%d]: got %v, want %v", i, out.Value(i), w)
		}
	}
}

func TestIsInEmpty(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.IsIn([]any{})
	for i := 0; i < out.Len(); i++ {
		if v, ok := out.Value(i).(bool); !ok || v {
			t.Fatalf("is_in empty[%d]: got %v, want false", i, out.Value(i))
		}
	}
}
