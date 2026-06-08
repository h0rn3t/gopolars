package lazyframe

// Ported from py-polars/tests/unit/lazyframe/test_predicates.py (py-1.28.1, representative subset)
//
// Predicate pushdown is an internal optimization; we verify it produces correct
// results (the observable contract), not that the rewrite occurred.

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestPredicateCorrectResult(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
		{Name: "b", Values: []any{int64(10), int64(20), int64(30), int64(40), int64(50)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Lazy().
		Filter(polars.Col("a").Ge(polars.Lit(int64(3)))).
		Filter(polars.Col("b").Le(polars.Lit(int64(40)))).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	// a in [3,4]
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2", out.Height())
	}
	a, _ := out.GetColumn("a")
	for i, w := range []int64{3, 4} {
		if v, _ := a.Value(i).(int64); v != w {
			t.Fatalf("a[%d]: got %v, want %d", i, a.Value(i), w)
		}
	}
}
