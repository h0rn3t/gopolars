package dataframe

// Ported from py-polars/tests/unit/dataframe/test_0_width_df.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestZeroWidthDF(t *testing.T) {
	t.Parallel()
	// Python: pl.DataFrame(height=5) creates a 5-row, 0-column DataFrame
	// gopolars creates empty DataFrame differently
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{}})
	if err != nil {
		t.Fatalf("empty df creation: %v", err)
	}
	if !df.IsEmpty() {
		t.Fatalf("empty df should be empty")
	}
	if df.Height() != 0 {
		t.Fatalf("empty df height: got %d, want 0", df.Height())
	}
	if df.Width() != 0 {
		t.Fatalf("empty df width: got %d, want 0", df.Width())
	}
	// Clear on an already-empty df should still work
	cleared := df.Clear()
	if !cleared.IsEmpty() {
		t.Fatalf("cleared empty df should be empty")
	}
	// Clone should work
	cloned := df.Clone()
	if cloned.Height() != df.Height() {
		t.Fatalf("clone height mismatch")
	}
	// Equals on empty df
	eq, err := df.Equals(df)
	if err != nil {
		t.Fatalf("equals: %v", err)
	}
	if !eq {
		t.Fatalf("empty df should equal itself")
	}
	// Estimated size
	size := df.EstimatedSize()
	if size != 0 {
		t.Fatalf("estimated_size: got %d, want 0", size)
	}
}

func TestZeroWidthDFColumns(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{}})
	if err != nil {
		t.Fatalf("empty df creation: %v", err)
	}
	cols := df.Columns()
	if len(cols) != 0 {
		t.Fatalf("columns: got %v, want []", cols)
	}
	schema := df.Schema()
	if len(schema) != 0 {
		t.Fatalf("schema: got %d fields, want 0", len(schema))
	}
}

func TestZeroWidthDFShape(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{}})
	if err != nil {
		t.Fatalf("empty df creation: %v", err)
	}
	shape := df.Shape()
	if shape[0] != 0 || shape[1] != 0 {
		t.Fatalf("shape: got %v, want [0 0]", shape)
	}
}
