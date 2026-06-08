package datatypes

// Ported from py-polars/tests/unit/datatypes/test_integer.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_integer_float_functions: is_finite/is_infinite/is_nan/is_not_nan on ints.
func TestIntegerFloatFunctions(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{{Name: "a", Values: []any{int64(1), int64(2)}}},
	})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Select(
		polars.Col("a").IsFinite().Alias("finite"),
		polars.Col("a").IsInfinite().Alias("infinite"),
		polars.Col("a").IsNan().Alias("nan"),
		polars.Col("a").IsNotNan().Alias("not_na"),
	)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	checkBool := func(col string, want []bool) {
		s, err := out.GetColumn(col)
		if err != nil {
			t.Fatalf("get %s: %v", col, err)
		}
		for i, w := range want {
			if v, ok := s.Value(i).(bool); !ok || v != w {
				t.Fatalf("%s[%d]: got %v, want %v", col, i, s.Value(i), w)
			}
		}
	}
	checkBool("finite", []bool{true, true})
	checkBool("infinite", []bool{false, false})
	checkBool("nan", []bool{false, false})
	checkBool("not_na", []bool{true, true})
}

// test_int_negate_operation: Series.not_() on Int64 is the bitwise complement
// (^v == -v-1), matching Polars.
func TestIntNegateOperation(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4), int64(50912341409)}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	out := s.Not_()
	if out.DataType() != polars.Int64 {
		t.Fatalf("not_ dtype: got %v, want Int64", out.DataType())
	}
	for i, w := range []int64{-2, -3, -4, -5, -50912341410} {
		if v, ok := out.Value(i).(int64); !ok || v != w {
			t.Fatalf("not_[%d]: got %T(%v), want int64 %d", i, out.Value(i), out.Value(i), w)
		}
	}
}
