package polars

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestSeriesArithmeticOperators pins the elementwise arithmetic contract for the
// Sub/Mul/Div operators (the Add sibling is covered separately). Int64 inputs
// stay Int64 for sub/mul but promote to Float64 for div.
func TestSeriesArithmeticOperators(t *testing.T) {
	a := newInt64Series(t, "a", []any{int64(1), int64(2), int64(3)})
	b := newInt64Series(t, "b", []any{int64(2), int64(2), int64(2)})

	sub, err := a.Sub(b)
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	if sub.DataType() != dtypes.Int64 {
		t.Errorf("Sub dtype = %s, want Int64", sub.DataType())
	}
	wantSub := []int64{-1, 0, 1}
	for i, w := range wantSub {
		if v, _ := sub.Value(i).(int64); v != w {
			t.Errorf("Sub[%d] = %v, want %d", i, sub.Value(i), w)
		}
	}

	mul, err := a.Mul(b)
	if err != nil {
		t.Fatalf("Mul: %v", err)
	}
	wantMul := []int64{2, 4, 6}
	for i, w := range wantMul {
		if v, _ := mul.Value(i).(int64); v != w {
			t.Errorf("Mul[%d] = %v, want %d", i, mul.Value(i), w)
		}
	}

	div, err := a.Div(b)
	if err != nil {
		t.Fatalf("Div: %v", err)
	}
	if div.DataType() != dtypes.Float64 {
		t.Errorf("Div dtype = %s, want Float64", div.DataType())
	}
	wantDiv := []float64{0.5, 1.0, 1.5}
	for i, w := range wantDiv {
		if v, _ := div.Value(i).(float64); v != w {
			t.Errorf("Div[%d] = %v, want %v", i, div.Value(i), w)
		}
	}
}

// TestSeriesArithmeticLengthMismatch confirms operators reject mismatched lengths.
func TestSeriesArithmeticLengthMismatch(t *testing.T) {
	a := newInt64Series(t, "a", []any{int64(1), int64(2), int64(3)})
	b := newInt64Series(t, "b", []any{int64(2), int64(2)})
	if _, err := a.Sub(b); err == nil {
		t.Error("Sub with mismatched lengths: expected error, got nil")
	}
	if _, err := a.Mul(b); err == nil {
		t.Error("Mul with mismatched lengths: expected error, got nil")
	}
	if _, err := a.Div(b); err == nil {
		t.Error("Div with mismatched lengths: expected error, got nil")
	}
}

// TestSeriesComparisonOperators pins the boolean output of the comparison
// operators (the Eq sibling is covered separately).
func TestSeriesComparisonOperators(t *testing.T) {
	a := newInt64Series(t, "a", []any{int64(1), int64(2), int64(3)})
	b := newInt64Series(t, "b", []any{int64(2), int64(2), int64(2)})

	cases := []struct {
		name string
		fn   func(Series) (Series, error)
		want []bool
	}{
		{"Ne", a.Ne, []bool{true, false, true}},
		{"Gt", a.Gt, []bool{false, false, true}},
		{"Ge", a.Ge, []bool{false, true, true}},
		{"Lt", a.Lt, []bool{true, false, false}},
		{"Le", a.Le, []bool{true, true, false}},
	}
	for _, tc := range cases {
		out, err := tc.fn(b)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if out.DataType() != dtypes.Boolean {
			t.Errorf("%s dtype = %s, want Boolean", tc.name, out.DataType())
		}
		for i, w := range tc.want {
			if v, _ := out.Value(i).(bool); v != w {
				t.Errorf("%s[%d] = %v, want %v", tc.name, i, out.Value(i), w)
			}
		}
	}
}

// TestSeriesHyperbolicArc covers the inverse hyperbolic unary functions, which
// delegate to math.Asinh/Acosh/Atanh.
func TestSeriesHyperbolicArc(t *testing.T) {
	s := newFloatSeries(t, "v", []any{0.0, 1.0, 2.0})

	arcsinh := s.Arcsinh()
	if arcsinh.Len() != 3 {
		t.Fatalf("Arcsinh len = %d, want 3", arcsinh.Len())
	}
	for i, in := range []float64{0.0, 1.0, 2.0} {
		got, _ := arcsinh.Value(i).(float64)
		if math.Abs(got-math.Asinh(in)) > 1e-9 {
			t.Errorf("Arcsinh[%d] = %v, want %v", i, got, math.Asinh(in))
		}
	}

	// Arccosh is defined for x >= 1; use 1,2,3 so every element is valid.
	c := newFloatSeries(t, "v", []any{1.0, 2.0, 3.0})
	arccosh := c.Arccosh()
	for i, in := range []float64{1.0, 2.0, 3.0} {
		got, _ := arccosh.Value(i).(float64)
		if math.Abs(got-math.Acosh(in)) > 1e-9 {
			t.Errorf("Arccosh[%d] = %v, want %v", i, got, math.Acosh(in))
		}
	}

	// Arctanh is defined for |x| < 1.
	tnh := newFloatSeries(t, "v", []any{-0.5, 0.0, 0.5})
	arctanh := tnh.Arctanh()
	for i, in := range []float64{-0.5, 0.0, 0.5} {
		got, _ := arctanh.Value(i).(float64)
		if math.Abs(got-math.Atanh(in)) > 1e-9 {
			t.Errorf("Arctanh[%d] = %v, want %v", i, got, math.Atanh(in))
		}
	}
}
