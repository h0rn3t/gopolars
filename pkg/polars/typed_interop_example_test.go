package polars_test

import (
	"fmt"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func ExampleNewDataFrameFromSeries() {
	id, err := polars.NewInt64Series("id", []int64{1, 2, 3}, []bool{false, true, false})
	if err != nil {
		fmt.Println(err)
		return
	}
	label, err := polars.NewStringSeries("label", []string{"a", "b", "c"}, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	df, err := polars.NewDataFrameFromSeries(id, label)
	if err != nil {
		fmt.Println(err)
		return
	}

	col, err := df.GetColumn("id")
	if err != nil {
		fmt.Println(err)
		return
	}
	ids, nulls, err := col.Int64Values()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(ids, nulls)
	// Output: [1 0 3] [false true false]
}
