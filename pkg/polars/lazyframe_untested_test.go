package polars

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// newAggFrame builds a numeric frame for frame-level aggregation tests.
func newAggFrame(t *testing.T) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		{Name: "b", Values: []any{10.0, 20.0, 30.0, 40.0}},
	}})
	if err != nil {
		t.Fatalf("newAggFrame: %v", err)
	}
	return df
}

// TestLazyFrameAggregations exercises the whole-frame reductions that delegate
// to NodeFrameAgg (the Sum sibling is covered elsewhere). Each collapses the
// frame to a single row.
func TestLazyFrameAggregations(t *testing.T) {
	cases := []struct {
		name string
		make func(LazyFrame) LazyFrame
	}{
		{"Max", func(l LazyFrame) LazyFrame { return l.Max() }},
		{"Min", func(l LazyFrame) LazyFrame { return l.Min() }},
		{"Mean", func(l LazyFrame) LazyFrame { return l.Mean() }},
		{"Median", func(l LazyFrame) LazyFrame { return l.Median() }},
		{"Std", func(l LazyFrame) LazyFrame { return l.Std() }},
		{"Var", func(l LazyFrame) LazyFrame { return l.Var() }},
		{"Quantile", func(l LazyFrame) LazyFrame { return l.Quantile(0.5) }},
		{"NullCount", func(l LazyFrame) LazyFrame { return l.NullCount() }},
		{"Count", func(l LazyFrame) LazyFrame { return l.Count() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			df := newAggFrame(t)
			out, err := tc.make(df.Lazy()).Collect(context.Background())
			if err != nil {
				t.Fatalf("%s Collect: %v", tc.name, err)
			}
			if out.Height() != 1 {
				t.Errorf("%s Height = %d, want 1", tc.name, out.Height())
			}
			if out.Width() != 2 {
				t.Errorf("%s Width = %d, want 2", tc.name, out.Width())
			}
		})
	}
}

// TestLazyFrameAggregationValues spot-checks the numeric result of Max/Min/Mean.
func TestLazyFrameAggregationValues(t *testing.T) {
	df := newAggFrame(t)

	maxOut, err := df.Lazy().Max().Collect(context.Background())
	if err != nil {
		t.Fatalf("Max Collect: %v", err)
	}
	if got := cellFloat(t, maxOut, "b", 0); got != 40.0 {
		t.Errorf("Max(b) = %v, want 40", got)
	}

	minOut, err := df.Lazy().Min().Collect(context.Background())
	if err != nil {
		t.Fatalf("Min Collect: %v", err)
	}
	if got := cellFloat(t, minOut, "b", 0); got != 10.0 {
		t.Errorf("Min(b) = %v, want 10", got)
	}

	meanOut, err := df.Lazy().Mean().Collect(context.Background())
	if err != nil {
		t.Fatalf("Mean Collect: %v", err)
	}
	if got := cellFloat(t, meanOut, "b", 0); got != 25.0 {
		t.Errorf("Mean(b) = %v, want 25", got)
	}
}

// cellFloat reads a numeric cell from a collected frame as float64.
func cellFloat(t *testing.T, df DataFrame, name string, i int) float64 {
	t.Helper()
	col, err := df.GetColumn(name)
	if err != nil {
		t.Fatalf("GetColumn(%q): %v", name, err)
	}
	switch x := col.Value(i).(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	default:
		return 0
	}
}

// cell reads a raw cell value from a collected frame.
func cell(t *testing.T, df DataFrame, name string, i int) any {
	t.Helper()
	col, err := df.GetColumn(name)
	if err != nil {
		t.Fatalf("GetColumn(%q): %v", name, err)
	}
	return col.Value(i)
}

// TestLazyCastCollect verifies Cast changes the column dtype through the lazy path.
func TestLazyCastCollect(t *testing.T) {
	df := newAggFrame(t)
	out, err := df.Lazy().
		Cast(map[string]dtypes.DataType{"a": dtypes.Float64}).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("Cast Collect: %v", err)
	}
	col, err := out.GetColumn("a")
	if err != nil {
		t.Fatalf("GetColumn(a): %v", err)
	}
	if dt := col.DataType(); dt != dtypes.Float64 {
		t.Errorf("Cast a dtype = %s, want Float64", dt)
	}
}

