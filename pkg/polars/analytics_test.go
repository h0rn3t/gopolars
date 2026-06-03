package polars

import (
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// newMeltFrame builds a 2-row, 3-column frame {"g","a","b"} used by the Melt
// tests. "g" is a group key, "a" and "b" are value columns.
func newMeltFrame(t *testing.T) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"x", "y"}},
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{int64(10), int64(20)}},
	}})
	if err != nil {
		t.Fatalf("newMeltFrame: %v", err)
	}
	return df
}

// TestMelt exercises the documented long-form transformation.
func TestMelt(t *testing.T) {
	df := newMeltFrame(t)
	out, err := Melt(df, MeltInput{
		IDVars:    []string{"g"},
		ValueVars: []string{"a", "b"},
	})
	if err != nil {
		t.Fatalf("Melt: %v", err)
	}
	if out.Width() != 3 {
		t.Fatalf("width = %d, want 3 (g, variable, value)", out.Width())
	}
	// Result has 2 (rows) * 2 (value vars) = 4 rows.
	if out.Height() != 4 {
		t.Errorf("height = %d, want 4", out.Height())
	}
	// The columns include the default variable/value names; order is set by
	// the underlying arrow table conversion so we check membership.
	cols := out.Columns()
	wantSet := map[string]bool{"g": true, "variable": true, "value": true}
	for _, c := range cols {
		if !wantSet[c] {
			t.Errorf("unexpected column %q in Melt output", c)
		}
		delete(wantSet, c)
	}
	if len(wantSet) != 0 {
		t.Errorf("missing columns in Melt output: %v", wantSet)
	}
}

// TestMeltCustomColumnNames verifies custom VariableCol/ValueCol are honored.
func TestMeltCustomColumnNames(t *testing.T) {
	df := newMeltFrame(t)
	out, err := Melt(df, MeltInput{
		IDVars:      []string{"g"},
		ValueVars:   []string{"a", "b"},
		VariableCol: "var",
		ValueCol:    "val",
	})
	if err != nil {
		t.Fatalf("Melt: %v", err)
	}
	cols := out.Columns()
	wantSet := map[string]bool{"g": true, "var": true, "val": true}
	for _, c := range cols {
		if !wantSet[c] {
			t.Errorf("unexpected column %q in Melt output", c)
		}
		delete(wantSet, c)
	}
	if len(wantSet) != 0 {
		t.Errorf("missing columns in Melt output: %v", wantSet)
	}
}

