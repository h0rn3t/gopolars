package arrow_test

import (
	"reflect"
	"testing"
	"time"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/arrow/memory"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

// TestDateTimeBinaryImport checks the bridge decodes Arrow Date32/Time64/Binary:
// Date32/Time64 map to Datetime (time.Time; no Date/Time dtype in gopolars) and
// Binary maps to the boxed Binary dtype.
func TestDateTimeBinaryImport(t *testing.T) {
	mem := memory.DefaultAllocator

	db := array.NewDate32Builder(mem)
	db.Append(goarrow.Date32FromTime(time.Date(2020, 12, 30, 0, 0, 0, 0, time.UTC)))
	dateArr := db.NewArray()
	defer dateArr.Release()

	tb := array.NewTime64Builder(mem, &goarrow.Time64Type{Unit: goarrow.Microsecond})
	tb.Append(goarrow.Time64(int64(23*3600+59*60+59) * 1_000_000)) // 23:59:59 in µs
	timeArr := tb.NewArray()
	defer timeArr.Release()

	bb := array.NewBinaryBuilder(mem, goarrow.BinaryTypes.Binary)
	bb.Append([]byte{0xde, 0xad})
	binArr := bb.NewArray()
	defer binArr.Release()

	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: "d", Type: dateArr.DataType(), Nullable: true},
		{Name: "t", Type: timeArr.DataType(), Nullable: true},
		{Name: "b", Type: goarrow.BinaryTypes.Binary, Nullable: true},
	}, nil)
	rec := array.NewRecordBatch(schema, []goarrow.Array{dateArr, timeArr, binArr}, 1)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	dd, _ := df.Series("d")
	if dd.DataType() != dtypes.Datetime || !dd.Value(0).(time.Time).Equal(time.Date(2020, 12, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("date col = %s %v", dd.DataType(), dd.Value(0))
	}
	tt, _ := df.Series("t")
	if !tt.Value(0).(time.Time).Equal(time.Date(1970, 1, 1, 23, 59, 59, 0, time.UTC)) {
		t.Fatalf("time col = %v", tt.Value(0))
	}
	bs, _ := df.Series("b")
	if bs.DataType() != dtypes.Binary || !reflect.DeepEqual(bs.Value(0), []byte{0xde, 0xad}) {
		t.Fatalf("binary col = %s %#v", bs.DataType(), bs.Value(0))
	}
}

// TestNestedRoundtrip checks that List and Struct columns survive a
// ToArrowRecord → FromArrowRecord round-trip (including null elements and a
// nested struct), exercising the Arrow nested bridge added for the SQL engine.
func TestNestedRoundtrip(t *testing.T) {
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "lst", Values: []any{
			[]any{int64(1), int64(2)},
			[]any{nil, int64(5)},
			nil,
		}},
		{Name: "st", Values: []any{
			map[string]any{"id": int64(7), "who": map[string]any{"n": "a"}},
			map[string]any{"id": int64(8), "who": map[string]any{"n": "b"}},
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

	got, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}

	lst, _ := got.Series("lst")
	if lst.DataType() != dtypes.List {
		t.Fatalf("lst dtype = %s, want list", lst.DataType())
	}
	if !reflect.DeepEqual(lst.Value(0), []any{int64(1), int64(2)}) {
		t.Fatalf("lst[0] = %#v", lst.Value(0))
	}
	if !reflect.DeepEqual(lst.Value(1), []any{nil, int64(5)}) {
		t.Fatalf("lst[1] = %#v", lst.Value(1))
	}
	if lst.Value(2) != nil {
		t.Fatalf("lst[2] = %#v, want nil", lst.Value(2))
	}

	st, _ := got.Series("st")
	if st.DataType() != dtypes.Struct {
		t.Fatalf("st dtype = %s, want struct", st.DataType())
	}
	m, ok := st.Value(0).(map[string]any)
	if !ok || m["id"] != int64(7) {
		t.Fatalf("st[0] = %#v", st.Value(0))
	}
	who, ok := m["who"].(map[string]any)
	if !ok || who["n"] != "a" {
		t.Fatalf("st[0].who = %#v", m["who"])
	}
	if st.Value(2) != nil {
		t.Fatalf("st[2] = %#v, want nil", st.Value(2))
	}
}

