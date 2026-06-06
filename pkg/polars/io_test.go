package polars

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// newIOTestFrame builds a 4-row, 2-column Int64+String frame used by IO
// round-trip tests.
func newIOTestFrame(t *testing.T) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		{Name: "b", Values: []any{"w", "x", "y", "z"}},
	}})
	if err != nil {
		t.Fatalf("newIOTestFrame: %v", err)
	}
	return df
}

// TestNewIOReturnsNonNil pins the documented constructor contract.
func TestNewIOReturnsNonNil(t *testing.T) {
	if NewIO() == nil {
		t.Fatalf("NewIO returned nil")
	}
}

// TestNewIOIndependentInstances verifies two NewIO calls both return a
// non-nil usable facade. The current implementation returns a value-typed
// facade so two calls compare equal, but each is independently usable.
func TestNewIOIndependentInstances(t *testing.T) {
	a := NewIO()
	b := NewIO()
	if a == nil || b == nil {
		t.Fatalf("NewIO returned nil: a=%v b=%v", a, b)
	}
	// Both must satisfy the IO interface and respond to ScanCSV without panic.
	if _, err := a.ScanCSV(ScanCSVInput{Path: "/tmp/none.csv"}); err != nil {
		t.Fatalf("a.ScanCSV: %v", err)
	}
	if _, err := b.ScanCSV(ScanCSVInput{Path: "/tmp/none.csv"}); err != nil {
		t.Fatalf("b.ScanCSV: %v", err)
	}
} // TestIORoundTripCSV writes a frame, then reads it back via IO.ReadCSV.
func TestIORoundTripCSV(t *testing.T) {
	io := NewIO()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.csv")
	df := newIOTestFrame(t)
	if err := df.WriteCSV(WriteCSVInput{Path: path, IncludeHeader: true}); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	got, err := io.ReadCSV(ReadCSVInput{Path: path, HasHeader: true})
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	if got.Height() != df.Height() {
		t.Errorf("ReadCSV height = %d, want %d", got.Height(), df.Height())
	}
	if got.Width() != df.Width() {
		t.Errorf("ReadCSV width = %d, want %d", got.Width(), df.Width())
	}
}

// TestIORoundTripParquet writes a frame, reads it back via IO.ReadParquet.
func TestIORoundTripParquet(t *testing.T) {
	io := NewIO()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.parquet")
	df := newIOTestFrame(t)
	if err := df.WriteParquet(WriteParquetInput{Path: path}); err != nil {
		t.Fatalf("WriteParquet: %v", err)
	}
	got, err := io.ReadParquet(ReadParquetInput{Path: path})
	if err != nil {
		t.Fatalf("ReadParquet: %v", err)
	}
	if got.Height() != df.Height() {
		t.Errorf("ReadParquet height = %d, want %d", got.Height(), df.Height())
	}
	if got.Width() != df.Width() {
		t.Errorf("ReadParquet width = %d, want %d", got.Width(), df.Width())
	}
}

// TestIORoundTripIPC writes a frame, reads it back via IO.ReadIPC.
func TestIORoundTripIPC(t *testing.T) {
	io := NewIO()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.ipc")
	df := newIOTestFrame(t)
	if err := df.WriteIPC(WriteIPCInput{Path: path}); err != nil {
		t.Fatalf("WriteIPC: %v", err)
	}
	got, err := io.ReadIPC(ReadIPCInput{Path: path})
	if err != nil {
		t.Fatalf("ReadIPC: %v", err)
	}
	if got.Height() != df.Height() {
		t.Errorf("ReadIPC height = %d, want %d", got.Height(), df.Height())
	}
}

