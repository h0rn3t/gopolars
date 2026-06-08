package dataframe

// Ported from py-polars/tests/unit/dataframe/test_partition_by.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFPartitionBy(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "group", Values: []any{"a", "a", "b", "b", "c"}},
			{Name: "value", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	parts, err := df.PartitionBy("group")
	if err != nil {
		t.Fatalf("partition_by: %v", err)
	}
	if len(parts) != 3 {
		t.Fatalf("partition count: got %d, want 3", len(parts))
	}
}

func TestDFPartitionByTwoColumns(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{"x", "x", "y", "y"}},
			{Name: "b", Values: []any{"m", "n", "m", "n"}},
			{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	parts, err := df.PartitionBy("a", "b")
	if err != nil {
		t.Fatalf("partition_by: %v", err)
	}
	if len(parts) != 4 {
		t.Fatalf("partition count: got %d, want 4", len(parts))
	}
}
