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
}

// FromArrowRecord imports an Arrow record batch into a DataFrame without
// allocating an intermediate []any for primitive dtypes (Float64, Int64,
// Boolean, String, Timestamp). Unsupported Arrow types fall back to a boxed
// slow path and return a non-nil warning via the second return value.
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
		for i := 0; i < n; i++ {
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
		for i := 0; i < n; i++ {
			if arr.IsValid(i) {
				vals[i] = timestampToTime(int64(a.Value(i)), unit)
			}
		}
		return chunk.NewTime(vals, nulls), nil

	case *array.Date32:
		vals := make([]time.Time, n)
		for i := 0; i < n; i++ {
			if arr.IsValid(i) {
				vals[i] = a.Value(i).ToTime()
			}
		}
		return chunk.NewTime(vals, nulls), nil

	case *array.Date64:
		vals := make([]time.Time, n)
		for i := 0; i < n; i++ {
			if arr.IsValid(i) {
				vals[i] = a.Value(i).ToTime()
			}
		}
		return chunk.NewTime(vals, nulls), nil

	case *array.Time32:
		unit := a.DataType().(*goarrow.Time32Type).Unit
		vals := make([]time.Time, n)
		for i := 0; i < n; i++ {
			if arr.IsValid(i) {
				vals[i] = a.Value(i).ToTime(unit)
			}
		}
		return chunk.NewTime(vals, nulls), nil

	case *array.Time64:
		unit := a.DataType().(*goarrow.Time64Type).Unit
		vals := make([]time.Time, n)
		for i := 0; i < n; i++ {
			if arr.IsValid(i) {
				vals[i] = a.Value(i).ToTime(unit)
			}
		}
		return chunk.NewTime(vals, nulls), nil

	case *array.Binary:
		boxed := make([]any, n)
		for i := 0; i < n; i++ {
			if arr.IsValid(i) {
				boxed[i] = append([]byte{}, a.Value(i)...)
			}
		}
		return chunk.NewBoxed(dtypes.Binary, boxed, nulls), nil

	case *array.LargeBinary:
		boxed := make([]any, n)
		for i := 0; i < n; i++ {
			if arr.IsValid(i) {
				boxed[i] = append([]byte{}, a.Value(i)...)
			}
		}
		return chunk.NewBoxed(dtypes.Binary, boxed, nulls), nil

	case *array.MonthDayNanoInterval:
		boxed := make([]any, n)
		for i := 0; i < n; i++ {
			if arr.IsValid(i) {
				boxed[i] = intervalToDuration(a.Value(i))
			}
		}
		return chunk.NewBoxed(dtypes.Duration, boxed, nulls), nil

	case *array.Decimal128:
		dt := a.DataType().(*goarrow.Decimal128Type)
		boxed := make([]any, n)
		for i := 0; i < n; i++ {
			if arr.IsValid(i) {
				boxed[i] = dtypes.DecimalValue(a.Value(i).ToString(dt.Scale))
			}
		}
		return chunk.NewBoxed(dtypes.Decimal, boxed, nulls), nil

	case *array.List, *array.LargeList, *array.FixedSizeList:
		return nestedToColumn(arr, n, dtypes.List), nil

	case *array.Struct:
		return nestedToColumn(arr, n, dtypes.Struct), nil

	default:
		// Unsupported Arrow type: box into []any. Clone string/[]byte values so
		// the boxed column does not alias a C-owned Arrow buffer that is freed on
		// the record's Release (see sliceStringBuffer).
		boxed := make([]any, n)
		for i := 0; i < n; i++ {
			if arr.IsNull(i) {
				continue
			}
			switch v := arr.GetOneForMarshal(i).(type) {
			case string:
				boxed[i] = strings.Clone(v)
			case []byte:
				boxed[i] = append([]byte(nil), v...)
			default:
				boxed[i] = v
			}
		}
		return chunk.NewBoxed(dtypes.String, boxed, nulls), nil
	}
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
		for i := 0; i < n; i++ {
			values[i] = s.Value(i)
		}
		return nestedColumnToArrow(values, col.Nulls(), alloc)
	}

	if s.DataType() == dtypes.Binary {
		b := array.NewBinaryBuilder(alloc, goarrow.BinaryTypes.Binary)
		b.Reserve(n)
		for i := 0; i < n; i++ {
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
	for i := 0; i < n; i++ {
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
	out := Table{Columns: map[string][]any{}}
	for _, name := range df.Columns() {
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
			for i := 0; i < n; i++ {
				values[i] = s.Value(i)
			}
		}

		out.Columns[name] = values
	}
	return out
}

// FromTable converts the legacy Table type back to a DataFrame.
func FromTable(t Table) (frame.DataFrame, error) {
	seriesInput := make([]frame.SeriesInput, 0, len(t.Columns))
	for k, v := range t.Columns {
		seriesInput = append(seriesInput, frame.SeriesInput{Name: k, Values: v})
	}
	return frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: seriesInput})
}
