package functions

// Ported from py-polars/tests/unit/functions/test_horizontal.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestSumHorizontal(t *testing.T) {
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
	result, err := df.SumHorizontal("sum_ab")
	if err != nil {
		t.Fatalf("sum_horizontal: %v", err)
	}
	if result.Height() != 3 {
		t.Fatalf("sum_horizontal height: got %d, want 3", result.Height())
	}
	s, _ := result.GetColumn("sum_ab")
	if v, ok := s.Value(0).(float64); !ok || v != 5.0 {
		t.Fatalf("sum_ab[0]: got %v, want 5.0", s.Value(0))
	}
	if v, ok := s.Value(1).(float64); !ok || v != 7.0 {
		t.Fatalf("sum_ab[1]: got %v, want 7.0", s.Value(1))
	}
	if v, ok := s.Value(2).(float64); !ok || v != 9.0 {
		t.Fatalf("sum_ab[2]: got %v, want 9.0", s.Value(2))
	}
}

func TestMeanHorizontal(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{float64(2.0), float64(4.0)}},
			{Name: "b", Values: []any{float64(4.0), float64(6.0)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.MeanHorizontal("mean_ab")
	if err != nil {
		t.Fatalf("mean_horizontal: %v", err)
	}
	s, _ := result.GetColumn("mean_ab")
	if v, ok := s.Value(0).(float64); !ok || v < 2.99 || v > 3.01 {
		t.Fatalf("mean_ab[0]: got %v, want ~3.0", s.Value(0))
	}
}

func TestMaxHorizontal(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(5), int64(3)}},
			{Name: "b", Values: []any{int64(4), int64(2), int64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.MaxHorizontal("max_ab")
	if err != nil {
		t.Fatalf("max_horizontal: %v", err)
	}
	s, _ := result.GetColumn("max_ab")
	if v, ok := s.Value(0).(float64); !ok || v != 4.0 {
		t.Fatalf("max_ab[0]: got %v, want 4.0", s.Value(0))
	}
	if v, ok := s.Value(1).(float64); !ok || v != 5.0 {
		t.Fatalf("max_ab[1]: got %v, want 5.0", s.Value(1))
	}
}

func TestMinHorizontal(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(5), int64(3)}},
			{Name: "b", Values: []any{int64(4), int64(2), int64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	result, err := df.MinHorizontal("min_ab")
	if err != nil {
		t.Fatalf("min_horizontal: %v", err)
	}
	s, _ := result.GetColumn("min_ab")
	if v, ok := s.Value(0).(float64); !ok || v != 1.0 {
		t.Fatalf("min_ab[0]: got %v, want 1.0", s.Value(0))
	}
	if v, ok := s.Value(1).(float64); !ok || v != 2.0 {
		t.Fatalf("min_ab[1]: got %v, want 2.0", s.Value(1))
	}
}
