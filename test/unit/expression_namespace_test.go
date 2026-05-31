package unit

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestExpressionNamespacesAndWhen(t *testing.T) {
	ts := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "name", Values: []any{"Kyiv", "Lviv"}},
			{Name: "score", Values: []any{int64(10), int64(30)}},
			{Name: "ts", Values: []any{ts, ts.AddDate(1, 0, 0)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}

	out, err := df.Select(
		polars.Col("name").StrLower().Alias("lower"),
		polars.Col("name").StrUpper().Alias("upper"),
		polars.Col("name").StrLen().Alias("len"),
		polars.Col("name").Contains(polars.Lit("yi")).Alias("contains"),
		polars.Col("ts").DtYear().Alias("year"),
		polars.When(
			polars.Col("score").Gt(polars.Lit(int64(20))),
			polars.Lit("high"),
			polars.Lit("low"),
		).Alias("bucket"),
	)
	if err != nil {
		t.Fatalf("select failed: %v", err)
	}

	table, err := out.ToArrow(polars.ToArrowInput{})
	if err != nil {
		t.Fatalf("to arrow failed: %v", err)
	}
	if table.Columns["lower"][0] != "kyiv" || table.Columns["upper"][1] != "LVIV" {
		t.Fatalf("unexpected string namespace results")
	}
	if table.Columns["len"][0] != int64(4) {
		t.Fatalf("unexpected string length")
	}
	if table.Columns["contains"][0] != true || table.Columns["contains"][1] != false {
		t.Fatalf("unexpected contains results")
	}
	if table.Columns["year"][0] != int64(2024) || table.Columns["year"][1] != int64(2025) {
		t.Fatalf("unexpected year extraction")
	}
	if table.Columns["bucket"][0] != "low" || table.Columns["bucket"][1] != "high" {
		t.Fatalf("unexpected when results")
	}
}
