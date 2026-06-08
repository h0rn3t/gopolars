package dataframe

// Ported from py-polars/tests/unit/dataframe/test_getitem.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFGetItemByColumnName(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
			{Name: "b", Values: []any{int64(3), int64(4)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	s, err := df.GetColumn("a")
	if err != nil {
		t.Fatalf("getitem by column name: %v", err)
	}
	if s.Len() != 2 {
		t.Fatalf("column 'a' length: got %d, want 2", s.Len())
	}
	if v, ok := s.Value(0).(int64); !ok || v != 1 {
		t.Fatalf("column 'a'[0]: got %v, want 1", s.Value(0))
	}
}

func TestDFGetItemSelectColumns(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
			{Name: "b", Values: []any{int64(3), int64(4)}},
			{Name: "c", Values: []any{int64(5), int64(6)}},
			{Name: "d", Values: []any{int64(7), int64(8)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	sub, err := df.SubSelectColumns("a", "d")
	if err != nil {
		t.Fatalf("sub_select_columns: %v", err)
	}
	cols := sub.Columns()
	if len(cols) != 2 || cols[0] != "a" || cols[1] != "d" {
		t.Fatalf("sub_select columns: got %v, want [a d]", cols)
	}
}

func TestDFGetItemSlice(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "b", Values: []any{float64(5), float64(6), float64(7), float64(8)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	sliced := df.Slice(1, 2)
	if sliced.Height() != 2 {
		t.Fatalf("slice height: got %d, want 2", sliced.Height())
	}
	s, _ := sliced.GetColumn("a")
	if v, ok := s.Value(0).(int64); !ok || v != 2 {
		t.Fatalf("slice[0]: got %v, want 2", s.Value(0))
	}
}

func TestDFGetItemRow(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{float64(1), float64(2), float64(3), float64(4)}},
			{Name: "b", Values: []any{int64(3), int64(4), int64(5), int64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	row, err := df.Row(1)
	if err != nil {
		t.Fatalf("row: %v", err)
	}
	if row["a"] != float64(2) {
		t.Fatalf("row[1][a]: got %v, want 2.0", row["a"])
	}
	if row["b"] != int64(4) {
		t.Fatalf("row[1][b]: got %v, want 4", row["b"])
	}
}

func TestDFGetItemByIndexAndColumn(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{float64(1), float64(2), float64(3), float64(4)}},
			{Name: "b", Values: []any{int64(3), int64(4), int64(5), int64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	val, err := df.Item(2, "b")
	if err != nil {
		t.Fatalf("item: %v", err)
	}
	if v, ok := val.(int64); !ok || v != 5 {
		t.Fatalf("item(2,b): got %v, want 5", val)
	}
}

func TestDFGetItemSubSelectColumns(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{int64(4), int64(5), int64(6)}},
			{Name: "c", Values: []any{int64(7), int64(8), int64(9)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	sub, err := df.SubSelectColumns("a", "c")
	if err != nil {
		t.Fatalf("sub_select_columns: %v", err)
	}
	cols := sub.Columns()
	if len(cols) != 2 || cols[0] != "a" || cols[1] != "c" {
		t.Fatalf("sub_select columns: got %v, want [a c]", cols)
	}
}

func TestDFGetItemEmptySlice(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
			{Name: "b", Values: []any{float64(5), float64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	sliced := df.Slice(0, 0)
	if sliced.Height() != 0 {
		t.Fatalf("empty slice height: got %d, want 0", sliced.Height())
	}
	if sliced.Width() != 2 {
		t.Fatalf("empty slice width: got %d, want 2", sliced.Width())
	}
}
