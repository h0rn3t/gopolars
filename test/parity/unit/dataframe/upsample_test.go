package dataframe

// Ported from py-polars/tests/unit/dataframe/test_upsample.py (py-1.28.1)

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFVStack(t *testing.T) {
	t.Parallel()
	df1, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
			{Name: "b", Values: []any{"x", "y"}},
		},
	})
	if err != nil {
		t.Fatalf("df1 creation: %v", err)
	}
	df2, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(3), int64(4)}},
			{Name: "b", Values: []any{"z", "w"}},
		},
	})
	if err != nil {
		t.Fatalf("df2 creation: %v", err)
	}
	stacked, err := df1.VStack(df2)
	if err != nil {
		t.Fatalf("vstack: %v", err)
	}
	if stacked.Height() != 4 {
		t.Fatalf("vstack height: got %d, want 4", stacked.Height())
	}
	if stacked.Width() != 2 {
		t.Fatalf("vstack width: got %d, want 2", stacked.Width())
	}
}

func TestDFExtend(t *testing.T) {
	t.Parallel()
	df1, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
			{Name: "b", Values: []any{"x", "y"}},
		},
	})
	if err != nil {
		t.Fatalf("df1 creation: %v", err)
	}
	df2, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(3), int64(4)}},
			{Name: "b", Values: []any{"z", "w"}},
		},
	})
	if err != nil {
		t.Fatalf("df2 creation: %v", err)
	}
	extended, err := df1.Extend(df2)
	if err != nil {
		t.Fatalf("extend: %v", err)
	}
	if extended.Height() != 4 {
		t.Fatalf("extend height: got %d, want 4", extended.Height())
	}
}

func TestDFConcatVertical(t *testing.T) {
	t.Parallel()
	df1, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("df1 creation: %v", err)
	}
	df2, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(3), int64(4)}},
		},
	})
	if err != nil {
		t.Fatalf("df2 creation: %v", err)
	}
	result, err := df1.Concat(polars.ConcatInput{Others: []polars.DataFrame{df2}, How: "vertical"})
	if err != nil {
		t.Fatalf("concat vertical: %v", err)
	}
	if result.Height() != 4 {
		t.Fatalf("concat height: got %d, want 4", result.Height())
	}
}

// upsample fills a regular time grid between the first and last timestamp,
// inserting null-valued rows for the missing intervals (matching Polars).
func TestDFUpsample(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "t", Values: []any{
				time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC),
			}},
			{Name: "v", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	out, err := df.Upsample("t", time.Hour)
	if err != nil {
		t.Fatalf("upsample: %v", err)
	}
	// grid 00,01,02,03 -> 4 rows; the 01:00 gap is filled with null v.
	if out.Height() != 4 {
		t.Fatalf("upsample height: got %d, want 4", out.Height())
	}
	v, _ := out.GetColumn("v")
	if v.Value(1) != nil {
		t.Fatalf("v[1] (gap row): got %v, want nil", v.Value(1))
	}
	if got, _ := v.Value(0).(int64); got != 1 {
		t.Fatalf("v[0]: got %v, want 1", v.Value(0))
	}
	if got, _ := v.Value(2).(int64); got != 2 {
		t.Fatalf("v[2]: got %v, want 2", v.Value(2))
	}
}
