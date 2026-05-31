package unit

import (
	"context"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestV09WaveCStructuralDataFrameAndLazy(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "k", Values: []any{"a", "a", "b", "b"}},
			{Name: "x", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "y", Values: []any{float64(1), float64(2), float64(3), float64(4)}},
			{Name: "ts", Values: []any{
				time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC),
			}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}
	if len(df.IterColumns()) != df.Width() || len(df.IterRows()) != df.Height() {
		t.Fatalf("iter methods mismatch")
	}
	if len(df.IterSlices(2)) != 2 {
		t.Fatalf("iter_slices mismatch")
	}
	if _, err := df.Item(1, "x"); err != nil {
		t.Fatalf("item failed: %v", err)
	}
	if df.IsDuplicated().Len() != df.Height() || df.IsUnique().Len() != df.Height() {
		t.Fatalf("duplicated/unique mismatch")
	}
	if _, err := df.JoinAsof(polars.JoinInput{Other: df, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: "asof"}); err != nil {
		t.Fatalf("join_asof failed: %v", err)
	}
	if _, err := df.MatchToSchema(dtypes.Schema{{Name: "z", Type: dtypes.Int64}}); err != nil {
		t.Fatalf("match_to_schema failed: %v", err)
	}
	if _, err := df.MergeSorted(df, "ts"); err != nil {
		t.Fatalf("merge_sorted failed: %v", err)
	}
	if df.NChunks() <= 0 {
		t.Fatalf("n_chunks mismatch")
	}
	if _, err := df.Pipe(func(in polars.DataFrame) (polars.DataFrame, error) { return in.Limit(2), nil }); err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	if df.Plot("ts", "x")["type"] == nil {
		t.Fatalf("plot metadata missing")
	}
	if len(df.Max()) == 0 || len(df.Min()) == 0 || len(df.Mean()) == 0 || len(df.Median()) == 0 || len(df.Product()) == 0 || len(df.Quantile(0.5)) == 0 {
		t.Fatalf("aggregate maps mismatch")
	}
	if _, err := df.MaxHorizontal("mx"); err != nil {
		t.Fatalf("max_horizontal failed: %v", err)
	}
	if _, err := df.MinHorizontal("mn"); err != nil {
		t.Fatalf("min_horizontal failed: %v", err)
	}
	if _, err := df.MeanHorizontal("avg"); err != nil {
		t.Fatalf("mean_horizontal failed: %v", err)
	}
	if _, err := df.Remove("y"); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	ctx := context.Background()
	lf := df.Lazy().ApproxNUnique("k").BottomK(2, "k")
	out, err := lf.Collect(ctx)
	if err != nil || out.Height() == 0 {
		t.Fatalf("lazy structural methods failed: %v", err)
	}
}
