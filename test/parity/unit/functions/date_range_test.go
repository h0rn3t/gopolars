package functions

// Ported from py-polars/tests/unit/functions/range/test_date_range.py (py-1.28.1)
//
// gopolars has no top-level pl.date_range / pl.datetime_range generator
// (interval, closed, eager). A range must be built in Go and fed into a Datetime
// column. We port a left-closed daily range built manually to document the
// available construction path, and mark the generator itself as a gap.

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestManualDailyRangeLeftClosed(t *testing.T) {
	t.Parallel()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 5) // exclusive end (closed="left")

	var vals []any
	for ts := start; ts.Before(end); ts = ts.AddDate(0, 0, 1) {
		vals = append(vals, ts)
	}
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{{Name: "date", Values: vals}},
	})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	if df.Height() != 5 {
		t.Fatalf("range height: got %d, want 5 (01..05, end excluded)", df.Height())
	}
	col, err := df.GetColumn("date")
	if err != nil {
		t.Fatalf("get date: %v", err)
	}
	first, ok := col.Value(0).(time.Time)
	if !ok || !first.Equal(start) {
		t.Fatalf("first: got %v, want %v", col.Value(0), start)
	}
	last, ok := col.Value(4).(time.Time)
	if !ok || !last.Equal(end.AddDate(0, 0, -1)) {
		t.Fatalf("last: got %v, want %v (end excluded)", col.Value(4), end.AddDate(0, 0, -1))
	}
}
