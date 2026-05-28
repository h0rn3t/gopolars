package csv

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	"github.com/eugeneshershen/gopolars/pkg/frame"
)

func TestCSVRoundtripWithSchema(t *testing.T) {
	src, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "city", Values: []any{"kyiv", "lviv"}},
			{Name: "ts", Values: []any{
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			}},
		},
	})
	if err != nil {
		t.Fatalf("побудова dataframe: %v", err)
	}

	path := filepath.Join(t.TempDir(), "data.csv")
	if err := Write(src, WriteInput{Path: path, IncludeHeader: true, Separator: ','}); err != nil {
		t.Fatalf("запис csv: %v", err)
	}

	readAll, err := Read(ReadInput{
		Path:      path,
		HasHeader: true,
		Separator: ',',
		Schema: dtypes.Schema{
			{Name: "id", Type: dtypes.Int64},
			{Name: "city", Type: dtypes.String},
			{Name: "ts", Type: dtypes.Datetime},
		},
	})
	if err != nil {
		t.Fatalf("читання csv: %v", err)
	}
	if readAll.Height() != 2 || readAll.Width() != 3 {
		t.Fatalf("неочікувана форма: %d x %d", readAll.Height(), readAll.Width())
	}

	projected, err := Read(ReadInput{Path: path, HasHeader: true, Separator: ',', Columns: []string{"city"}})
	if err != nil {
		t.Fatalf("читання з проєкцією: %v", err)
	}
	if projected.Width() != 1 {
		t.Fatalf("очікували 1 колонку, отримали %d", projected.Width())
	}
}

func TestCSVInferTypesWithoutHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no_header.csv")
	content := "1,true,3.5\n2,false,4.5\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("запис файлу: %v", err)
	}
	df, err := Read(ReadInput{Path: path, HasHeader: false, Separator: ','})
	if err != nil {
		t.Fatalf("читання без заголовка: %v", err)
	}
	if df.Width() != 3 || df.Height() != 2 {
		t.Fatalf("неочікувана форма: %d x %d", df.Height(), df.Width())
	}
}
