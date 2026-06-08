package operations

// Ported from py-polars/tests/unit/operations/test_fill_null.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestFillNullSeriesConstant(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, int64(3), nil}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.FillNull(int64(0))
	if err != nil {
		t.Fatalf("fill_null: %v", err)
	}
	for i, w := range []int64{1, 0, 3, 0} {
		if v, ok := out.Value(i).(int64); !ok || v != w {
			t.Fatalf("fill[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
	if out.NullCount() != 0 {
		t.Fatalf("null count after fill: got %d, want 0", out.NullCount())
	}
}

func TestFillNullNoNulls(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.FillNull(int64(99))
	if err != nil {
		t.Fatalf("fill_null: %v", err)
	}
	for i, w := range []int64{1, 2} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("fill[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}

// ForwardFill / BackwardFill strategies.
func TestFillNullForwardBackward(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, nil, int64(4)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	fwd := s.ForwardFill()
	for i, w := range []int64{1, 1, 1, 4} {
		if v, _ := fwd.Value(i).(int64); v != w {
			t.Fatalf("forward[%d]: got %v, want %d", i, fwd.Value(i), w)
		}
	}
	bwd := s.BackwardFill()
	for i, w := range []int64{1, 4, 4, 4} {
		if v, _ := bwd.Value(i).(int64); v != w {
			t.Fatalf("backward[%d]: got %v, want %d", i, bwd.Value(i), w)
		}
	}
}
