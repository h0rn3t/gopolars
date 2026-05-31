package unit

import (
	"context"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestPythonInteropSurface(t *testing.T) {
	t.Parallel()

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
			{Name: "b", Values: []any{float64(1.5), float64(2.5)}},
		},
	})
	if err != nil {
		t.Fatalf("створення DataFrame: %v", err)
	}
	col, err := df.GetColumn("a")
	if err != nil {
		t.Fatalf("GetColumn: %v", err)
	}

	if got := polars.PyArrayDataFrame(nil); got != nil {
		t.Fatalf("PyArrayDataFrame(nil) очікував nil, отримано %v", got)
	}
	if got := polars.PyArraySeries(nil); got != nil {
		t.Fatalf("PyArraySeries(nil) очікував nil, отримано %v", got)
	}
	if _, err := polars.PyArrowTableDataFrame(nil); err == nil {
		t.Fatal("PyArrowTableDataFrame(nil) очікував помилку")
	}
	if _, err := polars.PyArrowTableSeries(nil); err == nil {
		t.Fatal("PyArrowTableSeries(nil) очікував помилку")
	}

	if len(polars.PyArraySeries(col)) != 2 {
		t.Fatalf("PyArraySeries: неочікувана довжина")
	}
	if view := polars.PyDataFrameInterchange(df); view.Height() != df.Height() {
		t.Fatalf("PyDataFrameInterchange: висота %d != %d", view.Height(), df.Height())
	}

	table, err := polars.PyArrowTableDataFrame(df)
	if err != nil {
		t.Fatalf("PyArrowTableDataFrame: %v", err)
	}
	if len(table.Columns) != 2 {
		t.Fatalf("PyArrowTableDataFrame: очікували 2 колонки, отримали %d", len(table.Columns))
	}
	seriesTable, err := polars.PyArrowTableSeries(col)
	if err != nil {
		t.Fatalf("PyArrowTableSeries: %v", err)
	}
	if len(seriesTable.Columns) != 1 {
		t.Fatalf("PyArrowTableSeries: очікували 1 колонку")
	}
}

func TestGroupByMultipleAggregations(t *testing.T) {
	t.Parallel()

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "region", Values: []any{"north", "north", "south", "south", "south"}},
			{Name: "amount", Values: []any{int64(10), int64(20), int64(5), int64(15), int64(25)}},
			{Name: "sku", Values: []any{"A", "B", "A", "A", "C"}},
		},
	})
	if err != nil {
		t.Fatalf("створення DataFrame: %v", err)
	}

	agg, err := df.GroupBy("region").Agg(
		polars.Sum(polars.Col("amount")).Alias("total"),
		polars.Mean(polars.Col("amount")).Alias("avg"),
		polars.NUnique(polars.Col("sku")).Alias("sku_count"),
	)
	if err != nil {
		t.Fatalf("GroupBy.Agg: %v", err)
	}
	if agg.Height() != 2 || agg.Width() != 4 {
		t.Fatalf("неочікувана форма: %d x %d", agg.Height(), agg.Width())
	}

	northDF, err := agg.Filter(polars.Col("region").Eq(polars.Lit("north")))
	if err != nil {
		t.Fatalf("фільтр north: %v", err)
	}
	northTotal, err := northDF.Item(0, "total")
	if err != nil {
		t.Fatalf("Item north total: %v", err)
	}
	if northTotal != int64(30) {
		t.Fatalf("north total: очікували 30, отримали %v", northTotal)
	}

	southDF, err := agg.Filter(polars.Col("region").Eq(polars.Lit("south")))
	if err != nil {
		t.Fatalf("фільтр south: %v", err)
	}
	southSKU, err := southDF.Item(0, "sku_count")
	if err != nil {
		t.Fatalf("фільтр south: %v", err)
	}
	if southSKU != int64(2) {
		t.Fatalf("south sku_count: очікували 2, отримали %v", southSKU)
	}
}

