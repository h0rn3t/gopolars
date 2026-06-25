package conformance

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestTop30DataFrameAndExprConformanceV07(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "g", Values: []any{"a", "a", "b"}},
			{Name: "v", Values: []any{float64(1), float64(2), float64(3)}},
			{Name: "n", Values: []any{float64(1), nil, float64(3)}},
		},
	})
	_, err := df.Select(
		polars.Col("v").CumSum().Alias("cum"),
		polars.Col("v").Rank().Alias("rank"),
		polars.Col("v").CumSum().Over("g").Alias("over_cum"),
		polars.Col("n").FillNull(polars.Lit(float64(0))).Alias("fill_null"),
		polars.Col("v").RollingMean(2).Alias("roll_mean"),
	)
	if err != nil {
		t.Fatalf("top30 expr conformance failed: %v", err)
	}
	if _, err := df.NUnique("g"); err != nil {
		t.Fatalf("n_unique conformance failed: %v", err)
	}
	if df.DropNaNs("v").Height() == 0 {
		t.Fatalf("drop_nans conformance failed")
	}
}

func TestTop30LazyAndSeriesConformanceV07(t *testing.T) {
	ctx := context.Background()
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "v", Values: []any{int64(10), int64(20), int64(30)}},
			{Name: "ts", Values: []any{
				time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
			}},
		},
	})
	lf := df.Lazy().JoinWhere(polars.Col("v").Gt(polars.Lit(int64(10))))
	if (<-lf.CollectAsync(ctx)).Error != nil {
		t.Fatalf("collect_async conformance failed")
	}

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "s",
		DType:  polars.Float64,
		Values: []any{1.0, math.NaN(), nil, 4.0},
	})
	if err != nil {
		t.Fatalf("series create failed: %v", err)
	}
	if s.NullCount() != 1 || s.DropNans().Len() != 3 || len(s.ToList()) != 4 {
		t.Fatalf("series high-priority parity mismatch")
	}
}
