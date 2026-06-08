package operations

// Ported from py-polars/tests/unit/operations/test_bitwise.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Integer bitwise AND / OR operate element-wise.
func TestBitwiseAndOrInts(t *testing.T) {
	t.Parallel()
	a, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(0b1100), int64(0b1010)}})
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Int64, Values: []any{int64(0b1010), int64(0b0110)}})
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	and, err := a.BitwiseAnd(b)
	if err != nil {
		t.Fatalf("bitwise_and: %v", err)
	}
	for i, w := range []int64{0b1000, 0b0010} {
		if v, _ := and.Value(i).(int64); v != w {
			t.Fatalf("and[%d]: got %v, want %d", i, and.Value(i), w)
		}
	}
	or, err := a.BitwiseOr(b)
	if err != nil {
		t.Fatalf("bitwise_or: %v", err)
	}
	for i, w := range []int64{0b1110, 0b1110} {
		if v, _ := or.Value(i).(int64); v != w {
			t.Fatalf("or[%d]: got %v, want %d", i, or.Value(i), w)
		}
	}
}

// Bitwise &/|/^ on Boolean Series perform logical and/or/xor and return Boolean
// (matching Polars).
func TestBitwiseBoolean(t *testing.T) {
	t.Parallel()
	a, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Boolean, Values: []any{true, true, false, false}})
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Boolean, Values: []any{true, false, true, false}})
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	and, err := a.BitwiseAnd(b)
	if err != nil {
		t.Fatalf("bitwise_and: %v", err)
	}
	if and.DataType() != polars.Boolean {
		t.Fatalf("and dtype: got %v, want Boolean", and.DataType())
	}
	for i, w := range []bool{true, false, false, false} {
		if v, _ := and.Value(i).(bool); v != w {
			t.Fatalf("and[%d]: got %v, want %v", i, and.Value(i), w)
		}
	}
}
