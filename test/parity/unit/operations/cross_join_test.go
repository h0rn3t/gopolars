package operations

// Ported from py-polars/tests/unit/operations/test_cross_join.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestCrossJoinShape(t *testing.T) {
	t.Parallel()
	a, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "x", Values: []any{int64(1), int64(2)}},
	}})
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "y", Values: []any{int64(10), int64(20), int64(30)}},
	}})
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	out, err := a.Join(polars.JoinInput{Other: b, How: polars.JoinTypeCross})
	if err != nil {
		t.Fatalf("cross join: %v", err)
	}
	if out.Height() != 6 {
		t.Fatalf("cross height: got %d, want 6 (2x3)", out.Height())
	}
	if out.Width() != 2 {
		t.Fatalf("cross width: got %d, want 2", out.Width())
	}
}
