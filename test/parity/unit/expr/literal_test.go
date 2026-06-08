package expr

// Ported from py-polars/tests/unit/expr/test_literal.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestExprLiteralInt(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Lit(int64(42)).Alias("lit"))
	if err != nil {
		t.Fatalf("select lit int: %v", err)
	}
	if result.Height() != 3 {
		t.Fatalf("lit height: got %d, want 3", result.Height())
	}
}

func TestExprLiteralFloat(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Lit(float64(3.14)).Alias("lit"))
	if err != nil {
		t.Fatalf("select lit float: %v", err)
	}
	s, _ := result.GetColumn("lit")
	if s.Len() != 3 {
		t.Fatalf("lit len: got %d, want 3", s.Len())
	}
}

func TestExprLiteralString(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Lit("hello").Alias("lit"))
	if err != nil {
		t.Fatalf("select lit string: %v", err)
	}
	s, _ := result.GetColumn("lit")
	if s.Len() != 3 {
		t.Fatalf("lit len: got %d, want 3", s.Len())
	}
}

func TestExprLiteralBool(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Lit(true).Alias("lit"))
	if err != nil {
		t.Fatalf("select lit bool: %v", err)
	}
	s, _ := result.GetColumn("lit")
	if s.Len() != 3 {
		t.Fatalf("lit len: got %d, want 3", s.Len())
	}
}
func TestExprLiteralBroadcast(t *testing.T) {
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
		polars.Col("a").Add(polars.Lit(int64(10))).Alias("plus10"),
	)
	if err != nil {
		t.Fatalf("select broadcast: %v", err)
	}
	s, _ := result.GetColumn("plus10")
	if v, ok := s.Value(0).(int64); !ok || v != 11 {
		t.Fatalf("plus10[0]: got %v, want 11", s.Value(0))
	}
	if v, ok := s.Value(2).(int64); !ok || v != 13 {
		t.Fatalf("plus10[2]: got %v, want 13", s.Value(2))
	}
}

func TestExprLiteralInSelect(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Lit(int64(1)).Alias("one"),
		polars.Lit(int64(2)).Alias("two"),
		polars.Col("a").Alias("a"),
	)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if result.Width() != 3 {
		t.Fatalf("width: got %d, want 3", result.Width())
	}
}

func TestExprDtypeOfCol(t *testing.T) {
	t.Parallel()
	colA := polars.Col("a")
	if colA.Name() != "a" {
		t.Fatalf("col name: got %s, want a", colA.Name())
	}

	litVal := polars.Lit(int64(42))
	if litVal.Value() != int64(42) {
		t.Fatalf("lit value: got %v, want 42", litVal.Value())
	}
}

func TestExprCastLiteral(t *testing.T) {
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
		polars.Col("a").Cast(dtypes.Float64).Alias("a_float"),
	)
	if err != nil {
		t.Fatalf("select cast: %v", err)
	}
	dts := result.Dtypes()
	if dts[0] != polars.Float64 {
		t.Fatalf("cast dtype: got %v, want Float64", dts[0])
	}
}
