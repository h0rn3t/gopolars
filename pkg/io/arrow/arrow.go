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
		vals := make([]float64, n)
		for i := 0; i < n; i++ {
			if !nulls[i] {
				vals[i] = a.Value(i)
			}
		}
		return chunk.NewFloat64(vals, nulls), nil

	case *array.Int64:
		vals := make([]int64, n)
		for i := 0; i < n; i++ {
			if !nulls[i] {
				vals[i] = a.Value(i)
			}
		}
		return chunk.NewInt64(vals, nulls), nil

	case *array.Boolean:
		vals := make([]bool, n)
		for i := 0; i < n; i++ {
			if !nulls[i] {
				vals[i] = a.Value(i)
			}
		}
		return chunk.NewBool(vals, nulls), nil

	case *array.String:
		vals := make([]string, n)
		for i := 0; i < n; i++ {
			if !nulls[i] {
				// strings.Clone forces a Go-owned heap copy. a.Value(i) is an
				// unsafe view into the Arrow value buffer; for records imported
				// over the C Data Interface (e.g. ADBC read_database) that buffer
				// is C-owned and freed on the record's Release, so without a copy
				// the column would dangle into freed memory.
				vals[i] = strings.Clone(a.Value(i))
			}
		}
		return chunk.NewString(vals, nulls), nil

	case *array.LargeString:
		vals := make([]string, n)
		for i := 0; i < n; i++ {
			if !nulls[i] {
				vals[i] = strings.Clone(a.Value(i))
			}
		}
		return chunk.NewString(vals, nulls), nil

	case *array.Timestamp:
		vals := make([]time.Time, n)
		unit := a.DataType().(*goarrow.TimestampType).Unit
		for i := 0; i < n; i++ {
			if !nulls[i] {
				vals[i] = timestampToTime(int64(a.Value(i)), unit)
			}
		}
		return chunk.NewTime(vals, nulls), nil

	default:
		// Unsupported Arrow type: box into []any. Clone string/[]byte values so
		// the boxed column does not alias a C-owned Arrow buffer that is freed on
		// the record's Release (see the *array.String case).
		boxed := make([]any, n)
		for i := 0; i < n; i++ {
			if nulls[i] {
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
// validity bitmap. Returns a zeroed slice when there are no nulls.
func buildNullMask(arr goarrow.Array, n int) []bool {
	if arr.NullN() == 0 {
		return make([]bool, n)
	}
	nulls := make([]bool, n)
	for i := 0; i < n; i++ {
		nulls[i] = arr.IsNull(i)
	}
	return nulls
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
