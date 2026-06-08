package expr

// Ported from py-polars/tests/unit/expr/test_when_then.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestExprWhenBasic(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.When(polars.Col("a").Gt(polars.Lit(int64(2))), polars.Lit(int64(1)), polars.Lit(int64(0))).Alias("cond"),
	)
	if err != nil {
		t.Fatalf("select when: %v", err)
	}
	s, _ := result.GetColumn("cond")
	if v, ok := s.Value(0).(int64); !ok || v != 0 {
		t.Fatalf("cond[0]: got %v, want 0", s.Value(0))
	}
	if v, ok := s.Value(2).(int64); !ok || v != 1 {
		t.Fatalf("cond[2]: got %v, want 1", s.Value(2))
	}
}

func TestExprWhenWithColumn(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "b", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.When(polars.Col("a").Gt(polars.Lit(int64(2))), polars.Col("b"), polars.Lit(int64(0))).Alias("result"),
	)
	if err != nil {
		t.Fatalf("select when with column: %v", err)
	}
	s, _ := result.GetColumn("result")
	if v, ok := s.Value(0).(int64); !ok || v != 0 {
		t.Fatalf("result[0]: got %v, want 0", s.Value(0))
	}
	if v, ok := s.Value(2).(int64); !ok || v != 30 {
		t.Fatalf("result[2]: got %v, want 30", s.Value(2))
	}
}
