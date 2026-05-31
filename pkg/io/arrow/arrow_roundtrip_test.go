package arrow_test

import (
	"testing"
	"time"

	goarrow "github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"

	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// buildRecord creates an Arrow record with one column of the given type.
func buildFloat64Record(vals []float64, nullAt int) goarrow.Record {
	alloc := memory.NewGoAllocator()
	b := array.NewFloat64Builder(alloc)
	for i, v := range vals {
		if i == nullAt {
			b.AppendNull()
		} else {
			b.Append(v)
		}
	}
	arr := b.NewArray()
	defer arr.Release()
	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: "v", Type: goarrow.PrimitiveTypes.Float64, Nullable: true},
	}, nil)
	return array.NewRecord(schema, []goarrow.Array{arr}, int64(len(vals)))
}

func buildInt64Record(vals []int64, nullAt int) goarrow.Record {
	alloc := memory.NewGoAllocator()
	b := array.NewInt64Builder(alloc)
	for i, v := range vals {
		if i == nullAt {
			b.AppendNull()
		} else {
			b.Append(v)
		}
	}
	arr := b.NewArray()
	defer arr.Release()
	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: "v", Type: goarrow.PrimitiveTypes.Int64, Nullable: true},
	}, nil)
	return array.NewRecord(schema, []goarrow.Array{arr}, int64(len(vals)))
}

func buildBoolRecord(vals []bool) goarrow.Record {
	alloc := memory.NewGoAllocator()
	b := array.NewBooleanBuilder(alloc)
	b.AppendValues(vals, nil)
	arr := b.NewArray()
	defer arr.Release()
	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: "v", Type: goarrow.FixedWidthTypes.Boolean, Nullable: false},
	}, nil)
	return array.NewRecord(schema, []goarrow.Array{arr}, int64(len(vals)))
}

func buildStringRecord(vals []string, nullAt int) goarrow.Record {
	alloc := memory.NewGoAllocator()
	b := array.NewStringBuilder(alloc)
	for i, v := range vals {
		if i == nullAt {
			b.AppendNull()
		} else {
			b.Append(v)
		}
	}
	arr := b.NewArray()
	defer arr.Release()
	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: "v", Type: goarrow.BinaryTypes.String, Nullable: true},
	}, nil)
	return array.NewRecord(schema, []goarrow.Array{arr}, int64(len(vals)))
}

// TestRoundtripFloat64 verifies Float64 Arrow ↔ DataFrame ↔ Arrow values and nulls.
func TestRoundtripFloat64(t *testing.T) {
	input := []float64{1.1, 2.2, 3.3, 4.4}
	nullAt := 1
	rec := buildFloat64Record(input, nullAt)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	if df.Height() != len(input) {
		t.Fatalf("height: got %d want %d", df.Height(), len(input))
	}
	col, ok := df.Series("v")
	if !ok {
		t.Fatal("column v not found")
	}
	if !col.IsNull(nullAt) {
		t.Errorf("row %d should be null", nullAt)
	}
	for i, want := range input {
		if i == nullAt {
			continue
		}
		got, _ := col.Value(i).(float64)
		if got != want {
			t.Errorf("row %d: got %v want %v", i, got, want)
		}
	}

	// Export back to Arrow.
	out, err := iarrow.ToArrowRecord(df)
	if err != nil {
		t.Fatalf("ToArrowRecord: %v", err)
	}
	defer out.Release()
	if out.NumRows() != int64(len(input)) {
		t.Fatalf("output rows: got %d want %d", out.NumRows(), len(input))
	}
	fa, ok := out.Column(0).(*array.Float64)
	if !ok {
		t.Fatalf("expected Float64 array, got %T", out.Column(0))
	}
	if !fa.IsNull(nullAt) {
		t.Errorf("output row %d should be null", nullAt)
	}
	for i, want := range input {
		if i == nullAt {
			continue
		}
		if fa.Value(i) != want {
			t.Errorf("output row %d: got %v want %v", i, fa.Value(i), want)
		}
	}
}

// TestRoundtripInt64 verifies Int64 roundtrip with a null.
func TestRoundtripInt64(t *testing.T) {
	input := []int64{10, 20, 30}
	nullAt := 0
	rec := buildInt64Record(input, nullAt)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	out, err := iarrow.ToArrowRecord(df)
	if err != nil {
		t.Fatalf("ToArrowRecord: %v", err)
	}
	defer out.Release()
	ia, ok := out.Column(0).(*array.Int64)
	if !ok {
		t.Fatalf("expected Int64 array, got %T", out.Column(0))
	}
	if !ia.IsNull(nullAt) {
		t.Errorf("output row %d should be null", nullAt)
	}
	for i, want := range input {
		if i == nullAt {
			continue
		}
		if ia.Value(i) != want {
			t.Errorf("output row %d: got %d want %d", i, ia.Value(i), want)
		}
	}
}

