package toplevel

// Ported from py-polars/tests/unit/test_scalar.py (py-1.28.1, feasible subset).
//
// Tests needing pl.lit(None).cast(dtype) across Null/Int32/Binary/Array dtypes,
// pl.len()/pl.first() aggregations in select, or n_chunks introspection are
// omitted — those features are not present in gopolars (see PARITY_TRACKER GAP rows).

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_scalar_19957: a scalar literal added via with_columns broadcasts to every
// row, and survives a subsequent gather_every.
func TestScalarLitBroadcastGatherEvery(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "foo", Values: []any{int64(1), int64(1), int64(1), int64(1), int64(1)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	withBar, err := df.WithColumns(polars.Lit(int64(1)).Alias("bar"))
	if err != nil {
		t.Fatalf("with_columns: %v", err)
	}
	got := withBar.GatherEvery(2, 0)
	if got.Height() != 3 {
		t.Fatalf("height after gather_every(2): got %d, want 3", got.Height())
	}
	bar, _ := got.GetColumn("bar")
	foo, _ := got.GetColumn("foo")
	for i := 0; i < 3; i++ {
		if v, _ := bar.Value(i).(int64); v != 1 {
			t.Fatalf("bar[%d]: got %v, want 1", i, bar.Value(i))
		}
		if v, _ := foo.Value(i).(int64); v != 1 {
			t.Fatalf("foo[%d]: got %v, want 1", i, foo.Value(i))
		}
	}
}

// test_split_scalar_21581: a negative shift inside with_columns introduces a
// trailing null, a Boolean literal broadcasts to every row, and filtering on a
// comparison drops the null-predicate row (Polars treats a null predicate as
// false). Final: a=[1,2], next_a=[2,3], lit=[False,False].
func TestScalarShiftLitFilter(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", DType: polars.Float64, Values: []any{1.0, 2.0, 3.0}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	withCols, err := df.WithColumns(
		polars.Col("a").Shift(-1).Alias("next_a"),
		polars.Lit(true).Alias("lit"),
	)
	if err != nil {
		t.Fatalf("with_columns: %v", err)
	}
	// next_a = [2.0, 3.0, null] — shift(-1) moves values up and null-fills the tail.
	nextA, _ := withCols.GetColumn("next_a")
	for i, w := range []float64{2.0, 3.0} {
		if v, _ := nextA.Value(i).(float64); v != w {
			t.Fatalf("next_a[%d]: got %v, want %v", i, nextA.Value(i), w)
		}
	}
	if nextA.Value(2) != nil {
		t.Fatalf("next_a[2]: got %v, want nil (shift tail)", nextA.Value(2))
	}
	// Filter: `next_a != 99.0` is null on the trailing row, so that row is dropped.
	filtered, err := withCols.Filter(polars.Col("next_a").Ne(polars.Lit(99.0)))
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	out, err := filtered.WithColumns(polars.Lit(false).Alias("lit"))
	if err != nil {
		t.Fatalf("with_columns lit=false: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("height after filter: got %d, want 2", out.Height())
	}
	a, _ := out.GetColumn("a")
	na, _ := out.GetColumn("next_a")
	lit, _ := out.GetColumn("lit")
	for i, w := range []float64{1.0, 2.0} {
		if v, _ := a.Value(i).(float64); v != w {
			t.Fatalf("a[%d]: got %v, want %v", i, a.Value(i), w)
		}
	}
	for i, w := range []float64{2.0, 3.0} {
		if v, _ := na.Value(i).(float64); v != w {
			t.Fatalf("next_a[%d]: got %v, want %v", i, na.Value(i), w)
		}
	}
	for i := 0; i < 2; i++ {
		if v, _ := lit.Value(i).(bool); v != false {
			t.Fatalf("lit[%d]: got %v, want false", i, lit.Value(i))
		}
	}
}
