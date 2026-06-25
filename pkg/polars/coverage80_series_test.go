package polars

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// newDatetimeSeries builds a Datetime series.
func newDatetimeSeries(t *testing.T, name string, vals []any) Series {
	t.Helper()
	s, err := NewSeries(NewSeriesInput{Name: name, DType: dtypes.Datetime, Values: vals})
	if err != nil {
		t.Fatalf("newDatetimeSeries: %v", err)
	}
	return s
}

// TestSeriesFillNullSlowPath covers FillNull on a non-float64 column (the
// boxed slow path, not the float64 fast path).
func TestSeriesFillNullSlowPath(t *testing.T) {
	s := newStringSeries(t, "s", []any{"a", nil, "c", nil})
	out, err := s.FillNull("z")
	if err != nil {
		t.Fatalf("FillNull: %v", err)
	}
	if out.NullCount() != 0 {
		t.Errorf("NullCount after FillNull = %d, want 0", out.NullCount())
	}
	if v, _ := out.Value(1).(string); v != "z" {
		t.Errorf("Value(1) = %v, want z", out.Value(1))
	}
	if v, _ := out.Value(0).(string); v != "a" {
		t.Errorf("Value(0) = %v, want a (unchanged)", out.Value(0))
	}
}

// TestSeriesFillNullFloatFastPath covers the float64 typed fast path.
func TestSeriesFillNullFloatFastPath(t *testing.T) {
	s := newFloatSeries(t, "f", []any{1.0, nil, 3.0, nil})
	out, err := s.FillNull(9.0)
	if err != nil {
		t.Fatalf("FillNull: %v", err)
	}
	if out.NullCount() != 0 {
		t.Errorf("NullCount = %d, want 0", out.NullCount())
	}
	if v, _ := out.Value(1).(float64); v != 9.0 {
		t.Errorf("Value(1) = %v, want 9", out.Value(1))
	}
}

// TestSeriesFillNanSlowPath covers FillNan on an int64 column (slow path: no
// float64 backing, so nothing changes but the boxed loop runs).
func TestSeriesFillNanSlowPath(t *testing.T) {
	s := newInt64Series(t, "i", []any{int64(1), int64(2), int64(3), int64(4)})
	out, err := s.FillNan(0.0)
	if err != nil {
		t.Fatalf("FillNan: %v", err)
	}
	if out.Len() != 4 {
		t.Errorf("len = %d, want 4", out.Len())
	}
}

// TestSeriesFillNanFloatFastPath covers FillNan fast path replacing NaN values.
func TestSeriesFillNanFloatFastPath(t *testing.T) {
	s := newFloatSeries(t, "f", []any{1.0, nan(), 3.0, 4.0})
	out, err := s.FillNan(7.0)
	if err != nil {
		t.Fatalf("FillNan: %v", err)
	}
	if v, _ := out.Value(1).(float64); v != 7.0 {
		t.Errorf("Value(1) = %v, want 7", out.Value(1))
	}
}

// TestSeriesDropNansFloatFastPath covers DropNans removing NaN values.
func TestSeriesDropNansFloatFastPath(t *testing.T) {
	s := newFloatSeries(t, "f", []any{1.0, nan(), 3.0, nan()})
	out := s.DropNans()
	if out.Len() != 2 {
		t.Errorf("DropNans len = %d, want 2", out.Len())
	}
}

// TestSeriesDropNansSlowPath covers DropNans on an int64 column (slow path).
func TestSeriesDropNansSlowPath(t *testing.T) {
	s := newInt64Series(t, "i", []any{int64(1), int64(2), int64(3)})
	out := s.DropNans()
	if out.Len() != 3 {
		t.Errorf("DropNans len = %d, want 3 (no NaNs in int)", out.Len())
	}
}

// nan returns a NaN float64.
func nan() float64 {
	z := 0.0
	return z / z
}

// TestSeriesInterpolate covers linear interpolation between known points and
// the edge cases (leading/trailing nulls fall back to nearest).
func TestSeriesInterpolate(t *testing.T) {
	// nulls in the interior interpolate; leading null forward-fills.
	s := newFloatSeries(t, "f", []any{nil, 1.0, nil, 3.0})
	out := s.Interpolate()
	// index 0: nil with no left -> right value 1.0
	if v, _ := out.Value(0).(float64); v != 1.0 {
		t.Errorf("Interpolate[0] = %v, want 1.0 (leading null backfill)", out.Value(0))
	}
	// index 2: between 1.0 and 3.0 -> 2.0
	if v, _ := out.Value(2).(float64); v != 2.0 {
		t.Errorf("Interpolate[2] = %v, want 2.0", out.Value(2))
	}
}

