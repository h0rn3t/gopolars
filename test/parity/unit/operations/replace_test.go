package operations

// Ported from py-polars/tests/unit/operations/test_replace.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Replace a single value with another.
func TestReplaceSingle(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Replace(int64(2), int64(20))
	for i, w := range []int64{1, 20, 20, 3} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("replace[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}

// Replacing a value not present leaves the series unchanged.
func TestReplaceMissingNoop(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Replace(int64(99), int64(0))
	for i, w := range []int64{1, 2, 3} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("replace[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}

// Replace string values.
func TestReplaceString(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.String, Values: []any{"x", "y", "x"}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Replace("x", "z")
	for i, w := range []string{"z", "y", "z"} {
		if v, _ := out.Value(i).(string); v != w {
			t.Fatalf("replace[%d]: got %v, want %s", i, out.Value(i), w)
		}
	}
}
