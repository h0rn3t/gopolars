package unit

import (
	"context"
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestSQLWithCTEAndWindow(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "city", Values: []any{"kyiv", "kyiv", "lviv", "lviv"}},
			{Name: "value", Values: []any{int64(10), int64(20), int64(5), int64(15)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}
	query := `
WITH filtered AS (
  SELECT city, value FROM sales WHERE value > 9
)
SELECT city, sum(value) OVER (PARTITION BY city) AS city_sum
FROM filtered
ORDER BY city
`
	lf, err := polars.SQLFromDataFrame(context.Background(), df, query, "sales")
	if err != nil {
		t.Fatalf("sql parse failed: %v", err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if out.Height() != 3 {
		t.Fatalf("unexpected rows: %d", out.Height())
	}
	sumCol, ok := out.Series("city_sum")
	if !ok {
		t.Fatalf("missing city_sum")
	}
	cityCol, _ := out.Series("city")
	for i := 0; i < out.Height(); i++ {
		city := cityCol.Value(i).(string)
		sum := sumCol.Value(i).(int64)
		if city == "kyiv" && sum != 30 {
			t.Fatalf("unexpected kyiv sum")
		}
		if city == "lviv" && sum != 15 {
			t.Fatalf("unexpected lviv sum")
		}
	}
}
