package main

import (
	"fmt"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func main() {
	left, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "l", Values: []any{"a", "b"}},
		},
	})
	right, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(2), int64(3)}},
			{Name: "r", Values: []any{"x", "y"}},
		},
	})
	out, _ := left.Join(polars.JoinInput{
		Other:   right,
		LeftOn:  []string{"id"},
		RightOn: []string{"id"},
		How:     polars.JoinTypeFull,
	})
	fmt.Println(out.Height())
}
