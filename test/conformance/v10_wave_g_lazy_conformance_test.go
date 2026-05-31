package conformance

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestV10WaveGLazy(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(1), float64(2), float64(3)}},
			{Name: "y", Values: []any{"a", "b", "c"}},
			{Name: "ts", Values: []any{
				time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
			}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}
	lf := df.Lazy()

	if got := lf.Columns(); len(got) != 3 || got[0] != "x" {
		t.Fatalf("unexpected columns: got %v", got)
	}
	if got := lf.Width(); got != 3 {
		t.Fatalf("unexpected width: got %d want 3", got)
	}
	if len(lf.CollectSchema()) == 0 || len(lf.Schema()) == 0 {
		t.Fatalf("expected non-empty schema")
	}
	if len(lf.Dtypes()) != 3 {
		t.Fatalf("unexpected dtypes len: got %d want 3", len(lf.Dtypes()))
	}
	describe, err := lf.Describe()
	if err != nil {
		t.Fatalf("describe failed: %v", err)
	}
	if describe.Height() == 0 || describe.Width() == 0 {
		t.Fatalf("unexpected describe shape: %v", describe.Shape())
	}

	selected, err := lf.SelectSeq(polars.Col("x")).Collect(context.Background())
	if err != nil {
		t.Fatalf("select_seq collect failed: %v", err)
	}
	if selected.Width() != 1 {
		t.Fatalf("unexpected select_seq width: got %d want 1", selected.Width())
	}

	withCols, err := lf.WithColumnsSeq(polars.Col("x").Alias("x_copy")).Collect(context.Background())
	if err != nil {
		t.Fatalf("with_columns_seq collect failed: %v", err)
	}
	if _, err := withCols.GetColumn("x_copy"); err != nil {
		t.Fatalf("expected x_copy column: %v", err)
	}

	topK, err := lf.TopK(2, "x").Collect(context.Background())
	if err != nil {
		t.Fatalf("top_k collect failed: %v", err)
	}
	topKCol, err := topK.GetColumn("x")
	if err != nil {
		t.Fatalf("get top_k column failed: %v", err)
	}
	if topKCol.Value(0).(float64) != 3 || topKCol.Value(1).(float64) != 2 {
		t.Fatalf("unexpected top_k values: got [%v %v] want [3 2]", topKCol.Value(0), topKCol.Value(1))
	}

	removed, err := lf.Remove("y").Collect(context.Background())
	if err != nil {
		t.Fatalf("remove collect failed: %v", err)
	}
	if removed.Width() != 2 {
		t.Fatalf("unexpected remove width: got %d want 2", removed.Width())
	}

	other, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(1), float64(2), float64(3)}},
			{Name: "z", Values: []any{"u", "v", "w"}},
		},
	})
	if err != nil {
		t.Fatalf("new other dataframe failed: %v", err)
	}
	joined, err := lf.JoinAsof(polars.JoinInput{
		Other:   other,
		LeftOn:  []string{"x"},
		RightOn: []string{"x"},
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("join_asof collect failed: %v", err)
	}
	if joined.Width() < df.Width() {
		t.Fatalf("unexpected join_asof width: got %d want >= %d", joined.Width(), df.Width())
	}

	mapped, err := lf.MapBatches(func(batch polars.DataFrame) (polars.DataFrame, error) {
		return batch.Select(polars.Col("x"))
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("map_batches collect failed: %v", err)
	}
	if mapped.Width() != 1 {
		t.Fatalf("unexpected map_batches width: got %d want 1", mapped.Width())
	}

	mergeOther, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(4), float64(5), float64(6)}},
			{Name: "y", Values: []any{"d", "e", "f"}},
			{Name: "ts", Values: []any{
				time.Date(2026, 4, 1, 13, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 1, 14, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 1, 15, 0, 0, 0, time.UTC),
			}},
		},
	})
	if err != nil {
		t.Fatalf("new merge dataframe failed: %v", err)
	}
	merged, err := lf.MergeSorted(mergeOther.Lazy(), "x").Collect(context.Background())
	if err != nil {
		t.Fatalf("merge_sorted collect failed: %v", err)
	}
	if merged.Height() != df.Height()+mergeOther.Height() {
		t.Fatalf("unexpected merge_sorted height: got %d want %d", merged.Height(), df.Height()+mergeOther.Height())
	}

	piped, err := lf.Pipe(func(in polars.LazyFrame) polars.LazyFrame {
		return in.Select(polars.Col("x"))
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("pipe collect failed: %v", err)
	}
	if piped.Width() != 1 {
		t.Fatalf("unexpected pipe width: got %d want 1", piped.Width())
	}

	pipeWithSchema, err := lf.PipeWithSchema(func(in polars.LazyFrame, schema dtypes.Schema) polars.LazyFrame {
		if len(schema) == 0 {
			t.Fatalf("expected non-empty schema in pipe_with_schema")
		}
		return in.Limit(1)
	}, df.Schema()).Collect(context.Background())
	if err != nil {
		t.Fatalf("pipe_with_schema collect failed: %v", err)
	}
	if pipeWithSchema.Height() != 1 {
		t.Fatalf("unexpected pipe_with_schema height: got %d want 1", pipeWithSchema.Height())
	}

	rolling, err := lf.Rolling(polars.RollingMeanInput{
		By:     "ts",
		Value:  "x",
		Window: time.Hour,
		Output: "rx",
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("rolling collect failed: %v", err)
	}
	if _, err := rolling.GetColumn("rx"); err != nil {
		t.Fatalf("expected rolling output column: %v", err)
	}

	withContext, err := lf.WithContext(other.Lazy()).Collect(context.Background())
	if err != nil {
		t.Fatalf("with_context collect failed: %v", err)
	}
	if withContext.Height() != df.Height() {
		t.Fatalf("unexpected with_context height: got %d want %d", withContext.Height(), df.Height())
	}

	if show := lf.Show(2); show == "" {
		t.Fatalf("expected non-empty show output")
	}
	if cloned, err := lf.Lazy().Collect(context.Background()); err != nil || cloned.Height() != df.Height() {
		t.Fatalf("lazy clone collect failed: err=%v height=%d want=%d", err, cloned.Height(), df.Height())
	}

	if err := lf.SinkDelta(context.Background(), "/tmp/gopolars-v10-delta"); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected sink_delta not supported error, got %v", err)
	}
	if err := lf.SinkIceberg(context.Background(), "/tmp/gopolars-v10-iceberg"); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected sink_iceberg not supported error, got %v", err)
	}
}
