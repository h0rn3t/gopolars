package operations

// Ported from py-polars/tests/unit/operations/arithmetic/test_pow.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestPowSquare(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Pow(2)
	for i, w := range []float64{1, 4, 9, 16} {
		if toFloatAny(out.Value(i)) != w {
			t.Fatalf("pow2[%d]: got %v, want %v", i, out.Value(i), w)
		}
	}
}

func TestPowZero(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(5), int64(7)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Pow(0)
	for i := 0; i < out.Len(); i++ {
		if toFloatAny(out.Value(i)) != 1 {
			t.Fatalf("pow0[%d]: got %v, want 1", i, out.Value(i))
		}
	}
}
