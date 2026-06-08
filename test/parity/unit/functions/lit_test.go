package functions

// Ported from py-polars/tests/unit/functions/test_lit.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestLitInt(t *testing.T) {
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
		polars.Col("a"),
		polars.Lit(int64(10)).Alias("ten"),
	)
	if err != nil {
		t.Fatalf("select lit int: %v", err)
	}
	if result.Width() != 2 {
		t.Fatalf("width: got %d, want 2", result.Width())
	}
	s, _ := result.GetColumn("ten")
	if s.Len() != 3 {
		t.Fatalf("lit length: got %d, want 3", s.Len())
	}
}

func TestLitFloat(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Lit(float64(3.14)).Alias("pi"))
	if err != nil {
		t.Fatalf("select lit float: %v", err)
	}
	s, _ := result.GetColumn("pi")
	if v, ok := s.Value(0).(float64); !ok || v < 3.13 || v > 3.15 {
		t.Fatalf("pi: got %v, want ~3.14", s.Value(0))
	}
}

func TestLitString(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Lit("hello").Alias("greeting"))
	if err != nil {
		t.Fatalf("select lit string: %v", err)
	}
	s, _ := result.GetColumn("greeting")
	if v, ok := s.Value(0).(string); !ok || v != "hello" {
		t.Fatalf("greeting: got %v, want hello", s.Value(0))
	}
}

func TestLitBool(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Select(polars.Lit(true).Alias("flag"))
	if err != nil {
		t.Fatalf("select lit bool: %v", err)
	}
	s, _ := result.GetColumn("flag")
	if v, ok := s.Value(0).(bool); !ok || v != true {
		t.Fatalf("flag: got %v, want true", s.Value(0))
	}
}

func TestLitWithArithmetic(t *testing.T) {
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
		polars.Col("a").Add(polars.Lit(int64(100))).Alias("plus100"),
		polars.Col("a").Mul(polars.Lit(int64(2))).Alias("doubled"),
	)
	if err != nil {
		t.Fatalf("select lit arithmetic: %v", err)
	}
	s, _ := result.GetColumn("plus100")
	if v, ok := s.Value(0).(int64); !ok || v != 101 {
		t.Fatalf("plus100[0]: got %v, want 101", s.Value(0))
	}
	s2, _ := result.GetColumn("doubled")
	if v, ok := s2.Value(1).(int64); !ok || v != 4 {
		t.Fatalf("doubled[1]: got %v, want 4", s2.Value(1))
	}
}

func TestLitWithComparison(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.Filter(polars.Col("a").Gt(polars.Lit(int64(1))))
	if err != nil {
		t.Fatalf("filter col(a) > lit(1): %v", err)
	}
	if result.Height() != 2 {
		t.Fatalf("filter height: got %d, want 2", result.Height())
	}
}
