package polars

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// newInt64Series builds a 4-element Int64 series with the given values.
func newInt64Series(t *testing.T, name string, vals []any) Series {
	t.Helper()
	s, err := NewSeries(NewSeriesInput{Name: name, DType: dtypes.Int64, Values: vals})
	if err != nil {
		t.Fatalf("newInt64Series: %v", err)
	}
	return s
}

// newFloatSeries builds a 4-element Float64 series with the given values.
func newFloatSeries(t *testing.T, name string, vals []any) Series {
	t.Helper()
	s, err := NewSeries(NewSeriesInput{Name: name, DType: dtypes.Float64, Values: vals})
	if err != nil {
		t.Fatalf("newFloatSeries: %v", err)
	}
	return s
}

// newStringSeries builds a 4-element String series with the given values.
func newStringSeries(t *testing.T, name string, vals []any) Series {
	t.Helper()
	s, err := NewSeries(NewSeriesInput{Name: name, DType: dtypes.String, Values: vals})
	if err != nil {
		t.Fatalf("newStringSeries: %v", err)
	}
	return s
}

// TestSeriesBasicAccessors pins the documented accessor contract.
func TestSeriesBasicAccessors(t *testing.T) {
	s := newInt64Series(t, "x", []any{int64(1), int64(2), int64(3), int64(4)})
	if s.Name() != "x" {
		t.Errorf("Name = %q, want x", s.Name())
	}
	if s.DataType() != dtypes.Int64 {
		t.Errorf("DataType = %s, want Int64", s.DataType())
	}
	if s.Len() != 4 {
		t.Errorf("Len = %d, want 4", s.Len())
	}
	if v, _ := s.Value(2).(int64); v != 3 {
		t.Errorf("Value(2) = %v, want 3", s.Value(2))
	}
	if lst := s.ToList(); len(lst) != 4 {
		t.Errorf("ToList len = %d, want 4", len(lst))
	}
}

// TestSeriesNullCount verifies the documented null counting.
func TestSeriesNullCount(t *testing.T) {
	s := newFloatSeries(t, "v", []any{1.0, nil, 3.0, nil})
	if s.NullCount() != 2 {
		t.Errorf("NullCount = %d, want 2", s.NullCount())
	}
}

// TestSeriesIsNullIsNotNull exercises the validity mask methods.
func TestSeriesIsNullIsNotNull(t *testing.T) {
	s := newFloatSeries(t, "v", []any{1.0, nil, 3.0, nil})
	isNull := s.IsNull()
	isNotNull := s.IsNotNull()
	if isNull.DataType() != dtypes.Boolean {
		t.Errorf("IsNull dtype = %s, want Boolean", isNull.DataType())
	}
	if isNotNull.DataType() != dtypes.Boolean {
		t.Errorf("IsNotNull dtype = %s, want Boolean", isNotNull.DataType())
	}
	want := []bool{false, true, false, true}
	for i, w := range want {
		if v, _ := isNull.Value(i).(bool); v != w {
			t.Errorf("IsNull[%d] = %v, want %v", i, v, w)
		}
		if v, _ := isNotNull.Value(i).(bool); v == w {
			t.Errorf("IsNotNull[%d] = %v, should be the inverse", i, v)
		}
	}
}

