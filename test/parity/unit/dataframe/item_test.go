package dataframe

// Ported from py-polars/tests/unit/dataframe/test_item.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFItemInt(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(10), int64(20), int64(30)}},
			{Name: "b", Values: []any{"x", "y", "z"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	val, err := df.Item(0, "a")
	if err != nil {
		t.Fatalf("item: %v", err)
	}
	if v, ok := val.(int64); !ok || v != 10 {
		t.Fatalf("item(0,a): got %v, want 10", val)
	}
	val2, err := df.Item(2, "b")
	if err != nil {
		t.Fatalf("item: %v", err)
	}
	if v, ok := val2.(string); !ok || v != "z" {
		t.Fatalf("item(2,b): got %v, want z", val2)
	}
}

func TestDFItemFloat(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "f", Values: []any{float64(1.5), float64(2.5)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	val, err := df.Item(0, "f")
	if err != nil {
		t.Fatalf("item: %v", err)
	}
	if v, ok := val.(float64); !ok || v != 1.5 {
		t.Fatalf("item(0,f): got %v, want 1.5", val)
	}
}

func TestDFItemOutOfBounds(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	_, err = df.Item(5, "a")
	if err == nil {
		t.Fatalf("item out of bounds should error")
	}
}

func TestDFItemMissingColumn(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	_, err = df.Item(0, "nonexistent")
	if err == nil {
		t.Fatalf("item with missing column should error")
	}
}