// TestBridgeImportTypes drives the remaining Arrow import paths (Date64, Time32,
// LargeBinary, month_day_nano interval, Decimal128, FixedSizeList, and narrow leaves
// inside a List/Struct) through FromArrowRecord — they otherwise run only via the
// duckdb-tagged SQL engine, which the pure-Go coverage build does not compile.
func TestBridgeImportTypes(t *testing.T) {
	mem := memory.DefaultAllocator
	day := time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC)

	d64 := array.NewDate64Builder(mem)
	d64.Append(goarrow.Date64FromTime(day))
	a0 := d64.NewArray()
	defer a0.Release()

	t32 := array.NewTime32Builder(mem, &goarrow.Time32Type{Unit: goarrow.Second})
	t32.Append(goarrow.Time32(12 * 3600)) // 12:00:00
	a1 := t32.NewArray()
	defer a1.Release()

	lbin := array.NewBinaryBuilder(mem, goarrow.BinaryTypes.LargeBinary)
	lbin.Append([]byte{0x01, 0x02})
	a2 := lbin.NewArray()
	defer a2.Release()

	iv := array.NewMonthDayNanoIntervalBuilder(mem)
	iv.Append(goarrow.MonthDayNanoInterval{Months: 1, Days: 2, Nanoseconds: 3_000_000_000})
	a3 := iv.NewArray()
	defer a3.Release()

	dec := array.NewDecimal128Builder(mem, &goarrow.Decimal128Type{Precision: 5, Scale: 2})
	dec.Append(decimal128.FromI64(150)) // scale 2 -> "1.50"
	a4 := dec.NewArray()
	defer a4.Release()

	fsl := array.NewFixedSizeListBuilder(mem, 2, goarrow.PrimitiveTypes.Int64)
	fsl.Append(true)
	fsl.ValueBuilder().(*array.Int64Builder).AppendValues([]int64{7, 8}, nil)
	a5 := fsl.NewArray()
	defer a5.Release()

	li32 := array.NewListBuilder(mem, goarrow.PrimitiveTypes.Int32)
	li32.Append(true)
	li32.ValueBuilder().(*array.Int32Builder).AppendValues([]int32{10, 20}, nil)
	a6 := li32.NewArray()
	defer a6.Release()

	snarType := goarrow.StructOf(
		goarrow.Field{Name: "a", Type: goarrow.PrimitiveTypes.Int8, Nullable: true},
		goarrow.Field{Name: "b", Type: goarrow.PrimitiveTypes.Uint16, Nullable: true},
		goarrow.Field{Name: "c", Type: goarrow.PrimitiveTypes.Float32, Nullable: true},
	)
	sb := array.NewStructBuilder(mem, snarType)
	sb.Append(true)
	sb.FieldBuilder(0).(*array.Int8Builder).Append(1)
	sb.FieldBuilder(1).(*array.Uint16Builder).Append(2)
	sb.FieldBuilder(2).(*array.Float32Builder).Append(3.5)
	a7 := sb.NewArray()
	defer a7.Release()

	cols := []goarrow.Array{a0, a1, a2, a3, a4, a5, a6, a7}
	fields := []goarrow.Field{
		{Name: "d64", Type: a0.DataType(), Nullable: true},
		{Name: "t32", Type: a1.DataType(), Nullable: true},
		{Name: "lbin", Type: goarrow.BinaryTypes.LargeBinary, Nullable: true},
		{Name: "iv", Type: a3.DataType(), Nullable: true},
		{Name: "dec", Type: a4.DataType(), Nullable: true},
		{Name: "fsl", Type: a5.DataType(), Nullable: true},
		{Name: "li32", Type: a6.DataType(), Nullable: true},
		{Name: "snar", Type: snarType, Nullable: true},
	}
	rec := array.NewRecordBatch(goarrow.NewSchema(fields, nil), cols, 1)
	defer rec.Release()

	df, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	val := func(name string) any { s, _ := df.Series(name); return s.Value(0) }

	if !val("d64").(time.Time).Equal(day) {
		t.Fatalf("d64 = %v", val("d64"))
	}
	if !val("t32").(time.Time).Equal(time.Date(1970, 1, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("t32 = %v", val("t32"))
	}
	if !reflect.DeepEqual(val("lbin"), []byte{0x01, 0x02}) {
		t.Fatalf("lbin = %#v", val("lbin"))
	}
	if val("iv") != 30*24*time.Hour+2*24*time.Hour+3*time.Second {
		t.Fatalf("iv = %v", val("iv"))
	}
	if val("dec") != dtypes.DecimalValue("1.50") {
		t.Fatalf("dec = %#v", val("dec"))
	}
	if !reflect.DeepEqual(val("fsl"), []any{int64(7), int64(8)}) {
		t.Fatalf("fsl = %#v", val("fsl"))
	}
	if !reflect.DeepEqual(val("li32"), []any{int64(10), int64(20)}) {
		t.Fatalf("li32 = %#v", val("li32"))
	}
	snar, ok := val("snar").(map[string]any)
	if !ok || snar["a"] != int64(1) || snar["b"] != int64(2) || snar["c"] != float64(3.5) {
		t.Fatalf("snar = %#v", val("snar"))
	}
}

