package ipc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestReadMissingFile exercises the os.Open error path.
func TestReadMissingFile(t *testing.T) {
	if _, err := Read(ReadInput{Path: filepath.Join(t.TempDir(), "nope.ipc")}); err == nil {
		t.Error("Read missing file: expected error")
	}
}

// TestReadGarbage exercises the gob decode error path.
func TestReadGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garbage.ipc")
	if err := os.WriteFile(path, []byte("not a gob stream"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Read(ReadInput{Path: path}); err == nil {
		t.Error("Read garbage: expected decode error")
	}
}

// TestWriteMissingDir exercises the os.Create error path.
func TestWriteMissingDir(t *testing.T) {
	src, _ := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1)}},
	}})
	if err := Write(src, WriteInput{Path: filepath.Join(t.TempDir(), "nope", "out.ipc")}); err == nil {
		t.Error("Write to missing dir: expected error")
	}
}
