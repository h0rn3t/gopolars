package ipc

import (
	"path/filepath"
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
)

func TestIPCRoundtripWithProjection(t *testing.T) {
	src, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "city", Values: []any{"kyiv", "lviv"}},
		},
	})
	if err != nil {
		t.Fatalf("побудова dataframe: %v", err)
	}

	path := filepath.Join(t.TempDir(), "data.ipc")
	if err := Write(src, WriteInput{Path: path}); err != nil {
		t.Fatalf("запис ipc: %v", err)
	}

	readAll, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("читання ipc: %v", err)
	}
	if readAll.Height() != 2 || readAll.Width() != 2 {
		t.Fatalf("неочікувана форма: %d x %d", readAll.Height(), readAll.Width())
	}

	projected, err := Read(ReadInput{Path: path, Columns: []string{"city"}})
	if err != nil {
		t.Fatalf("читання з проєкцією: %v", err)
	}
	if projected.Width() != 1 {
		t.Fatalf("очікували 1 колонку, отримали %d", projected.Width())
	}
}
