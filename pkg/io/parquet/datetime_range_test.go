package parquet

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/schema"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestWriteParquetDatetimeRange checks that Datetime values outside the int64
// nanosecond range (1677-09-21 … 2262-04-11), flat and nested, survive a
// Parquet round-trip, that the column is a microsecond timestamp, and that
// sub-microsecond digits are truncated.
func TestWriteParquetDatetimeRange(t *testing.T) {
	far := time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC)
	yearOne := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	nanos := time.Date(2026, 5, 28, 12, 30, 45, 123456789, time.UTC)
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "ts", Values: []any{far, yearOne, nanos, nil}},
		{Name: "lst", Values: []any{[]any{far}, nil, nil, nil}},
		{Name: "st", Values: []any{map[string]any{"at": far}, nil, nil, nil}},
	}})
	if err != nil {
		t.Fatalf("FromAnyColumns: %v", err)
	}
	path := filepath.Join(t.TempDir(), "datetime.parquet")
	if err := Write(df, WriteInput{Path: path}); err != nil {
		t.Fatalf("Write(%q) error = %v", path, err)
	}

	rdr, err := file.OpenParquetFile(path, false)
	if err != nil {
		t.Fatalf("OpenParquetFile(%q): %v", path, err)
	}
	lt, ok := rdr.MetaData().Schema.Column(0).LogicalType().(schema.TimestampLogicalType)
	if !ok || lt.TimeUnit() != schema.TimeUnitMicros {
		t.Errorf("column ts logical type = %v, want a microsecond timestamp", rdr.MetaData().Schema.Column(0).LogicalType())
	}
	if err := rdr.Close(); err != nil {
		t.Fatalf("close %q: %v", path, err)
	}

	got, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("Read(%q) error = %v", path, err)
	}
	ts, _ := got.Series("ts")
	for i, want := range []time.Time{far, yearOne, time.Date(2026, 5, 28, 12, 30, 45, 123456000, time.UTC)} {
		if v, ok := ts.Value(i).(time.Time); !ok || !v.Equal(want) {
			t.Errorf("ts[%d] = %v, want %v", i, ts.Value(i), want)
		}
	}
	if v := ts.Value(3); v != nil {
		t.Errorf("ts[3] = %v, want nil", v)
	}
	lst, _ := got.Series("lst")
	if v, ok := lst.Value(0).([]any); !ok || len(v) != 1 || !v[0].(time.Time).Equal(far) {
		t.Errorf("lst[0] = %#v, want [%v]", lst.Value(0), far)
	}
	st, _ := got.Series("st")
	if v, ok := st.Value(0).(map[string]any); !ok || !v["at"].(time.Time).Equal(far) {
		t.Errorf("st[0] = %#v, want {at: %v}", st.Value(0), far)
	}
}

// TestWriteParquetDatetimeOutOfRange checks that an instant past the
// microsecond range fails the write with an error naming the column, before
// any file is created.
func TestWriteParquetDatetimeOutOfRange(t *testing.T) {
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "ts", Values: []any{time.Date(300000, 1, 1, 0, 0, 0, 0, time.UTC)}},
	}})
	if err != nil {
		t.Fatalf("FromAnyColumns: %v", err)
	}
	path := filepath.Join(t.TempDir(), "out_of_range.parquet")
	err = Write(df, WriteInput{Path: path})
	if err == nil || !strings.Contains(err.Error(), `column "ts"`) {
		t.Errorf("Write(year 300000) error = %v, want an error naming column \"ts\"", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("Write(year 300000) left %q behind (stat error = %v)", path, statErr)
	}
}

// TestLowCardinalityDatetimeCountsWrittenValues checks that the dictionary
// heuristic samples the microsecond values that are written: instants that
// differ only below a microsecond are one dictionary entry.
func TestLowCardinalityDatetimeCountsWrittenValues(t *testing.T) {
	base := time.Date(2026, 5, 28, 12, 30, 45, 123456000, time.UTC)
	vals := make([]time.Time, 100)
	for i := range vals {
		vals[i] = base.Add(time.Duration(i%1000) * time.Nanosecond)
	}
	if !lowCardinality(chunk.NewTime(vals, nil)) {
		t.Error("lowCardinality(100 instants within one microsecond) = false, want true")
	}
}
