package unit

import (
	"math"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestDataFrameAPISurfaceHeadTailDropRename(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "city", Values: []any{"kyiv", "lviv", "odesa"}},
			{Name: "value", Values: []any{int64(10), int64(20), int64(30)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}

	head := df.Head(2)
	if head.Height() != 2 {
		t.Fatalf("unexpected head height: %d", head.Height())
	}
	tail := df.Tail(2)
	if tail.Height() != 2 {
		t.Fatalf("unexpected tail height: %d", tail.Height())
	}
	renamed, err := df.Rename(map[string]string{"city": "location"})
	if err != nil {
		t.Fatalf("rename failed: %v", err)
	}
	if renamed.Columns()[1] != "location" {
		t.Fatalf("rename did not apply")
	}
	dropped, err := renamed.Drop("value")
	if err != nil {
		t.Fatalf("drop failed: %v", err)
	}
	if dropped.Width() != 2 {
		t.Fatalf("unexpected dropped width: %d", dropped.Width())
	}
}

func TestArrowRoundtripConstructorAndDTypes(t *testing.T) {
	ts := time.Date(2024, 8, 1, 10, 0, 0, 0, time.UTC)
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "amount", Values: []any{dtypes.DecimalValue("10.25"), dtypes.DecimalValue("20.50")}},
			{Name: "city", Values: []any{"kyiv", "lviv"}},
			{Name: "tags", Values: []any{[]any{"a", "b"}, []any{"x"}}},
			{Name: "meta", Values: []any{map[string]any{"k": "v1"}, map[string]any{"k": "v2"}}},
			{Name: "ts", Values: []any{ts, ts.Add(time.Hour)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}
	table, err := df.ToArrow(polars.ToArrowInput{})
	if err != nil {
		t.Fatalf("to arrow failed: %v", err)
	}
	rt, err := polars.NewDataFrameFromArrow(table)
	if err != nil {
		t.Fatalf("from arrow failed: %v", err)
	}
	if rt.Height() != 2 || rt.Width() != 5 {
		t.Fatalf("unexpected roundtrip shape")
	}
}

func TestNullNaNSemanticsInSortAndAgg(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "grp", Values: []any{"a", "a", "a", "b"}},
			{Name: "v", Values: []any{float64(2), math.NaN(), nil, float64(4)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}

	sorted, err := df.Sort(polars.SortInput{By: []string{"v"}, NullsLast: true})
	if err != nil {
		t.Fatalf("sort failed: %v", err)
	}
	table, err := sorted.ToArrow(polars.ToArrowInput{})
	if err != nil {
		t.Fatalf("to arrow failed: %v", err)
	}
	if table.Columns["v"][0] != float64(2) || table.Columns["v"][1] != float64(4) {
		t.Fatalf("unexpected sorted leading values")
	}

	agg, err := df.GroupBy("grp").Agg(polars.Mean(polars.Col("v")).Alias("m"))
	if err != nil {
		t.Fatalf("groupby mean failed: %v", err)
	}
	at, err := agg.ToArrow(polars.ToArrowInput{})
	if err != nil {
		t.Fatalf("agg to arrow failed: %v", err)
	}
	if len(at.Columns["m"]) != 2 {
		t.Fatalf("unexpected mean rows")
	}
}
