package csv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// TestReadMalformed exercises the csv.ReadAll error path (unbalanced quote).
func TestReadMalformed(t *testing.T) {
	path := writeFile(t, "bad.csv", "a,b\n\"unterminated,2\n")
	if _, err := Read(ReadInput{Path: path, HasHeader: true, Separator: ','}); err == nil {
		t.Error("Read malformed csv: expected error")
	}
}

// TestReadMissingFile exercises the os.Open error path.
func TestReadMissingFile(t *testing.T) {
	if _, err := Read(ReadInput{Path: filepath.Join(t.TempDir(), "nope.csv")}); err == nil {
		t.Error("Read missing file: expected error")
	}
}

// TestReadEmptyFile returns an empty frame for a zero-row file.
func TestReadEmptyFile(t *testing.T) {
	path := writeFile(t, "empty.csv", "")
	df, err := Read(ReadInput{Path: path, HasHeader: true, Separator: ','})
	if err != nil {
		t.Fatalf("Read empty: %v", err)
	}
	if df.Height() != 0 {
		t.Errorf("empty file Height = %d, want 0", df.Height())
	}
}

// TestReadNoHeaderAutoNames exercises the auto-naming path for headerless input.
func TestReadNoHeaderAutoNames(t *testing.T) {
	path := writeFile(t, "noheader.csv", "1,2,3\n4,5,6\n")
	df, err := Read(ReadInput{Path: path, HasHeader: false, Separator: ','})
	if err != nil {
		t.Fatalf("Read no-header: %v", err)
	}
	if df.Width() != 3 {
		t.Errorf("Width = %d, want 3", df.Width())
	}
	if _, ok := df.Series("column_1"); !ok {
		t.Error("expected auto-named column_1")
	}
}

// TestReadInferDatetimeAndEmpty drives the datetime inference branch and the
// empty-cell (null) handling in both inferColumn and parseWithType. The "n"
// column carries an empty cell on the first row.
func TestReadInferDatetimeAndEmpty(t *testing.T) {
	path := writeFile(t, "ts.csv", "ts,n\n2026-01-01T00:00:00Z,\n2026-01-02T00:00:00Z,5\n")
	df, err := Read(ReadInput{Path: path, HasHeader: true, Separator: ','})
	if err != nil {
		t.Fatalf("Read datetime: %v", err)
	}
	ts, ok := df.Series("ts")
	if !ok {
		t.Fatalf("missing ts column")
	}
	if ts.IsNull(0) {
		t.Error("ts row 0 should be a parsed timestamp, not null")
	}
	n, _ := df.Series("n")
	if !n.IsNull(0) {
		t.Error("n row 0 (empty cell) should be null")
	}
}

// TestWriteWithNull writes a frame containing a null cell, exercising the nil
// branch of Write.
func TestWriteWithNull(t *testing.T) {
	src, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), nil}},
	}})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}
	path := filepath.Join(t.TempDir(), "out.csv")
	if err := Write(src, WriteInput{Path: path, IncludeHeader: true, Separator: ','}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, _ := os.ReadFile(path)
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

// TestWriteMissingDir exercises the os.Create error path.
func TestWriteMissingDir(t *testing.T) {
	src, _ := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1)}},
	}})
	err := Write(src, WriteInput{Path: filepath.Join(t.TempDir(), "nope", "out.csv"), IncludeHeader: true})
	if err == nil {
		t.Error("Write to missing dir: expected error")
	}
}
