package polars

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/frame"
	iseries "github.com/h0rn3t/gopolars/pkg/series"
)

// ErrDTypeMismatch is returned, wrapped, by the typed Series accessors when the
// series does not hold the requested dtype.
var ErrDTypeMismatch = errors.New("polars: series dtype mismatch")

// Int64Values returns a copy of an Int64 series' values and its null mask.
func (s seriesFacade) Int64Values() ([]int64, []bool, error) {
	return typedValues(s, "Int64", (*chunk.Column).Int64s)
}

// Float64Values returns a copy of a Float64 series' values and its null mask.
func (s seriesFacade) Float64Values() ([]float64, []bool, error) {
	return typedValues(s, "Float64", (*chunk.Column).Float64s)
}

// StringValues returns a copy of a String, Categorical or Enum series' values
// and its null mask.
func (s seriesFacade) StringValues() ([]string, []bool, error) {
	return typedValues(s, "String, Categorical or Enum", (*chunk.Column).Strings)
}

// BoolValues returns a copy of a Boolean series' values and its null mask.
func (s seriesFacade) BoolValues() ([]bool, []bool, error) {
	return typedValues(s, "Boolean", (*chunk.Column).Bools)
}

// DatetimeValues returns a copy of a Datetime series' values and its null mask.
func (s seriesFacade) DatetimeValues() ([]time.Time, []bool, error) {
	return typedValues(s, "Datetime", (*chunk.Column).Times)
}

// typedValues copies a column's typed backing out of s. The mask is nil when
// the series has no nulls. Null slots are zeroed here: the chunk layer does not
// promise what its backing holds under a null.
func typedValues[T any](s seriesFacade, want string, backing func(*chunk.Column) ([]T, bool)) ([]T, []bool, error) {
	col := s.value.Column()
	var vals []T
	ok := col != nil
	if ok {
		vals, ok = backing(col)
	}
	if !ok {
		return nil, nil, fmt.Errorf("series %q is %s, want %s: %w", s.Name(), s.DataType(), want, ErrDTypeMismatch)
	}
	values := slices.Clone(vals)
	if col.NullCount() == 0 {
		return values, nil, nil
	}
	nulls := make([]bool, len(values))
	var zero T
	for i := range values {
		if col.IsNull(i) {
			nulls[i] = true
			values[i] = zero
		}
	}
	return values, nulls, nil
}

// NewInt64Series builds an Int64 series from values. A true entry in nulls
// marks that row null; a nil or empty nulls means no nulls, any other length
// must equal len(values). Both slices are copied.
func NewInt64Series(name string, values []int64, nulls []bool) (Series, error) {
	return newTypedSeries(name, values, nulls, chunk.NewInt64)
}

// NewFloat64Series builds a Float64 series from values; nulls follows
// [NewInt64Series].
func NewFloat64Series(name string, values []float64, nulls []bool) (Series, error) {
	return newTypedSeries(name, values, nulls, chunk.NewFloat64)
}

// NewStringSeries builds a String series from values; nulls follows
// [NewInt64Series].
func NewStringSeries(name string, values []string, nulls []bool) (Series, error) {
	return newTypedSeries(name, values, nulls, chunk.NewString)
}

// NewBoolSeries builds a Boolean series from values; nulls follows
// [NewInt64Series].
func NewBoolSeries(name string, values []bool, nulls []bool) (Series, error) {
	return newTypedSeries(name, values, nulls, chunk.NewBool)
}

// NewDatetimeSeries builds a Datetime series from values; nulls follows
// [NewInt64Series].
func NewDatetimeSeries(name string, values []time.Time, nulls []bool) (Series, error) {
	return newTypedSeries(name, values, nulls, chunk.NewTime)
}

// newTypedSeries copies values and nulls, because the chunk constructors take
// ownership of the slices they are given.
func newTypedSeries[T any](name string, values []T, nulls []bool, build func([]T, []bool) *chunk.Column) (Series, error) {
	if len(nulls) != 0 && len(nulls) != len(values) {
		return nil, fmt.Errorf("series %q: null mask has %d entries for %d values", name, len(nulls), len(values))
	}
	var mask []bool
	if len(nulls) != 0 {
		mask = slices.Clone(nulls)
	}
	return fromInternalSeries(iseries.FromColumn(name, build(slices.Clone(values), mask))), nil
}

// NewDataFrameFromSeries builds a DataFrame whose columns are the given series,
// in order. Series of different lengths or with duplicate names are an error;
// no series yields an empty DataFrame.
func NewDataFrameFromSeries(columns ...Series) (DataFrame, error) {
	internal := make([]iseries.Series, 0, len(columns))
	seen := make(map[string]struct{}, len(columns))
	for _, c := range columns {
		v, err := toInternalSeries(c)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[v.Name()]; dup {
			return nil, fmt.Errorf("new dataframe: duplicate column name %q", v.Name())
		}
		seen[v.Name()] = struct{}{}
		internal = append(internal, v)
	}
	return fromFrame(frame.New(frame.NewInput{Series: internal}))
}
