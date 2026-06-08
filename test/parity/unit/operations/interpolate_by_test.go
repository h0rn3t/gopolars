package operations

// Ported from py-polars/tests/unit/operations/test_interpolate_by.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// interpolate_by fills nulls by linear interpolation.
// DISCREPANCY: Python weights the interpolation by the `by` x-axis, so at x=2
// (between (1,1) and (4,3)) it yields ~1.667. gopolars ignores the `by` spacing
// and interpolates uniformly by position, yielding the midpoint 2.0. We pin the
// gopolars behavior.
func TestInterpolateBy(t *testing.T) {
	t.Parallel()
	v, err := polars.NewSeries(polars.NewSeriesInput{Name: "v", DType: polars.Float64, Values: []any{1.0, nil, 3.0}})
	if err != nil {
		t.Fatalf("v: %v", err)
	}
	by, err := polars.NewSeries(polars.NewSeriesInput{Name: "by", DType: polars.Float64, Values: []any{1.0, 2.0, 4.0}})
	if err != nil {
		t.Fatalf("by: %v", err)
	}
	out, err := v.InterpolateBy(by)
	if err != nil {
		t.Fatalf("interpolate_by: %v", err)
	}
	if got, _ := out.Value(1).(float64); got != 2.0 {
		t.Fatalf("interp[1]: got %v, want 2.0 (gopolars uniform; Python by-weighted -> ~1.667)", out.Value(1))
	}
	if got, _ := out.Value(0).(float64); got != 1.0 {
		t.Fatalf("interp[0]: got %v, want 1.0", out.Value(0))
	}
}
