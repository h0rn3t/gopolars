package operations

// Ported from py-polars/tests/unit/operations/test_replace_strict.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// replace_strict maps known values; here a simple single-value remap.
func TestReplaceStrict(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(1)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.ReplaceStrict(int64(1), int64(100))
	if err != nil {
		t.Fatalf("ReplaceStrict failed: %v", err)
	}
	for i, w := range []int64{100, 2, 100} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("replace_strict[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}
