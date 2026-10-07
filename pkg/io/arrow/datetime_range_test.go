package arrow_test

import (
	"math"
	"strings"
	"testing"
	"time"

	goarrow "github.com/apache/arrow-go/v18/arrow"

	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

var microTimestamp = &goarrow.TimestampType{Unit: goarrow.Microsecond}

// TestToArrowRecordDatetimeRange checks that Datetime exports as
// timestamp[us] over the whole microsecond range, including dates that do not
// fit int64 nanoseconds (1677-09-21 … 2262-04-11), and that sub-microsecond
// digits floor to the earlier microsecond.
func TestToArrowRecordDatetimeRange(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want time.Time
	}{
		{"far future", time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC), time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC)},
		{"year one", time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"sub-microsecond", time.Date(2026, 5, 28, 12, 30, 45, 123456789, time.UTC), time.Date(2026, 5, 28, 12, 30, 45, 123456000, time.UTC)},
		{"pre-epoch sub-microsecond", time.Date(1969, 12, 31, 23, 59, 59, 999999999, time.UTC), time.Date(1969, 12, 31, 23, 59, 59, 999999000, time.UTC)},
		{"non-UTC zone", time.Date(9999, 6, 1, 12, 0, 0, 0, time.FixedZone("UTC+2", 2*3600)), time.Date(9999, 6, 1, 10, 0, 0, 0, time.UTC)},
		{"last microsecond", time.UnixMicro(math.MaxInt64).Add(999 * time.Nanosecond), time.UnixMicro(math.MaxInt64)},
		{"first microsecond", time.UnixMicro(math.MinInt64), time.UnixMicro(math.MinInt64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
				{Name: "ts", Values: []any{tt.in, nil}},
			}})
			if err != nil {
				t.Fatalf("FromAnyColumns: %v", err)
			}
			rec, err := iarrow.ToArrowRecord(df)
			if err != nil {
				t.Fatalf("ToArrowRecord(%v) error = %v", tt.in, err)
			}
			defer rec.Release()
			if got := rec.Schema().Field(0).Type; !goarrow.TypeEqual(got, microTimestamp) {
				t.Errorf("ToArrowRecord(%v) type = %v, want %v", tt.in, got, microTimestamp)
			}
			back, err := iarrow.FromArrowRecord(rec)
			if err != nil {
				t.Fatalf("FromArrowRecord: %v", err)
			}
			s, _ := back.Series("ts")
			if got, ok := s.Value(0).(time.Time); !ok || !got.Equal(tt.want) {
				t.Errorf("round-trip of %v = %v, want %v", tt.in, s.Value(0), tt.want)
			}
			if got := s.Value(1); got != nil {
				t.Errorf("round-trip of null = %v, want nil", got)
			}
		})
	}
}

// TestToArrowRecordNestedDatetime checks that a time.Time inside a List or a
// Struct exports as timestamp[us] and keeps a date past 2262.
func TestToArrowRecordNestedDatetime(t *testing.T) {
	far := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "lst", Values: []any{[]any{far}}},
		{Name: "st", Values: []any{map[string]any{"ts": far}}},
	}})
	if err != nil {
		t.Fatalf("FromAnyColumns: %v", err)
	}
	rec, err := iarrow.ToArrowRecord(df)
	if err != nil {
		t.Fatalf("ToArrowRecord: %v", err)
	}
	defer rec.Release()
	if got := rec.Schema().Field(0).Type.(*goarrow.ListType).Elem(); !goarrow.TypeEqual(got, microTimestamp) {
		t.Errorf("list element type = %v, want %v", got, microTimestamp)
	}
	if got := rec.Schema().Field(1).Type.(*goarrow.StructType).Field(0).Type; !goarrow.TypeEqual(got, microTimestamp) {
		t.Errorf("struct field type = %v, want %v", got, microTimestamp)
	}
	back, err := iarrow.FromArrowRecord(rec)
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	lst, _ := back.Series("lst")
	if got, ok := lst.Value(0).([]any); !ok || len(got) != 1 || !got[0].(time.Time).Equal(far) {
		t.Errorf("lst[0] = %#v, want [%v]", lst.Value(0), far)
	}
	st, _ := back.Series("st")
	if got, ok := st.Value(0).(map[string]any); !ok || !got["ts"].(time.Time).Equal(far) {
		t.Errorf("st[0] = %#v, want {ts: %v}", st.Value(0), far)
	}
}

// TestToArrowRecordDatetimeOutOfRange checks that an instant whose
// microsecond count overflows int64 fails the export with an error naming
// the column instead of writing a different instant.
func TestToArrowRecordDatetimeOutOfRange(t *testing.T) {
	future := time.Date(300000, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		value any
	}{
		{"past the last microsecond", time.UnixMicro(math.MaxInt64).Add(time.Microsecond)},
		{"before the first microsecond", time.UnixMicro(math.MinInt64).Add(-time.Nanosecond)},
		{"year 300000", future},
		{"list", []any{future}},
		{"struct", map[string]any{"at": future}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
				{Name: "ts", Values: []any{tt.value}},
			}})
			if err != nil {
				t.Fatalf("FromAnyColumns: %v", err)
			}
			rec, err := iarrow.ToArrowRecord(df)
			if err == nil {
				rec.Release()
				t.Fatalf("ToArrowRecord(%v) error = nil, want an out-of-range error", tt.value)
			}
			if !strings.Contains(err.Error(), `column "ts"`) {
				t.Errorf("ToArrowRecord(%v) error = %q, want it to name column \"ts\"", tt.value, err)
			}
		})
	}
}
