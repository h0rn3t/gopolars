package csv

import (
	"bytes"
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// quotingSamples are the field values whose escaping must stay byte-identical to
// encoding/csv: the delimiter itself, quotes, CR/LF, leading and trailing
// spaces, the special `\.` field, empty and unicode values.
var quotingSamples = []string{
	"plain",
	"",
	"with,comma",
	"with;semicolon",
	"with\ttab",
	`with"quote`,
	`"fully quoted"`,
	`""`,
	"with\nnewline",
	"with\rcarriage",
	"with\r\ncrlf",
	" leading space",
	"trailing space ",
	" ",
	`\.`,
	`\\.`,
	"ключ",
	"emoji😀",
	"-42",
	"1e10",
}

// referenceCSV produces the bytes encoding/csv's Writer would produce for the
// same records and delimiter — the contract the hand-rolled writer must match.
func referenceCSV(t *testing.T, records [][]string, comma rune) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if comma != 0 {
		w.Comma = comma
	}
	for _, rec := range records {
		if err := w.Write(rec); err != nil {
			t.Fatalf("reference write: %v", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		t.Fatalf("reference flush: %v", err)
	}
	return buf.Bytes()
}

// TestWriteMatchesEncodingCSVByteForByte writes a string frame covering every
// quoting case through the accelerated writer and compares the file against
// encoding/csv's output, for several delimiters and with/without a header.
func TestWriteMatchesEncodingCSVByteForByte(t *testing.T) {
	for _, comma := range []rune{',', ';', '\t', '|', ' ', '-', '1'} {
		for _, header := range []bool{true, false} {
			// One column per sample keeps every value in a distinct field position,
			// and two rows prove the reused buffer does not leak between rows.
			names := make([]string, len(quotingSamples))
			cols := make([]frame.SeriesInput, len(quotingSamples))
			for i, sample := range quotingSamples {
				names[i] = "c" + string(rune('A'+i%26))
				cols[i] = frame.SeriesInput{
					Name:   names[i],
					Values: []any{sample, quotingSamples[len(quotingSamples)-1-i]},
					DType:  dtypes.String,
				}
			}
			df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: cols})
			if err != nil {
				t.Fatalf("build frame: %v", err)
			}

			path := filepath.Join(t.TempDir(), "out.csv")
			if err := Write(df, WriteInput{Path: path, IncludeHeader: header, Separator: comma}); err != nil {
				t.Fatalf("comma=%q header=%v: Write: %v", comma, header, err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read back: %v", err)
			}

			var records [][]string
			if header {
				records = append(records, names)
			}
			for row := 0; row < 2; row++ {
				rec := make([]string, len(quotingSamples))
				for i := range quotingSamples {
					if row == 0 {
						rec[i] = quotingSamples[i]
					} else {
						rec[i] = quotingSamples[len(quotingSamples)-1-i]
					}
				}
				records = append(records, rec)
			}
			want := referenceCSV(t, records, comma)

			if !bytes.Equal(got, want) {
				t.Fatalf("comma=%q header=%v: output differs from encoding/csv\n got: %q\nwant: %q", comma, header, got, want)
			}
		}
	}
}

// TestWriteTypedColumnsMatchEncodingCSV covers the typed appenders (int, float,
// bool, time) plus nulls against the same reference, including delimiters that
// collide with numeric text.
func TestWriteTypedColumnsMatchEncodingCSV(t *testing.T) {
	ts := time.Date(2026, 8, 4, 12, 30, 45, 123456789, time.UTC)
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "i", Values: []any{int64(-42), nil, int64(0), int64(1000000)}, DType: dtypes.Int64},
		// math.Copysign gives a real negative zero; the literal -0.0 does not (it is
		// folded to +0.0), and the writer must render it as "-0" like strconv does.
		{Name: "f", Values: []any{1.5, nil, math.Copysign(0, -1), 1e21}, DType: dtypes.Float64},
		{Name: "b", Values: []any{true, nil, false, true}, DType: dtypes.Boolean},
		{Name: "t", Values: []any{ts, nil, ts.Add(time.Hour), ts}, DType: dtypes.Datetime},
		{Name: "s", Values: []any{"a,b", nil, `q"q`, " sp"}, DType: dtypes.String},
	}})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}

	// The expected cell text is the *unescaped* rendering the writer documents
	// (nulls empty, RFC3339Nano times, strconv formatting); encoding/csv then does
	// the escaping. Deriving it from cellAppender would be circular, since that is
	// the code under test and it already applies quoting.
	names := df.Columns()
	records := [][]string{names}
	for row := 0; row < df.Height(); row++ {
		rec := make([]string, len(names))
		for j, name := range names {
			s, _ := df.Series(name)
			switch v := s.Value(row).(type) {
			case nil:
				rec[j] = ""
			case int64:
				rec[j] = strconv.FormatInt(v, 10)
			case float64:
				rec[j] = strconv.FormatFloat(v, 'g', -1, 64)
			case bool:
				rec[j] = strconv.FormatBool(v)
			case time.Time:
				rec[j] = v.Format(time.RFC3339Nano)
			case string:
				rec[j] = v
			default:
				t.Fatalf("unexpected cell type %T", v)
			}
		}
		records = append(records, rec)
	}

	for _, comma := range []rune{',', ';', '-', '1', ':'} {
		path := filepath.Join(t.TempDir(), "typed.csv")
		if err := Write(df, WriteInput{Path: path, IncludeHeader: true, Separator: comma}); err != nil {
			t.Fatalf("comma=%q: Write: %v", comma, err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if want := referenceCSV(t, records, comma); !bytes.Equal(got, want) {
			t.Fatalf("comma=%q: typed output differs from encoding/csv\n got: %q\nwant: %q", comma, got, want)
		}
	}
}

// TestWriteRejectsInvalidSeparator checks the delimiter validation encoding/csv
// used to perform for us is still enforced.
func TestWriteRejectsInvalidSeparator(t *testing.T) {
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{"x"}, DType: dtypes.String},
	}})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}
	for _, comma := range []rune{'"', '\r', '\n'} {
		path := filepath.Join(t.TempDir(), "bad.csv")
		if err := Write(df, WriteInput{Path: path, Separator: comma}); err == nil {
			t.Fatalf("separator %q: expected an error, got nil", comma)
		}
	}
}

