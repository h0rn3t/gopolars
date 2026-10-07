package arrow_test

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/float16"
	"github.com/apache/arrow-go/v18/arrow/memory"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

func recordOf(name string, arr goarrow.Array) goarrow.RecordBatch {
	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: name, Type: arr.DataType(), Nullable: true},
	}, nil)
	return array.NewRecordBatch(schema, []goarrow.Array{arr}, int64(arr.Len()))
}

// TestFromArrowLargeString covers the *array.LargeString case.
func TestFromArrowLargeString(t *testing.T) {
	alloc := memory.NewGoAllocator()
	b := array.NewLargeStringBuilder(alloc)
	b.Append("alpha")
	b.AppendNull()
	b.Append("gamma")
	arr := b.NewArray()
	defer arr.Release()
	rec := recordOf("v", arr)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	s, _ := df.Series("v")
	if v, _ := s.Value(0).(string); v != "alpha" {
		t.Errorf("row 0 = %v, want alpha", s.Value(0))
	}
	if !s.IsNull(1) {
		t.Error("row 1 should be null")
	}
}

// TestFromArrowRecordConvertsTypes checks that Arrow types without a native
// column are converted losslessly at import and that every row reads back.
func TestFromArrowRecordConvertsTypes(t *testing.T) {
	const long = "a string longer than twelve bytes"
	cases := []struct {
		name      string
		build     func(mem memory.Allocator) goarrow.Array
		wantDtype dtypes.DataType
		want      []any
	}{
		{
			name: "int8",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewInt8Builder(mem)
				defer b.Release()
				b.AppendValues([]int8{math.MinInt8, 0, math.MaxInt8}, []bool{true, false, true})
				return b.NewArray()
			},
			wantDtype: dtypes.Int64,
			want:      []any{int64(math.MinInt8), nil, int64(math.MaxInt8)},
		},
		{
			name: "int16",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewInt16Builder(mem)
				defer b.Release()
				b.AppendValues([]int16{math.MinInt16, math.MaxInt16}, nil)
				return b.NewArray()
			},
			wantDtype: dtypes.Int64,
			want:      []any{int64(math.MinInt16), int64(math.MaxInt16)},
		},
		{
			name: "int32 with null",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewInt32Builder(mem)
				defer b.Release()
				b.AppendValues([]int32{7, 0, 9}, []bool{true, false, true})
				return b.NewArray()
			},
			wantDtype: dtypes.Int64,
			want:      []any{int64(7), nil, int64(9)},
		},
		{
			name: "int32 slice",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewInt32Builder(mem)
				defer b.Release()
				b.AppendValues([]int32{1, 2, 0, 4}, []bool{true, true, false, true})
				arr := b.NewArray()
				defer arr.Release()
				return array.NewSlice(arr, 1, 4)
			},
			wantDtype: dtypes.Int64,
			want:      []any{int64(2), nil, int64(4)},
		},
		{
			name: "uint8",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewUint8Builder(mem)
				defer b.Release()
				b.AppendValues([]uint8{0, math.MaxUint8}, nil)
				return b.NewArray()
			},
			wantDtype: dtypes.Int64,
			want:      []any{int64(0), int64(math.MaxUint8)},
		},
		{
			name: "uint16",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewUint16Builder(mem)
				defer b.Release()
				b.AppendValues([]uint16{math.MaxUint16}, nil)
				return b.NewArray()
			},
			wantDtype: dtypes.Int64,
			want:      []any{int64(math.MaxUint16)},
		},
		{
			name: "uint32",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewUint32Builder(mem)
				defer b.Release()
				b.AppendValues([]uint32{math.MaxUint32, 0}, []bool{true, false})
				return b.NewArray()
			},
			wantDtype: dtypes.Int64,
			want:      []any{int64(math.MaxUint32), nil},
		},
		{
			name: "uint64 within int64",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewUint64Builder(mem)
				defer b.Release()
				// The null slot holds a value above MaxInt64: it must not count.
				b.AppendValues([]uint64{0, math.MaxInt64, math.MaxUint64}, []bool{true, true, false})
				return b.NewArray()
			},
			wantDtype: dtypes.Int64,
			want:      []any{int64(0), int64(math.MaxInt64), nil},
		},
		{
			name: "float16",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewFloat16Builder(mem)
				defer b.Release()
				b.AppendValues([]float16.Num{float16.New(1.5), {}, float16.New(-0.25)}, []bool{true, false, true})
				return b.NewArray()
			},
			wantDtype: dtypes.Float64,
			want:      []any{1.5, nil, -0.25},
		},
		{
			name: "float32",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewFloat32Builder(mem)
				defer b.Release()
				b.AppendValues([]float32{1.5, 0, 0.1}, []bool{true, false, true})
				return b.NewArray()
			},
			wantDtype: dtypes.Float64,
			want:      []any{1.5, nil, float64(float32(0.1))},
		},
		{
			name: "string view",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewStringViewBuilder(mem)
				defer b.Release()
				b.AppendValues([]string{"short", "", long}, []bool{true, false, true})
				return b.NewArray()
			},
			wantDtype: dtypes.String,
			want:      []any{"short", nil, long},
		},
		{
			name: "binary view",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewBinaryViewBuilder(mem)
				defer b.Release()
				b.AppendValues([][]byte{[]byte("ab"), nil, []byte(long)}, []bool{true, false, true})
				return b.NewArray()
			},
			wantDtype: dtypes.Binary,
			want:      []any{[]byte("ab"), nil, []byte(long)},
		},
		{
			name: "dictionary of strings with null",
			build: func(mem memory.Allocator) goarrow.Array {
				dt := &goarrow.DictionaryType{IndexType: goarrow.PrimitiveTypes.Int32, ValueType: goarrow.BinaryTypes.String}
				b := array.NewDictionaryBuilder(mem, dt).(*array.BinaryDictionaryBuilder)
				defer b.Release()
				for _, v := range []string{"a", "", "b", "a"} {
					if v == "" {
						b.AppendNull()
						continue
					}
					if err := b.AppendString(v); err != nil {
						t.Fatalf("AppendString(%q): %v", v, err)
					}
				}
				return b.NewArray()
			},
			wantDtype: dtypes.String,
			want:      []any{"a", nil, "b", "a"},
		},
		{
			name: "dictionary of int32",
			build: func(mem memory.Allocator) goarrow.Array {
				dt := &goarrow.DictionaryType{IndexType: goarrow.PrimitiveTypes.Int8, ValueType: goarrow.PrimitiveTypes.Int32}
				b := array.NewDictionaryBuilder(mem, dt).(*array.Int32DictionaryBuilder)
				defer b.Release()
				for _, v := range []int32{10, 20, 10} {
					if err := b.Append(v); err != nil {
						t.Fatalf("Append(%d): %v", v, err)
					}
				}
				b.AppendNull()
				return b.NewArray()
			},
			wantDtype: dtypes.Int64,
			want:      []any{int64(10), int64(20), int64(10), nil},
		},
		{
			name: "duration ms",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewDurationBuilder(mem, &goarrow.DurationType{Unit: goarrow.Millisecond})
				defer b.Release()
				b.AppendValues([]goarrow.Duration{1500, 0, -3}, []bool{true, false, true})
				return b.NewArray()
			},
			wantDtype: dtypes.Duration,
			want:      []any{1500 * time.Millisecond, nil, -3 * time.Millisecond},
		},
		{
			name: "duration s",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewDurationBuilder(mem, &goarrow.DurationType{Unit: goarrow.Second})
				defer b.Release()
				b.Append(90)
				return b.NewArray()
			},
			wantDtype: dtypes.Duration,
			want:      []any{90 * time.Second},
		},
		{
			name: "duration ns extremes",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewDurationBuilder(mem, &goarrow.DurationType{Unit: goarrow.Nanosecond})
				defer b.Release()
				b.AppendValues([]goarrow.Duration{math.MinInt64, math.MaxInt64}, nil)
				return b.NewArray()
			},
			wantDtype: dtypes.Duration,
			want:      []any{time.Duration(math.MinInt64), time.Duration(math.MaxInt64)},
		},
		{
			name:      "null type",
			build:     func(memory.Allocator) goarrow.Array { return array.NewNull(3) },
			wantDtype: dtypes.String,
			want:      []any{nil, nil, nil},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			arr := tc.build(memory.NewGoAllocator())
			defer arr.Release()
			rec := recordOf("v", arr)
			defer rec.Release()

			df, err := iarrow.FromArrowRecord(rec)
			if err != nil {
				t.Fatalf("FromArrowRecord(%s) error = %v, want nil", arr.DataType(), err)
			}
			s, ok := df.Series("v")
			if !ok {
				t.Fatalf("FromArrowRecord(%s) columns = %v, want [v]", arr.DataType(), df.Columns())
			}
			if got := s.DataType(); got != tc.wantDtype {
				t.Errorf("FromArrowRecord(%s) dtype = %s, want %s", arr.DataType(), got, tc.wantDtype)
			}
			got := make([]any, s.Len())
			for i := range got {
				got[i] = s.Value(i)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("FromArrowRecord(%s) values = %#v, want %#v", arr.DataType(), got, tc.want)
			}
		})
	}
}