// TestBridgeExportNested round-trips List/Struct columns with diverse leaf types
// through ToArrowRecord -> FromArrowRecord, exercising the export inference/append
// helpers (inferArrowType, appendArrowValue, toInt64/toFloat64/toStr).
func TestBridgeExportNested(t *testing.T) {
	t1 := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "li", Values: []any{[]any{int64(1), int64(2)}}},
		{Name: "lf", Values: []any{[]any{1.5, 2.5}}},
		{Name: "ls", Values: []any{[]any{"a", "b"}}},
		{Name: "lb", Values: []any{[]any{true, false}}},
		{Name: "lt", Values: []any{[]any{t1}}},
		{Name: "st", Values: []any{map[string]any{
			"i": int64(7), "f": 1.25, "s": "x", "b": true, "dv": dtypes.DecimalValue("9.9"),
		}}},
	}})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}
	rec, err := iarrow.ToArrowRecord(df)
	if err != nil {
		t.Fatalf("ToArrowRecord: %v", err)
	}
	defer rec.Release()
	got, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	v := func(name string) any { s, _ := got.Series(name); return s.Value(0) }

	if !reflect.DeepEqual(v("li"), []any{int64(1), int64(2)}) {
		t.Fatalf("li = %#v", v("li"))
	}
	if !reflect.DeepEqual(v("lf"), []any{1.5, 2.5}) {
		t.Fatalf("lf = %#v", v("lf"))
	}
	if !reflect.DeepEqual(v("ls"), []any{"a", "b"}) {
		t.Fatalf("ls = %#v", v("ls"))
	}
	if !reflect.DeepEqual(v("lb"), []any{true, false}) {
		t.Fatalf("lb = %#v", v("lb"))
	}
	lt, ok := v("lt").([]any)
	if !ok || len(lt) != 1 || !lt[0].(time.Time).Equal(t1) {
		t.Fatalf("lt = %#v", v("lt"))
	}
	st, ok := v("st").(map[string]any)
	if !ok || st["i"] != int64(7) || st["f"] != 1.25 || st["s"] != "x" || st["b"] != true || st["dv"] != "9.9" {
		t.Fatalf("st = %#v", v("st"))
	}
}
