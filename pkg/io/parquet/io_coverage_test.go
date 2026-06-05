package parquet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/parquet-go/parquet-go"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.parquet")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

// TestReadMissingFile exercises the path where neither the parquet reader nor
// the JSON fallback can open the file.
func TestReadMissingFile(t *testing.T) {
	if _, err := Read(ReadInput{Path: filepath.Join(t.TempDir(), "nope.parquet")}); err == nil {
		t.Error("Read missing file: expected error")
	}
}

// TestReadJSONFallback reads a plain-JSON payload through the fallback branch
// (the file is not a real parquet container).
func TestReadJSONFallback(t *testing.T) {
	path := writeFile(t, `{"columns":[{"name":"a","type":"int64","values":[1,2,null]}]}`)
	df, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("Read fallback: %v", err)
	}
	if df.Height() != 3 || df.Width() != 1 {
		t.Errorf("shape = %dx%d, want 3x1", df.Height(), df.Width())
	}
}

// TestReadFallbackInvalidJSON exercises the fallback unmarshal error path.
func TestReadFallbackInvalidJSON(t *testing.T) {
	path := writeFile(t, "definitely not parquet and not json {")
	if _, err := Read(ReadInput{Path: path}); err == nil {
		t.Error("Read invalid fallback: expected error")
	}
}

// TestDecodeValueTypeErrors drives every typed error branch of decodeValues via
// the JSON fallback: each payload carries a value of the wrong JSON kind.
func TestDecodeValueTypeErrors(t *testing.T) {
	cases := map[string]string{
		"int64-from-string":    `{"columns":[{"name":"a","type":"int64","values":["x"]}]}`,
		"float64-from-string":  `{"columns":[{"name":"a","type":"float64","values":["x"]}]}`,
		"bool-from-string":     `{"columns":[{"name":"a","type":"bool","values":["x"]}]}`,
		"datetime-bad-parse":   `{"columns":[{"name":"a","type":"datetime","values":["notadate"]}]}`,
		"datetime-from-number": `{"columns":[{"name":"a","type":"datetime","values":[123]}]}`,
		"decimal-from-number":  `{"columns":[{"name":"a","type":"decimal","values":[123]}]}`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeFile(t, doc)
			if _, err := Read(ReadInput{Path: path}); err == nil {
				t.Errorf("%s: expected decode error", name)
			}
		})
	}
}

// TestDecodeValueValidTypes drives the successful typed branches incl. datetime,
// decimal and the unknown-type Sprintf fallback.
func TestDecodeValueValidTypes(t *testing.T) {
	doc := `{"columns":[
	  {"name":"i","type":"int64","values":[1,null]},
	  {"name":"f","type":"float64","values":[1.5,2.5]},
	  {"name":"b","type":"bool","values":[true,false]},
	  {"name":"t","type":"datetime","values":["2026-01-01T00:00:00Z","2026-01-02T00:00:00Z"]},
	  {"name":"d","type":"decimal","values":["1.50","2.50"]},
	  {"name":"s","type":"string","values":["x","y"]},
	  {"name":"u","type":"weirdtype","values":[123,456]}
	]}`
	path := writeFile(t, doc)
	df, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("Read valid types: %v", err)
	}
	if df.Width() != 7 {
		t.Errorf("Width = %d, want 7", df.Width())
	}
}

// TestReadEmptyRealParquet exercises the len(rows)==0 branch of the real-parquet
// path by writing a container with zero envelope rows.
func TestReadEmptyRealParquet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.parquet")
	if err := parquet.WriteFile(path, []parquetEnvelope{}); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	df, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if df.Height() != 0 {
		t.Errorf("Height = %d, want 0", df.Height())
	}
}

// TestReadRealParquetBadPayload exercises the json.Unmarshal error in the
// real-parquet path: a valid container whose envelope payload is not valid JSON.
func TestReadRealParquetBadPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "badpayload.parquet")
	if err := parquet.WriteFile(path, []parquetEnvelope{{Payload: "{invalid"}}); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := Read(ReadInput{Path: path}); err == nil {
		t.Error("Read bad payload: expected json error")
	}
}

// TestReadColumnProjection exercises the column-selection branch of payloadToFrame.
func TestReadColumnProjection(t *testing.T) {
	path := writeFile(t, `{"columns":[{"name":"a","type":"int64","values":[1]},{"name":"b","type":"int64","values":[2]}]}`)
	df, err := Read(ReadInput{Path: path, Columns: []string{"a"}})
	if err != nil {
		t.Fatalf("Read projection: %v", err)
	}
	if df.Width() != 1 {
		t.Errorf("Width = %d, want 1", df.Width())
	}
}

// TestWriteRealParquetRoundtrip writes a real parquet file (the happy path) and
// reads it back through the parquet reader.
func TestWriteRealParquetRoundtrip(t *testing.T) {
	src, _ := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
	}})
	path := filepath.Join(t.TempDir(), "real.parquet")
	if err := Write(src, WriteInput{Path: path}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	df, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("Read back: %v", err)
	}
	if df.Height() != 2 {
		t.Errorf("Height = %d, want 2", df.Height())
	}
}