// TestFromArrowUnsupportedType checks that a value that cannot be represented
// and an Arrow type without a mapping fail at import, naming the column,
// instead of producing a column that panics when read.
func TestFromArrowUnsupportedType(t *testing.T) {
	cases := []struct {
		name    string
		build   func(mem memory.Allocator) goarrow.Array
		wantErr []string
	}{
		{
			name: "uint64 above MaxInt64",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewUint64Builder(mem)
				defer b.Release()
				b.AppendValues([]uint64{1, 1 << 63}, nil)
				return b.NewArray()
			},
			wantErr: []string{`column "v"`, "overflow"},
		},
		{
			name: "duration above time.Duration",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewDurationBuilder(mem, &goarrow.DurationType{Unit: goarrow.Second})
				defer b.Release()
				b.Append(math.MaxInt64 / 1000)
				return b.NewArray()
			},
			wantErr: []string{`column "v"`, "overflow"},
		},
		{
			name: "map",
			build: func(mem memory.Allocator) goarrow.Array {
				b := array.NewMapBuilder(mem, goarrow.BinaryTypes.String, goarrow.PrimitiveTypes.Int64, false)
				defer b.Release()
				b.Append(true)
				b.KeyBuilder().(*array.StringBuilder).Append("k")
				b.ItemBuilder().(*array.Int64Builder).Append(1)
				return b.NewArray()
			},
			wantErr: []string{`column "v"`, "unsupported arrow type map<"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			arr := tc.build(memory.NewGoAllocator())
			defer arr.Release()
			rec := recordOf("v", arr)
			defer rec.Release()

			_, err := iarrow.FromArrowRecord(rec)
			if err == nil {
				t.Fatalf("FromArrowRecord(%s) error = nil, want an error containing %q", arr.DataType(), tc.wantErr)
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("FromArrowRecord(%s) error = %q, want it to contain %q", arr.DataType(), err, want)
				}
			}
		})
	}
}

