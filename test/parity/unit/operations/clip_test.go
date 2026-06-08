package operations

// Ported from py-polars/tests/unit/operations/test_clip.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// clip bounds values to [lower, upper].
func TestClipBoth(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(-5), int64(0), int64(5), int64(10), int64(15)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Clip(0, 10)
	want := []float64{0, 0, 5, 10, 10}
	for i, w := range want {
		switch v := out.Value(i).(type) {
		case int64:
			if float64(v) != w {
				t.Fatalf("clip[%d]: got %d, want %v", i, v, w)
			}
		case float64:
			if v != w {
				t.Fatalf("clip[%d]: got %v, want %v", i, v, w)
			}
		default:
			t.Fatalf("clip[%d]: unexpected type %T", i, out.Value(i))
		}
	}
}

func TestClipFloat(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{-1.5, 2.5, 11.0}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Clip(0.0, 10.0)
	want := []float64{0.0, 2.5, 10.0}
	for i, w := range want {
		if v, _ := out.Value(i).(float64); v != w {
			t.Fatalf("clip[%d]: got %v, want %v", i, out.Value(i), w)
		}
	}
}
