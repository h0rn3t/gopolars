package json

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

// TestReadMissingFile exercises the os.ReadFile error path.
func TestReadMissingFile(t *testing.T) {
	if _, err := Read(ReadInput{Path: filepath.Join(t.TempDir(), "nope.json")}); err == nil {
		t.Error("Read missing file: expected error")
	}
}

// TestReadInvalidJSON exercises the json.Unmarshal error path.
func TestReadInvalidJSON(t *testing.T) {
	path := writeFile(t, "bad.json", "{not json")
	if _, err := Read(ReadInput{Path: path}); err == nil {
		t.Error("Read invalid json: expected error")
	}
}

// TestReadEmptyArray returns an empty frame for a zero-row document.
func TestReadEmptyArray(t *testing.T) {
	path := writeFile(t, "empty.json", "[]")
	df, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("Read []: %v", err)
	}
	if df.Height() != 0 {
		t.Errorf("Height = %d, want 0", df.Height())
	}
}

// TestReadInfersTypes drives normalizeValue and every inferType branch:
// integral float→int64, non-integral float, bool, string, and an all-null column.
func TestReadInfersTypes(t *testing.T) {
	doc := `[
	  {"i": 1, "f": 1.5, "b": true, "s": "x", "n": null},
	  {"i": 2, "f": 2.5, "b": false, "s": "y", "n": null}
	]`
	path := writeFile(t, "typed.json", doc)
	df, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("Read typed: %v", err)
	}
	if df.Height() != 2 || df.Width() != 5 {
		t.Errorf("shape = %dx%d, want 2x5", df.Height(), df.Width())
	}
}

// TestReadColumnProjection exercises the column-selection branch.
func TestReadColumnProjection(t *testing.T) {
	path := writeFile(t, "proj.json", `[{"a":1,"b":2},{"a":3,"b":4}]`)
	df, err := Read(ReadInput{Path: path, Columns: []string{"a"}})
	if err != nil {
		t.Fatalf("Read projection: %v", err)
	}
	if df.Width() != 1 {
		t.Errorf("Width = %d, want 1", df.Width())
	}
}

// TestReadNDJSONMissingFile exercises readNDJSON os.Open error.
func TestReadNDJSONMissingFile(t *testing.T) {
	if _, err := Read(ReadInput{Path: filepath.Join(t.TempDir(), "nope.ndjson"), NDJSON: true}); err == nil {
		t.Error("Read missing ndjson: expected error")
	}
}

// TestReadNDJSONBadLine exercises the per-line unmarshal error path.
func TestReadNDJSONBadLine(t *testing.T) {
	path := writeFile(t, "bad.ndjson", "{\"a\":1}\n{not json}\n")
	if _, err := Read(ReadInput{Path: path, NDJSON: true}); err == nil {
		t.Error("Read bad ndjson line: expected error")
	}
}

// TestWriteVariantsRoundtrip exercises NDJSON, pretty and compact writers.
func TestWriteVariantsRoundtrip(t *testing.T) {
	src, _ := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
	}})
	for _, in := range []WriteInput{
		{NDJSON: true},
		{Pretty: true},
		{},
	} {
		in.Path = filepath.Join(t.TempDir(), "out.json")
		if err := Write(src, in); err != nil {
			t.Errorf("Write %+v: %v", in, err)
		}
	}
}

// TestWriteMissingDir exercises the os.Create error path.
func TestWriteMissingDir(t *testing.T) {
	src, _ := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1)}},
	}})
	if err := Write(src, WriteInput{Path: filepath.Join(t.TempDir(), "nope", "out.json")}); err == nil {
		t.Error("Write to missing dir: expected error")
	}
}
