package unit

import (
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestWindowRollingDynamicPivotMelt(t *testing.T) {
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "city", Values: []any{"kyiv", "kyiv", "lviv", "lviv"}},
			{Name: "kind", Values: []any{"a", "b", "a", "b"}},
			{Name: "ts", Values: []any{base, base.Add(time.Hour), base, base.Add(time.Hour)}},
			{Name: "value", Values: []any{int64(10), int64(20), int64(5), int64(15)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}

	windowed, err := polars.WindowSum(df, polars.WindowSumInput{
		PartitionBy: []string{"city"},
		OrderBy:     "ts",
		Value:       "value",
		Output:      "running_value",
	})
	if err != nil {
		t.Fatalf("window sum failed: %v", err)
	}
	if windowed.Width() != 5 {
		t.Fatalf("unexpected windowed width")
	}

	rolled, err := polars.RollingMean(windowed, polars.RollingMeanInput{
		By:      "ts",
		Value:   "value",
		Window:  time.Hour,
		MinRows: 1,
		Output:  "roll_mean",
	})
	if err != nil {
		t.Fatalf("rolling mean failed: %v", err)
	}
	if rolled.Width() != 6 {
		t.Fatalf("unexpected rolled width")
	}

	dyn, err := polars.GroupByDynamic(rolled, polars.DynamicGroupInput{
		By:      "ts",
		Every:   time.Hour,
		AggExpr: polars.Sum(polars.Col("value")).Alias("sum_value"),
	})
	if err != nil {
		t.Fatalf("dynamic group failed: %v", err)
	}
	if dyn.Height() == 0 {
		t.Fatalf("unexpected dynamic group result")
	}

	melted, err := polars.Melt(df, polars.MeltInput{
		IDVars:      []string{"city"},
		ValueVars:   []string{"value"},
		VariableCol: "var",
		ValueCol:    "val",
	})
	if err != nil {
		t.Fatalf("melt failed: %v", err)
	}
	if melted.Height() != df.Height() {
		t.Fatalf("unexpected melt height")
	}

	pivoted, err := polars.Pivot(df, polars.PivotInput{
		Index:   "city",
		Columns: "kind",
		Values:  "value",
		Agg:     "sum",
	})
	if err != nil {
		t.Fatalf("pivot failed: %v", err)
	}
	if pivoted.Height() != 2 {
		t.Fatalf("unexpected pivot height")
	}
}
