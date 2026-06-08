package operations

// Ported from py-polars/tests/unit/operations/map/test_map_elements.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// map_elements applies a per-element function.
func TestMapElements(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.MapElements(func(v any) any {
		if n, ok := v.(int64); ok {
			return n * 10
		}
		return v
	})
	if err != nil {
		t.Fatalf("map_elements: %v", err)
	}
	for i, w := range []int64{10, 20, 30} {
		switch v := out.Value(i).(type) {
		case int64:
			if v != w {
				t.Fatalf("map[%d]: got %d, want %d", i, v, w)
			}
		case float64:
			if v != float64(w) {
				t.Fatalf("map[%d]: got %v, want %d", i, v, w)
			}
		default:
			t.Fatalf("map[%d]: unexpected type %T", i, out.Value(i))
		}
	}
}