// TestWriteParallelMatchesSequential checks the concurrent formatting path
// produces byte-identical output to the sequential one. It compares a frame
// above parallelWriteThreshold written under several worker counts against the
// single-worker output, which is what pins the wave-ordering: a worker's block
// must land in row order, and its reused buffer must not leak across waves.
func TestWriteParallelMatchesSequential(t *testing.T) {
	// Enough rows to span several waves at any of the worker counts below.
	const rows = parallelWriteThreshold + csvChunkRows*3 + 17
	ints := make([]any, rows)
	floats := make([]any, rows)
	strs := make([]any, rows)
	for i := 0; i < rows; i++ {
		ints[i] = int64(i)
		floats[i] = float64(i) * 1.5
		switch i % 4 {
		case 0:
			strs[i] = "plain"
		case 1:
			strs[i] = "has,comma"
		case 2:
			strs[i] = `has"quote`
		default:
			strs[i] = " leading"
		}
		if i%17 == 0 {
			floats[i] = nil
			strs[i] = nil
		}
	}
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "i", Values: ints, DType: dtypes.Int64},
		{Name: "f", Values: floats, DType: dtypes.Float64},
		{Name: "s", Values: strs, DType: dtypes.String},
	}})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}

	original := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(original)

	write := func(procs int) []byte {
		runtime.GOMAXPROCS(procs)
		path := filepath.Join(t.TempDir(), "p.csv")
		if err := Write(df, WriteInput{Path: path, IncludeHeader: true, Separator: ','}); err != nil {
			t.Fatalf("GOMAXPROCS=%d: Write: %v", procs, err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		return got
	}

	sequential := write(1)
	for _, procs := range []int{2, 3, 8, 16} {
		if got := write(procs); !bytes.Equal(got, sequential) {
			t.Fatalf("GOMAXPROCS=%d: parallel output differs from sequential (%d vs %d bytes)", procs, len(got), len(sequential))
		}
	}

	// And the sequential baseline itself still matches encoding/csv.
	records := [][]string{{"i", "f", "s"}}
	for i := 0; i < rows; i++ {
		rec := make([]string, 3)
		rec[0] = strconv.FormatInt(ints[i].(int64), 10)
		if floats[i] == nil {
			rec[1] = ""
		} else {
			rec[1] = strconv.FormatFloat(floats[i].(float64), 'g', -1, 64)
		}
		if strs[i] == nil {
			rec[2] = ""
		} else {
			rec[2] = strs[i].(string)
		}
		records = append(records, rec)
	}
	if want := referenceCSV(t, records, ','); !bytes.Equal(sequential, want) {
		t.Fatalf("output differs from encoding/csv (%d vs %d bytes)", len(sequential), len(want))
	}
}

// TestWriteBlockBoundaryIsTransparent checks a frame large enough to cross the
// output block boundary many times still matches the reference exactly — the
// block flush must not drop or duplicate anything.
func TestWriteBlockBoundaryIsTransparent(t *testing.T) {
	const rows = 20000
	vals := make([]any, rows)
	for i := range vals {
		// A wide value so the frame spans many 64 KiB blocks, with quoting in play.
		vals[i] = "row,with\"quote and padding padding padding padding padding"
	}
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "s", Values: vals, DType: dtypes.String},
	}})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}

	path := filepath.Join(t.TempDir(), "big.csv")
	if err := Write(df, WriteInput{Path: path, IncludeHeader: true, Separator: ','}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	records := [][]string{{"s"}}
	for i := 0; i < rows; i++ {
		records = append(records, []string{vals[i].(string)})
	}
	if want := referenceCSV(t, records, ','); !bytes.Equal(got, want) {
		t.Fatalf("large write differs from encoding/csv (got %d bytes, want %d)", len(got), len(want))
	}
}

// TestWriteReadRoundTrip checks the round-trip contract still holds through the
// gopolars reader.
func TestWriteReadRoundTrip(t *testing.T) {
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "i", Values: []any{int64(1), int64(2), int64(3)}, DType: dtypes.Int64},
		{Name: "s", Values: []any{"a,b", "plain", `q"q`}, DType: dtypes.String},
	}})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}
	path := filepath.Join(t.TempDir(), "rt.csv")
	if err := Write(df, WriteInput{Path: path, IncludeHeader: true, Separator: ','}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	back, err := Read(ReadInput{Path: path, HasHeader: true, Separator: ','})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got, want := back.Columns(), df.Columns(); len(got) != len(want) {
		t.Fatalf("columns=%v, want %v", got, want)
	}
	if back.Height() != df.Height() {
		t.Fatalf("height=%d, want %d", back.Height(), df.Height())
	}
	for _, name := range df.Columns() {
		src, _ := df.Series(name)
		dst, ok := back.Series(name)
		if !ok {
			t.Fatalf("column %s missing after round-trip", name)
		}
		for i := 0; i < df.Height(); i++ {
			if got, want := dst.Value(i), src.Value(i); got != want {
				t.Fatalf("column %s row %d: got %v, want %v", name, i, got, want)
			}
		}
	}
}
