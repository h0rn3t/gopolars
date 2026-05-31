package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func main() {
	tmp := filepath.Join(".", "examples_lazy_pushdown.parquet")
	base, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "value", Values: []any{int64(10), int64(20), int64(30)}},
		},
	})
	_ = base.WriteParquet(polars.WriteParquetInput{Path: tmp})
	io := polars.NewIO()
	lf, _ := io.ScanParquet(polars.ScanParquetInput{Path: tmp})
	out, _ := lf.Filter(polars.Col("id").Gt(polars.Lit(int64(1)))).
		Select(polars.Col("value")).
		Collect(context.Background())
	fmt.Println(out.Height(), out.Width())
}
