package functions

// Ported from py-polars/tests/unit/functions/test_nth.py (py-1.28.1)
//
// gopolars has no top-level pl.nth(index...) expression that selects the Nth
// column by position. We port the intent using column-by-index selection via
// Columns() + SubSelectColumns, and document the gap.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func nthDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
			{Name: "b", Values: []any{int64(3), int64(4)}},
			{Name: "c", Values: []any{int64(5), int64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	return df
}

// nthColumnName resolves a (possibly negative) column index like pl.nth.
func nthColumnName(cols []string, idx int) string {
	if idx < 0 {
		idx += len(cols)
	}
	return cols[idx]
}

// test_nth: pl.nth(0) -> "a", pl.nth(-1) -> "c"
func TestNthSingle(t *testing.T) {
	t.Parallel()
	df := nthDF(t)
	cols := df.Columns()

	for _, tc := range []struct {
		idx  int
		want string
	}{
		{0, "a"},
		{-1, "c"},
	} {
		out, err := df.SubSelectColumns(nthColumnName(cols, tc.idx))
		if err != nil {
			t.Fatalf("nth(%d): %v", tc.idx, err)
		}
		if got := out.Columns(); len(got) != 1 || got[0] != tc.want {
			t.Fatalf("nth(%d): got %v, want [%s]", tc.idx, got, tc.want)
		}
	}
}

// test_nth: pl.nth(2, 1) -> ["c", "b"]; pl.nth([2, -2, 0]) -> ["c", "b", "a"]
func TestNthMultiple(t *testing.T) {
	t.Parallel()
	df := nthDF(t)
	cols := df.Columns()

	for _, tc := range []struct {
		idxs []int
		want []string
	}{
		{[]int{2, 1}, []string{"c", "b"}},
		{[]int{2, -2, 0}, []string{"c", "b", "a"}},
	} {
		names := make([]string, len(tc.idxs))
		for i, idx := range tc.idxs {
			names[i] = nthColumnName(cols, idx)
		}
		out, err := df.SubSelectColumns(names...)
		if err != nil {
			t.Fatalf("nth(%v): %v", tc.idxs, err)
		}
		got := out.Columns()
		if len(got) != len(tc.want) {
			t.Fatalf("nth(%v): got %v, want %v", tc.idxs, got, tc.want)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Fatalf("nth(%v): got %v, want %v", tc.idxs, got, tc.want)
			}
		}
	}
}