// TestSeriesInterpolateTrailingNull covers the trailing-null branch.
func TestSeriesInterpolateTrailingNull(t *testing.T) {
	s := newFloatSeries(t, "f", []any{1.0, 2.0, nil})
	out := s.Interpolate()
	if v, _ := out.Value(2).(float64); v != 2.0 {
		t.Errorf("Interpolate[2] = %v, want 2.0 (trailing null fwd-fill)", out.Value(2))
	}
}

// TestSeriesCastNumeric covers Cast across int/float/string/bool with nulls.
func TestSeriesCastNumeric(t *testing.T) {
	s := newInt64Series(t, "i", []any{int64(1), nil, int64(3)})

	asFloat, err := s.Cast(dtypes.Float64)
	if err != nil {
		t.Fatalf("Cast Float64: %v", err)
	}
	if asFloat.DataType() != dtypes.Float64 {
		t.Errorf("cast dtype = %s, want Float64", asFloat.DataType())
	}
	if asFloat.NullCount() != 1 {
		t.Errorf("null preserved across cast: NullCount = %d, want 1", asFloat.NullCount())
	}

	asStr, err := s.Cast(dtypes.String)
	if err != nil {
		t.Fatalf("Cast String: %v", err)
	}
	if v, _ := asStr.Value(0).(string); v != "1" {
		t.Errorf("cast to string[0] = %v, want 1", asStr.Value(0))
	}

	asBool, err := s.Cast(dtypes.Boolean)
	if err != nil {
		t.Fatalf("Cast Boolean: %v", err)
	}
	if v, _ := asBool.Value(0).(bool); !v {
		t.Errorf("cast 1 to bool = %v, want true", asBool.Value(0))
	}
}

// TestSeriesCastStringToNumeric covers parsing string->int/float in castAny.
func TestSeriesCastStringToNumeric(t *testing.T) {
	s := newStringSeries(t, "s", []any{"10", "20"})
	asInt, err := s.Cast(dtypes.Int64)
	if err != nil {
		t.Fatalf("Cast Int64: %v", err)
	}
	if v, _ := asInt.Value(0).(int64); v != 10 {
		t.Errorf("string->int[0] = %v, want 10", asInt.Value(0))
	}

	asFloat, err := s.Cast(dtypes.Float64)
	if err != nil {
		t.Fatalf("Cast Float64: %v", err)
	}
	if v, _ := asFloat.Value(1).(float64); v != 20.0 {
		t.Errorf("string->float[1] = %v, want 20", asFloat.Value(1))
	}
}

// TestSeriesCastBoolToString covers stringifying bool and casting bool->float.
func TestSeriesCastBoolFloat(t *testing.T) {
	s, err := NewSeries(NewSeriesInput{Name: "b", DType: dtypes.Boolean, Values: []any{true, false}})
	if err != nil {
		t.Fatalf("bool series: %v", err)
	}
	asFloat, err := s.Cast(dtypes.Float64)
	if err != nil {
		t.Fatalf("Cast Float64: %v", err)
	}
	if v, _ := asFloat.Value(0).(float64); v != 1.0 {
		t.Errorf("true->float = %v, want 1", asFloat.Value(0))
	}
	if v, _ := asFloat.Value(1).(float64); v != 0.0 {
		t.Errorf("false->float = %v, want 0", asFloat.Value(1))
	}
}

// TestSeriesCastUnsupportedReturnsError covers the castAny error path.
func TestSeriesCastUnsupportedReturnsError(t *testing.T) {
	s := newStringSeries(t, "s", []any{"not-a-time"})
	if _, err := s.Cast(dtypes.Datetime); err == nil {
		t.Fatalf("Cast bad string to Datetime returned nil error, want non-nil")
	}
}

// TestSeriesDtYear covers the dt.Year happy path.
func TestSeriesDtYear(t *testing.T) {
	t0 := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC)
	s := newDatetimeSeries(t, "d", []any{t0, nil, t1})
	out, err := s.Dt().Year()
	if err != nil {
		t.Fatalf("Year: %v", err)
	}
	if v, _ := out.Value(0).(int64); v != 2021 {
		t.Errorf("Year[0] = %v, want 2021", out.Value(0))
	}
	if out.Value(1) != nil {
		t.Errorf("Year[1] = %v, want nil", out.Value(1))
	}
	if v, _ := out.Value(2).(int64); v != 2022 {
		t.Errorf("Year[2] = %v, want 2022", out.Value(2))
	}
}

