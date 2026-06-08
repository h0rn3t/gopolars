package dataframe

// Ported from py-polars/tests/unit/dataframe/test_equals.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFEqualsSame(t *testing.T) {
	t.Parallel()
	df1, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "foo", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "bar", Values: []any{float64(6), float64(7), float64(8)}},
			{Name: "ham", Values: []any{"a", "b", "c"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	df2, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "foo", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "bar", Values: []any{float64(6), float64(7), float64(8)}},
			{Name: "ham", Values: []any{"a", "b", "c"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	eq, err := df1.Equals(df2)
	if err != nil {
		t.Fatalf("equals: %v", err)
	}
	if !eq {
		t.Fatalf("identical dataframes should be equal")
	}
}

func TestDFEqualsSelf(t *testing.T) {
	t.Parallel()
	df1, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "foo", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	eq, err := df1.Equals(df1)
	if err != nil {
		t.Fatalf("equals self: %v", err)
	}
	if !eq {
		t.Fatalf("dataframe should equal itself")
	}
}

func TestDFEqualsDifferentValues(t *testing.T) {
	t.Parallel()
	df1, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "foo", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "bar", Values: []any{float64(6), float64(7), float64(8)}},
			{Name: "ham", Values: []any{"a", "b", "c"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	df2, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "foo", Values: []any{int64(3), int64(2), int64(1)}},
			{Name: "bar", Values: []any{float64(8), float64(7), float64(6)}},
			{Name: "ham", Values: []any{"c", "b", "a"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	eq, err := df1.Equals(df2)
	if err != nil {
		t.Fatalf("equals: %v", err)
	}
	if eq {
		t.Fatalf("dataframes with different values should not be equal")
	}
}

func TestDFEqualsDifferentColumns(t *testing.T) {
	t.Parallel()
	df1, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "foo", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "bar", Values: []any{float64(6), float64(7), float64(8)}},
			{Name: "ham", Values: []any{"a", "b", "c"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	df3, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{float64(6), float64(7), float64(8)}},
			{Name: "c", Values: []any{"a", "b", "c"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	eq, err := df1.Equals(df3)
	if err != nil {
		t.Fatalf("equals: %v", err)
	}
	if eq {
		t.Fatalf("dataframes with different column names should not be equal")
	}
}

func TestDFEqualsWithNulls(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), nil, int64(3)}},
			{Name: "b", Values: []any{float64(1), float64(2), nil}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	df2, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), nil, int64(3)}},
			{Name: "b", Values: []any{float64(1), float64(2), nil}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	eq, err := df.Equals(df2)
	if err != nil {
		t.Fatalf("equals: %v", err)
	}
	// DISCREPANCY: Python's .equals() treats nulls as equal by default.
	// gopolars behavior with nulls needs verification.
	if !eq {
		t.Fatalf("dataframes with same nulls should be equal")
	}
}
