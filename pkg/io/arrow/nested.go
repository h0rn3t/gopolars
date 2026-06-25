package arrow

import (
	"fmt"
	"strings"
	"time"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/arrow/memory"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// arrowValueAt extracts a single Go-native value from an Arrow array at row i,
// recursing through List/Struct so nested values become []any / map[string]any.
// Narrow numeric leaves (int8/16/32, uint*, float32, decimal128) are widened to
// int64/float64 to match gopolars' native dtypes. Returns nil for null rows.
func arrowValueAt(arr goarrow.Array, i int) any {
	if arr.IsNull(i) {
		return nil
	}
	switch a := arr.(type) {
	case *array.Float64:
		return a.Value(i)
	case *array.Float32:
		return float64(a.Value(i))
	case *array.Int64:
		return a.Value(i)
	case *array.Int32:
		return int64(a.Value(i))
	case *array.Int16:
		return int64(a.Value(i))
	case *array.Int8:
		return int64(a.Value(i))
	case *array.Uint64:
		return int64(a.Value(i))
	case *array.Uint32:
		return int64(a.Value(i))
	case *array.Uint16:
		return int64(a.Value(i))
	case *array.Uint8:
		return int64(a.Value(i))
	case *array.Boolean:
		return a.Value(i)
	case *array.String:
		return strings.Clone(a.Value(i))
	case *array.LargeString:
		return strings.Clone(a.Value(i))
	case *array.Timestamp:
		unit := a.DataType().(*goarrow.TimestampType).Unit
		return timestampToTime(int64(a.Value(i)), unit)
	case *array.Date32:
		return a.Value(i).ToTime()
	case *array.Date64:
		return a.Value(i).ToTime()
	case *array.Time32:
		return a.Value(i).ToTime(a.DataType().(*goarrow.Time32Type).Unit)
	case *array.Time64:
		return a.Value(i).ToTime(a.DataType().(*goarrow.Time64Type).Unit)
	case *array.Binary:
		return append([]byte(nil), a.Value(i)...)
	case *array.LargeBinary:
		return append([]byte(nil), a.Value(i)...)
	case *array.MonthDayNanoInterval:
		return intervalToDuration(a.Value(i))
	case *array.Decimal128:
		dt := a.DataType().(*goarrow.Decimal128Type)
		return a.Value(i).ToFloat64(dt.Scale)
	case *array.List:
		start, end := a.ValueOffsets(i)
		return sliceValues(a.ListValues(), int(start), int(end))
	case *array.LargeList:
		start, end := a.ValueOffsets(i)
		return sliceValues(a.ListValues(), int(start), int(end))
	case *array.FixedSizeList:
		start, end := a.ValueOffsets(i)
		return sliceValues(a.ListValues(), int(start), int(end))
	case *array.Struct:
		st := a.DataType().(*goarrow.StructType)
		m := make(map[string]any, a.NumField())
		for j := 0; j < a.NumField(); j++ {
			m[st.Field(j).Name] = arrowValueAt(a.Field(j), i)
		}
		return m
	default:
		return arr.GetOneForMarshal(i)
	}
}

// intervalToDuration flattens an Arrow month_day_nano interval into a fixed
// time.Duration. polars' Duration has no calendar component, so days are treated
// as 24h and months (absent in fixed intervals) as 30 days.
func intervalToDuration(iv goarrow.MonthDayNanoInterval) time.Duration {
	return time.Duration(iv.Months)*30*24*time.Hour +
		time.Duration(iv.Days)*24*time.Hour +
		time.Duration(iv.Nanoseconds)
}

// sliceValues extracts child[start:end] as a []any of native values.
func sliceValues(child goarrow.Array, start, end int) []any {
	out := make([]any, 0, end-start)
	for k := start; k < end; k++ {
		out = append(out, arrowValueAt(child, k))
	}
	return out
}

// nestedToColumn builds a boxed List/Struct gopolars column from an Arrow
// List/Struct array.
func nestedToColumn(arr goarrow.Array, n int, dtype dtypes.DataType) *chunk.Column {
	nulls := buildNullMask(arr, n)
	boxed := make([]any, n)
	for i := 0; i < n; i++ {
		if !nulls[i] {
			boxed[i] = arrowValueAt(arr, i)
		}
	}
	return chunk.NewBoxed(dtype, boxed, nulls)
}

// inferArrowType determines the Arrow element type for a boxed List/Struct
// column by scanning its values (and, recursively, their leaves). Empty/all-null
// input defaults to String so an Arrow array can still be built.
func inferArrowType(values []any) (goarrow.DataType, error) {
	for _, v := range values {
		if v == nil {
			continue
		}
		switch x := v.(type) {
		case int64:
			return goarrow.PrimitiveTypes.Int64, nil
		case float64:
			return goarrow.PrimitiveTypes.Float64, nil
		case string:
			return goarrow.BinaryTypes.String, nil
		case dtypes.DecimalValue:
			return goarrow.BinaryTypes.String, nil
		case bool:
			return goarrow.FixedWidthTypes.Boolean, nil
		case time.Time:
			return &goarrow.TimestampType{Unit: goarrow.Nanosecond}, nil
		case []any:
			child, err := inferArrowType(flattenLists(values))
			if err != nil {
				return nil, err
			}
			return goarrow.ListOf(child), nil
		case map[string]any:
			return inferStructType(values)
		default:
			return nil, fmt.Errorf("cannot infer arrow type for %T", x)
		}
	}
	return goarrow.BinaryTypes.String, nil
}

// flattenLists concatenates the element slices of every non-nil []any value.
func flattenLists(values []any) []any {
	var out []any
	for _, v := range values {
		if lst, ok := v.([]any); ok {
			out = append(out, lst...)
		}
	}
	return out
}

// inferStructType builds a StructType from the union of field names (ordered by
// first appearance), inferring each field's type from its values across rows.
func inferStructType(values []any) (goarrow.DataType, error) {
	var order []string
	seen := map[string]bool{}
	for _, v := range values {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		for k := range m {
			if !seen[k] {
				seen[k] = true
				order = append(order, k)
			}
		}
	}
	// Stable field order: maps iterate randomly, so sort for determinism.
	sortStrings(order)
	fields := make([]goarrow.Field, 0, len(order))
	for _, name := range order {
		col := make([]any, 0, len(values))
		for _, v := range values {
			if m, ok := v.(map[string]any); ok {
				col = append(col, m[name])
			}
		}
		ft, err := inferArrowType(col)
		if err != nil {
			return nil, fmt.Errorf("struct field %q: %w", name, err)
		}
		fields = append(fields, goarrow.Field{Name: name, Type: ft, Nullable: true})
	}
	return goarrow.StructOf(fields...), nil
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// nestedColumnToArrow builds an Arrow array for a boxed List/Struct column.
func nestedColumnToArrow(values []any, nulls []bool, mem memory.Allocator) (goarrow.Array, goarrow.DataType, error) {
	dt, err := inferArrowType(values)
	if err != nil {
		return nil, nil, err
	}
	b := array.NewBuilder(mem, dt)
	defer b.Release()
	for i, v := range values {
		if nulls != nil && nulls[i] {
			b.AppendNull()
			continue
		}
		if err := appendArrowValue(b, dt, v); err != nil {
			return nil, nil, err
		}
	}
	return b.NewArray(), dt, nil
}

// appendArrowValue recursively appends a Go-native value to an Arrow builder
// whose type is dt. A nil value appends a null.
func appendArrowValue(b array.Builder, dt goarrow.DataType, v any) error {
	if v == nil {
		b.AppendNull()
		return nil
	}
	switch bb := b.(type) {
	case *array.Int64Builder:
		bb.Append(toInt64(v))
	case *array.Float64Builder:
		bb.Append(toFloat64(v))
	case *array.StringBuilder:
		bb.Append(toStr(v))
	case *array.BooleanBuilder:
		bb.Append(v.(bool))
	case *array.TimestampBuilder:
		bb.Append(goarrow.Timestamp(v.(time.Time).UnixNano()))
	case *array.Decimal128Builder:
		bb.Append(decimal128.FromI64(toInt64(v)))
	case *array.ListBuilder:
		lst, ok := v.([]any)
		if !ok {
			return fmt.Errorf("expected []any for list, got %T", v)
		}
		bb.Append(true)
		child := dt.(*goarrow.ListType).Elem()
		vb := bb.ValueBuilder()
		for _, e := range lst {
			if err := appendArrowValue(vb, child, e); err != nil {
				return err
			}
		}
	case *array.StructBuilder:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("expected map[string]any for struct, got %T", v)
		}
		st := dt.(*goarrow.StructType)
		bb.Append(true)
		for j := 0; j < bb.NumField(); j++ {
			f := st.Field(j)
			if err := appendArrowValue(bb.FieldBuilder(j), f.Type, m[f.Name]); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported builder %T", b)
	}
	return nil
}

func toInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	default:
		return 0
	}
}

func toStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case dtypes.DecimalValue:
		return string(x)
	default:
		return fmt.Sprintf("%v", v)
	}
}
