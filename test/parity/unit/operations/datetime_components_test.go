package operations

// Ported from py-polars/tests/unit/operations/namespaces/temporal/test_datetime.py
// (py-1.28.1, representative subset for the dt component accessors gopolars exposes)

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func dtComponentsDF(t *testing.T) polars.DataFrame {
	t.Helper()
	// 2024-03-04 is a Monday; 2024-03-06 is a Wednesday.
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "ts", DType: polars.Datetime, Values: []any{
			time.Date(2024, 3, 4, 9, 0, 0, 0, time.UTC),   // Monday
			time.Date(2024, 3, 6, 17, 30, 0, 0, time.UTC), // Wednesday
			time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),  // Sunday
		}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

func dtIntCol(t *testing.T, df polars.DataFrame, e polars.Expr) []int64 {
	t.Helper()
	out, err := df.Select(e.Alias("v"))
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	col, _ := out.GetColumn("v")
	got := make([]int64, col.Len())
	for i := range got {
		v, ok := col.Value(i).(int64)
		if !ok {
			t.Fatalf("value[%d] not int64: %v", i, col.Value(i))
		}
		got[i] = v
	}
	return got
}

func TestDtMonthDayHour(t *testing.T) {
	t.Parallel()
	df := dtComponentsDF(t)
	if got := dtIntCol(t, df, polars.Col("ts").DtMonth()); got[0] != 3 || got[1] != 3 {
		t.Fatalf("month: got %v, want [3 3 ...]", got)
	}
	if got := dtIntCol(t, df, polars.Col("ts").DtDay()); got[0] != 4 || got[1] != 6 {
		t.Fatalf("day: got %v, want [4 6 ...]", got)
	}
	if got := dtIntCol(t, df, polars.Col("ts").DtHour()); got[0] != 9 || got[1] != 17 {
		t.Fatalf("hour: got %v, want [9 17 ...]", got)
	}
}

// dt.weekday() uses Polars/ISO weekday numbering: Monday=1 .. Sunday=7.
func TestDtWeekday(t *testing.T) {
	t.Parallel()
	df := dtComponentsDF(t)
	got := dtIntCol(t, df, polars.Col("ts").DtWeekday())
	// Monday=1, Wednesday=3, Sunday=7.
	if got[0] != 1 || got[1] != 3 || got[2] != 7 {
		t.Fatalf("weekday: got %v, want [1 3 7] (ISO Mon=1..Sun=7)", got)
	}
}
