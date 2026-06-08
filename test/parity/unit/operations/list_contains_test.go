package operations

// Ported from py-polars/tests/unit/operations/namespaces/list/test_list.py
// (py-1.28.1, representative subset for list.contains)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// list.contains(value) returns a Boolean per row: whether the sub-list holds it.
func TestListContains(t *testing.T) {
	t.Parallel()
	out, err := listDF(t).Select(polars.Col("l").ListContains(polars.Lit(int64(4))).Alias("has4"))
	if err != nil {
		t.Fatalf("list.contains: %v", err)
	}
	col, _ := out.GetColumn("has4")
	for i, w := range []bool{false, true, false} {
		if v, _ := col.Value(i).(bool); v != w {
			t.Fatalf("contains[%d]: got %v, want %v", i, col.Value(i), w)
		}
	}
}
