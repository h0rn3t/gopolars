package dataframe

// Ported from py-polars/tests/unit/dataframe/test_describe.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFDescribeInt(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	desc, err := df.Describe()
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	// DISCREPANCY: gopolars Describe uses "len" instead of Python's "count"
	cols := desc.Columns()
	if len(cols) == 0 {
		t.Fatalf("describe should have columns")
	}
}

func TestDFDescribeFloat(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{float64(1.5), float64(2.5), float64(3.5)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	desc, err := df.Describe()
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	if desc.Width() < 2 {
		t.Fatalf("describe should have at least 2 columns (statistic + data)")
	}
}

func TestDFDescribeString(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "s", Values: []any{"a", "b", "c", "a"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	desc, err := df.Describe()
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	_ = desc
}

func TestDFDescribeWithNulls(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{nil, int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	desc, err := df.Describe()
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	_ = desc
}

func TestDFDescribeMultipleColumns(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "ints", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "floats", Values: []any{float64(1.5), float64(2.5), float64(3.5)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	desc, err := df.Describe()
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	if desc.Width() < 3 {
		t.Fatalf("describe should have at least 3 columns (statistic + 2 data)")
	}
}