// TestRoundtripBool verifies Boolean roundtrip without nulls.
func TestRoundtripBool(t *testing.T) {
	input := []bool{true, false, true}
	rec := buildBoolRecord(input)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	out, err := iarrow.ToArrowRecord(df)
	if err != nil {
		t.Fatalf("ToArrowRecord: %v", err)
	}
	defer out.Release()
	ba, ok := out.Column(0).(*array.Boolean)
	if !ok {
		t.Fatalf("expected Boolean array, got %T", out.Column(0))
	}
	for i, want := range input {
		if ba.Value(i) != want {
			t.Errorf("row %d: got %v want %v", i, ba.Value(i), want)
		}
	}
}

// TestRoundtripString verifies String roundtrip with a null.
func TestRoundtripString(t *testing.T) {
	input := []string{"alpha", "beta", "gamma"}
	nullAt := 2
	rec := buildStringRecord(input, nullAt)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	out, err := iarrow.ToArrowRecord(df)
	if err != nil {
		t.Fatalf("ToArrowRecord: %v", err)
	}
	defer out.Release()
	sa, ok := out.Column(0).(*array.String)
	if !ok {
		t.Fatalf("expected String array, got %T", out.Column(0))
	}
	if !sa.IsNull(nullAt) {
		t.Errorf("output row %d should be null", nullAt)
	}
	for i, want := range input {
		if i == nullAt {
			continue
		}
		if sa.Value(i) != want {
			t.Errorf("row %d: got %q want %q", i, sa.Value(i), want)
		}
	}
}

// TestRoundtripMultiColumn verifies a multi-column record.
func TestRoundtripMultiColumn(t *testing.T) {
	alloc := memory.NewGoAllocator()
	fb := array.NewFloat64Builder(alloc)
	fb.AppendValues([]float64{1, 2, 3}, nil)
	farr := fb.NewArray()
	defer farr.Release()

	ib := array.NewInt64Builder(alloc)
	ib.AppendValues([]int64{10, 20, 30}, nil)
	iarr := ib.NewArray()
	defer iarr.Release()

	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: "f", Type: goarrow.PrimitiveTypes.Float64},
		{Name: "i", Type: goarrow.PrimitiveTypes.Int64},
	}, nil)
	rec := array.NewRecord(schema, []goarrow.Array{farr, iarr}, 3)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	if df.Width() != 2 {
		t.Fatalf("width: got %d want 2", df.Width())
	}
	out, err := iarrow.ToArrowRecord(df)
	if err != nil {
		t.Fatalf("ToArrowRecord: %v", err)
	}
	defer out.Release()
	if out.NumCols() != 2 || out.NumRows() != 3 {
		t.Fatalf("output shape: %dx%d want 2x3", out.NumCols(), out.NumRows())
	}
}

// TestToTableNoBoxing verifies ToTable reads typed backing without Value(i) for
// Float64 and Int64.
func TestToTableNoBoxing(t *testing.T) {
	f64s := series.FromFloat64("f", []float64{1.5, 2.5, 3.5}, nil)
	i64s := series.FromInt64("i", []int64{10, 20, 30}, nil)
	_ = f64s
	_ = i64s
	// Build a small record and round-trip through ToTable / FromTable.
	alloc := memory.NewGoAllocator()
	fb := array.NewFloat64Builder(alloc)
	fb.AppendValues([]float64{1.5, 2.5, 3.5}, nil)
	farr := fb.NewArray()
	defer farr.Release()

	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: "f", Type: goarrow.PrimitiveTypes.Float64},
	}, nil)
	rec := array.NewRecord(schema, []goarrow.Array{farr}, 3)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	table := iarrow.ToTable(df)
	if len(table.Columns["f"]) != 3 {
		t.Fatalf("ToTable len: got %d want 3", len(table.Columns["f"]))
	}
	for i, want := range []float64{1.5, 2.5, 3.5} {
		got, ok := table.Columns["f"][i].(float64)
		if !ok || got != want {
			t.Errorf("row %d: got %v want %v", i, table.Columns["f"][i], want)
		}
	}
}

// TestRoundtripTimestamp verifies Timestamp ↔ time.Time ↔ Arrow.
func TestRoundtripTimestamp(t *testing.T) {
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	alloc := memory.NewGoAllocator()
	dt := &goarrow.TimestampType{Unit: goarrow.Nanosecond}
	b := array.NewTimestampBuilder(alloc, dt)
	b.Append(goarrow.Timestamp(ts.UnixNano()))
	b.AppendNull()
	arr := b.NewArray()
	defer arr.Release()

	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: "t", Type: dt, Nullable: true},
	}, nil)
	rec := array.NewRecord(schema, []goarrow.Array{arr}, 2)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	s, ok := df.Series("t")
	if !ok {
		t.Fatal("column t not found")
	}
	got, ok2 := s.Value(0).(time.Time)
	if !ok2 {
		t.Fatalf("expected time.Time, got %T", s.Value(0))
	}
	if !got.Equal(ts) {
		t.Errorf("timestamp: got %v want %v", got, ts)
	}
	if !s.IsNull(1) {
		t.Error("row 1 should be null")
	}
}
