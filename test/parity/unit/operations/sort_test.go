package operations

// Ported from py-polars/tests/unit/operations/test_sort.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func sortDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(3), int64(1), int64(2)}},
		{Name: "b", Values: []any{"z", "x", "y"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

func TestSortAscending(t *testing.T) {
	t.Parallel()
	out, err := sortDF(t).Sort(polars.SortInput{By: []string{"a"}, Descending: []bool{false}})
	if err != nil {
		t.Fatalf("sort asc: %v", err)
	}
	a, _ := out.GetColumn("a")
	for i, w := range []int64{1, 2, 3} {
		if v, _ := a.Value(i).(int64); v != w {
			t.Fatalf("asc[%d]: got %v, want %d", i, a.Value(i), w)
		}
	}
}

func TestSortDescending(t *testing.T) {
	t.Parallel()
	out, err := sortDF(t).Sort(polars.SortInput{By: []string{"a"}, Descending: []bool{true}})
	if err != nil {
		t.Fatalf("sort desc: %v", err)
	}
	a, _ := out.GetColumn("a")
	for i, w := range []int64{3, 2, 1} {
		if v, _ := a.Value(i).(int64); v != w {
			t.Fatalf("desc[%d]: got %v, want %d", i, a.Value(i), w)
		}
	}
}

// Sorting carries the other columns along.
func TestSortKeepsRows(t *testing.T) {
	t.Parallel()
	out, err := sortDF(t).Sort(polars.SortInput{By: []string{"a"}, Descending: []bool{false}})
	if err != nil {
		t.Fatalf("sort: %v", err)
	}
	b, _ := out.GetColumn("b")
	for i, w := range []string{"x", "y", "z"} {
		if v, _ := b.Value(i).(string); v != w {
			t.Fatalf("b[%d]: got %v, want %s", i, b.Value(i), w)
		}
	}
}

// Series-level sort with nulls: Polars' default (nulls_last=False) places nulls
// first for BOTH ascending and descending.
func TestSortSeriesWithNulls(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "x", DType: polars.Int64, Values: []any{int64(3), nil, int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	asc := s.Sort(false)
	if asc.Len() != 4 || asc.Value(0) != nil {
		t.Fatalf("asc[0]: got %v, want nil (nulls first)", asc.Value(0))
	}
	for i, w := range []int64{1, 2, 3} {
		if v, _ := asc.Value(i + 1).(int64); v != w {
			t.Fatalf("asc[%d]: got %v, want %d", i+1, asc.Value(i+1), w)
		}
	}
	// Descending: nulls still first, then values descending.
	desc := s.Sort(true)
	if desc.Value(0) != nil {
		t.Fatalf("desc[0]: got %v, want nil (nulls first regardless of direction)", desc.Value(0))
	}
	for i, w := range []int64{3, 2, 1} {
		if v, _ := desc.Value(i + 1).(int64); v != w {
			t.Fatalf("desc[%d]: got %v, want %d", i+1, desc.Value(i+1), w)
		}
	}
}
