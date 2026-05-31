package parquet

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

func TestParquetRoundtripAllDtypes(t *testing.T) {
	ts := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "score", Values: []any{float64(1.5), float64(2.5)}},
			{Name: "active", Values: []any{true, false}},
			{Name: "ts", Values: []any{ts, ts.Add(time.Hour)}},
			{Name: "amount", Values: []any{dtypes.DecimalValue("10.50"), dtypes.DecimalValue("20.00")}},
			{Name: "label", Values: []any{"alpha", "beta"}},
		},
	})
	if err != nil {
		t.Fatalf("побудова dataframe: %v", err)
	}

	path := filepath.Join(t.TempDir(), "all_dtypes.parquet")
	if err := Write(df, WriteInput{Path: path}); err != nil {
		t.Fatalf("запис parquet: %v", err)
	}

	readAll, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("читання parquet: %v", err)
	}
	if readAll.Height() != 2 || readAll.Width() != 6 {
		t.Fatalf("неочікувана форма: %d x %d", readAll.Height(), readAll.Width())
	}

	projected, err := Read(ReadInput{Path: path, Columns: []string{"id", "label"}})
	if err != nil {
		t.Fatalf("читання з проєкцією: %v", err)
	}
	if projected.Width() != 2 {
		t.Fatalf("очікували 2 колонки, отримали %d", projected.Width())
	}
}

func TestParquetJSONFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.json")
	payload := `{"columns":[{"name":"id","type":"int64","values":[1,2]}]}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("запис fallback-файлу: %v", err)
	}
	df, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("читання JSON fallback: %v", err)
	}
	if df.Height() != 2 {
		t.Fatalf("очікували 2 рядки, отримали %d", df.Height())
	}
}

func TestDecodeValuesErrors(t *testing.T) {
	if _, err := decodeValues([]any{float64(1)}, string(dtypes.Boolean)); err == nil {
		t.Fatal("очікували помилку для bool")
	}
	if _, err := decodeValues([]any{"not-a-time"}, string(dtypes.Datetime)); err == nil {
		t.Fatal("очікували помилку для datetime")
	}
}
