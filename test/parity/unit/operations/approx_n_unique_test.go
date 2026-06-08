package operations

// Ported from py-polars/tests/unit/operations/unique/test_approx_n_unique.py (py-1.28.1).
//
// DISCREPANCY: Python's (deprecated) `DataFrame.approx_n_unique()` returns a
// one-row DataFrame of per-COLUMN approximate distinct counts (a=2, b=1 for the
// data below). gopolars `DataFrame.ApproxNUnique(cols...)` is an alias of
// NUnique and returns a single int = the number of unique ROWS. This test
// asserts gopolars' actual behavior.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestApproxNUniqueDataFrameRows(t *testing.T) {
	t.Parallel()
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(2)}},
		{Name: "b", Values: []any{int64(2), int64(2), int64(2)}},
	}})
	// Unique rows of (a,b): (1,2),(2,2),(2,2) -> 2.
	if got, err := df.ApproxNUnique(); err != nil || got != 2 {
		t.Fatalf("df.approx_n_unique(): got %d err %v, want 2 (Python returns per-col a=2,b=1)", got, err)
	}
	// Per-column counts are available via the subset form.
	if got, _ := df.ApproxNUnique("a"); got != 2 {
		t.Fatalf("df.approx_n_unique([a]): got %d, want 2", got)
	}
	if got, _ := df.ApproxNUnique("b"); got != 1 {
		t.Fatalf("df.approx_n_unique([b]): got %d, want 1", got)
	}
}
