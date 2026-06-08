package expr

// Ported from py-polars/tests/unit/expr/test_dunders.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestExprDundersAdd(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{int64(4), int64(5), int64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").Add(polars.Col("b")).Alias("a_plus_b"),
	)
	if err != nil {
		t.Fatalf("select add: %v", err)
	}
	s, _ := result.GetColumn("a_plus_b")
	if v, ok := s.Value(0).(int64); !ok || v != 5 {
		t.Fatalf("a_plus_b[0]: got %v, want 5", s.Value(0))
	}
}

func TestExprDundersSub(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(10), int64(20), int64(30)}},
			{Name: "b", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").Sub(polars.Col("b")).Alias("a_minus_b"),
	)
	if err != nil {
		t.Fatalf("select sub: %v", err)
	}
	s, _ := result.GetColumn("a_minus_b")
	if v, ok := s.Value(0).(int64); !ok || v != 9 {
		t.Fatalf("a_minus_b[0]: got %v, want 9", s.Value(0))
	}
}

func TestExprDundersMul(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(2), int64(3), int64(4)}},
			{Name: "b", Values: []any{int64(5), int64(6), int64(7)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").Mul(polars.Col("b")).Alias("a_mul_b"),
	)
	if err != nil {
		t.Fatalf("select mul: %v", err)
	}
	s, _ := result.GetColumn("a_mul_b")
	if v, ok := s.Value(0).(int64); !ok || v != 10 {
		t.Fatalf("a_mul_b[0]: got %v, want 10", s.Value(0))
	}
}

func TestExprDundersDiv(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{float64(10.0), float64(20.0), float64(30.0)}},
			{Name: "b", Values: []any{float64(2.0), float64(4.0), float64(5.0)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").Div(polars.Col("b")).Alias("a_div_b"),
	)
	if err != nil {
		t.Fatalf("select div: %v", err)
	}
	s, _ := result.GetColumn("a_div_b")
	if v, ok := s.Value(0).(float64); !ok || v != 5.0 {
		t.Fatalf("a_div_b[0]: got %v, want 5.0", s.Value(0))
	}
}

func TestExprDundersNeg(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(-2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").Neg().Alias("neg_a"),
	)
	if err != nil {
		t.Fatalf("select neg: %v", err)
	}
	s, _ := result.GetColumn("neg_a")
	// Negating an Int64 column preserves the Int64 dtype (matching Polars).
	if s.DataType() != polars.Int64 {
		t.Fatalf("neg dtype: got %v, want Int64", s.DataType())
	}
	if v, ok := s.Value(0).(int64); !ok || v != -1 {
		t.Fatalf("neg_a[0]: got %T(%v), want int64 -1", s.Value(0), s.Value(0))
	}
	if v, ok := s.Value(1).(int64); !ok || v != 2 {
		t.Fatalf("neg_a[1]: got %T(%v), want int64 2", s.Value(1), s.Value(1))
	}
}

func TestExprDundersEqNe(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{int64(1), int64(5), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").Eq(polars.Col("b")).Alias("eq"),
		polars.Col("a").Ne(polars.Col("b")).Alias("ne"),
	)
	if err != nil {
		t.Fatalf("select eq/ne: %v", err)
	}
	s, _ := result.GetColumn("eq")
	if v, ok := s.Value(0).(bool); !ok || v != true {
		t.Fatalf("eq[0]: got %v, want true", s.Value(0))
	}
	if v, ok := s.Value(1).(bool); !ok || v != false {
		t.Fatalf("eq[1]: got %v, want false", s.Value(1))
	}
}

func TestExprDundersAndOr(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{true, true, false, false}},
			{Name: "b", Values: []any{true, false, true, false}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.Col("a").And(polars.Col("b")).Alias("and_ab"),
		polars.Col("a").Or(polars.Col("b")).Alias("or_ab"),
	)
	if err != nil {
		t.Fatalf("select and/or: %v", err)
	}
	sAnd, _ := result.GetColumn("and_ab")
	if v, ok := sAnd.Value(0).(bool); !ok || v != true {
		t.Fatalf("and[0]: got %v, want true", sAnd.Value(0))
	}
	if v, ok := sAnd.Value(1).(bool); !ok || v != false {
		t.Fatalf("and[1]: got %v, want false", sAnd.Value(1))
	}
}
