package operations

// Ported from py-polars/tests/unit/operations/test_reshape.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Reshape to a 1-D flat shape returns all elements in order.
// DISCREPANCY: gopolars Reshape requires positive dimensions (no -1 "infer"),
// so the flatten shape is given explicitly as the length.
func TestReshapeFlat(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(10), int64(20), int64(30), int64(40)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.Reshape(4)
	if err != nil {
		t.Fatalf("reshape(4): %v", err)
	}
	if out.Len() != 4 {
		t.Fatalf("reshape len: got %d, want 4", out.Len())
	}
	for i, w := range []int64{10, 20, 30, 40} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("reshape[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}

// DISCREPANCY: Python reshape((2,2)) yields a nested Array Series (2 rows of
// length-2). gopolars has no fixed-size Array dtype, so a 2-D reshape keeps the
// data flat (length preserved). We pin the gopolars behavior.
func TestReshape2DFlat(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.Reshape(2, 2)
	if err != nil {
		t.Fatalf("reshape(2,2): %v", err)
	}
	if out.Len() != 4 {
		t.Fatalf("reshape(2,2) len: got %d, want 4 (gopolars keeps flat; Python -> 2 rows)", out.Len())
	}
}
