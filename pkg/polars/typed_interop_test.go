package polars

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	iseries "github.com/h0rn3t/gopolars/pkg/series"
)

func mustSeries(t *testing.T, name string, dtype dtypes.DataType, values []any) Series {
	t.Helper()
	s, err := NewSeries(NewSeriesInput{Name: name, DType: dtype, Values: values})
	if err != nil {
		t.Fatalf("NewSeries(%q, %s): %v", name, dtype, err)
	}
	return s
}

func TestTypedValueAccessors(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		series     Series
		read       func(Series) (any, []bool, error)
		wantValues any
		wantNulls  []bool
	}{
		{
			name:   "int64_with_null",
			series: mustSeries(t, "a", dtypes.Int64, []any{int64(1), nil, int64(3)}),
			read: func(s Series) (any, []bool, error) {
				return s.Int64Values()
			},
			wantValues: []int64{1, 0, 3},
			wantNulls:  []bool{false, true, false},
		},
		{
			name:   "float64_without_null",
			series: mustSeries(t, "a", dtypes.Float64, []any{1.5, 2.5}),
			read: func(s Series) (any, []bool, error) {
				return s.Float64Values()
			},
			wantValues: []float64{1.5, 2.5},
		},
		{
			name:   "string_with_null",
			series: mustSeries(t, "a", dtypes.String, []any{"x", nil}),
			read: func(s Series) (any, []bool, error) {
				return s.StringValues()
			},
			wantValues: []string{"x", ""},
			wantNulls:  []bool{false, true},
		},
		{
			name:   "categorical_as_strings",
			series: mustSeries(t, "a", dtypes.Categorical, []any{"x", "y"}),
			read: func(s Series) (any, []bool, error) {
				return s.StringValues()
			},
			wantValues: []string{"x", "y"},
		},
		{
			name:   "enum_as_strings",
			series: mustSeries(t, "a", dtypes.Enum, []any{"x"}),
			read: func(s Series) (any, []bool, error) {
				return s.StringValues()
			},
			wantValues: []string{"x"},
		},
		{
			name:   "bool_with_null",
			series: mustSeries(t, "a", dtypes.Boolean, []any{true, nil}),
			read: func(s Series) (any, []bool, error) {
				return s.BoolValues()
			},
			wantValues: []bool{true, false},
			wantNulls:  []bool{false, true},
		},
		{
			name:   "datetime_with_null",
			series: mustSeries(t, "a", dtypes.Datetime, []any{nil, ts}),
			read: func(s Series) (any, []bool, error) {
				return s.DatetimeValues()
			},
			wantValues: []time.Time{{}, ts},
			wantNulls:  []bool{true, false},
		},
		{
			// The backing under a null slot holds 7; the accessor reports zero.
			name:   "null_slot_zeroed",
			series: fromInternalSeries(iseries.FromColumn("a", chunk.NewFloat64([]float64{1, 7}, []bool{false, true}))),
			read: func(s Series) (any, []bool, error) {
				return s.Float64Values()
			},
			wantValues: []float64{1, 0},
			wantNulls:  []bool{false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			values, nulls, err := tt.read(tt.series)
			if err != nil {
				t.Fatalf("%s accessor error = %v, want nil", tt.name, err)
			}
			if !equalValues(values, tt.wantValues) {
				t.Errorf("%s accessor values = %v, want %v", tt.name, values, tt.wantValues)
			}
			if !slices.Equal(nulls, tt.wantNulls) || (nulls == nil) != (tt.wantNulls == nil) {
				t.Errorf("%s accessor nulls = %#v, want %#v", tt.name, nulls, tt.wantNulls)
			}
		})
	}
}

// equalValues compares typed slices returned by the accessors.
func equalValues(got, want any) bool {
	switch w := want.(type) {
	case []int64:
		g, ok := got.([]int64)
		return ok && slices.Equal(g, w)
	case []float64:
		g, ok := got.([]float64)
		return ok && slices.Equal(g, w)
	case []string:
		g, ok := got.([]string)
		return ok && slices.Equal(g, w)
	case []bool:
		g, ok := got.([]bool)
		return ok && slices.Equal(g, w)
	case []time.Time:
		g, ok := got.([]time.Time)
		return ok && slices.EqualFunc(g, w, time.Time.Equal)
	default:
		return false
	}
}

func TestTypedAccessorDTypeMismatch(t *testing.T) {
	t.Parallel()
	s := mustSeries(t, "a", dtypes.String, []any{"x"})
	values, nulls, err := s.Int64Values()
	if !errors.Is(err, ErrDTypeMismatch) {
		t.Errorf("Int64Values() on String error = %v, want ErrDTypeMismatch", err)
	}
	if values != nil || nulls != nil {
		t.Errorf("Int64Values() on String = %v, %v, want nil, nil", values, nulls)
	}
}

func TestTypedAccessorReturnsCopy(t *testing.T) {
	t.Parallel()
	df, err := NewDataFrameFromSeries(mustSeries(t, "s", dtypes.String, []any{"x", nil}))
	if err != nil {
		t.Fatalf("NewDataFrameFromSeries: %v", err)
	}
	col, err := df.GetColumn("s")
	if err != nil {
		t.Fatalf("GetColumn(s): %v", err)
	}
	values, nulls, err := col.StringValues()
	if err != nil {
		t.Fatalf("StringValues(): %v", err)
	}
	values[0] = "mutated"
	nulls[1] = false

	if got := col.Value(0); got != "x" {
		t.Errorf("Series.Value(0) after mutating the copy = %v, want x", got)
	}
	if got, _ := df.Item(1, "s"); got != nil {
		t.Errorf("DataFrame.Item(1, s) after mutating the copy = %v, want nil", got)
	}
}

