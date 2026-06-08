package operations

// Ported from py-polars/tests/unit/operations/test_filter.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func filterDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
		{Name: "b", Values: []any{"a", "b", "c", "d", "e"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

func TestFilterGt(t *testing.T) {
	t.Parallel()
	out, err := filterDF(t).Filter(polars.Col("a").Gt(polars.Lit(int64(2))))
	if err != nil {
		t.Fatalf("filter gt: %v", err)
	}
	if out.Height() != 3 {
		t.Fatalf("height: got %d, want 3 (a=3,4,5)", out.Height())
	}
	a, _ := out.GetColumn("a")
	if v, _ := a.Value(0).(int64); v != 3 {
		t.Fatalf("first: got %v, want 3", a.Value(0))
	}
}

func TestFilterEq(t *testing.T) {
	t.Parallel()
	out, err := filterDF(t).Filter(polars.Col("b").Eq(polars.Lit("c")))
	if err != nil {
		t.Fatalf("filter eq: %v", err)
	}
	if out.Height() != 1 {
		t.Fatalf("height: got %d, want 1", out.Height())
	}
	a, _ := out.GetColumn("a")
	if v, _ := a.Value(0).(int64); v != 3 {
		t.Fatalf("a: got %v, want 3", a.Value(0))
	}
}

func TestFilterAndCondition(t *testing.T) {
	t.Parallel()
	pred := polars.Col("a").Gt(polars.Lit(int64(1))).And(polars.Col("a").Lt(polars.Lit(int64(5))))
	out, err := filterDF(t).Filter(pred)
	if err != nil {
		t.Fatalf("filter and: %v", err)
	}
	if out.Height() != 3 {
		t.Fatalf("height: got %d, want 3 (a=2,3,4)", out.Height())
	}
}

func TestFilterEmptyResult(t *testing.T) {
	t.Parallel()
	out, err := filterDF(t).Filter(polars.Col("a").Gt(polars.Lit(int64(100))))
	if err != nil {
		t.Fatalf("filter empty: %v", err)
	}
	if out.Height() != 0 {
		t.Fatalf("height: got %d, want 0", out.Height())
	}
}
