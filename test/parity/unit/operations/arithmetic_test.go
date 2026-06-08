package operations

// Ported from py-polars/tests/unit/operations/arithmetic/test_arithmetic.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func arithPair(t *testing.T) (polars.Series, polars.Series) {
	t.Helper()
	a, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(10), int64(20), int64(30)}})
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	return a, b
}

func TestSeriesAdd(t *testing.T) {
	t.Parallel()
	a, b := arithPair(t)
	out, err := a.Add(b)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	for i, w := range []float64{11, 22, 33} {
		if toFloatAny(out.Value(i)) != w {
			t.Fatalf("add[%d]: got %v, want %v", i, out.Value(i), w)
		}
	}
}

func TestSeriesSub(t *testing.T) {
	t.Parallel()
	a, b := arithPair(t)
	out, err := a.Sub(b)
	if err != nil {
		t.Fatalf("sub: %v", err)
	}
	for i, w := range []float64{9, 18, 27} {
		if toFloatAny(out.Value(i)) != w {
			t.Fatalf("sub[%d]: got %v, want %v", i, out.Value(i), w)
		}
	}
}

func TestSeriesMul(t *testing.T) {
	t.Parallel()
	a, b := arithPair(t)
	out, err := a.Mul(b)
	if err != nil {
		t.Fatalf("mul: %v", err)
	}
	for i, w := range []float64{10, 40, 90} {
		if toFloatAny(out.Value(i)) != w {
			t.Fatalf("mul[%d]: got %v, want %v", i, out.Value(i), w)
		}
	}
}

// String "+" concatenates element-wise (matching Polars).
func TestStringConcatViaAdd(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "x", Values: []any{"a", "b", "c"}},
		{Name: "y", Values: []any{"1", "2", "3"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Select(polars.Col("x").Add(polars.Col("y")).Alias("xy"))
	if err != nil {
		t.Fatalf("string add: %v", err)
	}
	s, _ := out.GetColumn("xy")
	for i, w := range []string{"a1", "b2", "c3"} {
		if v, _ := s.Value(i).(string); v != w {
			t.Fatalf("xy[%d]: got %v, want %s", i, s.Value(i), w)
		}
	}
}

func TestSeriesDiv(t *testing.T) {
	t.Parallel()
	a, b := arithPair(t)
	out, err := a.Div(b)
	if err != nil {
		t.Fatalf("div: %v", err)
	}
	for i, w := range []float64{10, 10, 10} {
		if toFloatAny(out.Value(i)) != w {
			t.Fatalf("div[%d]: got %v, want %v", i, out.Value(i), w)
		}
	}
}
