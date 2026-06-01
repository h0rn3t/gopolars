package series

import (
	"fmt"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// Series is a named column. Its canonical storage is a typed *chunk.Column;
// the legacy []any input accepted by New is converted to a typed chunk once at
// construction time and not retained.
type Series struct {
	name string
	col  *chunk.Column
}

// New builds a Series from legacy []any values, converting them to a typed
// chunk. The []any is not kept as canonical storage.
func New(name string, dtype dtypes.DataType, values []any) (Series, error) {
	col, err := chunk.FromAny(dtype, values)
	if err != nil {
		return Series{}, fmt.Errorf("invalid value for series %s: %w", name, err)
	}
	return Series{name: name, col: col}, nil
}

// FromColumn wraps an existing typed chunk as a named Series (zero-copy).
func FromColumn(name string, col *chunk.Column) Series {
	return Series{name: name, col: col}
}

// FromInt64 builds an Int64 series directly from a typed slice (zero-copy;
// ownership of values/nulls transfers to the series).
func FromInt64(name string, values []int64, nulls []bool) Series {
	return Series{name: name, col: chunk.NewInt64(values, nulls)}
}

// FromFloat64 builds a Float64 series directly from a typed slice (zero-copy).
func FromFloat64(name string, values []float64, nulls []bool) Series {
	return Series{name: name, col: chunk.NewFloat64(values, nulls)}
}

// FromString builds a String series directly from a typed slice (zero-copy).
func FromString(name string, values []string, nulls []bool) Series {
	return Series{name: name, col: chunk.NewString(values, nulls)}
}

// FromBool builds a Boolean series directly from a typed slice (zero-copy).
func FromBool(name string, values []bool, nulls []bool) Series {
	return Series{name: name, col: chunk.NewBool(values, nulls)}
}

func (s Series) Name() string {
	return s.name
}

func (s Series) DataType() dtypes.DataType {
	if s.col == nil {
		return ""
	}
	return s.col.DataType()
}

func (s Series) Len() int {
	return s.col.Len()
}

func (s Series) IsNull(i int) bool {
	return s.col.IsNull(i)
}

func (s Series) Value(i int) any {
	return s.col.ValueAt(i)
}

// Column returns the underlying typed chunk for internal zero-copy access.
// It may be nil for the zero-value Series.
func (s Series) Column() *chunk.Column {
	return s.col
}

func (s Series) Clone() Series {
	if s.col == nil {
		return s
	}
	return Series{name: s.name, col: s.col.Clone()}
}

// Rename returns a Series with a new name that SHARES the source's underlying
// typed column (metadata-only, no buffer copy). The shared column is marked so
// the copy-on-write contract protects it from any future in-place mutation.
func (s Series) Rename(name string) Series {
	if s.col != nil {
		s.col.MarkShared()
	}
	return Series{name: name, col: s.col}
}

func (s Series) Slice(indexes []int) Series {
	return Series{name: s.name, col: s.col.Slice(indexes)}
}

// Filter keeps rows where mask[i] is true.
func (s Series) Filter(mask []bool) Series {
	return Series{name: s.name, col: s.col.Filter(mask)}
}

func (s Series) Shift(periods int) Series {
	if periods == 0 || s.Len() == 0 {
		return s.Clone()
	}
	return Series{name: s.name, col: s.col.Shift(periods)}
}
