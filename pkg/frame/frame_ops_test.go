package frame

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

func TestDataFrameTailReverseRename(t *testing.T) {
	t.Parallel()

	df, err := FromAnyColumns(FromAnyColumnsInput{
		Columns: []SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "city", Values: []any{"a", "b", "c"}},
		},
	})
	if err != nil {
		t.Fatalf("from columns: %v", err)
	}

	tail := df.Tail(2)
	idTail, _ := tail.Series("id")
	if tail.Height() != 2 || idTail.Value(0) != int64(2) {
		t.Fatalf("tail: height=%d first=%v", tail.Height(), idTail.Value(0))
	}

	rev := df.Reverse()
	cityRev, _ := rev.Series("city")
	if cityRev.Value(0) != "c" {
		t.Fatalf("reverse: %v", cityRev.Value(0))
	}

	renamed, err := df.Rename(map[string]string{"city": "location"})
	if err != nil || renamed.Columns()[1] != "location" {
		t.Fatalf("rename: err=%v cols=%v", err, renamed.Columns())
	}
}

func TestConcatHorizontalAndUpdate(t *testing.T) {
	t.Parallel()

	left, err := FromAnyColumns(FromAnyColumnsInput{
		Columns: []SeriesInput{{Name: "a", Values: []any{int64(1), int64(2)}}},
	})
	if err != nil {
		t.Fatalf("left: %v", err)
	}
	right, err := FromAnyColumns(FromAnyColumnsInput{
		Columns: []SeriesInput{{Name: "a", Values: []any{int64(10), int64(20)}}},
	})
	if err != nil {
		t.Fatalf("right: %v", err)
	}

	joined, err := ConcatHorizontal(left, right)
	if err != nil {
		t.Fatalf("concat horizontal: %v", err)
	}
	if joined.Width() != 2 {
		t.Fatalf("очікували 2 колонки після горизонтального concat, отримали %d", joined.Width())
	}

	patch, err := FromAnyColumns(FromAnyColumnsInput{
		Columns: []SeriesInput{{Name: "a", Values: []any{int64(99), int64(88)}}},
	})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	updated, err := left.Update(patch)
	aCol, _ := updated.Series("a")
	if err != nil || aCol.Value(0) != int64(99) {
		t.Fatalf("update: err=%v val=%v", err, aCol.Value(0))
	}
}

func TestShiftWithRowIndexSetSortedUnnestUnpivot(t *testing.T) {
	t.Parallel()

	df, err := FromAnyColumns(FromAnyColumnsInput{
		Columns: []SeriesInput{
			{Name: "k", Values: []any{"x", "y"}},
			{Name: "v", Values: []any{int64(1), int64(2)}},
			{Name: "meta", Values: []any{
				map[string]any{"p": int64(10)},
				map[string]any{"p": int64(20)},
			}},
		},
	})
	if err != nil {
		t.Fatalf("from columns: %v", err)
	}

	withIdx, err := df.WithRowIndex("idx", 100)
	if err != nil || withIdx.Height() != 2 {
		t.Fatalf("with row index: %v height=%d", err, withIdx.Height())
	}

	shifted, err := df.Shift(1)
	vShift, _ := shifted.Series("v")
	if err != nil || !vShift.IsNull(0) {
		t.Fatalf("shift frame: err=%v null0=%v", err, vShift.IsNull(0))
	}

	sorted, err := df.SetSorted("v")
	if err != nil || sorted.Height() != 2 {
		t.Fatalf("set sorted: %v", err)
	}

	flat, err := df.Unnest("meta")
	if err != nil || flat.Width() < 2 {
		t.Fatalf("unnest: err=%v width=%d", err, flat.Width())
	}

	melted, err := df.Unpivot([]string{"k"}, []string{"v"}, "var", "val")
	if err != nil || melted.Height() != 2 {
		t.Fatalf("unpivot: err=%v height=%d", err, melted.Height())
	}
}

func TestGroupByMinMaxUsesExtreme(t *testing.T) {
	t.Parallel()

	df, err := FromAnyColumns(FromAnyColumnsInput{
		Columns: []SeriesInput{
			{Name: "g", Values: []any{"a", "a", "b"}},
			{Name: "v", Values: []any{int64(3), int64(7), int64(1)}},
		},
	})
	if err != nil {
		t.Fatalf("from columns: %v", err)
	}

	out, err := df.GroupBy("g").Agg(
		expr.Min(expr.Col("v")).Alias("min_v"),
		expr.Max(expr.Col("v")).Alias("max_v"),
	)
	if err != nil {
		t.Fatalf("agg min/max: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("groups: %d", out.Height())
	}
}
