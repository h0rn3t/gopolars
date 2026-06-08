package operations

// Ported from py-polars/tests/unit/operations/test_statistics.py (py-1.28.1, representative subset)

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func statsSeries(t *testing.T) polars.Series {
	t.Helper()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{1.0, 2.0, 3.0, 4.0, 5.0}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	return s
}

func TestMean(t *testing.T) {
	t.Parallel()
	if got := statsSeries(t).Mean(); math.Abs(got-3.0) > 1e-9 {
		t.Fatalf("mean: got %v, want 3.0", got)
	}
}

func TestMedian(t *testing.T) {
	t.Parallel()
	if got := statsSeries(t).Median(); math.Abs(got-3.0) > 1e-9 {
		t.Fatalf("median: got %v, want 3.0", got)
	}
}

// Sample standard deviation (ddof=1): std of [1..5] is sqrt(2.5).
func TestStd(t *testing.T) {
	t.Parallel()
	if got := statsSeries(t).Std(); math.Abs(got-math.Sqrt(2.5)) > 1e-9 {
		t.Fatalf("std: got %v, want %v", got, math.Sqrt(2.5))
	}
}

// Sample variance (ddof=1) of [1..5] is 2.5.
func TestVar(t *testing.T) {
	t.Parallel()
	if got := statsSeries(t).Var(); math.Abs(got-2.5) > 1e-9 {
		t.Fatalf("var: got %v, want 2.5", got)
	}
}

func TestQuantileMedian(t *testing.T) {
	t.Parallel()
	if got := statsSeries(t).Quantile(0.5); math.Abs(got-3.0) > 1e-9 {
		t.Fatalf("quantile(0.5): got %v, want 3.0", got)
	}
}

func TestSum(t *testing.T) {
	t.Parallel()
	if got := statsSeries(t).Sum(); math.Abs(got-15.0) > 1e-9 {
		t.Fatalf("sum: got %v, want 15.0", got)
	}
}
