package unit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	iarrow "github.com/eugeneshershen/gopolars/pkg/io/arrow"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestParquetAndArrowRoundtrip(t *testing.T) {
	dir := t.TempDir()
	parquetPath := filepath.Join(dir, "data.parquet")

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "city", Values: []any{"kyiv", "lviv", "odesa"}},
			{Name: "value", Values: []any{float64(10.5), float64(20.5), float64(30.5)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}

	if err := df.WriteParquet(polars.WriteParquetInput{Path: parquetPath}); err != nil {
		t.Fatalf("write parquet failed: %v", err)
	}

	io := polars.NewIO()
	readAll, err := io.ReadParquet(polars.ReadParquetInput{Path: parquetPath})
	if err != nil {
		t.Fatalf("read parquet failed: %v", err)
	}
	if readAll.Width() != 3 || readAll.Height() != 3 {
		t.Fatalf("unexpected read parquet shape: %d x %d", readAll.Height(), readAll.Width())
	}

	scan, err := io.ScanParquet(polars.ScanParquetInput{Path: parquetPath, Columns: []string{"id", "city"}})
	if err != nil {
		t.Fatalf("scan parquet failed: %v", err)
	}
	collected, err := scan.Collect(context.Background())
	if err != nil {
		t.Fatalf("scan parquet collect failed: %v", err)
	}
	if collected.Width() != 2 {
		t.Fatalf("expected projection width 2, got %d", collected.Width())
	}

	table, err := df.ToArrow(polars.ToArrowInput{})
	if err != nil {
		t.Fatalf("to arrow failed: %v", err)
	}
	fromArrow, err := iarrow.FromTable(table)
	if err != nil {
		t.Fatalf("from arrow failed: %v", err)
	}
	if fromArrow.Height() != df.Height() || fromArrow.Width() != df.Width() {
		t.Fatalf("arrow roundtrip shape mismatch")
	}
}

func TestSQLPlanGoldenAndAPIParity(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "city", Values: []any{"kyiv", "kyiv", "lviv", "lviv"}},
			{Name: "value", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}

	query := "SELECT city, sum(value) as total FROM sales WHERE value > 10 GROUP BY city ORDER BY city LIMIT 2"
	sqlLF, err := polars.SQLFromDataFrame(context.Background(), df, query, "sales")
	if err != nil {
		t.Fatalf("parse sql failed: %v", err)
	}

	apiLF := df.Lazy().
		Filter(polars.Col("value").Gt(polars.Lit(int64(10)))).
		GroupBy("city").
		Agg(polars.Sum(polars.Col("value")).Alias("total")).
		Select(polars.Col("city"), polars.Col("total")).
		Sort(polars.SortInput{By: []string{"city"}}).
		Limit(2)

	sqlExplain := sqlLF.Explain(true)
	apiExplain := apiLF.Explain(true)
	if sqlExplain != apiExplain {
		t.Fatalf("sql/api explain mismatch:\nsql=%s\napi=%s", sqlExplain, apiExplain)
	}

	goldenPath := filepath.Join("testdata", "sql_explain.golden")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden failed: %v", err)
	}
	if strings.TrimSpace(string(want)) != strings.TrimSpace(sqlExplain) {
		t.Fatalf("golden mismatch:\nwant=%s\ngot=%s", string(want), sqlExplain)
	}

	sqlOut, err := sqlLF.Collect(context.Background())
	if err != nil {
		t.Fatalf("sql collect failed: %v", err)
	}
	apiOut, err := apiLF.Collect(context.Background())
	if err != nil {
		t.Fatalf("api collect failed: %v", err)
	}
	if sqlOut.Height() != apiOut.Height() || sqlOut.Width() != apiOut.Width() {
		t.Fatalf("sql/api output shape mismatch")
	}
}
