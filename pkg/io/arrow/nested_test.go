package arrow_test

import (
	"reflect"
	"testing"
	"time"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
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