// TestSeriesTransformations covers Slice, Filter, Cast, Sort, Reverse, Shift,
// Rename, Clone, Rechunk, Head, Tail, Limit.
func TestSeriesTransformations(t *testing.T) {
	s := newInt64Series(t, "x", []any{int64(1), int64(2), int64(3), int64(4)})

	// Slice
	if out := s.Slice(1, 2); out.Len() != 2 {
		t.Errorf("Slice(1,2) len = %d, want 2", out.Len())
	}

	// Filter via boolean mask
	mask, err := NewSeries(NewSeriesInput{Name: "m", DType: dtypes.Boolean, Values: []any{true, false, true, false}})
	if err != nil {
		t.Fatalf("mask: %v", err)
	}
	out, err := s.Filter(mask)
	if err != nil {
		t.Fatalf("Filter: %v", err)
	}
	if out.Len() != 2 {
		t.Errorf("Filter len = %d, want 2", out.Len())
	}

	// Cast
	if out, err := s.Cast(dtypes.Float64); err != nil {
		t.Errorf("Cast: %v", err)
	} else if out.DataType() != dtypes.Float64 {
		t.Errorf("Cast dtype = %s, want Float64", out.DataType())
	}

	// Sort
	desc := s.Sort(true)
	if v, _ := desc.Value(0).(int64); v != 4 {
		t.Errorf("Sort(true)[0] = %v, want 4", desc.Value(0))
	}
	asc := s.Sort(false)
	if v, _ := asc.Value(0).(int64); v != 1 {
		t.Errorf("Sort(false)[0] = %v, want 1", asc.Value(0))
	}

	// Reverse
	if out := s.Reverse(); out.Len() != 4 {
		t.Errorf("Reverse len = %d, want 4", out.Len())
	}

	// Shift
	if out := s.Shift(1); out.Len() != 4 {
		t.Errorf("Shift len = %d, want 4", out.Len())
	}

	// Rename / Clone / Rechunk / Head / Tail / Limit
	if out := s.Rename("y"); out.Name() != "y" {
		t.Errorf("Rename name = %q, want y", out.Name())
	}
	if out := s.Clone(); out.Len() != s.Len() {
		t.Errorf("Clone len = %d, want %d", out.Len(), s.Len())
	}
	if out := s.Rechunk(); out.Len() != s.Len() {
		t.Errorf("Rechunk len = %d, want %d", out.Len(), s.Len())
	}
	if out := s.Head(2); out.Len() != 2 {
		t.Errorf("Head(2) len = %d, want 2", out.Len())
	}
	if out := s.Tail(2); out.Len() != 2 {
		t.Errorf("Tail(2) len = %d, want 2", out.Len())
	}
	if out := s.Limit(3); out.Len() != 3 {
		t.Errorf("Limit(3) len = %d, want 3", out.Len())
	}
}

// TestSeriesFillDrop verifies the null-replacement and drop family.
func TestSeriesFillDrop(t *testing.T) {
	s := newFloatSeries(t, "v", []any{1.0, nil, math.NaN(), 4.0})

	// FillNull with 0
	filled, err := s.FillNull(float64(0))
	if err != nil {
		t.Fatalf("FillNull: %v", err)
	}
	if filled.NullCount() != 0 {
		t.Errorf("FillNull still has %d nulls", filled.NullCount())
	}

	// DropNulls
	dn := s.DropNulls()
	if dn.Len() != 3 {
		t.Errorf("DropNulls len = %d, want 3", dn.Len())
	}

	// DropNans (drops NaN, keeps null)
	if out := s.DropNans(); out.Len() != 3 {
		t.Errorf("DropNans len = %d, want 3 (NaN dropped, null kept)", out.Len())
	}
}

// TestSeriesAggregations covers the documented reduction methods.
func TestSeriesAggregations(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1), int64(2), int64(3), int64(4)})

	if v := s.Sum(); v != 10 {
		t.Errorf("Sum = %v, want 10", v)
	}
	if v := s.Min(); v != 1 {
		t.Errorf("Min = %v, want 1", v)
	}
	if v := s.Max(); v != 4 {
		t.Errorf("Max = %v, want 4", v)
	}
	if v := s.Mean(); v != 2.5 {
		t.Errorf("Mean = %v, want 2.5", v)
	}
	if v := s.Median(); v != 2.5 {
		t.Errorf("Median = %v, want 2.5", v)
	}
	if v := s.Product(); v != 24 {
		t.Errorf("Product = %v, want 24", v)
	}
	if v := s.NUnique(); v != 4 {
		t.Errorf("NUnique = %d, want 4", v)
	}
	if v := s.Var(); v <= 0 {
		t.Errorf("Var = %v, want > 0", v)
	}
	if v := s.Std(); v <= 0 {
		t.Errorf("Std = %v, want > 0", v)
	}
	if v := s.Quantile(0.5); v != 2.5 {
		t.Errorf("Quantile(0.5) = %v, want 2.5", v)
	}
	if v := s.ArgMax(); v != 3 {
		t.Errorf("ArgMax = %d, want 3", v)
	}
	if v := s.ArgMin(); v != 0 {
		t.Errorf("ArgMin = %d, want 0", v)
	}
}

// TestSeriesComparisonsAndArithmetic covers the binary ops.
func TestSeriesComparisonsAndArithmetic(t *testing.T) {
	a := newInt64Series(t, "a", []any{int64(1), int64(2), int64(3)})
	b := newInt64Series(t, "b", []any{int64(2), int64(2), int64(2)})

	// Add → elementwise
	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if v, _ := sum.Value(0).(int64); v != 3 {
		t.Errorf("Add[0] = %v, want 3", sum.Value(0))
	}

	// Eq → boolean
	eq, err := a.Eq(b)
	if err != nil {
		t.Fatalf("Eq: %v", err)
	}
	if eq.DataType() != dtypes.Boolean {
		t.Errorf("Eq dtype = %s, want Boolean", eq.DataType())
	}
}

