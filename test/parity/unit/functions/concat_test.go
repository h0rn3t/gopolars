package functions

// Ported from py-polars/tests/unit/functions/test_concat.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFConcatVertical(t *testing.T) {
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
	result, err := df1.Concat(polars.ConcatInput{Others: []polars.DataFrame{df2}, How: "vertical"})
	if err != nil {
		t.Fatalf("concat vertical: %v", err)
	}
	if result.Height() != 4 {
		t.Fatalf("concat height: got %d, want 4", result.Height())
	}
	if result.Width() != 2 {
		t.Fatalf("concat width: got %d, want 2", result.Width())
	}
}

func TestDFConcatHorizontal(t *testing.T) {
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
			{Name: "b", Values: []any{int64(3), int64(4)}},
		},
	})
	if err != nil {
		t.Fatalf("df2 creation: %v", err)
	}
	result, err := df1.Concat(polars.ConcatInput{Others: []polars.DataFrame{df2}, How: "horizontal"})
	if err != nil {
		t.Fatalf("concat horizontal: %v", err)
	}
	if result.Width() != 2 {
		t.Fatalf("concat width: got %d, want 2", result.Width())
	}
	cols := result.Columns()
	if len(cols) != 2 {
		t.Fatalf("concat columns: got %v, want 2", cols)
	}
}

func TestDFVStackConcat(t *testing.T) {
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
	result, err := df1.VStack(df2)
	if err != nil {
		t.Fatalf("vstack: %v", err)
	}
	if result.Height() != 4 {
		t.Fatalf("vstack height: got %d, want 4", result.Height())
	}
}

func TestDFExtendConcat(t *testing.T) {
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
	result, err := df1.Extend(df2)
	if err != nil {
		t.Fatalf("extend: %v", err)
	}
	if result.Height() != 4 {
		t.Fatalf("extend height: got %d, want 4", result.Height())
	}
}
