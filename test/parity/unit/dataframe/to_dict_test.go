package dataframe

// Ported from py-polars/tests/unit/dataframe/test_to_dict.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFToDict(t *testing.T) {
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
	dict := df.ToDict()
	if len(dict) != 2 {
		t.Fatalf("dict keys: got %d, want 2", len(dict))
	}
	aVals, ok := dict["a"]
	if !ok {
		t.Fatalf("dict missing key 'a'")
	}
	if len(aVals) != 3 {
		t.Fatalf("dict[a] length: got %d, want 3", len(aVals))
	}
	bVals, ok := dict["b"]
	if !ok {
		t.Fatalf("dict missing key 'b'")
	}
	if len(bVals) != 3 {
		t.Fatalf("dict[b] length: got %d, want 3", len(bVals))
	}
}

func TestDFToDicts(t *testing.T) {
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
	dicts := df.ToDicts()
	if len(dicts) != 2 {
		t.Fatalf("dicts length: got %d, want 2", len(dicts))
	}
	if dicts[0]["a"] != int64(1) {
		t.Fatalf("dicts[0][a]: got %v, want 1", dicts[0]["a"])
	}
	if dicts[0]["b"] != "x" {
		t.Fatalf("dicts[0][b]: got %v, want x", dicts[0]["b"])
	}
	if dicts[1]["a"] != int64(2) {
		t.Fatalf("dicts[1][a]: got %v, want 2", dicts[1]["a"])
	}
}

func TestDFToDictsWithNulls(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{nil, int64(2)}},
			{Name: "b", Values: []any{"x", nil}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	dicts := df.ToDicts()
	if len(dicts) != 2 {
		t.Fatalf("dicts length: got %d, want 2", len(dicts))
	}
	if dicts[0]["a"] != nil {
		t.Fatalf("dicts[0][a]: got %v, want nil", dicts[0]["a"])
	}
	if dicts[1]["b"] != nil {
		t.Fatalf("dicts[1][b]: got %v, want nil", dicts[1]["b"])
	}
}
