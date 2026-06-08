package dataframe

// Ported from py-polars/tests/unit/dataframe/test_from_dict.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestFromDictBasic(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{float64(4), float64(5), float64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("from_dict: %v", err)
	}
	if df.Height() != 3 {
		t.Fatalf("height: got %d, want 3", df.Height())
	}
	if df.Width() != 2 {
		t.Fatalf("width: got %d, want 2", df.Width())
	}
	cols := df.Columns()
	if len(cols) != 2 || cols[0] != "a" || cols[1] != "b" {
		t.Fatalf("columns: got %v, want [a b]", cols)
	}
}

func TestFromDictWithNulls(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{nil, int64(2), int64(3)}},
			{Name: "b", Values: []any{"x", nil, "z"}},
		},
	})
	if err != nil {
		t.Fatalf("from_dict with nulls: %v", err)
	}
	if df.Height() != 3 {
		t.Fatalf("height: got %d, want 3", df.Height())
	}
	nc := df.NullCount()
	if nc["a"] != 1 {
		t.Fatalf("null_count[a]: got %d, want 1", nc["a"])
	}
	if nc["b"] != 1 {
		t.Fatalf("null_count[b]: got %d, want 1", nc["b"])
	}
}

func TestFromDictDifferentDtypes(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "int_col", Values: []any{int64(1), int64(2)}},
			{Name: "float_col", Values: []any{float64(1.5), float64(2.5)}},
			{Name: "str_col", Values: []any{"a", "b"}},
			{Name: "bool_col", Values: []any{true, false}},
		},
	})
	if err != nil {
		t.Fatalf("from_dict mixed dtypes: %v", err)
	}
	if df.Width() != 4 {
		t.Fatalf("width: got %d, want 4", df.Width())
	}
	schema := df.Schema()
	foundInt, foundFloat, foundStr, foundBool := false, false, false, false
	for _, f := range schema {
		if f.Name == "int_col" && f.Type == polars.Int64 {
			foundInt = true
		}
		if f.Name == "float_col" && f.Type == polars.Float64 {
			foundFloat = true
		}
		if f.Name == "str_col" && f.Type == polars.String {
			foundStr = true
		}
		if f.Name == "bool_col" && f.Type == polars.Boolean {
			foundBool = true
		}
	}
	if !foundInt {
		t.Fatalf("schema missing int_col with Int64")
	}
	if !foundFloat {
		t.Fatalf("schema missing float_col with Float64")
	}
	if !foundStr {
		t.Fatalf("schema missing str_col with String")
	}
	if !foundBool {
		t.Fatalf("schema missing bool_col with Boolean")
	}
}

func TestFromDictSingleRow(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1)}},
			{Name: "b", Values: []any{"x"}},
		},
	})
	if err != nil {
		t.Fatalf("from_dict single row: %v", err)
	}
	if df.Height() != 1 {
		t.Fatalf("height: got %d, want 1", df.Height())
	}
}

func TestFromDictSingleColumn(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("from_dict single col: %v", err)
	}
	if df.Width() != 1 {
		t.Fatalf("width: got %d, want 1", df.Width())
	}
	if df.Height() != 3 {
		t.Fatalf("height: got %d, want 3", df.Height())
	}
}

// A dict with an all-null column requires pinning the column dtype (inference
// can't determine it); the resulting column is fully null with that dtype.
func TestFromDictAllNulls(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{nil, nil}, DType: polars.Float64},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	if df.Height() != 2 || df.Width() != 1 {
		t.Fatalf("shape: got %dx%d, want 2x1", df.Height(), df.Width())
	}
	x, _ := df.GetColumn("x")
	if x.DataType() != polars.Float64 {
		t.Fatalf("dtype: got %v, want Float64", x.DataType())
	}
	if x.NullCount() != 2 {
		t.Fatalf("null count: got %d, want 2", x.NullCount())
	}
}