// TestSeriesCatCodes covers cat.Codes assigning codes by first appearance.
func TestSeriesCatCodes(t *testing.T) {
	s, err := NewSeries(NewSeriesInput{Name: "c", DType: dtypes.Categorical, Values: []any{"x", "y", "x", nil, "z"}})
	if err != nil {
		t.Fatalf("categorical series: %v", err)
	}
	out, err := s.Cat().Codes()
	if err != nil {
		t.Fatalf("Codes: %v", err)
	}
	if v, _ := out.Value(0).(int64); v != 0 {
		t.Errorf("Codes[0] = %v, want 0", out.Value(0))
	}
	if v, _ := out.Value(1).(int64); v != 1 {
		t.Errorf("Codes[1] = %v, want 1", out.Value(1))
	}
	if v, _ := out.Value(2).(int64); v != 0 {
		t.Errorf("Codes[2] = %v, want 0 (x repeats)", out.Value(2))
	}
	if out.Value(3) != nil {
		t.Errorf("Codes[3] = %v, want nil", out.Value(3))
	}
	if v, _ := out.Value(4).(int64); v != 2 {
		t.Errorf("Codes[4] = %v, want 2", out.Value(4))
	}
}

// TestSeriesStructField covers struct.Field extracting a typed field.
func TestSeriesStructField(t *testing.T) {
	s, err := NewSeries(NewSeriesInput{Name: "rec", DType: dtypes.Struct, Values: []any{
		map[string]any{"x": int64(10), "y": "a"},
		map[string]any{"x": int64(20), "y": "b"},
		nil,
	}})
	if err != nil {
		t.Fatalf("struct series: %v", err)
	}
	out, err := s.Struct().Field("x")
	if err != nil {
		t.Fatalf("Field x: %v", err)
	}
	if out.DataType() != dtypes.Int64 {
		t.Errorf("Field x dtype = %s, want Int64", out.DataType())
	}
	if v, _ := out.Value(1).(int64); v != 20 {
		t.Errorf("Field x[1] = %v, want 20", out.Value(1))
	}
	if out.Value(2) != nil {
		t.Errorf("Field x[2] = %v, want nil", out.Value(2))
	}
}

// TestSeriesStructFieldEmptyName covers the empty-field-name error.
func TestSeriesStructFieldEmptyName(t *testing.T) {
	s, err := NewSeries(NewSeriesInput{Name: "rec", DType: dtypes.Struct, Values: []any{
		map[string]any{"x": int64(10)},
	}})
	if err != nil {
		t.Fatalf("struct series: %v", err)
	}
	if _, err := s.Struct().Field(""); err == nil {
		t.Fatalf("Field with empty name returned nil error, want non-nil")
	}
}

// TestSeriesStrNamespaceWithNulls covers Lower/Upper/Len null handling.
func TestSeriesStrNamespaceWithNulls(t *testing.T) {
	s := newStringSeries(t, "s", []any{"Ab", nil, "Cd"})

	lower, err := s.Str().Lower()
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	if v, _ := lower.Value(0).(string); v != "ab" {
		t.Errorf("Lower[0] = %v, want ab", lower.Value(0))
	}
	if lower.Value(1) != nil {
		t.Errorf("Lower[1] = %v, want nil", lower.Value(1))
	}

	upper, err := s.Str().Upper()
	if err != nil {
		t.Fatalf("Upper: %v", err)
	}
	if v, _ := upper.Value(2).(string); v != "CD" {
		t.Errorf("Upper[2] = %v, want CD", upper.Value(2))
	}
	if upper.Value(1) != nil {
		t.Errorf("Upper[1] = %v, want nil", upper.Value(1))
	}

	length, err := s.Str().Len()
	if err != nil {
		t.Fatalf("Len: %v", err)
	}
	if v, _ := length.Value(0).(int64); v != 2 {
		t.Errorf("Len[0] = %v, want 2", length.Value(0))
	}
	if length.Value(1) != nil {
		t.Errorf("Len[1] = %v, want nil", length.Value(1))
	}
}

// TestSeriesArrListLenWithNull covers arr.ListLen null handling.
func TestSeriesArrListLenWithNull(t *testing.T) {
	s, err := NewSeries(NewSeriesInput{Name: "l", DType: dtypes.List, Values: []any{
		[]any{int64(1), int64(2)},
		nil,
		[]any{int64(3)},
	}})
	if err != nil {
		t.Fatalf("list series: %v", err)
	}
	out, err := s.Arr().ListLen()
	if err != nil {
		t.Fatalf("ListLen: %v", err)
	}
	if v, _ := out.Value(0).(int64); v != 2 {
		t.Errorf("ListLen[0] = %v, want 2", out.Value(0))
	}
	if out.Value(1) != nil {
		t.Errorf("ListLen[1] = %v, want nil", out.Value(1))
	}
}
