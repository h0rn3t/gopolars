package unit

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestScanCSVIsLazyAndCollectAppliesPlan(t *testing.T) {
	io := polars.NewIO()
	lf, err := io.ScanCSV(polars.ScanCSVInput{
		Path:      filepath.Join(t.TempDir(), "missing.csv"),
		HasHeader: true,
		Separator: ',',
	})
	if err != nil {
		t.Fatalf("scan should not fail eagerly: %v", err)
	}
	if _, err := lf.Collect(context.Background()); err == nil {
		t.Fatalf("collect must fail for missing source file")
	}
}

func TestScanParquetPushdownCollect(t *testing.T) {
	base, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "city", Values: []any{"kyiv", "lviv", "odesa"}},
			{Name: "value", Values: []any{int64(10), int64(20), int64(30)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}
	path := filepath.Join(t.TempDir(), "data.parquet")
	if err := base.WriteParquet(polars.WriteParquetInput{Path: path}); err != nil {
		t.Fatalf("write parquet failed: %v", err)
	}
	io := polars.NewIO()
	lazy, err := io.ScanParquet(polars.ScanParquetInput{Path: path})
	if err != nil {
		t.Fatalf("scan parquet failed: %v", err)
	}
	lf, err := lazy.
		Filter(polars.Col("id").Gt(polars.Lit(int64(1)))).
		Select(polars.Col("city"), polars.Col("value")).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if lf.Height() != 2 || lf.Width() != 2 {
		t.Fatalf("unexpected pushed-down shape: %d x %d", lf.Height(), lf.Width())
	}
}
