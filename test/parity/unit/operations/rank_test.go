package operations

// Ported from py-polars/tests/unit/operations/test_rank.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// rank assigns ordinal ranks to values (ascending).
func TestRankAscending(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(30), int64(10), int64(20)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Rank()
	// values 30,10,20 -> ranks 3,1,2
	want := []float64{3, 1, 2}
	for i, w := range want {
		switch v := out.Value(i).(type) {
		case int64:
			if float64(v) != w {
				t.Fatalf("rank[%d]: got %d, want %v", i, v, w)
			}
		case float64:
			if v != w {
				t.Fatalf("rank[%d]: got %v, want %v", i, v, w)
			}
		default:
			t.Fatalf("rank[%d]: unexpected type %T", i, out.Value(i))
		}
	}
}

func TestRankLength(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{1.5, 2.5, 0.5, 3.5}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if out := s.Rank(); out.Len() != 4 {
		t.Fatalf("rank len: got %d, want 4", out.Len())
	}
}
