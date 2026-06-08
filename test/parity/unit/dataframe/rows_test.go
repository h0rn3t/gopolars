package dataframe

// Ported from py-polars/tests/unit/dataframe/test_rows.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFRows(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{"x", "y", "z"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	rows := df.Rows()
	if len(rows) != 3 {
		t.Fatalf("rows length: got %d, want 3", len(rows))
	}
	if rows[0]["a"] != int64(1) {
		t.Fatalf("rows[0][a]: got %v, want 1", rows[0]["a"])
	}
	if rows[0]["b"] != "x" {
		t.Fatalf("rows[0][b]: got %v, want x", rows[0]["b"])
	}
}

func TestDFRowsByName(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{"x", "y", "z"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	row, err := df.Row(1)
	if err != nil {
		t.Fatalf("row by index: %v", err)
	}
	if row["a"] != int64(2) {
		t.Fatalf("row[1][a]: got %v, want 2", row["a"])
	}
	if row["b"] != "y" {
		t.Fatalf("row[1][b]: got %v, want y", row["b"])
	}
}

func TestDFRowsSingleRow(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(42)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	rows := df.Rows()
	if len(rows) != 1 {
		t.Fatalf("rows length: got %d, want 1", len(rows))
	}
	if rows[0]["a"] != int64(42) {
		t.Fatalf("rows[0][a]: got %v, want 42", rows[0]["a"])
	}
}

func TestDFRowsEmpty(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{}})
	if err != nil {
		t.Fatalf("empty df creation: %v", err)
	}
	rows := df.Rows()
	if len(rows) != 0 {
		t.Fatalf("rows length on empty df: got %d, want 0", len(rows))
	}
}

func TestDFIterRows(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
			{Name: "b", Values: []any{"x", "y"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	iterRows := df.IterRows()
	if len(iterRows) != 2 {
		t.Fatalf("iter_rows length: got %d, want 2", len(iterRows))
	}
}