// TestFromArrowTimestampUnits covers timestampToTime for Second/Millisecond/
// Microsecond units (Nanosecond is covered elsewhere).
func TestFromArrowTimestampUnits(t *testing.T) {
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name string
		unit goarrow.TimeUnit
		val  int64
	}{
		{"second", goarrow.Second, base.Unix()},
		{"millisecond", goarrow.Millisecond, base.UnixMilli()},
		{"microsecond", goarrow.Microsecond, base.UnixMicro()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			alloc := memory.NewGoAllocator()
			dt := &goarrow.TimestampType{Unit: tc.unit}
			b := array.NewTimestampBuilder(alloc, dt)
			b.Append(goarrow.Timestamp(tc.val))
			arr := b.NewArray()
			defer arr.Release()
			rec := recordOf("t", arr)
			defer rec.Release()

			df, err := iarrow.FromArrowRecord(rec)
			if err != nil {
				t.Fatalf("FromArrowRecord: %v", err)
			}
			s, _ := df.Series("t")
			got, ok := s.Value(0).(time.Time)
			if !ok {
				t.Fatalf("expected time.Time, got %T", s.Value(0))
			}
			if !got.Equal(base) {
				t.Errorf("%s: got %v, want %v", tc.name, got, base)
			}
		})
	}
}

// TestToArrowBoolWithNull covers the AppendNull path of the Bools branch.
func TestToArrowBoolWithNull(t *testing.T) {
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "b", Values: []any{true, nil, false}},
	}})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}
	rec, err := iarrow.ToArrowRecord(df)
	if err != nil {
		t.Fatalf("ToArrowRecord: %v", err)
	}
	defer rec.Release()
	ba, ok := rec.Column(0).(*array.Boolean)
	if !ok {
		t.Fatalf("expected Boolean array, got %T", rec.Column(0))
	}
	if !ba.IsNull(1) {
		t.Error("row 1 should be null")
	}
}

// TestToArrowTimeColumn covers the Times branch of columnToArrowArray.
func TestToArrowTimeColumn(t *testing.T) {
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "t", Values: []any{
			time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			nil,
		}},
	}})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}
	rec, err := iarrow.ToArrowRecord(df)
	if err != nil {
		t.Fatalf("ToArrowRecord: %v", err)
	}
	defer rec.Release()
	ts, ok := rec.Column(0).(*array.Timestamp)
	if !ok {
		t.Fatalf("expected Timestamp array, got %T", rec.Column(0))
	}
	if !ts.IsNull(1) {
		t.Error("row 1 should be null")
	}
}

// TestToArrowBoxedFallback covers the boxed string-fallback branch for a dtype
// without a typed or nested Arrow backing (Decimal), including the AppendNull path.
func TestToArrowBoxedFallback(t *testing.T) {
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "s", Values: []any{
			dtypes.DecimalValue("1.5"),
			nil,
		}},
	}})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}
	rec, err := iarrow.ToArrowRecord(df)
	if err != nil {
		t.Fatalf("ToArrowRecord: %v", err)
	}
	defer rec.Release()
	sa, ok := rec.Column(0).(*array.String)
	if !ok {
		t.Fatalf("expected String array (boxed fallback), got %T", rec.Column(0))
	}
	if !sa.IsNull(1) {
		t.Error("row 1 should be null")
	}
}
