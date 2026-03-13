package unit

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestV09WaveBDataFrameSeriesLazyExprSQL(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "g", Values: []any{"a", "a", "b", "b"}},
			{Name: "v", Values: []any{float64(1), float64(2), float64(3), float64(4)}},
			{Name: "ts", Values: []any{
				time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC),
			}},
		},
	})
	if err != nil {
		t.Fatalf("new df failed: %v", err)
	}

	if df.Rechunk().Height() != df.Height() {
		t.Fatalf("rechunk mismatch")
	}
	if len(df.ToNumpy()) != df.Height() || len(df.ToPandas()) != df.Height() {
		t.Fatalf("to_numpy/to_pandas mismatch")
	}
	parts, err := df.PartitionBy("g")
	if err != nil || len(parts) != 2 {
		t.Fatalf("partition_by failed")
	}
	up, err := df.Upsample("ts", time.Hour)
	if err != nil || up.Height() != df.Height() {
		t.Fatalf("upsample failed")
	}

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "s",
		DType:  polars.Float64,
		Values: []any{1.0, nil, 3.0, math.NaN()},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.Describe()["len"] == nil {
		t.Fatalf("describe missing fields")
	}
	if _, err := s.Hist(3); err != nil {
		t.Fatalf("hist failed: %v", err)
	}
	if s.Interpolate().Len() != s.Len() {
		t.Fatalf("interpolate failed")
	}
	if len(s.ToNumpy()) != s.Len() || len(s.ToPandas()) != s.Len() {
		t.Fatalf("series to_numpy/to_pandas mismatch")
	}

	out, err := df.Select(
		polars.Col("v").Clip(polars.Lit(float64(1.5)), polars.Lit(float64(3.5))).Alias("clip"),
		polars.Col("v").Round().Alias("round"),
		polars.Col("v").Pow(polars.Lit(float64(2))).Alias("pow"),
		polars.Col("v").Log().Alias("log"),
		polars.Col("v").Sqrt().Alias("sqrt"),
		polars.Col("ts").Dt().Alias("dt"),
		polars.Col("g").Str().Alias("str"),
		polars.Col("v").List().Alias("list"),
		polars.Col("v").Struct().Alias("struct"),
	)
	if err != nil || out.Height() != df.Height() {
		t.Fatalf("expr medium methods failed: %v", err)
	}

	ctx := context.Background()
	lf := df.Lazy().Cache().Remote("local")
	if lf.ShowGraph() == "" {
		t.Fatalf("show_graph failed")
	}
	payload, err := lf.Serialize()
	if err != nil || len(payload) == 0 {
		t.Fatalf("serialize failed: %v", err)
	}
	lf2, err := lf.Deserialize(payload)
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if (<-lf2.SinkBatches(ctx, 2)).Error != nil {
		t.Fatalf("sink_batches failed")
	}

	sqlCtx := polars.NewSQLContext()
	if err := sqlCtx.RegisterGlobals(map[string]polars.DataFrame{"t": df}); err != nil {
		t.Fatalf("register_globals failed: %v", err)
	}
	lf3, err := sqlCtx.ExecuteGlobal(ctx, "SELECT g FROM t WHERE g = 'a'")
	if err != nil {
		t.Fatalf("execute_global failed: %v", err)
	}
	out3, err := lf3.Collect(ctx)
	if err != nil || out3.Height() != 2 {
		t.Fatalf("execute_global output mismatch")
	}
}
