package unit

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
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

func TestV07LazyTop30Methods(t *testing.T) {
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
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "out.ndjson")
	if err := lf.SinkNDJSON(ctx, polars.WriteJSONInput{Path: path}); err != nil {
		t.Fatalf("sink_ndjson failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("ndjson file not written")
	}
}

func TestV09SeriesHighPriorityMethods(t *testing.T) {
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "s",
		DType:  polars.Float64,
		Values: []any{1.0, math.NaN(), nil, 4.0},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.NullCount() != 1 {
		t.Fatalf("expected null_count=1")
	}
	filled, err := s.FillNan(0.0)
	if err != nil {
		t.Fatalf("fill_nan failed: %v", err)
	}
	if v := filled.Value(1); v == nil || v.(float64) != 0.0 {
		t.Fatalf("fill_nan result mismatch")
	}
	dropped := s.DropNans()
	if dropped.Len() != 3 {
		t.Fatalf("drop_nans result mismatch")
	}
	if len(s.ToList()) != s.Len() {
		t.Fatalf("to_list mismatch")
	}
	if s.RollingMean(2).Len() != s.Len() {
		t.Fatalf("rolling_mean len mismatch")
	}
	if s.RollingSum(2).Len() != s.Len() {
		t.Fatalf("rolling_sum len mismatch")
	}
	if s.RollingMin(2).Len() != s.Len() {
		t.Fatalf("rolling_min len mismatch")
	}
	if s.RollingMax(2).Len() != s.Len() {
		t.Fatalf("rolling_max len mismatch")
	}
}
