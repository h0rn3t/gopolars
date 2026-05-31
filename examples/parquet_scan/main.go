package main

import (
	"fmt"
	"path/filepath"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func main() {
	tmp := filepath.Join(".", "examples_parquet_scan.parquet")
	base, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "city", Values: []any{"kyiv", "lviv"}},
		},
	})
	_ = base.WriteParquet(polars.WriteParquetInput{Path: tmp})
	io := polars.NewIO()
	out, _ := io.ReadParquet(polars.ReadParquetInput{Path: tmp, Columns: []string{"city"}})
	fmt.Println(out.Width())
}
