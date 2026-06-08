package lazyframe

// Ported from py-polars/tests/unit/lazyframe/test_explain.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// explain returns a non-empty textual query plan.
func TestExplainNonEmpty(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	lf := df.Lazy().Filter(polars.Col("a").Gt(polars.Lit(int64(1)))).Select(polars.Col("a"))
	if plan := lf.Explain(false); plan == "" {
		t.Fatal("explain(unoptimized) returned empty plan")
	}
	if plan := lf.Explain(true); plan == "" {
		t.Fatal("explain(optimized) returned empty plan")
	}
}
