package operations

// Ported from py-polars/tests/unit/operations/test_join.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func joinLeft(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "a", Values: []any{"x", "y", "z"}},
	}})
	if err != nil {
		t.Fatalf("left df: %v", err)
	}
	return df
}

func joinRight(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(2), int64(3), int64(4)}},
		{Name: "b", Values: []any{"p", "q", "r"}},
	}})
	if err != nil {
		t.Fatalf("right df: %v", err)
	}
	return df
}

func TestJoinInner(t *testing.T) {
	t.Parallel()
	out, err := joinLeft(t).Join(polars.JoinInput{
		Other: joinRight(t), LeftOn: []string{"k"}, RightOn: []string{"k"}, How: polars.JoinTypeInner,
	})
	if err != nil {
		t.Fatalf("inner join: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("inner height: got %d, want 2 (k=2,3)", out.Height())
	}
}

func TestJoinLeft(t *testing.T) {
	t.Parallel()
	out, err := joinLeft(t).Join(polars.JoinInput{
		Other: joinRight(t), LeftOn: []string{"k"}, RightOn: []string{"k"}, How: polars.JoinTypeLeft,
	})
	if err != nil {
		t.Fatalf("left join: %v", err)
	}
	if out.Height() != 3 {
		t.Fatalf("left height: got %d, want 3 (all left rows)", out.Height())
	}
}

func TestJoinSemiAnti(t *testing.T) {
	t.Parallel()
	semi, err := joinLeft(t).Join(polars.JoinInput{
		Other: joinRight(t), LeftOn: []string{"k"}, RightOn: []string{"k"}, How: polars.JoinTypeSemi,
	})
	if err != nil {
		t.Fatalf("semi join: %v", err)
	}
	if semi.Height() != 2 {
		t.Fatalf("semi height: got %d, want 2 (k=2,3)", semi.Height())
	}
	anti, err := joinLeft(t).Join(polars.JoinInput{
		Other: joinRight(t), LeftOn: []string{"k"}, RightOn: []string{"k"}, How: polars.JoinTypeAnti,
	})
	if err != nil {
		t.Fatalf("anti join: %v", err)
	}
	if anti.Height() != 1 {
		t.Fatalf("anti height: got %d, want 1 (k=1)", anti.Height())
	}
}

func TestJoinCross(t *testing.T) {
	t.Parallel()
	out, err := joinLeft(t).Join(polars.JoinInput{Other: joinRight(t), How: polars.JoinTypeCross})
	if err != nil {
		t.Fatalf("cross join: %v", err)
	}
	if out.Height() != 9 {
		t.Fatalf("cross height: got %d, want 9 (3x3)", out.Height())
	}
}