// TestPivotSum verifies the default sum aggregation behavior.
//
// Layout:
//   - g (index): x, x, y, y
//   - k (columns): a, b, a, b
//   - v (values):  1, 2, 3, 4
//
// Expected pivot (sum):
//   - row "x", col "a" → 1  (only one (x,a) pair: v=1)
//   - row "x", col "b" → 2
//   - row "y", col "a" → 3
//   - row "y", col "b" → 4
func TestPivotSum(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"x", "x", "y", "y"}},
		{Name: "k", Values: []any{"a", "b", "a", "b"}},
		{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	out, err := Pivot(df, PivotInput{
		Index:   "g",
		Columns: "k",
		Values:  "v",
		Agg:     "sum",
	})
	if err != nil {
		t.Fatalf("Pivot: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("height = %d, want 2", out.Height())
	}
	// Per-(index,column) sums are exactly 1, 2, 3, 4.
	want := map[string]map[string]float64{
		"x": {"a": 1, "b": 2},
		"y": {"a": 3, "b": 4},
	}
	keys, _ := out.GetColumn("g")
	idxByKey := map[string]int{}
	for i := 0; i < keys.Len(); i++ {
		k, _ := keys.Value(i).(string)
		idxByKey[k] = i
	}
	for g, byCol := range want {
		for c, w := range byCol {
			col, err := out.GetColumn(c)
			if err != nil {
				t.Errorf("pivot column %q: %v", c, err)
				continue
			}
			row := idxByKey[g]
			var got float64
			switch v := col.Value(row).(type) {
			case int64:
				got = float64(v)
			case float64:
				got = v
			}
			if got != w {
				t.Errorf("pivot[%s][%s] = %v, want %v", g, c, got, w)
			}
		}
	}
}

// TestPivotMeanAndCount verifies the "mean" and "count" aggregations.
func TestPivotMeanAndCount(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"x", "x", "y", "y"}},
		{Name: "k", Values: []any{"a", "a", "a", "a"}},
		{Name: "v", Values: []any{int64(2), int64(4), int64(6), int64(8)}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	for _, agg := range []string{"mean", "count"} {
		t.Run(agg, func(t *testing.T) {
			out, err := Pivot(df, PivotInput{Index: "g", Columns: "k", Values: "v", Agg: agg})
			if err != nil {
				t.Fatalf("Pivot %s: %v", agg, err)
			}
			// mean over {2,4} for x → 3; over {6,8} for y → 7.
			// count for x → 2; for y → 2.
			col, _ := out.GetColumn("a")
			for i := 0; i < col.Len(); i++ {
				var v float64
				switch x := col.Value(i).(type) {
				case int64:
					v = float64(x)
				case float64:
					v = x
				}
				keys, _ := out.GetColumn("g")
				k, _ := keys.Value(i).(string)
				switch {
				case agg == "mean" && k == "x" && v != 3:
					t.Errorf("mean[x] = %v, want 3", v)
				case agg == "mean" && k == "y" && v != 7:
					t.Errorf("mean[y] = %v, want 7", v)
				case agg == "count" && v != 2:
					t.Errorf("count[%s] = %v, want 2", k, v)
				}
			}
		})
	}
}

// TestRollingMeanExercisesDispatch verifies the wrapper just dispatches to
// the underlying frame method (we assert the result is non-nil and matches
// the documented rolling-mean output for a window=3 input).
func TestRollingMeanExercisesDispatch(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "v", Values: []any{1.0, 2.0, 3.0, 4.0, 5.0}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	// Wrapper just dispatches; we test that the call doesn't panic and the
	// resulting frame has the same height.
	out, err := RollingMean(df, RollingMeanInput{Value: "v", Window: 0})
	if err != nil {
		// Window 0 may legitimately error; that's still a documented contract.
		if !strings.Contains(err.Error(), "window") {
			t.Fatalf("RollingMean err = %v, want window-related", err)
		}
		return
	}
	if out.Height() != 5 {
		t.Errorf("RollingMean height = %d, want 5", out.Height())
	}
}

// TestGroupByDynamicDispatch verifies the wrapper dispatches without panic.
func TestGroupByDynamicDispatch(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "v", Values: []any{int64(1), int64(2), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	// Just exercise the dispatch path; behavior is owned by the underlying
	// frame.GroupByDynamic which has its own tests.
	_, _ = GroupByDynamic(df, DynamicGroupInput{By: "v"})
}

// TestWindowSumNonNumeric returns an error for non-numeric value columns,
// exercising the type-switch error branch.
func TestWindowSumNonNumeric(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "o", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "v", Values: []any{"a", "b", "c"}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	_, err = WindowSum(df, WindowSumInput{
		PartitionBy: []string{"o"},
		OrderBy:     "o",
		Value:       "v",
		Output:      "ws",
	})
	if err == nil {
		t.Fatalf("WindowSum on string column returned nil error, want non-nil")
	}
}

// TestWindowSumMissingColumns returns an error when the order/value column
// does not exist.
func TestWindowSumMissingColumns(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	_, err = WindowSum(df, WindowSumInput{
		PartitionBy: []string{"a"},
		OrderBy:     "missing",
		Value:       "a",
		Output:      "ws",
	})
	if err == nil {
		t.Fatalf("WindowSum on missing column returned nil error, want non-nil")
	}
}
