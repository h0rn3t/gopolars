package dataframe

// Ported from py-polars/tests/unit/dataframe/test_vstack.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFVStackBasic(t *testing.T) {
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
	s, _ := stacked.GetColumn("a")
	if s.Len() != 4 {
		t.Fatalf("vstack column a len: got %d, want 4", s.Len())
	}
}

func TestDFVStackAlias(t *testing.T) {
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
	stacked, err := df1.Vstack(df2)
	if err != nil {
		t.Fatalf("vstack (lowercase): %v", err)
	}
	if stacked.Height() != 2 {
		t.Fatalf("vstack height: got %d, want 2", stacked.Height())
	}
}

func TestDFVStackSingleRow(t *testing.T) {
	t.Parallel()
	df1, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df1 creation: %v", err)
	}
	df2, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(4)}},
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
}
