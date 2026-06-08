package datatypes

// Ported from py-polars/tests/unit/datatypes/test_bool.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func newBoolSeries(t *testing.T, name string, vals []any) polars.Series {
	t.Helper()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.Boolean, Values: vals})
	if err != nil {
		t.Fatalf("new bool series: %v", err)
	}
	return s
}

// test_bool_sum_empty: empty bool sum is 0.
func TestBoolSumEmpty(t *testing.T) {
	t.Parallel()
	s := newBoolSeries(t, "x", []any{})
	if got := s.Sum(); got != 0 {
		t.Fatalf("empty bool sum: got %v, want 0", got)
	}
}

// test_bool_min_max: min/max over a Boolean Series follow truthiness and ignore
// nulls. gopolars Min()/Max() return float64 (1.0 true / 0.0 false); we assert
// truthiness, matching Python's boolean min/max.
func TestBoolMinMax(t *testing.T) {
	t.Parallel()
	cases := []struct {
		vals []any
		min  bool
		max  bool
	}{
		{[]any{nil, true}, true, true},
		{[]any{nil, true, false}, false, true},
		{[]any{false, true}, false, true},
		{[]any{true, true}, true, true},
		{[]any{false, false}, false, false},
	}
	for _, c := range cases {
		s := newBoolSeries(t, "x", c.vals)
		if got := s.Min() != 0; got != c.min {
			t.Fatalf("min(%v): got %v, want %v", c.vals, got, c.min)
		}
		if got := s.Max() != 0; got != c.max {
			t.Fatalf("max(%v): got %v, want %v", c.vals, got, c.max)
		}
	}
}

// Boolean sum counts the true values.
func TestBoolSum(t *testing.T) {
	t.Parallel()
	s := newBoolSeries(t, "x", []any{true, false, true, nil, true})
	if got := s.Sum(); got != 3 {
		t.Fatalf("bool sum: got %v, want 3 (count of true)", got)
	}
}

// test_bool_literal_expressions: col & / | / ^ with bool literals.
func TestBoolLiteralExpressions(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{{Name: "x", Values: []any{false, true}}},
	})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	check := func(e polars.Expr, want []bool) {
		out, err := df.Select(e.Alias("r"))
		if err != nil {
			t.Fatalf("select: %v", err)
		}
		s, _ := out.GetColumn("r")
		for i, w := range want {
			if v, ok := s.Value(i).(bool); !ok || v != w {
				t.Fatalf("got %v at %d, want %v", s.Value(i), i, w)
			}
		}
	}
	check(polars.Col("x").And(polars.Lit(false)), []bool{false, false})
	check(polars.Col("x").And(polars.Lit(true)), []bool{false, true})
	check(polars.Col("x").Or(polars.Lit(false)), []bool{false, true})
	check(polars.Col("x").Or(polars.Lit(true)), []bool{true, true})
	check(polars.Col("x").Xor(polars.Lit(false)), []bool{false, true})
	check(polars.Col("x").Xor(polars.Lit(true)), []bool{true, false})
}
