package operations

// Ported from py-polars null-comparison semantics exercised across
// operations/test_filter.py and datatypes null handling (py-1.28.1).
//
// Polars: a comparison against a null cell yields null (not true/false), boolean
// combinators use three-valued (Kleene) logic, and filter keeps only rows whose
// predicate is True (null and False both drop).

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func nullCmpDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), nil, int64(4)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// filter(a != 2) drops the null row (null predicate is not True).
func TestFilterNeDropsNull(t *testing.T) {
	t.Parallel()
	out, err := nullCmpDF(t).Filter(polars.Col("a").Ne(polars.Lit(int64(2))))
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	// a=[1,2,null,4]; a!=2 -> [T, F, null, T]; survivors: 1 and 4.
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2", out.Height())
	}
	col, _ := out.GetColumn("a")
	for i, w := range []int64{1, 4} {
		if v, _ := col.Value(i).(int64); v != w {
			t.Fatalf("a[%d]: got %v, want %d", i, col.Value(i), w)
		}
	}
}

// filter(a == 2) also drops the null row and keeps only the match.
func TestFilterEqDropsNull(t *testing.T) {
	t.Parallel()
	out, err := nullCmpDF(t).Filter(polars.Col("a").Eq(polars.Lit(int64(2))))
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if out.Height() != 1 {
		t.Fatalf("height: got %d, want 1", out.Height())
	}
}

// select(a > 1) produces a Boolean column whose null row stays null.
func TestSelectCompareKeepsNull(t *testing.T) {
	t.Parallel()
	out, err := nullCmpDF(t).Select(polars.Col("a").Gt(polars.Lit(int64(1))).Alias("gt"))
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	col, _ := out.GetColumn("gt")
	// [1>1, 2>1, null>1, 4>1] -> [false, true, null, true]
	want := []any{false, true, nil, true}
	for i, w := range want {
		if col.Value(i) != w {
			t.Fatalf("gt[%d]: got %v, want %v", i, col.Value(i), w)
		}
	}
}

// Kleene AND inside a filter: (a > 1) AND (a < 4). The null row is null in both
// operands -> null -> dropped; only a=2,3.. that satisfy both survive (here a=2).
func TestFilterKleeneAnd(t *testing.T) {
	t.Parallel()
	out, err := nullCmpDF(t).Filter(
		polars.Col("a").Gt(polars.Lit(int64(1))).And(polars.Col("a").Lt(polars.Lit(int64(4)))),
	)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	// a=[1,2,null,4]: (>1 & <4) -> [F, T, null, F]; survivor: 2.
	if out.Height() != 1 {
		t.Fatalf("height: got %d, want 1", out.Height())
	}
	col, _ := out.GetColumn("a")
	if v, _ := col.Value(0).(int64); v != 2 {
		t.Fatalf("a[0]: got %v, want 2", col.Value(0))
	}
}
