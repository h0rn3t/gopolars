package operations

// Ported from py-polars/tests/unit/operations/aggregation/test_horizontal.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func horizDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "b", Values: []any{int64(10), int64(20), int64(30)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

func firstColValues(t *testing.T, df polars.DataFrame, name string) polars.Series {
	t.Helper()
	s, err := df.GetColumn(name)
	if err != nil {
		t.Fatalf("get %s: %v", name, err)
	}
	return s
}

func TestSumHorizontal(t *testing.T) {
	t.Parallel()
	out, err := horizDF(t).SumHorizontal("sum")
	if err != nil {
		t.Fatalf("sum_horizontal: %v", err)
	}
	s := firstColValues(t, out, "sum")
	for i, w := range []float64{11, 22, 33} {
		got := toFloatAny(s.Value(i))
		if got != w {
			t.Fatalf("sum_h[%d]: got %v, want %v", i, s.Value(i), w)
		}
	}
}

func TestMaxHorizontal(t *testing.T) {
	t.Parallel()
	out, err := horizDF(t).MaxHorizontal("max")
	if err != nil {
		t.Fatalf("max_horizontal: %v", err)
	}
	s := firstColValues(t, out, "max")
	for i, w := range []float64{10, 20, 30} {
		if toFloatAny(s.Value(i)) != w {
			t.Fatalf("max_h[%d]: got %v, want %v", i, s.Value(i), w)
		}
	}
}

func TestMinHorizontal(t *testing.T) {
	t.Parallel()
	out, err := horizDF(t).MinHorizontal("min")
	if err != nil {
		t.Fatalf("min_horizontal: %v", err)
	}
	s := firstColValues(t, out, "min")
	for i, w := range []float64{1, 2, 3} {
		if toFloatAny(s.Value(i)) != w {
			t.Fatalf("min_h[%d]: got %v, want %v", i, s.Value(i), w)
		}
	}
}

func TestMeanHorizontal(t *testing.T) {
	t.Parallel()
	out, err := horizDF(t).MeanHorizontal("mean")
	if err != nil {
		t.Fatalf("mean_horizontal: %v", err)
	}
	s := firstColValues(t, out, "mean")
	for i, w := range []float64{5.5, 11, 16.5} {
		if toFloatAny(s.Value(i)) != w {
			t.Fatalf("mean_h[%d]: got %v, want %v", i, s.Value(i), w)
		}
	}
}

func toFloatAny(v any) float64 {
	switch x := v.(type) {
	case int64:
		return float64(x)
	case float64:
		return x
	default:
		return -1e18
	}
}
