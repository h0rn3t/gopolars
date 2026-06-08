package parityhelpers

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func NewInt64Series(t *testing.T, name string, values []int64) polars.Series {
	t.Helper()
	anyVals := make([]any, len(values))
	for i, v := range values {
		anyVals[i] = v
	}
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.Int64, Values: anyVals})
	if err != nil {
		t.Fatalf("NewInt64Series(%q) failed: %v", name, err)
	}
	return s
}

func NewInt64SeriesWithNulls(t *testing.T, name string, values []int64, nullMask []bool) polars.Series {
	t.Helper()
	anyVals := make([]any, len(values))
	for i, v := range values {
		if nullMask[i] {
			anyVals[i] = nil
		} else {
			anyVals[i] = v
		}
	}
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.Int64, Values: anyVals})
	if err != nil {
		t.Fatalf("NewInt64SeriesWithNulls(%q) failed: %v", name, err)
	}
	return s
}

func NewFloat64Series(t *testing.T, name string, values []float64) polars.Series {
	t.Helper()
	anyVals := make([]any, len(values))
	for i, v := range values {
		anyVals[i] = v
	}
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.Float64, Values: anyVals})
	if err != nil {
		t.Fatalf("NewFloat64Series(%q) failed: %v", name, err)
	}
	return s
}

func NewFloat64SeriesWithNulls(t *testing.T, name string, values []float64, nullMask []bool) polars.Series {
	t.Helper()
	anyVals := make([]any, len(values))
	for i, v := range values {
		if nullMask[i] {
			anyVals[i] = nil
		} else {
			anyVals[i] = v
		}
	}
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.Float64, Values: anyVals})
	if err != nil {
		t.Fatalf("NewFloat64SeriesWithNulls(%q) failed: %v", name, err)
	}
	return s
}

func NewStringSeries(t *testing.T, name string, values []string) polars.Series {
	t.Helper()
	anyVals := make([]any, len(values))
	for i, v := range values {
		anyVals[i] = v
	}
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.String, Values: anyVals})
	if err != nil {
		t.Fatalf("NewStringSeries(%q) failed: %v", name, err)
	}
	return s
}

func NewStringSeriesWithNulls(t *testing.T, name string, values []string, nullMask []bool) polars.Series {
	t.Helper()
	anyVals := make([]any, len(values))
	for i, v := range values {
		if nullMask[i] {
			anyVals[i] = nil
		} else {
			anyVals[i] = v
		}
	}
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.String, Values: anyVals})
	if err != nil {
		t.Fatalf("NewStringSeriesWithNulls(%q) failed: %v", name, err)
	}
	return s
}

func NewBoolSeries(t *testing.T, name string, values []bool) polars.Series {
	t.Helper()
	anyVals := make([]any, len(values))
	for i, v := range values {
		anyVals[i] = v
	}
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.Boolean, Values: anyVals})
	if err != nil {
		t.Fatalf("NewBoolSeries(%q) failed: %v", name, err)
	}
	return s
}

func NewBoolSeriesWithNulls(t *testing.T, name string, values []bool, nullMask []bool) polars.Series {
	t.Helper()
	anyVals := make([]any, len(values))
	for i, v := range values {
		if nullMask[i] {
			anyVals[i] = nil
		} else {
			anyVals[i] = v
		}
	}
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.Boolean, Values: anyVals})
	if err != nil {
		t.Fatalf("NewBoolSeriesWithNulls(%q) failed: %v", name, err)
	}
	return s
}

func NewDF(t *testing.T, columns ...frame.SeriesInput) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: columns})
	if err != nil {
		t.Fatalf("NewDataFrame failed: %v", err)
	}
	return df
}

func Col(name string, values ...any) frame.SeriesInput {
	return frame.SeriesInput{Name: name, Values: values}
}

func AssertFloatApprox(t *testing.T, got, want float64, eps float64) {
	t.Helper()
	if math.IsNaN(got) && math.IsNaN(want) {
		return
	}
	if math.IsNaN(got) || math.IsNaN(want) {
		t.Fatalf("float mismatch: got %v, want %v", got, want)
	}
	if math.Abs(got-want) > eps {
		t.Fatalf("float mismatch: got %v, want %v (eps=%v)", got, want, eps)
	}
}

func AssertSeriesLen(t *testing.T, s polars.Series, expected int) {
	t.Helper()
	if s.Len() != expected {
		t.Fatalf("series length mismatch: got %d, want %d", s.Len(), expected)
	}
}

func AssertSeriesName(t *testing.T, s polars.Series, expected string) {
	t.Helper()
	if s.Name() != expected {
		t.Fatalf("series name mismatch: got %q, want %q", s.Name(), expected)
	}
}