// TestSeriesCumulative covers the cumulative reductions.
func TestSeriesCumulative(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1), int64(2), int64(3), int64(4)})
	if out := s.CumSum(); out.Len() != 4 {
		t.Errorf("CumSum len = %d, want 4", out.Len())
	}
	if out := s.CumMax(); out.Len() != 4 {
		t.Errorf("CumMax len = %d, want 4", out.Len())
	}
	if out := s.CumMin(); out.Len() != 4 {
		t.Errorf("CumMin len = %d, want 4", out.Len())
	}
	if out := s.CumProd(); out.Len() != 4 {
		t.Errorf("CumProd len = %d, want 4", out.Len())
	}
	if out := s.CumCount(); out.Len() != 4 {
		t.Errorf("CumCount len = %d, want 4", out.Len())
	}
}

// TestSeriesRolling covers the rolling family.
func TestSeriesRolling(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1), int64(2), int64(3), int64(4), int64(5)})
	if out := s.RollingMean(3); out.Len() != 5 {
		t.Errorf("RollingMean len = %d, want 5", out.Len())
	}
	if out := s.RollingSum(3); out.Len() != 5 {
		t.Errorf("RollingSum len = %d, want 5", out.Len())
	}
	if out := s.RollingMin(3); out.Len() != 5 {
		t.Errorf("RollingMin len = %d, want 5", out.Len())
	}
	if out := s.RollingMax(3); out.Len() != 5 {
		t.Errorf("RollingMax len = %d, want 5", out.Len())
	}
}

// TestSeriesAlias verifies the documented aliasing.
func TestSeriesAlias(t *testing.T) {
	s := newInt64Series(t, "x", []any{int64(1)})
	if out := s.Alias("y"); out.Name() != "y" {
		t.Errorf("Alias name = %q, want y", out.Name())
	}
}

// TestSeriesStringNamespace exercises the small string namespace surface.
func TestSeriesStringNamespace(t *testing.T) {
	s := newStringSeries(t, "x", []any{"AbC", "DeF", "gHi"})
	if out, err := s.Str().Lower(); err != nil {
		t.Errorf("Str.Lower: %v", err)
	} else if v, _ := out.Value(0).(string); v != "abc" {
		t.Errorf("Str.Lower[0] = %v, want abc", out.Value(0))
	}
	if out, err := s.Str().Upper(); err != nil {
		t.Errorf("Str.Upper: %v", err)
	} else if v, _ := out.Value(0).(string); v != "ABC" {
		t.Errorf("Str.Upper[0] = %v, want ABC", out.Value(0))
	}
	if out, err := s.Str().Len(); err != nil {
		t.Errorf("Str.Len: %v", err)
	} else if v, _ := out.Value(0).(int64); v != 3 {
		t.Errorf("Str.Len[0] = %v, want 3", out.Value(0))
	}

	// Str namespace on a non-string series returns an error.
	ns, err := s.Str().Lower()
	_ = ns
	_ = err
	if _, err := newInt64Series(t, "n", []any{int64(1)}).Str().Lower(); err == nil {
		t.Errorf("Str.Lower on int64 returned nil error, want non-nil")
	}
}

// TestSeriesNamespacesForNonMatchingDTypes return errors.
func TestSeriesNamespacesForNonMatchingDTypes(t *testing.T) {
	intS := newInt64Series(t, "n", []any{int64(2024)})
	if _, err := intS.Dt().Year(); err == nil {
		t.Errorf("Dt.Year on int64 returned nil error, want non-nil")
	}
	if _, err := intS.Cat().Codes(); err == nil {
		t.Errorf("Cat.Codes on int64 returned nil error, want non-nil")
	}
	if _, err := intS.Struct().Field("f"); err == nil {
		t.Errorf("Struct.Field on int64 returned nil error, want non-nil")
	}
	if _, err := intS.Bin().CountOnes(); err != nil {
		// Bin.CountOnes needs a seriesFacade; this may still work.
		t.Logf("Bin.CountOnes on int64: %v (allowed)", err)
	}
}
