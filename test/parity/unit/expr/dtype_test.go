package expr

// Ported from py-polars/tests/unit/expr/test_dtype_of.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestExprDtypeInference(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{float64(1.5), float64(2.5), float64(3.5)}},
			{Name: "c", Values: []any{"x", "y", "z"}},
			{Name: "d", Values: []any{true, false, true}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Col("a"))
	if err != nil {
		t.Fatalf("select col(a): %v", err)
	}
	dts := result.Dtypes()
	if dts[0] != polars.Int64 {
		t.Fatalf("dtype of col(a): got %v, want Int64", dts[0])
	}

	result2, err := df.Select(polars.Col("b"))
	if err != nil {
		t.Fatalf("select col(b): %v", err)
	}
	dts2 := result2.Dtypes()
	if dts2[0] != polars.Float64 {
		t.Fatalf("dtype of col(b): got %v, want Float64", dts2[0])
	}

	result3, err := df.Select(polars.Col("c"))
	if err != nil {
		t.Fatalf("select col(c): %v", err)
	}
	dts3 := result3.Dtypes()
	if dts3[0] != polars.String {
		t.Fatalf("dtype of col(c): got %v, want String", dts3[0])
	}
}

func TestExprArithmeticDtypeResult(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{float64(1.0), float64(2.0), float64(3.0)}},
			{Name: "b", Values: []any{float64(1.5), float64(2.5), float64(3.5)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").Add(polars.Col("b")).Alias("sum"),
	)
	if err != nil {
		t.Fatalf("select float + float: %v", err)
	}
	dts := result.Dtypes()
	if dts[0] != polars.Float64 {
		t.Fatalf("dtype of float+float: got %v, want Float64", dts[0])
	}
}

func TestExprBoolResult(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").Gt(polars.Lit(int64(1))).Alias("gt1"),
	)
	if err != nil {
		t.Fatalf("select gt: %v", err)
	}
	dts := result.Dtypes()
	if dts[0] != polars.Boolean {
		t.Fatalf("dtype of gt result: got %v, want Boolean", dts[0])
	}
}
