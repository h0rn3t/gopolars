package arrow_test

import (
	"testing"
	"time"

	goarrow "github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"

	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

func recordOf(name string, arr goarrow.Array) goarrow.Record {
	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: name, Type: arr.DataType(), Nullable: true},
	}, nil)
	return array.NewRecord(schema, []goarrow.Array{arr}, int64(arr.Len()))
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

// TestFromArrowUnsupportedType covers the default boxed branch using an Arrow
// type without a typed mapping (Int32).
func TestFromArrowUnsupportedType(t *testing.T) {
	alloc := memory.NewGoAllocator()
	b := array.NewInt32Builder(alloc)
	b.Append(7)
	b.AppendNull()
	b.Append(9)
	arr := b.NewArray()
	defer arr.Release()
	rec := recordOf("v", arr)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	if df.Height() != 3 {
		t.Errorf("Height = %d, want 3", df.Height())
	}
	s, _ := df.Series("v")
	if !s.IsNull(1) {
		t.Error("row 1 should be null")
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
// without a typed backing (struct), including the AppendNull path.
func TestToArrowBoxedFallback(t *testing.T) {
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "s", Values: []any{
			map[string]any{"x": int64(1)},
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
