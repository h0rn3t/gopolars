package csv

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// goldenFrame builds a mixed-dtype frame covering the formatting edge cases the
// accelerated writer must reproduce byte-for-byte: negative ints, shortest-form
// floats (2/3, 1e-9), bools, quoted strings (comma, newline), datetimes, and
// nulls in every column.
func goldenFrame(t *testing.T) frame.DataFrame {
	t.Helper()
	ts := time.Date(2026, 5, 28, 12, 30, 45, 123456789, time.UTC)
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(-2), nil}},
			{Name: "score", Values: []any{float64(1.5), float64(-3.25), nil}},
			{Name: "ratio", Values: []any{float64(0.1), float64(2.0 / 3.0), float64(1e-9)}},
			{Name: "active", Values: []any{true, false, nil}},
			{Name: "label", Values: []any{"alpha", "with,comma", "line\nbreak"}},
			{Name: "ts", Values: []any{ts, nil, ts.Add(time.Hour)}},
		},
	})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}
	return df
}

// golden is the exact byte output produced by the previous row-at-a-time writer
// (captured before the acceleration). The accelerated writer must match it.
const golden = "id,score,ratio,active,label,ts\n" +
	"1,1.5,0.1,true,alpha,2026-05-28T12:30:45.123456789Z\n" +
	"-2,-3.25,0.6666666666666666,false,\"with,comma\",\n" +
	",,1e-09,,\"line\nbreak\",2026-05-28T13:30:45.123456789Z\n"

func TestWriteCSVByteIdentical(t *testing.T) {
	path := filepath.Join(t.TempDir(), "g.csv")
	if err := Write(goldenFrame(t), WriteInput{Path: path, IncludeHeader: true, Separator: ','}); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != golden {
		t.Fatalf("CSV output changed.\n got: %q\nwant: %q", string(got), golden)
	}
}

func TestWriteCSVRoundTrip(t *testing.T) {
	df := goldenFrame(t)
	path := filepath.Join(t.TempDir(), "rt.csv")
	if err := Write(df, WriteInput{Path: path, IncludeHeader: true, Separator: ','}); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := Read(ReadInput{Path: path, HasHeader: true, Separator: ','})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Height() != df.Height() || got.Width() != df.Width() {
		t.Fatalf("shape changed: got %dx%d want %dx%d", got.Height(), got.Width(), df.Height(), df.Width())
	}
	wantCols := df.Columns()
	for i, name := range got.Columns() {
		if name != wantCols[i] {
			t.Fatalf("column %d: got %q want %q", i, name, wantCols[i])
		}
	}
}
