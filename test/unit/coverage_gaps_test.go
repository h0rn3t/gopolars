package unit

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestCoverageGapsPublicAPI(t *testing.T) {
	t.Parallel()

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "g", Values: []any{"a", "a", "b"}},
			{Name: "v", Values: []any{int64(3), int64(7), int64(1)}},
		},
	})
	if err != nil {
		t.Fatalf("dataframe: %v", err)
	}

	stacked, err := df.Vstack(df)
	if err != nil || stacked.Height() != 6 {
		t.Fatalf("vstack: err=%v height=%d", err, stacked.Height())
	}

	agg, err := df.GroupBy("g").Agg(
		polars.Min(polars.Col("v")),
		polars.Max(polars.Col("v")),
		polars.Count(),
	)
	if err != nil || agg.Height() != 2 {
		t.Fatalf("min/max/count agg: err=%v height=%d", err, agg.Height())
	}

	ctx := context.Background()
	io := polars.NewIO()
	lf, err := io.ScanJSON(polars.ScanJSONInput{Path: "missing.json"})
	if err != nil {
		t.Fatalf("scan json: %v", err)
	}
	lf = df.Lazy().
		Head(2).
		Tail(1).
		First().
		Last().
		Slice(0, 1).
		GatherEvery(2, 0).
		Reverse().
		Rename(map[string]string{"g": "grp"}).
		FillNull(int64(0)).
		DropNulls("v").
		DropNaNs("v")
	if _, err := io.SQL(ctx, "SELECT v FROM df"); err != nil {
		t.Fatalf("sql lazy: %v", err)
	}
	if _, err := lf.Collect(ctx); err != nil {
		t.Fatalf("lazy collect: %v", err)
	}
}
