// Package arrow bridges gopolars frames and Apache Arrow records/tables.
//
// Native path (zero []any allocation for primitive dtypes):
//
//	FromArrowRecord(rec arrow.RecordBatch) → frame.DataFrame
//	ToArrowRecord(df frame.DataFrame) → arrow.RecordBatch
//
// Legacy path (kept for gob-encoded IPC compatibility):
//
//	ToTable / FromTable use the custom Table type that stores []any per column.
package arrow

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// Table is the legacy columnar representation used by gob-encoded IPC files.
type Table struct {
	Columns map[string][]any
	// Names is the column order. A name without an entry in Columns is
	// skipped; columns missing from Names follow in ascending name order, so a
	// table without Names (older IPC files, hand-built tables) reads
	// alphabetically. gob ignores the field when it is absent on either side.
	Names []string
}

// FromArrowRecord imports an Arrow record batch into a DataFrame without
// allocating an intermediate []any for primitive dtypes (Float64, Int64,
// Boolean, String, Timestamp). Narrow integers widen to Int64, 16- and 32-bit
// floats to Float64, string and binary views become String and Binary,
// dictionary arrays are decoded, durations become Duration and the null type
// an all-null column. An Arrow type without a mapping, or a uint64 value above
// math.MaxInt64, makes the import fail with an error naming the column.
func FromArrowRecord(rec goarrow.RecordBatch) (frame.DataFrame, error) {
	schema := rec.Schema()
	n := int(rec.NumRows())
	cols := make([]series.Series, 0, rec.NumCols())

	for i := 0; i < int(rec.NumCols()); i++ {
		name := schema.Field(i).Name
		arr := rec.Column(i)
		col, err := arrowArrayToColumn(arr, n)
		if err != nil {
			return frame.DataFrame{}, fmt.Errorf("column %q: %w", name, err)
		}
		cols = append(cols, series.FromColumn(name, col))
	}
	return frame.New(frame.NewInput{Series: cols})
}

// ToArrowRecord exports a DataFrame to an Arrow record batch without
// round-tripping through []any for primitive dtypes.
func ToArrowRecord(df frame.DataFrame) (goarrow.RecordBatch, error) {
	alloc := memory.NewGoAllocator()
	fields := make([]goarrow.Field, 0, df.Width())
	arrays := make([]goarrow.Array, 0, df.Width())

	for _, name := range df.Columns() {
		s, _ := df.Series(name)
		arr, arrowType, err := columnToArrowArray(s, alloc)
		if err != nil {
			return nil, fmt.Errorf("column %q: %w", name, err)
		}
		fields = append(fields, goarrow.Field{Name: name, Type: arrowType, Nullable: true})
		arrays = append(arrays, arr)
	}

	schema := goarrow.NewSchema(fields, nil)
	rec := array.NewRecordBatch(schema, arrays, int64(df.Height()))
	for _, a := range arrays {
		a.Release()
	}
	return rec, nil
}