func AssertDFShape(t *testing.T, df polars.DataFrame, expectedRows, expectedCols int) {
	t.Helper()
	if df.Height() != expectedRows || df.Width() != expectedCols {
		t.Fatalf("DataFrame shape mismatch: got %dx%d, want %dx%d", df.Height(), df.Width(), expectedRows, expectedCols)
	}
}

func AssertDFColumns(t *testing.T, df polars.DataFrame, expected []string) {
	t.Helper()
	cols := df.Columns()
	if len(cols) != len(expected) {
		t.Fatalf("columns count mismatch: got %v, want %v", cols, expected)
	}
	for i, col := range cols {
		if col != expected[i] {
			t.Fatalf("column name mismatch at index %d: got %q, want %q", i, col, expected[i])
		}
	}
}

func AssertInt64Value(t *testing.T, s polars.Series, idx int, expected int64) {
	t.Helper()
	v := s.Value(idx)
	got, ok := v.(int64)
	if !ok {
		t.Fatalf("value at index %d is not int64: got %T(%v)", idx, v, v)
	}
	if got != expected {
		t.Fatalf("int64 value mismatch at index %d: got %d, want %d", idx, got, expected)
	}
}

func AssertFloat64Value(t *testing.T, s polars.Series, idx int, expected float64, eps float64) {
	t.Helper()
	v := s.Value(idx)
	got, ok := v.(float64)
	if !ok {
		t.Fatalf("value at index %d is not float64: got %T(%v)", idx, v, v)
	}
	AssertFloatApprox(t, got, expected, eps)
}

func AssertStringValue(t *testing.T, s polars.Series, idx int, expected string) {
	t.Helper()
	v := s.Value(idx)
	got, ok := v.(string)
	if !ok {
		t.Fatalf("value at index %d is not string: got %T(%v)", idx, v, v)
	}
	if got != expected {
		t.Fatalf("string value mismatch at index %d: got %q, want %q", idx, got, expected)
	}
}

func AssertBoolValue(t *testing.T, s polars.Series, idx int, expected bool) {
	t.Helper()
	v := s.Value(idx)
	got, ok := v.(bool)
	if !ok {
		t.Fatalf("value at index %d is not bool: got %T(%v)", idx, v, v)
	}
	if got != expected {
		t.Fatalf("bool value mismatch at index %d: got %v, want %v", idx, got, expected)
	}
}

func AssertNullValue(t *testing.T, s polars.Series, idx int) {
	t.Helper()
	v := s.Value(idx)
	if v != nil {
		t.Fatalf("expected null at index %d, got %T(%v)", idx, v, v)
	}
}

func AssertSeriesValuesInt64(t *testing.T, s polars.Series, expected []int64) {
	t.Helper()
	if s.Len() != len(expected) {
		t.Fatalf("series length mismatch: got %d, want %d", s.Len(), len(expected))
	}
	for i, exp := range expected {
		v := s.Value(i)
		if v == nil {
			t.Fatalf("unexpected null at index %d", i)
		}
		got, ok := v.(int64)
		if !ok {
			t.Fatalf("value at index %d is not int64: got %T(%v)", i, v, v)
		}
		if got != exp {
			t.Fatalf("int64 value mismatch at index %d: got %d, want %d", i, got, exp)
		}
	}
}

func AssertSeriesValuesFloat64(t *testing.T, s polars.Series, expected []float64, eps float64) {
	t.Helper()
	if s.Len() != len(expected) {
		t.Fatalf("series length mismatch: got %d, want %d", s.Len(), len(expected))
	}
	for i, exp := range expected {
		v := s.Value(i)
		if v == nil {
			if !math.IsNaN(exp) {
				t.Fatalf("unexpected null at index %d, expected %v", i, exp)
			}
			continue
		}
		got, ok := v.(float64)
		if !ok {
			t.Fatalf("value at index %d is not float64: got %T(%v)", i, v, v)
		}
		if math.IsNaN(exp) {
			if !math.IsNaN(got) {
				t.Fatalf("float64 value mismatch at index %d: got %v, want NaN", i, got)
			}
		} else if math.Abs(got-exp) > eps {
			t.Fatalf("float64 value mismatch at index %d: got %v, want %v (eps=%v)", i, got, exp, eps)
		}
	}
}

func AssertSeriesValuesString(t *testing.T, s polars.Series, expected []string) {
	t.Helper()
	if s.Len() != len(expected) {
		t.Fatalf("series length mismatch: got %d, want %d", s.Len(), len(expected))
	}
	for i, exp := range expected {
		v := s.Value(i)
		if exp == "" && v == nil {
			continue
		}
		got, ok := v.(string)
		if !ok {
			t.Fatalf("value at index %d is not string: got %T(%v)", i, v, v)
		}
		if got != exp {
			t.Fatalf("string value mismatch at index %d: got %q, want %q", i, got, exp)
		}
	}
}
