package frame

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

func covFrame(t *testing.T, cols ...SeriesInput) DataFrame {
	t.Helper()
	df, err := FromAnyColumns(FromAnyColumnsInput{Columns: cols})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	return df
}

// TestSortMixedAndNulls drives the sort comparator (compareSortValues / lessAny)
// over a column containing nulls, asserting null placement and order.
func TestSortMixedAndNulls(t *testing.T) {
	t.Parallel()

	df := covFrame(t, SeriesInput{Name: "v", Values: []any{int64(3), nil, int64(1), int64(2)}})

	asc, err := df.Sort(SortInput{By: []string{"v"}, Descending: []bool{false}})
	if err != nil {
		t.Fatalf("sort asc: %v", err)
	}
	col, _ := asc.Series("v")
	// Non-null values must appear in ascending order somewhere in the column.
	var seen []int64
	for i := 0; i < col.Len(); i++ {
		if v := col.Value(i); v != nil {
			seen = append(seen, v.(int64))
		}
	}
	if len(seen) != 3 || seen[0] != 1 || seen[1] != 2 || seen[2] != 3 {
		t.Fatalf("sorted non-null order = %v, want [1 2 3]", seen)
	}

	desc, err := df.Sort(SortInput{By: []string{"v"}, Descending: []bool{true}})
	if err != nil {
		t.Fatalf("sort desc: %v", err)
	}
	dcol, _ := desc.Series("v")
	var dseen []int64
	for i := 0; i < dcol.Len(); i++ {
		if v := dcol.Value(i); v != nil {
			dseen = append(dseen, v.(int64))
		}
	}
	if len(dseen) != 3 || dseen[0] != 3 || dseen[2] != 1 {
		t.Fatalf("desc non-null order = %v, want [3 2 1]", dseen)
	}
}

// TestShift covers DataFrame.Shift (evalShift) forward and backward.
func TestShift(t *testing.T) {
	t.Parallel()

	df := covFrame(t, SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3)}})

	fwd, err := df.Shift(1)
	if err != nil {
		t.Fatalf("shift +1: %v", err)
	}
	c, _ := fwd.Series("v")
	if c.Value(0) != nil || c.Value(1) != int64(1) || c.Value(2) != int64(2) {
		t.Fatalf("shift +1 = [%v %v %v], want [nil 1 2]", c.Value(0), c.Value(1), c.Value(2))
	}

	back, err := df.Shift(-1)
	if err != nil {
		t.Fatalf("shift -1: %v", err)
	}
	b, _ := back.Series("v")
	if b.Value(0) != int64(2) || b.Value(2) != nil {
		t.Fatalf("shift -1 = [%v .. %v], want [2 .. nil]", b.Value(0), b.Value(2))
	}
}

// TestGroupByMedian covers the medianOf aggregation path.
func TestGroupByMedian(t *testing.T) {
	t.Parallel()

	df := covFrame(t,
		SeriesInput{Name: "g", Values: []any{"a", "a", "a", "b"}},
		SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(10)}},
	)

	out, err := df.GroupBy("g").Agg(expr.Median(expr.Col("v")).Alias("med"))
	if err != nil {
		t.Fatalf("groupby median: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("median groups = %d, want 2", out.Height())
	}
	// Group "a" median of {1,2,3} = 2; group "b" median of {10} = 10.
	rows := out.ToDicts()
	for _, r := range rows {
		med, _ := toFloatAny(r["med"])
		switch r["g"] {
		case "a":
			if med != 2 {
				t.Errorf("group a median = %v, want 2", r["med"])
			}
		case "b":
			if med != 10 {
				t.Errorf("group b median = %v, want 10", r["med"])
			}
		}
	}
}

func toFloatAny(v any) (float64, bool) {
	switch t := v.(type) {
	case int64:
		return float64(t), true
	case float64:
		return t, true
	default:
		return 0, false
	}
}