// TestIORoundTripJSON writes a frame, reads it back via IO.ReadJSON.
func TestIORoundTripJSON(t *testing.T) {
	io := NewIO()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.json")
	df := newIOTestFrame(t)
	if err := df.WriteJSON(WriteJSONInput{Path: path, NDJSON: true}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	got, err := io.ReadJSON(ReadJSONInput{Path: path, NDJSON: true})
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if got.Height() != df.Height() {
		t.Errorf("ReadJSON height = %d, want %d", got.Height(), df.Height())
	}
}

// TestIOReadMissingFile returns an error.
func TestIOReadMissingFile(t *testing.T) {
	io := NewIO()
	if _, err := io.ReadCSV(ReadCSVInput{Path: "/does/not/exist.csv"}); err == nil {
		t.Fatalf("ReadCSV on missing file returned nil error, want non-nil")
	}
}

// TestIOWriteUnwritablePath returns an error.
func TestIOWriteUnwritablePath(t *testing.T) {
	df := newIOTestFrame(t)
	err := df.WriteCSV(WriteCSVInput{Path: "/this/does/not/exist/out.csv"})
	if err == nil {
		t.Fatalf("WriteCSV to unwritable path returned nil error, want non-nil")
	}
}

// TestIOScanCSVFollowedByCollect exercises the Scan + Collect path.
func TestIOScanCSVFollowedByCollect(t *testing.T) {
	io := NewIO()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.csv")
	df := newIOTestFrame(t)
	if err := df.WriteCSV(WriteCSVInput{Path: path, IncludeHeader: true}); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	lf, err := io.ScanCSV(ScanCSVInput{Path: path, HasHeader: true})
	if err != nil {
		t.Fatalf("ScanCSV: %v", err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if out.Height() != df.Height() {
		t.Errorf("ScanCSV Collect height = %d, want %d", out.Height(), df.Height())
	}
}

// TestIOScanParquetFollowedByCollect exercises the Scan + Collect path.
func TestIOScanParquetFollowedByCollect(t *testing.T) {
	io := NewIO()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.parquet")
	df := newIOTestFrame(t)
	if err := df.WriteParquet(WriteParquetInput{Path: path}); err != nil {
		t.Fatalf("WriteParquet: %v", err)
	}
	lf, err := io.ScanParquet(ScanParquetInput{Path: path})
	if err != nil {
		t.Fatalf("ScanParquet: %v", err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if out.Height() != df.Height() {
		t.Errorf("ScanParquet Collect height = %d, want %d", out.Height(), df.Height())
	}
}

// TestIOScanIPCReturnsLazyFrame verifies ScanIPC returns a non-nil LF.
func TestIOScanIPCReturnsLazyFrame(t *testing.T) {
	io := NewIO()
	lf, err := io.ScanIPC(ScanIPCInput{Path: "/tmp/none.ipc"})
	if err != nil {
		t.Fatalf("ScanIPC: %v", err)
	}
	if lf == nil {
		t.Fatalf("ScanIPC returned nil LF")
	}
}

// TestIOScanJSONReturnsLazyFrame verifies ScanJSON returns a non-nil LF.
func TestIOScanJSONReturnsLazyFrame(t *testing.T) {
	io := NewIO()
	lf, err := io.ScanJSON(ScanJSONInput{Path: "/tmp/none.json"})
	if err != nil {
		t.Fatalf("ScanJSON: %v", err)
	}
	if lf == nil {
		t.Fatalf("ScanJSON returned nil LF")
	}
}

// TestIOReadCSVHasHeaderFalse exercises the no-header CSV path.
func TestIOReadCSVHasHeaderFalse(t *testing.T) {
	io := NewIO()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "noheader.csv")
	// Write a CSV with no header.
	if err := writeRawFile(t, path, "1,2\n3,4\n"); err != nil {
		t.Fatalf("writeRawFile: %v", err)
	}
	got, err := io.ReadCSV(ReadCSVInput{Path: path, HasHeader: false})
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	if got.Height() != 2 {
		t.Errorf("height = %d, want 2 (no header rows)", got.Height())
	}
}

// TestIOReadParquetColumnProjection verifies the Columns option is honored.
func TestIOReadParquetColumnProjection(t *testing.T) {
	io := NewIO()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.parquet")
	df := newIOTestFrame(t)
	if err := df.WriteParquet(WriteParquetInput{Path: path}); err != nil {
		t.Fatalf("WriteParquet: %v", err)
	}
	got, err := io.ReadParquet(ReadParquetInput{Path: path, Columns: []string{"a"}})
	if err != nil {
		t.Fatalf("ReadParquet with projection: %v", err)
	}
	if got.Width() != 1 {
		t.Errorf("projected width = %d, want 1", got.Width())
	}
	if got.Columns()[0] != "a" {
		t.Errorf("projected column = %q, want a", got.Columns()[0])
	}
}

// TestIOSQLReturnsLazyFrame verifies IO.SQL returns a non-nil LazyFrame.
func TestIOSQLReturnsLazyFrame(t *testing.T) {
	io := NewIO()
	// Use a query that the underlying parser will accept; we don't care about
	// the columns, only that the wrapper dispatches and returns a non-nil LF.
	lf, err := io.SQL(context.Background(), "SELECT 1 AS x")
	if err != nil {
		// The parser may reject this on its own; either way, the call should
		// return a non-nil LF or a non-nil err, never a nil LF + nil err.
		if lf != nil {
			t.Errorf("io.SQL returned non-nil LazyFrame with err: %v", err)
		}
		return
	}
	if lf == nil {
		t.Fatalf("io.SQL returned nil LazyFrame with no error")
	}
}

// writeRawFile writes a string to a file path, used by tests that need
// hand-crafted CSV/JSON input.
func writeRawFile(t *testing.T, path, content string) error {
	t.Helper()
	return writeFile(path, []byte(content))
}
