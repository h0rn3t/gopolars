package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestMeltPivotParity(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "city", Values: []any{"kyiv", "lviv"}},
			{Name: "a", Values: []any{int64(1), int64(2)}},
			{Name: "b", Values: []any{int64(10), int64(20)}},
		},
	})
	melted, err := df.Melt(polars.MeltInput{IDVars: []string{"city"}, ValueVars: []string{"a", "b"}, VariableCol: "metric", ValueCol: "val"})
	if err != nil {
		t.Fatalf("melt failed: %v", err)
	}
	if melted.Height() != 4 {
		t.Fatalf("unexpected melt rows")
	}
	pivoted, err := melted.Pivot(polars.PivotInput{Index: "city", Columns: "metric", Values: "val", Agg: "sum"})
	if err != nil {
		t.Fatalf("pivot failed: %v", err)
	}
	if pivoted.Width() != 3 {
		t.Fatalf("unexpected pivot width")
	}
	lazy, err := df.Lazy().Melt(polars.MeltInput{IDVars: []string{"city"}, ValueVars: []string{"a", "b"}, VariableCol: "metric", ValueCol: "val"}).Collect(context.Background())
	if err != nil {
		t.Fatalf("lazy melt failed: %v", err)
	}
	if lazy.Height() != melted.Height() {
		t.Fatalf("lazy/eager melt mismatch")
	}
}

func TestLazySinkCollectParity(t *testing.T) {
	root := t.TempDir()
	parquetPath := filepath.Join(root, "out.parquet")
	csvPath := filepath.Join(root, "out.csv")
	ipcPath := filepath.Join(root, "out.ipc")
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "city", Values: []any{"kyiv", "lviv"}},
		},
	})
	lf := df.Lazy().Filter(polars.Col("id").Gt(polars.Lit(int64(0))))
	collected, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if err := lf.SinkParquet(context.Background(), polars.WriteParquetInput{Path: parquetPath}); err != nil {
		t.Fatalf("sink parquet failed: %v", err)
	}
	if err := lf.SinkCSV(context.Background(), polars.WriteCSVInput{Path: csvPath, IncludeHeader: true, Separator: ','}); err != nil {
		t.Fatalf("sink csv failed: %v", err)
	}
	if err := lf.SinkIPC(context.Background(), polars.WriteIPCInput{Path: ipcPath}); err != nil {
		t.Fatalf("sink ipc failed: %v", err)
	}
	if _, err := os.Stat(parquetPath); err != nil {
		t.Fatalf("missing parquet sink output")
	}
	io := polars.NewIO()
	readBack, err := io.ReadParquet(polars.ReadParquetInput{Path: parquetPath})
	if err != nil {
		t.Fatalf("readback failed: %v", err)
	}
	if readBack.Height() != collected.Height() {
		t.Fatalf("sink/collect mismatch")
	}
}
