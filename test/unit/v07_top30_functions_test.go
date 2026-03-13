package unit

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestV07DataFrameTop30Utilities(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "v", Values: []any{1.0, 2.0, nil, 4.0}},
			{Name: "nanv", Values: []any{1.0, math.NaN(), 3.0, 4.0}},
		},
	})
	if df.IsEmpty() {
		t.Fatalf("expected non-empty dataframe")
	}
	if df.EstimatedSize() <= 0 {
		t.Fatalf("estimated size should be positive")
	}
	if _, err := df.NUnique("id"); err != nil {
		t.Fatalf("n_unique failed: %v", err)
	}
	nulls := df.NullCount()
	if nulls["v"] == 0 {
		t.Fatalf("expected null count for v")
	}
	dicts := df.ToDicts()
	if len(dicts) != df.Height() {
		t.Fatalf("to_dicts row mismatch")
	}
	withCount, err := df.WithRowCount("row_nr", 10)
	if err != nil || withCount.Height() != df.Height() {
		t.Fatalf("with_row_count failed")
	}
	withIndex, err := df.WithRowIndex("row_idx", 0)
	if err != nil || withIndex.Height() != df.Height() {
		t.Fatalf("with_row_index failed")
	}
	sample := df.Sample(2, 42)
	if sample.Height() != 2 {
		t.Fatalf("sample height mismatch")
	}
	dropNaN := df.DropNaNs("nanv")
	if dropNaN.Height() != 3 {
		t.Fatalf("drop_nans mismatch")
	}
	fillNaN, err := df.FillNaN(99.0)
	if err != nil {
		t.Fatalf("fill_nan failed: %v", err)
	}
	_ = fillNaN
}

func TestV07ExprTop30Methods(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "g", Values: []any{"a", "a", "b", "b"}},
			{Name: "v", Values: []any{float64(1), float64(2), float64(2), float64(4)}},
			{Name: "n", Values: []any{float64(1), math.NaN(), float64(3), float64(4)}},
		},
	})
	out, err := df.Select(
		polars.Col("v").CumSum().Alias("cum_sum"),
		polars.Col("v").CumCount().Alias("cum_count"),
		polars.Col("v").Rank().Alias("rank"),
		polars.Col("v").CumSum().Over("g").Alias("over_cum"),
		polars.Col("v").Replace(polars.Lit(float64(2)), polars.Lit(float64(20))).Alias("replace"),
		polars.Col("v").FillNull(polars.Lit(float64(0))).Alias("fill_null"),
		polars.Col("n").FillNaN(polars.Lit(float64(0))).Alias("fill_nan"),
		polars.Col("v").RollingMin(2).Alias("roll_min"),
		polars.Col("v").RollingMax(2).Alias("roll_max"),
		polars.Col("v").RollingMean(2).Alias("roll_mean"),
		polars.Col("v").RollingSum(2).Alias("roll_sum"),
		polars.Col("v").RollingStd(2).Alias("roll_std"),
	)
	if err != nil {
		t.Fatalf("expr top30 select failed: %v", err)
	}
	if out.Height() != df.Height() {
		t.Fatalf("expr output height mismatch")
	}
}

func TestV07LazyTop30MethodsAndSQLContext(t *testing.T) {
	ctx := context.Background()
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "v", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
			{Name: "ts", Values: []any{
				time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC),
			}},
		},
	})
	lf := df.Lazy().Inspect().JoinWhere(polars.Col("v").Gt(polars.Lit(int64(15))))
	asyncResult := <-lf.CollectAsync(ctx)
	if asyncResult.Error != nil {
		t.Fatalf("collect_async failed: %v", asyncResult.Error)
	}
	batchCount := 0
	for r := range lf.CollectBatches(ctx, 2) {
		if r.Error != nil {
			t.Fatalf("collect_batches failed: %v", r.Error)
		}
		batchCount++
	}
	if batchCount == 0 {
		t.Fatalf("expected batches")
	}
	profOut, profile, err := lf.Profile(ctx)
	if err != nil {
		t.Fatalf("profile failed: %v", err)
	}
	if profOut.Height() == 0 || profile["duration_ms"] == nil {
		t.Fatalf("invalid profile output")
	}
	sqlLF, err := lf.SQL(ctx, "SELECT id FROM t WHERE v > 20", "t")
	if err != nil {
		t.Fatalf("lazy sql failed: %v", err)
	}
	sqlOut, err := sqlLF.Collect(ctx)
	if err != nil || sqlOut.Height() == 0 {
		t.Fatalf("lazy sql collect failed")
	}
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "out.ndjson")
	if err := lf.SinkNDJSON(ctx, polars.WriteJSONInput{Path: path}); err != nil {
		t.Fatalf("sink_ndjson failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("ndjson file not written")
	}
	sqlCtx := polars.NewSQLContext()
	if err := sqlCtx.Register("t", df); err != nil {
		t.Fatalf("sqlcontext register failed: %v", err)
	}
	if len(sqlCtx.Tables()) != 1 {
		t.Fatalf("sqlcontext tables mismatch")
	}
	ctxLF, err := sqlCtx.Execute(ctx, "SELECT id FROM t WHERE v >= 30")
	if err != nil {
		t.Fatalf("sqlcontext execute failed: %v", err)
	}
	ctxOut, err := ctxLF.Collect(ctx)
	if err != nil || ctxOut.Height() != 2 {
		t.Fatalf("sqlcontext collect mismatch")
	}
	sqlCtx.Unregister("t")
	if len(sqlCtx.Tables()) != 0 {
		t.Fatalf("sqlcontext unregister failed")
	}
}
