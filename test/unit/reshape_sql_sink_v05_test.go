package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
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

func TestSQLSubqueryAndSetOps(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "value", Values: []any{int64(10), int64(20), int64(30)}},
		},
	})
	query := `
SELECT id FROM (SELECT id, value FROM t WHERE value >= 20) s WHERE id > 1
`
	lf, err := polars.SQLFromDataFrame(context.Background(), df, query, "t")
	if err != nil {
		t.Fatalf("subquery parse failed: %v", err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("subquery collect failed: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("unexpected subquery rows")
	}
	unionQuery := `SELECT id FROM t WHERE id <= 2 UNION SELECT id FROM t WHERE id >= 2`
	lf2, err := polars.SQLFromDataFrame(context.Background(), df, unionQuery, "t")
	if err != nil {
		t.Fatalf("union parse failed: %v", err)
	}
	unionOut, err := lf2.Collect(context.Background())
	if err != nil {
		t.Fatalf("union collect failed: %v", err)
	}
	if unionOut.Height() != 3 {
		t.Fatalf("unexpected union rows")
	}

	intersectQuery := `SELECT id FROM t WHERE id <= 2 INTERSECT SELECT id FROM t WHERE id >= 2`
	lf3, err := polars.SQLFromDataFrame(context.Background(), df, intersectQuery, "t")
	if err != nil {
		t.Fatalf("intersect parse failed: %v", err)
	}
	intersectOut, err := lf3.Collect(context.Background())
	if err != nil {
		t.Fatalf("intersect collect failed: %v", err)
	}
	if intersectOut.Height() != 1 {
		t.Fatalf("unexpected intersect rows: %d", intersectOut.Height())
	}

	exceptQuery := `SELECT id FROM t WHERE id <= 2 EXCEPT SELECT id FROM t WHERE id >= 2`
	lf4, err := polars.SQLFromDataFrame(context.Background(), df, exceptQuery, "t")
	if err != nil {
		t.Fatalf("except parse failed: %v", err)
	}
	exceptOut, err := lf4.Collect(context.Background())
	if err != nil {
		t.Fatalf("except collect failed: %v", err)
	}
	if exceptOut.Height() != 1 {
		t.Fatalf("unexpected except rows: %d", exceptOut.Height())
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