// TestLazyCastEmptyMappingNoop confirms an empty Cast mapping is a no-op.
func TestLazyCastEmptyMappingNoop(t *testing.T) {
	df := newAggFrame(t)
	out, err := df.Lazy().Cast(map[string]dtypes.DataType{}).Collect(context.Background())
	if err != nil {
		t.Fatalf("Cast empty Collect: %v", err)
	}
	col, err := out.GetColumn("a")
	if err != nil {
		t.Fatalf("GetColumn(a): %v", err)
	}
	if col.DataType() != dtypes.Int64 {
		t.Errorf("empty Cast changed dtype to %s", col.DataType())
	}
}

// TestLazyShiftCollect verifies Shift preserves height and nulls the leading row.
func TestLazyShiftCollect(t *testing.T) {
	df := newAggFrame(t)
	out, err := df.Lazy().Shift(1).Collect(context.Background())
	if err != nil {
		t.Fatalf("Shift Collect: %v", err)
	}
	if out.Height() != 4 {
		t.Errorf("Shift Height = %d, want 4", out.Height())
	}
	if v := cell(t, out, "a", 0); v != nil {
		t.Errorf("Shift(1) row 0 = %v, want nil", v)
	}
}

// TestLazyWithRowIndexCollect verifies WithRowIndex/WithRowCount add an index column.
func TestLazyWithRowIndexCollect(t *testing.T) {
	df := newAggFrame(t)
	for _, name := range []string{"idx"} {
		out, err := df.Lazy().WithRowIndex(name, 0).Collect(context.Background())
		if err != nil {
			t.Fatalf("WithRowIndex Collect: %v", err)
		}
		if out.Width() != 3 {
			t.Errorf("WithRowIndex Width = %d, want 3", out.Width())
		}
		if _, err := out.GetColumn(name); err != nil {
			t.Errorf("WithRowIndex did not add column %q: %v", name, err)
		}
	}

	// WithRowCount is an alias for WithRowIndex.
	out, err := df.Lazy().WithRowCount("rc", 5).Collect(context.Background())
	if err != nil {
		t.Fatalf("WithRowCount Collect: %v", err)
	}
	if _, err := out.GetColumn("rc"); err != nil {
		t.Errorf("WithRowCount did not add column rc: %v", err)
	}
}

// TestLazySetSortedCollect confirms SetSorted is a transparent marker.
func TestLazySetSortedCollect(t *testing.T) {
	df := newAggFrame(t)
	out, err := df.Lazy().SetSorted("a").Collect(context.Background())
	if err != nil {
		t.Fatalf("SetSorted Collect: %v", err)
	}
	if out.Height() != 4 || out.Width() != 2 {
		t.Errorf("SetSorted changed shape: %dx%d, want 4x2", out.Height(), out.Width())
	}
}

// TestLazyClearCollect confirms Clear yields an empty frame keeping the schema.
func TestLazyClearCollect(t *testing.T) {
	df := newAggFrame(t)
	out, err := df.Lazy().Clear().Collect(context.Background())
	if err != nil {
		t.Fatalf("Clear Collect: %v", err)
	}
	if out.Height() != 0 {
		t.Errorf("Clear Height = %d, want 0", out.Height())
	}
	if out.Width() != 2 {
		t.Errorf("Clear Width = %d, want 2 (schema preserved)", out.Width())
	}
}

// TestLazyUpdateCollect exercises Update against another LazyFrame. The other
// plan must carry at least one node, so it is given a passthrough Filter.
func TestLazyUpdateCollect(t *testing.T) {
	df := newAggFrame(t)
	other := df.Lazy().Filter(Col("a").Gt(Lit(int64(0))))
	out, err := df.Lazy().Update(other).Collect(context.Background())
	if err != nil {
		t.Fatalf("Update Collect: %v", err)
	}
	if out.Height() != 4 {
		t.Errorf("Update Height = %d, want 4", out.Height())
	}
}

// newNestedFrame builds a frame with a list column and a struct column for the
// reshape/nested lazy tests.
func newNestedFrame(t *testing.T) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2)}},
		{Name: "tags", Values: []any{[]any{"a", "b"}, []any{"c"}}},
		{Name: "meta", Values: []any{
			map[string]any{"x": int64(1), "y": "u"},
			map[string]any{"x": int64(2), "y": "v"},
		}},
	}})
	if err != nil {
		t.Fatalf("newNestedFrame: %v", err)
	}
	return df
}

// TestLazyExplodeCollect verifies Explode expands a list column row-wise.
func TestLazyExplodeCollect(t *testing.T) {
	df := newNestedFrame(t)
	out, err := df.Lazy().Explode("tags").Collect(context.Background())
	if err != nil {
		t.Fatalf("Explode Collect: %v", err)
	}
	if out.Height() != 3 {
		t.Errorf("Explode Height = %d, want 3", out.Height())
	}
}

