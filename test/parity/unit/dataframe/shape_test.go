package dataframe

// Ported from py-polars/tests/unit/dataframe/test_shape.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFShape(t *testing.T) {
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
	shape := df.Shape()
	if shape[0] != 3 {
		t.Fatalf("shape[0]: got %d, want 3", shape[0])
	}
	if shape[1] != 2 {
		t.Fatalf("shape[1]: got %d, want 2", shape[1])
	}
}

func TestDFShapeEmpty(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{}})
	if err != nil {
		t.Fatalf("empty df creation: %v", err)
	}
	shape := df.Shape()
	if shape[0] != 0 || shape[1] != 0 {
		t.Fatalf("empty shape: got %v, want [0 0]", shape)
	}
}

func TestDFShapeSingleColumn(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	shape := df.Shape()
	if shape[0] != 2 || shape[1] != 1 {
		t.Fatalf("shape: got %v, want [2 1]", shape)
	}
}

func TestDFHeightWidth(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "b", Values: []any{float64(1), float64(2), float64(3), float64(4)}},
			{Name: "c", Values: []any{"a", "b", "c", "d"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	if df.Height() != 4 {
		t.Fatalf("height: got %d, want 4", df.Height())
	}
	if df.Width() != 3 {
		t.Fatalf("width: got %d, want 3", df.Width())
	}
}
