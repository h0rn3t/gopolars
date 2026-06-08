package operations

// Ported from py-polars/tests/unit/operations/test_hist.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// hist buckets values into `bins` bins, returning a frame with bin + count columns
// whose counts sum to the number of (non-null) observations.
func TestHist(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.Hist(2)
	if err != nil {
		t.Fatalf("hist: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("bins: got %d, want 2", out.Height())
	}
	// find the count column and check counts sum to 6
	cnt, err := out.GetColumn("count")
	if err != nil {
		t.Fatalf("hist count column: %v", err)
	}
	total := 0.0
	for i := 0; i < cnt.Len(); i++ {
		total += toFloatAny(cnt.Value(i))
	}
	if total != 6 {
		t.Fatalf("hist counts sum: got %v, want 6", total)
	}
}