// TestLazyFlattenStructCollect verifies FlattenStruct expands a struct column
// into prefixed scalar columns.
func TestLazyFlattenStructCollect(t *testing.T) {
	df := newNestedFrame(t)
	out, err := df.Lazy().FlattenStruct("meta", "meta_").Collect(context.Background())
	if err != nil {
		t.Fatalf("FlattenStruct Collect: %v", err)
	}
	if _, err := out.GetColumn("meta_x"); err != nil {
		t.Errorf("FlattenStruct missing column meta_x: %v", err)
	}
}

// TestLazyUnnestCollect verifies Unnest expands a struct column into its fields.
func TestLazyUnnestCollect(t *testing.T) {
	df := newNestedFrame(t)
	out, err := df.Lazy().Unnest("meta").Collect(context.Background())
	if err != nil {
		t.Fatalf("Unnest Collect: %v", err)
	}
	if _, err := out.GetColumn("x"); err != nil {
		t.Errorf("Unnest missing field column x: %v", err)
	}
}

// newWideFrame builds a wide frame for Unpivot/Pivot lazy tests.
func newWideFrame(t *testing.T) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "city", Values: []any{"kyiv", "lviv"}},
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{int64(10), int64(20)}},
	}})
	if err != nil {
		t.Fatalf("newWideFrame: %v", err)
	}
	return df
}

// TestLazyUnpivotCollect verifies Unpivot melts value columns into long form.
func TestLazyUnpivotCollect(t *testing.T) {
	df := newWideFrame(t)
	out, err := df.Lazy().Unpivot(MeltInput{
		IDVars:      []string{"city"},
		ValueVars:   []string{"a", "b"},
		VariableCol: "metric",
		ValueCol:    "val",
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("Unpivot Collect: %v", err)
	}
	if out.Height() != 4 {
		t.Errorf("Unpivot Height = %d, want 4", out.Height())
	}
}

// TestLazyPivotCollect verifies Pivot reshapes long form back to wide.
func TestLazyPivotCollect(t *testing.T) {
	df := newWideFrame(t)
	melted := df.Lazy().Unpivot(MeltInput{
		IDVars:      []string{"city"},
		ValueVars:   []string{"a", "b"},
		VariableCol: "metric",
		ValueCol:    "val",
	})
	out, err := melted.Pivot(PivotInput{
		Index:   "city",
		Columns: "metric",
		Values:  "val",
		Agg:     "sum",
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("Pivot Collect: %v", err)
	}
	if out.Width() != 3 {
		t.Errorf("Pivot Width = %d, want 3", out.Width())
	}
}

// TestLazyFillNaNCollect verifies FillNaN replaces NaN values.
func TestLazyFillNaNCollect(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "b", Values: []any{1.0, math.NaN(), 3.0}},
	}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	out, err := df.Lazy().FillNaN(0).Collect(context.Background())
	if err != nil {
		t.Fatalf("FillNaN Collect: %v", err)
	}
	got := cellFloat(t, out, "b", 1)
	if math.IsNaN(got) {
		t.Errorf("FillNaN left NaN at row 1")
	}
	if got != 0 {
		t.Errorf("FillNaN[1] = %v, want 0", got)
	}
}

// TestLazyInterpolateCollect verifies Interpolate fills interior nulls.
func TestLazyInterpolateCollect(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "b", Values: []any{1.0, nil, 3.0}},
	}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	out, err := df.Lazy().Interpolate("b").Collect(context.Background())
	if err != nil {
		t.Fatalf("Interpolate Collect: %v", err)
	}
	if out.Height() != 3 {
		t.Errorf("Interpolate Height = %d, want 3", out.Height())
	}
	if v := cell(t, out, "b", 1); v == nil {
		t.Errorf("Interpolate did not fill interior null")
	}
}

// TestLazyGroupByDynamicCollect exercises the lazy temporal GroupByDynamic path.
func TestLazyGroupByDynamicCollect(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "ts", Values: []any{
			time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 10, 11, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 10, 13, 0, 0, 0, time.UTC),
		}},
		{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
	}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	out, err := df.Lazy().GroupByDynamic(DynamicGroupInput{
		By:      "ts",
		Every:   2 * time.Hour,
		Period:  2 * time.Hour,
		Closed:  "left",
		Label:   "left",
		AggExpr: Sum(Col("v")),
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("GroupByDynamic Collect: %v", err)
	}
	if out.Height() == 0 {
		t.Errorf("GroupByDynamic produced empty frame")
	}
}
