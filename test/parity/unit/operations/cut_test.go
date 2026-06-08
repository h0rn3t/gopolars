package operations

// Ported from py-polars/tests/unit/operations/test_cut.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// cut bins values into intervals defined by breakpoints.
func TestCut(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{1.0, 2.0, 3.0, 4.0, 5.0}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.Cut([]float64{2.5, 4.5})
	if err != nil {
		t.Fatalf("cut: %v", err)
	}
	if out.Len() != 5 {
		t.Fatalf("cut len: got %d, want 5", out.Len())
	}
	// values below 2.5 share a bin label; values 3,4 share the middle bin.
	if out.Value(0) == nil {
		t.Fatalf("cut[0] is unexpectedly null")
	}
}

func TestCutNumberOfDistinctBins(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{0.0, 10.0, 20.0}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.Cut([]float64{5.0, 15.0})
	if err != nil {
		t.Fatalf("cut: %v", err)
	}
	seen := map[string]bool{}
	for i := 0; i < out.Len(); i++ {
		if v, ok := out.Value(i).(string); ok {
			seen[v] = true
		}
	}
	// three values, three breaks-defined bins
	if len(seen) != 3 {
		t.Fatalf("distinct bins: got %d (%v), want 3", len(seen), seen)
	}
}