// arrowArrayToColumn converts a single Arrow array into a typed chunk.Column.
func arrowArrayToColumn(arr goarrow.Array, n int) (*chunk.Column, error) {
	nulls := buildNullMask(arr, n)
	switch a := arr.(type) {
	case *array.Float64:
		return chunk.NewFloat64(cloneZeroingNulls(a.Float64Values(), nulls), nulls), nil

	case *array.Int64:
		return chunk.NewInt64(cloneZeroingNulls(a.Int64Values(), nulls), nulls), nil

	case *array.Boolean:
		vals := make([]bool, n)
		for i := range n {
			if arr.IsValid(i) {
				vals[i] = a.Value(i)
			}
		}
		return chunk.NewBool(vals, nulls), nil

	// A zero-length array may carry no offsets buffer, which ValueOffsets
	// cannot slice.
	case *array.String:
		if n == 0 {
			return chunk.NewString([]string{}, nil), nil
		}
		return chunk.NewString(sliceStringBuffer(a.ValueBytes(), a.ValueOffsets(), nulls), nulls), nil

	case *array.LargeString:
		if n == 0 {
			return chunk.NewString([]string{}, nil), nil
		}
		return chunk.NewString(sliceStringBuffer(a.ValueBytes(), a.ValueOffsets(), nulls), nulls), nil

	case *array.Timestamp:
		vals := make([]time.Time, n)
		unit := a.DataType().(*goarrow.TimestampType).Unit
		for i := range n {
			if arr.IsValid(i) {
				vals[i] = timestampToTime(int64(a.Value(i)), unit)
			}
		}
		return chunk.NewTime(vals, nulls), nil

	case *array.Date32:
		vals := make([]time.Time, n)
		for i := range n {
			if arr.IsValid(i) {
				vals[i] = a.Value(i).ToTime()
			}
		}
		return chunk.NewTime(vals, nulls), nil

	case *array.Date64:
		vals := make([]time.Time, n)
		for i := range n {
			if arr.IsValid(i) {
				vals[i] = a.Value(i).ToTime()
			}
		}
		return chunk.NewTime(vals, nulls), nil

	case *array.Time32:
		unit := a.DataType().(*goarrow.Time32Type).Unit
		vals := make([]time.Time, n)
		for i := range n {
			if arr.IsValid(i) {
				vals[i] = a.Value(i).ToTime(unit)
			}
		}
		return chunk.NewTime(vals, nulls), nil

	case *array.Time64:
		unit := a.DataType().(*goarrow.Time64Type).Unit
		vals := make([]time.Time, n)
		for i := range n {
			if arr.IsValid(i) {
				vals[i] = a.Value(i).ToTime(unit)
			}
		}
		return chunk.NewTime(vals, nulls), nil

	case *array.Binary:
		boxed := make([]any, n)
		for i := range n {
			if arr.IsValid(i) {
				boxed[i] = append([]byte{}, a.Value(i)...)
			}
		}
		return chunk.NewBoxed(dtypes.Binary, boxed, nulls), nil

	case *array.LargeBinary:
		boxed := make([]any, n)
		for i := range n {
			if arr.IsValid(i) {
				boxed[i] = append([]byte{}, a.Value(i)...)
			}
		}
		return chunk.NewBoxed(dtypes.Binary, boxed, nulls), nil

	case *array.MonthDayNanoInterval:
		boxed := make([]any, n)
		for i := range n {
			if arr.IsValid(i) {
				boxed[i] = intervalToDuration(a.Value(i))
			}
		}
		return chunk.NewBoxed(dtypes.Duration, boxed, nulls), nil

	case *array.Decimal128:
		dt := a.DataType().(*goarrow.Decimal128Type)
		boxed := make([]any, n)
		for i := range n {
			if arr.IsValid(i) {
				boxed[i] = dtypes.DecimalValue(a.Value(i).ToString(dt.Scale))
			}
		}
		return chunk.NewBoxed(dtypes.Decimal, boxed, nulls), nil

	case *array.List, *array.LargeList, *array.FixedSizeList:
		return nestedToColumn(arr, n, dtypes.List), nil

	case *array.Struct:
		return nestedToColumn(arr, n, dtypes.Struct), nil

	// gopolars has no 8/16/32-bit dtypes: narrow numbers widen losslessly.
	case *array.Int8:
		return chunk.NewInt64(widenZeroingNulls[int64](a.Int8Values(), nulls), nulls), nil
	case *array.Int16:
		return chunk.NewInt64(widenZeroingNulls[int64](a.Int16Values(), nulls), nulls), nil
	case *array.Int32:
		return chunk.NewInt64(widenZeroingNulls[int64](a.Int32Values(), nulls), nulls), nil
	case *array.Uint8:
		return chunk.NewInt64(widenZeroingNulls[int64](a.Uint8Values(), nulls), nulls), nil
	case *array.Uint16:
		return chunk.NewInt64(widenZeroingNulls[int64](a.Uint16Values(), nulls), nulls), nil
	case *array.Uint32:
		return chunk.NewInt64(widenZeroingNulls[int64](a.Uint32Values(), nulls), nulls), nil
	case *array.Float32:
		return chunk.NewFloat64(widenZeroingNulls[float64](a.Float32Values(), nulls), nulls), nil

	case *array.Uint64:
		vals := make([]int64, n)
		for i, v := range a.Uint64Values() {
			if nulls != nil && nulls[i] {
				continue
			}
			if v > math.MaxInt64 {
				return nil, fmt.Errorf("uint64 value %d at row %d overflows int64", v, i)
			}
			vals[i] = int64(v)
		}
		return chunk.NewInt64(vals, nulls), nil

	case *array.Float16:
		vals := make([]float64, n)
		for i, v := range a.Values() {
			if nulls == nil || !nulls[i] {
				vals[i] = float64(v.Float32())
			}
		}
		return chunk.NewFloat64(vals, nulls), nil

	// View values point into Arrow buffers that may be C-owned (see
	// sliceStringBuffer), so each one is copied.
	case *array.StringView:
		vals := make([]string, n)
		for i := range n {
			if arr.IsValid(i) {
				vals[i] = strings.Clone(a.Value(i))
			}
		}
		return chunk.NewString(vals, nulls), nil

	case *array.BinaryView:
		boxed := make([]any, n)
		for i := range n {
			if arr.IsValid(i) {
				boxed[i] = append([]byte{}, a.Value(i)...)
			}
		}
		return chunk.NewBoxed(dtypes.Binary, boxed, nulls), nil

	case *array.Dictionary:
		dict, err := arrowArrayToColumn(a.Dictionary(), a.Dictionary().Len())
		if err != nil {
			return nil, fmt.Errorf("dictionary values: %w", err)
		}
		indices := make([]int, n)
		for i := range n {
			if a.IsNull(i) {
				indices[i] = -1 // Gather yields a null row
				continue
			}
			indices[i] = a.GetValueIndex(i)
		}
		return dict.Gather(indices), nil

	case *array.Duration:
		unit := a.DataType().(*goarrow.DurationType).Unit
		mult := unit.Multiplier()
		boxed := make([]any, n)
		for i, v := range a.DurationValues() {
			if nulls != nil && nulls[i] {
				continue
			}
			d := time.Duration(v)
			if d < math.MinInt64/mult || d > math.MaxInt64/mult {
				return nil, fmt.Errorf("duration %d%s at row %d overflows time.Duration", v, unit, i)
			}
			boxed[i] = d * mult
		}
		return chunk.NewBoxed(dtypes.Duration, boxed, nulls), nil

	// The null type carries no validity bitmap, so the mask is built here.
	case *array.Null:
		return chunk.NewString(make([]string, n), slices.Repeat([]bool{true}, n)), nil

	default:
		return nil, fmt.Errorf("unsupported arrow type %s", arr.DataType())
	}
}

