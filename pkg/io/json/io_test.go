package json

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

func TestJSONRoundtripPretty(t *testing.T) {
	src, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "city", Values: []any{"kyiv", "lviv"}},
		},
	})
	if err != nil {
		t.Fatalf("побудова dataframe: %v", err)
	}

	path := filepath.Join(t.TempDir(), "data.json")
	if err := Write(src, WriteInput{Path: path, Pretty: true}); err != nil {
		t.Fatalf("запис json: %v", err)
	}
	readAll, err := Read(ReadInput{Path: path, Schema: dtypes.Schema{{Name: "id", Type: dtypes.Int64}}})
	if err != nil {
		t.Fatalf("читання json: %v", err)
	}
	if readAll.Height() != 2 || readAll.Width() != 2 {
		t.Fatalf("неочікувана форма: %d x %d", readAll.Height(), readAll.Width())
	}
}

func TestNDJSONRoundtrip(t *testing.T) {
	ts := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	src, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1)}},
			{Name: "ts", Values: []any{ts}},
		},
	})
	if err != nil {
		t.Fatalf("побудова dataframe: %v", err)
	}

	path := filepath.Join(t.TempDir(), "lines.ndjson")
	if err := Write(src, WriteInput{Path: path, NDJSON: true}); err != nil {
		t.Fatalf("запис ndjson: %v", err)
	}
	readBack, err := Read(ReadInput{Path: path, NDJSON: true, Columns: []string{"id"}})
	if err != nil {
		t.Fatalf("читання ndjson: %v", err)
	}
	if readBack.Width() != 1 || readBack.Height() != 1 {
		t.Fatalf("неочікувана форма: %d x %d", readBack.Height(), readBack.Width())
	}
}
