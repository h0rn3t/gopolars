package operations

// Ported from py-polars/tests/unit/operations/test_group_by_dynamic.py (py-1.28.1, representative subset)

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// group_by_dynamic buckets rows into fixed-width time windows and aggregates each.
func TestGroupByDynamicHourly(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "t", Values: []any{
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 1, 1, 0, 30, 0, 0, time.UTC),
			time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
			time.Date(2024, 1, 1, 1, 15, 0, 0, time.UTC),
		}},
		{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.GroupByDynamic(polars.DynamicGroupInput{
		By:      "t",
		Every:   time.Hour,
		Period:  time.Hour,
		AggExpr: polars.Sum(polars.Col("v")).Alias("s"),
	})
	if err != nil {
		t.Fatalf("group_by_dynamic: %v", err)
	}
	// two hourly windows: [00:00,01:00) -> 1+2=3, [01:00,02:00) -> 3+4=7
	if out.Height() != 2 {
		t.Fatalf("windows: got %d, want 2", out.Height())
	}
	s, err := out.GetColumn("s")
	if err != nil {
		t.Fatalf("get s: %v", err)
	}
	for i, w := range []float64{3, 7} {
		if toFloatAny(s.Value(i)) != w {
			t.Fatalf("window sum[%d]: got %v, want %v", i, s.Value(i), w)
		}
	}
}
