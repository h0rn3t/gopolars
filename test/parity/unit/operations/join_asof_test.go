package operations

// Ported from py-polars/tests/unit/operations/test_join_asof.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// A backward as-of join matches each left row with the most recent right row
// whose key is <= the left key.
func TestJoinAsofBackward(t *testing.T) {
	t.Parallel()
	left, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "t", Values: []any{int64(1), int64(5), int64(10)}},
		{Name: "l", Values: []any{"a", "b", "c"}},
	}})
	if err != nil {
		t.Fatalf("left: %v", err)
	}
	right, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "t", Values: []any{int64(1), int64(4), int64(8)}},
		{Name: "r", Values: []any{int64(100), int64(400), int64(800)}},
	}})
	if err != nil {
		t.Fatalf("right: %v", err)
	}
	out, err := left.JoinAsof(polars.JoinInput{
		Other: right, LeftOn: []string{"t"}, RightOn: []string{"t"}, AsofDirection: "backward",
	})
	if err != nil {
		t.Fatalf("join_asof: %v", err)
	}
	// every left row keeps its row (left join semantics)
	if out.Height() != 3 {
		t.Fatalf("height: got %d, want 3", out.Height())
	}
	r, err := out.GetColumn("r")
	if err != nil {
		t.Fatalf("get r: %v", err)
	}
	// t=1 -> r=100; t=5 -> nearest <=5 is 4 -> 400; t=10 -> nearest <=10 is 8 -> 800
	for i, w := range []int64{100, 400, 800} {
		if v, _ := r.Value(i).(int64); v != w {
			t.Fatalf("r[%d]: got %v, want %d", i, r.Value(i), w)
		}
	}
}
