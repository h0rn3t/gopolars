package lazyframe

// Ported from py-polars/tests/unit/lazyframe/test_async.py and test_collect_all.py
// (py-1.28.1, representative subsets).

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func asyncDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// test_collect_all (analogue): collecting the same plan twice is deterministic.
func TestCollectTwice(t *testing.T) {
	t.Parallel()
	lf := asyncDF(t).Lazy().Filter(polars.Col("a").Gt(polars.Lit(int64(1))))
	out1, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect 1: %v", err)
	}
	out2, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect 2: %v", err)
	}
	if out1.Height() != out2.Height() || out1.Height() != 2 {
		t.Fatalf("collect heights: %d vs %d, want 2", out1.Height(), out2.Height())
	}
}

// test_async: CollectAsync delivers the same result over its channel.
func TestCollectAsync(t *testing.T) {
	t.Parallel()
	lf := asyncDF(t).Lazy().Select(polars.Col("a"))
	ch := lf.CollectAsync(context.Background())
	res := <-ch
	if res.Error != nil {
		t.Fatalf("async collect: %v", res.Error)
	}
	if res.DataFrame.Height() != 3 {
		t.Fatalf("async height: got %d, want 3", res.DataFrame.Height())
	}
}
