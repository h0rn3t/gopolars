package dataframe

// Ported from py-polars/tests/unit/dataframe/test_extend.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFExtendBasic(t *testing.T) {
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
	s, _ := extended.GetColumn("a")
	if s.Len() != 4 {
		t.Fatalf("extend column a len: got %d, want 4", s.Len())
	}
}

func TestDFExtendSingleRow(t *testing.T) {
	t.Parallel()
	df1, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1)}},
		},
	})
	if err != nil {
		t.Fatalf("df1 creation: %v", err)
	}
	df2, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("df2 creation: %v", err)
	}
	extended, err := df1.Extend(df2)
	if err != nil {
		t.Fatalf("extend: %v", err)
	}
	if extended.Height() != 2 {
		t.Fatalf("extend height: got %d, want 2", extended.Height())
	}
}
