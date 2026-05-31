package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestPartitionAwareParquetDatasetScan(t *testing.T) {
	root := t.TempDir()
	y2024 := filepath.Join(root, "year=2024")
	y2025 := filepath.Join(root, "year=2025")
	_ = os.MkdirAll(y2024, 0o755)
	_ = os.MkdirAll(y2025, 0o755)
	dfA, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1)}},
			{Name: "value", Values: []any{int64(10)}},
		},
	})
	dfB, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(2)}},
			{Name: "value", Values: []any{int64(20)}},
		},
	})
	_ = dfA.WriteParquet(polars.WriteParquetInput{Path: filepath.Join(y2024, "part.parquet")})
	_ = dfB.WriteParquet(polars.WriteParquetInput{Path: filepath.Join(y2025, "part.parquet")})

	io := polars.NewIO()
	lf, err := io.ScanParquet(polars.ScanParquetInput{Path: root})
	if err != nil {
		t.Fatalf("scan parquet failed: %v", err)
	}
	out, err := lf.Filter(polars.Col("year").Eq(polars.Lit("2025"))).Collect(context.Background())
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if out.Height() != 1 {
		t.Fatalf("unexpected rows: %d", out.Height())
	}
	year, _ := out.Series("year")
	if year.Value(0) != "2025" {
		t.Fatalf("expected pruned partition value")
	}
}

func TestExplainDiagnosticsSchema(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{{Name: "id", Values: []any{int64(1), int64(2)}}},
	})
	diag := df.Lazy().Filter(polars.Col("id").Gt(polars.Lit(int64(1)))).ExplainDiagnostics(true)
	for _, k := range []string{"schema_version", "logical_nodes", "optimized_nodes", "stateful_pipeline", "plan"} {
		if _, ok := diag[k]; !ok {
			t.Fatalf("missing diagnostics key %s", k)
		}
	}
}
