package macro

import (
	"context"
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func BenchmarkE2ELazyGroupBySortLimit(b *testing.B) {
	n := 50000
	city := make([]any, n)
	value := make([]any, n)
	for i := 0; i < n; i++ {
		if i%3 == 0 {
			city[i] = "kyiv"
		} else if i%3 == 1 {
			city[i] = "lviv"
		} else {
			city[i] = "odesa"
		}
		value[i] = int64(i % 1000)
	}
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "city", Values: city},
			{Name: "value", Values: value},
		},
	})
	if err != nil {
		b.Fatalf("new dataframe failed: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := df.
			Lazy().
			Filter(polars.Col("value").Gt(polars.Lit(int64(100)))).
			GroupBy("city").
			Agg(polars.Sum(polars.Col("value")).Alias("total")).
			Sort(polars.SortInput{By: []string{"total"}, Descending: []bool{true}}).
			Limit(2).
			Collect(context.Background())
		if err != nil {
			b.Fatalf("pipeline failed: %v", err)
		}
	}
}
