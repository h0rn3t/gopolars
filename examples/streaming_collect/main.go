package main

import (
	"context"
	"fmt"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func main() {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "value", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
		},
	})
	out, _ := df.Lazy().
		Filter(polars.Col("id").Gt(polars.Lit(int64(1)))).
		CollectStreaming(context.Background(), 2)
	fmt.Println(out.Height())
}