// widenZeroingNulls copies a narrow Arrow value buffer into a Go-owned int64 or
// float64 slice and zeroes the null slots, whose contents Arrow leaves undefined.
func widenZeroingNulls[T int64 | float64, S int8 | int16 | int32 | uint8 | uint16 | uint32 | float32](values []S, nulls []bool) []T {
	out := make([]T, len(values))
	for i, v := range values {
		if nulls == nil || !nulls[i] {
			out[i] = T(v)
		}
	}
	return out
}

// columnToArrowArray converts one series column to an Arrow array using the
// typed backing slice when available (no []any allocation for primitives).
func columnToArrowArray(s series.Series, alloc memory.Allocator) (goarrow.Array, goarrow.DataType, error) {
	col := s.Column()
	n := s.Len()

	if f64s, ok := col.Float64s(); ok {
		b := array.NewFloat64Builder(alloc)
		b.Reserve(n)
		nulls := col.Nulls()
		for i, v := range f64s {
			if nulls != nil && nulls[i] {
				b.AppendNull()
			} else {
				b.Append(v)
			}
		}
		return b.NewArray(), goarrow.PrimitiveTypes.Float64, nil
	}

	if i64s, ok := col.Int64s(); ok {
		b := array.NewInt64Builder(alloc)
		b.Reserve(n)
		nulls := col.Nulls()
		for i, v := range i64s {
			if nulls != nil && nulls[i] {
				b.AppendNull()
			} else {
				b.Append(v)
			}
		}
		return b.NewArray(), goarrow.PrimitiveTypes.Int64, nil
	}

	if bools, ok := col.Bools(); ok {
		b := array.NewBooleanBuilder(alloc)
		b.Reserve(n)
		nulls := col.Nulls()
		for i, v := range bools {
			if nulls != nil && nulls[i] {
				b.AppendNull()
			} else {
				b.Append(v)
			}
		}
		return b.NewArray(), goarrow.FixedWidthTypes.Boolean, nil
	}

	if strs, ok := col.Strings(); ok {
		b := array.NewStringBuilder(alloc)
		b.Reserve(n)
		nulls := col.Nulls()
		for i, v := range strs {
			if nulls != nil && nulls[i] {
				b.AppendNull()
			} else {
				b.Append(v)
			}
		}
		return b.NewArray(), goarrow.BinaryTypes.String, nil
	}

	if tims, ok := col.Times(); ok {
		dt := &goarrow.TimestampType{Unit: goarrow.Nanosecond}
		b := array.NewTimestampBuilder(alloc, dt)
		b.Reserve(n)
		nulls := col.Nulls()
		for i, v := range tims {
			if nulls != nil && nulls[i] {
				b.AppendNull()
			} else {
				b.Append(goarrow.Timestamp(v.UnixNano()))
			}
		}
		return b.NewArray(), dt, nil
	}

	if dt := s.DataType(); dt == dtypes.List || dt == dtypes.Struct {
		values := make([]any, n)
		for i := range n {
			values[i] = s.Value(i)
		}
		return nestedColumnToArrow(values, alloc)
	}

	if s.DataType() == dtypes.Binary {
		b := array.NewBinaryBuilder(alloc, goarrow.BinaryTypes.Binary)
		b.Reserve(n)
		for i := range n {
			if v, ok := s.Value(i).([]byte); ok {
				b.Append(v)
			} else {
				b.AppendNull()
			}
		}
		return b.NewArray(), goarrow.BinaryTypes.Binary, nil
	}

	// Boxed fallback: build a string array from Value(i).
	b := array.NewStringBuilder(alloc)
	b.Reserve(n)
	for i := range n {
		v := s.Value(i)
		if v == nil {
			b.AppendNull()
		} else {
			b.Append(fmt.Sprintf("%v", v))
		}
	}
	return b.NewArray(), goarrow.BinaryTypes.String, nil
}

