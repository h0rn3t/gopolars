package operations

// Ported from py-polars/tests/unit/operations/test_inequality_join.py (py-1.28.1, representative subset)
//
// gopolars exposes inequality/predicate joins via DataFrame.JoinWhere(predicate).

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// JoinWhere keeps the rows satisfying the predicate.
func TestJoinWherePredicate(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "x", Values: []any{int64(1), int64(3), int64(5), int64(7)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.JoinWhere(polars.Col("x").Gt(polars.Lit(int64(3))))
	if err != nil {
		t.Fatalf("join_where: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2 (x=5,7)", out.Height())
	}
	x, _ := out.GetColumn("x")
	if v, _ := x.Value(0).(int64); v != 5 {
		t.Fatalf("first: got %v, want 5", x.Value(0))
	}
}
