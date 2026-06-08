package operations

// Ported from py-polars/tests/unit/operations/test_shift.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func shiftSeries(t *testing.T) polars.Series {
	t.Helper()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	return s
}

// Positive shift moves values down, filling the head with nulls.
func TestShiftPositive(t *testing.T) {
	t.Parallel()
	out := shiftSeries(t).Shift(1)
	if out.Value(0) != nil {
		t.Fatalf("shift[0]: got %v, want nil", out.Value(0))
	}
	for i, w := range []int64{1, 2, 3} {
		if v, _ := out.Value(i + 1).(int64); v != w {
			t.Fatalf("shift[%d]: got %v, want %d", i+1, out.Value(i+1), w)
		}
	}
}

// Negative shift moves values up, filling the tail with nulls.
func TestShiftNegative(t *testing.T) {
	t.Parallel()
	out := shiftSeries(t).Shift(-1)
	for i, w := range []int64{2, 3, 4} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("shift[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
	if out.Value(3) != nil {
		t.Fatalf("shift[3]: got %v, want nil", out.Value(3))
	}
}

// Shift by zero is identity.
func TestShiftZero(t *testing.T) {
	t.Parallel()
	out := shiftSeries(t).Shift(0)
	for i, w := range []int64{1, 2, 3, 4} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("shift0[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}
