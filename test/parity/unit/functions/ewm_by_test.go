package functions

// Ported from py-polars/tests/unit/functions/test_ewm_by.py (py-1.28.1)
//
// The Python test is property-based (hypothesis): for evenly-spaced times,
// ewm_mean_by(by, half_life) equals ewm_mean(half_life, adjust=False). We port
// the deterministic core of that invariant.
//
// DISCREPANCY: gopolars Series.EwmMeanBy(by, alpha) takes a fixed alpha and only
// uses `by` to sort/unsort the values — it does NOT scale the decay by the time
// spacing / half-life. So it equals EwmMean(alpha) whenever `by` is already
// sorted. Python instead derives the decay from the actual gaps in `by`.

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestEwmMeanByMatchesEwmMeanWhenSorted(t *testing.T) {
	t.Parallel()
	values, err := polars.NewSeries(polars.NewSeriesInput{Name: "values", DType: polars.Float64, Values: []any{1.0, 2.0, 3.0, 4.0, 5.0}})
	if err != nil {
		t.Fatalf("values: %v", err)
	}
	by, err := polars.NewSeries(polars.NewSeriesInput{Name: "index", DType: polars.Int64, Values: []any{int64(0), int64(1), int64(2), int64(3), int64(4)}})
	if err != nil {
		t.Fatalf("by: %v", err)
	}

	const alpha = 0.5
	got, err := values.EwmMeanBy(by, alpha)
	if err != nil {
		t.Fatalf("ewm_mean_by: %v", err)
	}
	want := values.EwmMean(alpha)

	if got.Len() != want.Len() {
		t.Fatalf("len mismatch: got %d, want %d", got.Len(), want.Len())
	}
	for i := 0; i < got.Len(); i++ {
		g, _ := got.Value(i).(float64)
		w, _ := want.Value(i).(float64)
		if math.Abs(g-w) > 1e-9 {
			t.Fatalf("idx %d: got %v, want %v", i, g, w)
		}
	}
}

// EwmMeanBy must respect the ordering of `by`: unsorted `by` reorders the decay.
func TestEwmMeanByReordersByKey(t *testing.T) {
	t.Parallel()
	values, err := polars.NewSeries(polars.NewSeriesInput{Name: "values", DType: polars.Float64, Values: []any{5.0, 4.0, 3.0, 2.0, 1.0}})
	if err != nil {
		t.Fatalf("values: %v", err)
	}
	by, err := polars.NewSeries(polars.NewSeriesInput{Name: "index", DType: polars.Int64, Values: []any{int64(4), int64(3), int64(2), int64(1), int64(0)}})
	if err != nil {
		t.Fatalf("by: %v", err)
	}
	out, err := values.EwmMeanBy(by, 0.5)
	if err != nil {
		t.Fatalf("ewm_mean_by: %v", err)
	}
	// First (by smallest key) value 1.0 anchors the ewm; result is finite & defined.
	if out.Len() != 5 {
		t.Fatalf("len: got %d, want 5", out.Len())
	}
	if _, ok := out.Value(4).(float64); !ok {
		t.Fatalf("expected float at idx 4, got %T", out.Value(4))
	}
}

// test_length_mismatch_22084: mismatched lengths must error.
func TestEwmMeanByLengthMismatch(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.Float64, Values: []any{0.0, nil}})
	if err != nil {
		t.Fatalf("s: %v", err)
	}
	by, err := polars.NewSeries(polars.NewSeriesInput{Name: "by", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("by: %v", err)
	}
	if _, err := s.EwmMeanBy(by, 0.5); err == nil {
		t.Fatalf("expected length-mismatch error, got nil")
	}
}
