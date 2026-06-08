package functions

// Ported from py-polars/tests/unit/functions/test_repeat.py (py-1.28.1)
//
// gopolars has no top-level pl.repeat / pl.ones / pl.zeros constructors. Its
// Series.RepeatBy(n) matches Python's Expr.repeat_by: element i becomes a sub-list
// of n copies (List dtype).

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// repeat_by(2) on ints yields a List Series [[1,1],[2,2],[3,3]].
func TestRepeatByInts(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	out := s.RepeatBy(2)
	if out.DataType() != polars.List {
		t.Fatalf("dtype: got %v, want List", out.DataType())
	}
	if out.Len() != 3 {
		t.Fatalf("len: got %d, want 3", out.Len())
	}
	for i, base := range []int64{1, 2, 3} {
		row, ok := out.Value(i).([]any)
		if !ok || len(row) != 2 {
			t.Fatalf("row %d: got %T(%v), want list of 2", i, out.Value(i), out.Value(i))
		}
		for _, v := range row {
			if x, ok := v.(int64); !ok || x != base {
				t.Fatalf("row %d element: got %v, want %d", i, v, base)
			}
		}
	}
}

func TestRepeatByStrings(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.String, Values: []any{"foo", "bar"}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	out := s.RepeatBy(3)
	if out.Len() != 2 {
		t.Fatalf("len: got %d, want 2", out.Len())
	}
	row0, ok := out.Value(0).([]any)
	if !ok || len(row0) != 3 {
		t.Fatalf("row 0: got %T(%v), want list of 3", out.Value(0), out.Value(0))
	}
	if v, _ := row0[0].(string); v != "foo" {
		t.Fatalf("row0[0]: got %v, want foo", row0[0])
	}
}

// test_repeat_n_zero: n == 0 yields per-element empty lists, length unchanged.
func TestRepeatByZero(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	out := s.RepeatBy(0)
	if out.Len() != 2 {
		t.Fatalf("repeat_by(0) len: got %d, want 2", out.Len())
	}
	row, ok := out.Value(0).([]any)
	if !ok || len(row) != 0 {
		t.Fatalf("row 0: got %T(%v), want empty list", out.Value(0), out.Value(0))
	}
}

// Analogue of pl.repeat(value, n): build a constant series via ExtendConstant on an
// empty series. ExtendConstant(value, n) appends n copies of value.
func TestRepeatViaExtendConstant(t *testing.T) {
	t.Parallel()
	empty, err := polars.NewSeries(polars.NewSeriesInput{Name: "repeat", DType: polars.Int64, Values: []any{}})
	if err != nil {
		t.Fatalf("new empty series: %v", err)
	}
	out := empty.ExtendConstant(int64(7), 5)
	if out.Len() != 5 {
		t.Fatalf("len: got %d, want 5", out.Len())
	}
	for i := 0; i < 5; i++ {
		if v, ok := out.Value(i).(int64); !ok || v != 7 {
			t.Fatalf("idx %d: got %v, want 7", i, out.Value(i))
		}
	}
}
