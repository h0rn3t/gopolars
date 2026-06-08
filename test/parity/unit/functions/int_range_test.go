package functions

// Ported from py-polars/tests/unit/functions/range/test_int_range.py (py-1.28.1)
//
// gopolars has no top-level pl.int_range(start, end, step) / pl.arange generator
// expression. The nearest available primitive is DataFrame.WithRowIndex, which
// materialises a 0..n contiguous integer column; we port that to document what
// IS available, plus a gap marker for the general range generator.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestWithRowIndexAsIntRange(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
		},
	})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.WithRowIndex("idx", 0)
	if err != nil {
		t.Fatalf("with_row_index: %v", err)
	}
	idx, err := out.GetColumn("idx")
	if err != nil {
		t.Fatalf("get idx: %v", err)
	}
	for i := 0; i < 4; i++ {
		got := idx.Value(i)
		switch v := got.(type) {
		case int64:
			if int(v) != i {
				t.Fatalf("idx[%d]: got %d, want %d", i, v, i)
			}
		case uint32:
			if int(v) != i {
				t.Fatalf("idx[%d]: got %d, want %d", i, v, i)
			}
		default:
			t.Fatalf("idx[%d]: unexpected type %T", i, got)
		}
	}
}

// WithRowIndex honours a non-zero offset (analogue of int_range start).
func TestWithRowIndexOffset(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.WithRowIndex("idx", 100)
	if err != nil {
		t.Fatalf("with_row_index offset: %v", err)
	}
	idx, err := out.GetColumn("idx")
	if err != nil {
		t.Fatalf("get idx: %v", err)
	}
	first := idx.Value(0)
	switch v := first.(type) {
	case int64:
		if v != 100 {
			t.Fatalf("idx[0]: got %d, want 100", v)
		}
	case uint32:
		if v != 100 {
			t.Fatalf("idx[0]: got %d, want 100", v)
		}
	default:
		t.Fatalf("idx[0]: unexpected type %T", first)
	}
}