func TestDataFrameCleaningPipeline(t *testing.T) {
	t.Parallel()

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "name", Values: []any{"  Kyiv ", "lviv", "kharkiv", "odesa"}},
			{Name: "score", Values: []any{int64(80), int64(55), int64(90), int64(40)}},
		},
	})
	if err != nil {
		t.Fatalf("створення DataFrame: %v", err)
	}

	cleaned, err := df.WithColumns(
		polars.Col("name").StrLower().Alias("name_norm"),
		polars.Col("score").Gt(polars.Lit(int64(60))).Alias("passed"),
	)
	if err != nil {
		t.Fatalf("WithColumns: %v", err)
	}
	sorted, err := cleaned.Sort(polars.SortInput{By: []string{"score"}, Descending: []bool{true}})
	if err != nil {
		t.Fatalf("Sort: %v", err)
	}
	top, err := sorted.Filter(polars.Col("passed").Eq(polars.Lit(true)))
	if err != nil {
		t.Fatalf("Filter: %v", err)
	}
	if top.Height() != 2 {
		t.Fatalf("очікували 2 рядки з passed=true, отримали %d", top.Height())
	}

	firstScore, err := top.Item(0, "score")
	if err != nil {
		t.Fatalf("Item score: %v", err)
	}
	if firstScore != int64(90) {
		t.Fatalf("перший score після sort desc: очікували 90, отримали %v", firstScore)
	}
}

func TestSeriesNamespacesDirect(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	strSeries, err := polars.NewSeries(polars.NewSeriesInput{
		Name: "s", DType: dtypes.String, Values: []any{" hello ", "WORLD"},
	})
	if err != nil {
		t.Fatal(err)
	}
	dtSeries, err := polars.NewSeries(polars.NewSeriesInput{
		Name: "ts", DType: dtypes.Datetime, Values: []any{ts},
	})
	if err != nil {
		t.Fatal(err)
	}
	listSeries, err := polars.NewSeries(polars.NewSeriesInput{
		Name: "tags", DType: dtypes.List, Values: []any{[]any{"a", "b", "c"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	structSeries, err := polars.NewSeries(polars.NewSeriesInput{
		Name: "meta", DType: dtypes.Struct, Values: []any{map[string]any{"city": "kyiv"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	upper, err := strSeries.Str().Upper()
	if err != nil {
		t.Fatalf("Str.Upper: %v", err)
	}
	if v, _ := upper.Value(1).(string); v != "WORLD" {
		t.Fatalf("Upper: %v", upper.Value(1))
	}

	year, err := dtSeries.Dt().Year()
	if err != nil {
		t.Fatalf("Dt.Year: %v", err)
	}
	if v, _ := year.Value(0).(int64); v != 2024 {
		t.Fatalf("Year: %v", year.Value(0))
	}

	listLen, err := listSeries.Arr().ListLen()
	if err != nil {
		t.Fatalf("Arr.ListLen: %v", err)
	}
	if v, _ := listLen.Value(0).(int64); v != 3 {
		t.Fatalf("ListLen: %v", listLen.Value(0))
	}

	city, err := structSeries.Struct().Field("city")
	if err != nil {
		t.Fatalf("Struct.Field: %v", err)
	}
	if v, _ := city.Value(0).(string); v != "kyiv" {
		t.Fatalf("Field city: %v", city.Value(0))
	}
}

func TestLazyWithColumnsFilterCollect(t *testing.T) {
	t.Parallel()

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "value", Values: []any{int64(10), int64(25), int64(30), int64(5)}},
		},
	})
	if err != nil {
		t.Fatalf("створення DataFrame: %v", err)
	}

	out, err := df.Lazy().
		WithColumns(polars.Col("value").Mul(polars.Lit(int64(2))).Alias("doubled")).
		Filter(polars.Col("doubled").Ge(polars.Lit(int64(40)))).
		Select(polars.Col("id"), polars.Col("doubled")).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("lazy pipeline: %v", err)
	}
	if out.Height() != 2 || out.Width() != 2 {
		t.Fatalf("неочікувана форма: %d x %d", out.Height(), out.Width())
	}

	doubled, err := out.Item(0, "doubled")
	if err != nil {
		t.Fatalf("Item doubled: %v", err)
	}
	if doubled != int64(50) {
		t.Fatalf("doubled[0]: очікували 50, отримали %v", doubled)
	}
}

func TestSeriesRankAndQuantile(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name: "v", DType: dtypes.Float64, Values: []any{10.0, 20.0, 30.0, 40.0},
	})
	if err != nil {
		t.Fatal(err)
	}

	ranked := s.Rank()
	if ranked.Len() != 4 {
		t.Fatalf("Rank len: %d", ranked.Len())
	}
	if ranked.Value(0) != int64(1) || ranked.Value(3) != int64(4) {
		t.Fatalf("Rank values: %v", ranked.ToList())
	}

	q50 := s.Quantile(0.5)
	if q50 != 25.0 {
		t.Fatalf("Quantile(0.5): очікували 25.0, отримали %v", q50)
	}
}