func TestTypedAccessorAllocsIndependentOfLength(t *testing.T) {
	allocsFor := func(n int) float64 {
		s, err := NewInt64Series("a", make([]int64, n), nil)
		if err != nil {
			t.Fatalf("NewInt64Series: %v", err)
		}
		return testing.AllocsPerRun(5, func() {
			if _, _, err := s.Int64Values(); err != nil {
				t.Fatalf("Int64Values(): %v", err)
			}
		})
	}
	if small, large := allocsFor(100), allocsFor(10_000); small != large {
		t.Errorf("Int64Values() allocs: %v for 100 rows, %v for 10000 rows, want equal", small, large)
	}
}

func TestTypedSeriesConstructors(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		build     func() (Series, error)
		wantDType dtypes.DataType
		want      []any
	}{
		{"int64", func() (Series, error) {
			return NewInt64Series("id", []int64{1, 2, 3}, []bool{false, true, false})
		}, dtypes.Int64, []any{int64(1), nil, int64(3)}},
		{"float64_no_mask", func() (Series, error) {
			return NewFloat64Series("v", []float64{1.5}, nil)
		}, dtypes.Float64, []any{1.5}},
		{"string_empty_mask", func() (Series, error) {
			return NewStringSeries("s", []string{"x", "y"}, []bool{})
		}, dtypes.String, []any{"x", "y"}},
		{"bool", func() (Series, error) {
			return NewBoolSeries("b", []bool{true, false}, []bool{false, true})
		}, dtypes.Boolean, []any{true, nil}},
		{"datetime", func() (Series, error) {
			return NewDatetimeSeries("ts", []time.Time{ts}, nil)
		}, dtypes.Datetime, []any{ts}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, err := tt.build()
			if err != nil {
				t.Fatalf("%s constructor error = %v, want nil", tt.name, err)
			}
			if s.DataType() != tt.wantDType {
				t.Errorf("%s constructor dtype = %s, want %s", tt.name, s.DataType(), tt.wantDType)
			}
			got := make([]any, s.Len())
			for i := range got {
				got[i] = s.Value(i)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("%s constructor values = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestTypedSeriesConstructorRejectsMaskLength(t *testing.T) {
	t.Parallel()
	s, err := NewInt64Series("id", []int64{1, 2, 3}, []bool{false, true})
	if err == nil || s != nil {
		t.Errorf("NewInt64Series(3 values, 2-entry mask) = %v, %v, want nil, error", s, err)
	}
}

func TestTypedSeriesConstructorCopiesInput(t *testing.T) {
	t.Parallel()
	values := []float64{1, 2}
	nulls := []bool{false, false}
	s, err := NewFloat64Series("v", values, nulls)
	if err != nil {
		t.Fatalf("NewFloat64Series: %v", err)
	}
	values[0] = 99
	nulls[1] = true
	if got := []any{s.Value(0), s.Value(1)}; !slices.Equal(got, []any{1.0, 2.0}) {
		t.Errorf("NewFloat64Series values after mutating the input = %v, want [1 2]", got)
	}
}

func TestNewDataFrameFromSeries(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	id, _ := NewInt64Series("id", []int64{1, 2}, nil)
	when, _ := NewDatetimeSeries("ts", []time.Time{ts, ts.Add(time.Hour)}, nil)
	short, _ := NewInt64Series("short", []int64{1}, nil)
	dup, _ := NewInt64Series("id", []int64{3, 4}, nil)

	df, err := NewDataFrameFromSeries(id, when)
	if err != nil {
		t.Fatalf("NewDataFrameFromSeries(id, ts) error = %v, want nil", err)
	}
	if got := df.Columns(); !slices.Equal(got, []string{"id", "ts"}) {
		t.Errorf("NewDataFrameFromSeries(id, ts) columns = %v, want [id ts]", got)
	}
	if got := df.Dtypes(); !slices.Equal(got, []dtypes.DataType{dtypes.Int64, dtypes.Datetime}) {
		t.Errorf("NewDataFrameFromSeries(id, ts) dtypes = %v, want [Int64 Datetime]", got)
	}
	if got, _ := df.Item(1, "ts"); got != ts.Add(time.Hour) {
		t.Errorf("NewDataFrameFromSeries(id, ts).Item(1, ts) = %v, want %v", got, ts.Add(time.Hour))
	}

	for name, columns := range map[string][]Series{
		"duplicate_names": {id, dup},
		"length_mismatch": {id, short},
	} {
		if df, err := NewDataFrameFromSeries(columns...); err == nil || df != nil {
			t.Errorf("NewDataFrameFromSeries(%s) = %v, %v, want nil, error", name, df, err)
		}
	}

	empty, err := NewDataFrameFromSeries()
	if err != nil || empty == nil || empty.Width() != 0 {
		t.Errorf("NewDataFrameFromSeries() = %v, %v, want an empty frame", empty, err)
	}
}
