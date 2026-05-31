package micro

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func BenchmarkFilterInt64(b *testing.B) {
	n := 20000
	ids := make([]any, n)
	values := make([]any, n)
	for i := 0; i < n; i++ {
		ids[i] = int64(i)
		values[i] = int64(i % 100)
	}
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: ids},
			{Name: "value", Values: values},
		},
	})
	if err != nil {
		b.Fatalf("new dataframe failed: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := df.Filter(polars.Col("value").Gt(polars.Lit(int64(50))))
		if err != nil {
			b.Fatalf("filter failed: %v", err)
		}
	}
}