// buildNullMask creates a bool slice (true == null) from the Arrow array's
// validity bitmap. Returns nil when there are no nulls, which chunk columns
// treat as "no nulls" without spending n bytes on an all-false mask.
func buildNullMask(arr goarrow.Array, n int) []bool {
	if arr.NullN() == 0 {
		return nil
	}
	nulls := make([]bool, n)
	for i := range n {
		nulls[i] = arr.IsNull(i)
	}
	return nulls
}

// cloneZeroingNulls copies an Arrow value buffer into a Go-owned slice and
// zeroes the null slots, whose contents Arrow leaves undefined.
func cloneZeroingNulls[T int64 | float64](values []T, nulls []bool) []T {
	out := slices.Clone(values)
	for i, null := range nulls {
		if null {
			out[i] = 0
		}
	}
	return out
}

// sliceStringBuffer copies an Arrow string value buffer into one Go string and
// slices every non-null value out of it: one allocation per column instead of
// one per value, and no reference into the Arrow buffer, which is C-owned and
// freed on Release for records imported over the C Data Interface (ADBC).
// offsets are absolute positions in the array's buffer, as ValueOffsets
// returns them, so sliced arrays rebase on offsets[0].
func sliceStringBuffer[O int32 | int64](data []byte, offsets []O, nulls []bool) []string {
	buf := string(data)
	base := offsets[0]
	vals := make([]string, len(offsets)-1)
	for i := range vals {
		if nulls == nil || !nulls[i] {
			vals[i] = buf[offsets[i]-base : offsets[i+1]-base]
		}
	}
	return vals
}

func timestampToTime(v int64, unit goarrow.TimeUnit) time.Time {
	switch unit {
	case goarrow.Second:
		return time.Unix(v, 0).UTC()
	case goarrow.Millisecond:
		return time.UnixMilli(v).UTC()
	case goarrow.Microsecond:
		return time.UnixMicro(v).UTC()
	default: // Nanosecond
		return time.Unix(0, v).UTC()
	}
}

// ToTable builds the legacy Table (map[string][]any) for gob-serialized IPC.
// For primitive dtypes the typed backing is read directly to avoid per-element
// interface boxing.
func ToTable(df frame.DataFrame) Table {
	out := Table{Columns: map[string][]any{}, Names: df.Columns()}
	for _, name := range out.Names {
		s, _ := df.Series(name)
		col := s.Column()
		n := s.Len()
		values := make([]any, n)

		if f64s, ok := col.Float64s(); ok {
			nulls := col.Nulls()
			for i, v := range f64s {
				if nulls == nil || !nulls[i] {
					values[i] = v
				}
			}
		} else if i64s, ok := col.Int64s(); ok {
			nulls := col.Nulls()
			for i, v := range i64s {
				if nulls == nil || !nulls[i] {
					values[i] = v
				}
			}
		} else if bools, ok := col.Bools(); ok {
			nulls := col.Nulls()
			for i, v := range bools {
				if nulls == nil || !nulls[i] {
					values[i] = v
				}
			}
		} else if strs, ok := col.Strings(); ok {
			nulls := col.Nulls()
			for i, v := range strs {
				if nulls == nil || !nulls[i] {
					values[i] = v
				}
			}
		} else if tims, ok := col.Times(); ok {
			nulls := col.Nulls()
			for i, v := range tims {
				if nulls == nil || !nulls[i] {
					values[i] = v
				}
			}
		} else {
			// Boxed fallback via Value(i).
			for i := range n {
				values[i] = s.Value(i)
			}
		}

		out.Columns[name] = values
	}
	return out
}

// FromTable converts the legacy Table type back to a DataFrame, with columns
// in the order Table.Names describes.
func FromTable(t Table) (frame.DataFrame, error) {
	rest := maps.Clone(t.Columns)
	seriesInput := make([]frame.SeriesInput, 0, len(t.Columns))
	for _, name := range t.Names {
		if values, ok := rest[name]; ok {
			seriesInput = append(seriesInput, frame.SeriesInput{Name: name, Values: values})
			delete(rest, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(rest)) {
		seriesInput = append(seriesInput, frame.SeriesInput{Name: name, Values: rest[name]})
	}
	return frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: seriesInput})
}
