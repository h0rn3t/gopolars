package dataframe

// Ported from py-polars/tests/unit/dataframe/test_null_count.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFNullCount(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), nil, int64(3), nil}},
			{Name: "b", Values: []any{"x", "y", nil, nil}},
			{Name: "c", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	nc := df.NullCount()
	if nc["a"] != 2 {
		t.Fatalf("null_count[a]: got %d, want 2", nc["a"])
	}
	if nc["b"] != 2 {
		t.Fatalf("null_count[b]: got %d, want 2", nc["b"])
	}
	if nc["c"] != 0 {
		t.Fatalf("null_count[c]: got %d, want 0", nc["c"])
	}
}

func TestDFNullCountNoNulls(t *testing.T) {
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
	nc := df.NullCount()
	if nc["a"] != 0 {
		t.Fatalf("null_count[a]: got %d, want 0", nc["a"])
	}
	if nc["b"] != 0 {
		t.Fatalf("null_count[b]: got %d, want 0", nc["b"])
	}
}

// An all-null column is created by pinning its dtype (the inference path cannot
// determine a dtype with no non-null values); its null count equals its height.
func TestDFNullCountAllNulls(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{nil, nil, nil}, DType: polars.Int64},
			{Name: "b", Values: []any{int64(1), nil, int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	nc := df.NullCount()
	if nc["a"] != 3 {
		t.Fatalf("null_count[a]: got %d, want 3", nc["a"])
	}
	if nc["b"] != 1 {
		t.Fatalf("null_count[b]: got %d, want 1", nc["b"])
	}
}
