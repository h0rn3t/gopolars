package operations

// Ported from py-polars/tests/unit/operations/namespaces/list/test_list.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func listDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "l", Values: []any{[]any{int64(1), int64(2), int64(3)}, []any{int64(4)}, []any{int64(5), int64(6)}}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// list.len returns the length of each sub-list.
func TestListLen(t *testing.T) {
	t.Parallel()
	out, err := listDF(t).Select(polars.Col("l").ListLen().Alias("n"))
	if err != nil {
		t.Fatalf("list.len failed: %v", err)
	}
	s, _ := out.GetColumn("n")
	for i, w := range []int64{3, 1, 2} {
		if toFloatAny(s.Value(i)) != float64(w) {
			t.Fatalf("list.len[%d]: got %v, want %d", i, s.Value(i), w)
		}
	}
}

// list.get(0) returns the first element of each sub-list.
func TestListGet(t *testing.T) {
	t.Parallel()
	out, err := listDF(t).Select(polars.Col("l").ListGet(polars.Lit(int64(0))).Alias("first"))
	if err != nil {
		t.Fatalf("list.get failed: %v", err)
	}
	s, _ := out.GetColumn("first")
	for i, w := range []int64{1, 4, 5} {
		if toFloatAny(s.Value(i)) != float64(w) {
			t.Fatalf("list.get[%d]: got %v, want %d", i, s.Value(i), w)
		}
	}
}
