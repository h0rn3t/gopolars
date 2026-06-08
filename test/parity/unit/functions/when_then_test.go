package functions

// Ported from py-polars/tests/unit/functions/test_when_then.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestWhenThenBasic(t *testing.T) {
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
		t.Fatalf("select when/then: %v", err)
	}
	s, _ := result.GetColumn("cond")
	if v, ok := s.Value(0).(int64); !ok || v != 0 {
		t.Fatalf("cond[0]: got %v, want 0", s.Value(0))
	}
	if v, ok := s.Value(2).(int64); !ok || v != 1 {
		t.Fatalf("cond[2]: got %v, want 1", s.Value(2))
	}
}

func TestWhenThenWithString(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "val", Values: []any{int64(10), int64(20), int64(30)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.When(polars.Col("val").Gt(polars.Lit(int64(15))), polars.Lit("high"), polars.Lit("low")).Alias("category"),
	)
	if err != nil {
		t.Fatalf("select when/then string: %v", err)
	}
	s, _ := result.GetColumn("category")
	if v, ok := s.Value(0).(string); !ok || v != "low" {
		t.Fatalf("category[0]: got %v, want low", s.Value(0))
	}
	if v, ok := s.Value(1).(string); !ok || v != "high" {
		t.Fatalf("category[1]: got %v, want high", s.Value(1))
	}
}

func TestWhenThenWithColumnResult(t *testing.T) {
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
		t.Fatalf("select when/then with column: %v", err)
	}
	s, _ := result.GetColumn("result")
	// a[0]=1 -> not gt 2 -> 0, a[2]=3 -> gt 2 -> b[2]=30
	if v, ok := s.Value(0).(int64); !ok || v != 0 {
		t.Fatalf("result[0]: got %v, want 0", s.Value(0))
	}
	if v, ok := s.Value(2).(int64); !ok || v != 30 {
		t.Fatalf("result[2]: got %v, want 30", s.Value(2))
	}
}

func TestWhenThenWithFilter(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(
		polars.When(polars.Col("x").Ge(polars.Lit(int64(3))), polars.Lit("pass"), polars.Lit("fail")).Alias("status"),
	)
	if err != nil {
		t.Fatalf("select when/then ge: %v", err)
	}
	s, _ := result.GetColumn("status")
	if v, ok := s.Value(0).(string); !ok || v != "fail" {
		t.Fatalf("status[0]: got %v, want fail", s.Value(0))
	}
	if v, ok := s.Value(4).(string); !ok || v != "pass" {
		t.Fatalf("status[4]: got %v, want pass", s.Value(4))
	}
}
