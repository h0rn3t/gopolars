package exec

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// execNode runs a single logical node through the engine and fails on error.
func execNode(t *testing.T, src frame.DataFrame, n logical.Node) frame.DataFrame {
	t.Helper()
	out, err := New().Execute(context.Background(), src, []logical.Node{n})
	if err != nil {
		t.Fatalf("execute %s: %v", n.Type, err)
	}
	return out
}

// execNodeErr runs a single node and asserts it errors.
func execNodeErr(t *testing.T, src frame.DataFrame, n logical.Node) {
	t.Helper()
	if _, err := New().Execute(context.Background(), src, []logical.Node{n}); err == nil {
		t.Fatalf("expected error for %s node", n.Type)
	}
}

// TestExecuteExplodeNode covers the NodeExplode arm on a list column.
func TestExecuteExplodeNode(t *testing.T) {
	t.Parallel()

	src := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}},
		frame.SeriesInput{Name: "tags", DType: dtypes.List, Values: []any{
			[]any{int64(10), int64(11)},
			[]any{int64(20)},
		}},
	)
	out := execNode(t, src, logical.Node{Type: logical.NodeExplode, Columns: []string{"tags"}})
	if out.Height() != 3 {
		t.Fatalf("explode height = %d, want 3", out.Height())
	}

	// Missing column -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeExplode, Columns: []string{"nope"}})
}

// TestExecuteFlattenNode covers NodeFlatten (success + missing-target + missing
// column error).
func TestExecuteFlattenNode(t *testing.T) {
	t.Parallel()

	src := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}},
		frame.SeriesInput{Name: "rec", DType: dtypes.Struct, Values: []any{
			map[string]any{"x": int64(10)},
			map[string]any{"x": int64(20)},
		}},
	)
	out := execNode(t, src, logical.Node{Type: logical.NodeFlatten, Columns: []string{"rec"}, Prefix: "rec_"})
	if _, ok := out.Series("rec_x"); !ok {
		t.Fatal("flatten did not produce rec_x")
	}

	// No target column -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeFlatten, Columns: nil})
	// Missing column -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeFlatten, Columns: []string{"missing"}})
}

// TestExecuteUnnestNode covers NodeUnnest on a struct column.
func TestExecuteUnnestNode(t *testing.T) {
	t.Parallel()

	src := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1)}},
		frame.SeriesInput{Name: "rec", DType: dtypes.Struct, Values: []any{
			map[string]any{"x": int64(10), "y": int64(11)},
		}},
	)
	out := execNode(t, src, logical.Node{Type: logical.NodeUnnest, Columns: []string{"rec"}})
	if _, ok := out.Series("x"); !ok {
		t.Fatal("unnest did not produce x")
	}
}

// TestExecuteMeltUnpivotNodes covers NodeMelt and NodeUnpivot, plus their
// incomplete-metadata error arms.
func TestExecuteMeltUnpivotNodes(t *testing.T) {
	t.Parallel()

	src := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}},
		frame.SeriesInput{Name: "a", Values: []any{int64(10), int64(20)}},
		frame.SeriesInput{Name: "b", Values: []any{int64(30), int64(40)}},
	)

	// Columns = [id, a, b]; Strings = [varName, valName, idCount("1")].
	melt := execNode(t, src, logical.Node{
		Type:    logical.NodeMelt,
		Columns: []string{"id", "a", "b"},
		Strings: []string{"variable", "value", "1"},
	})
	// 2 rows * 2 value vars = 4 rows.
	if melt.Height() != 4 {
		t.Fatalf("melt height = %d, want 4", melt.Height())
	}
	if _, ok := melt.Series("variable"); !ok {
		t.Fatal("melt missing variable column")
	}

	unpivot := execNode(t, src, logical.Node{
		Type:    logical.NodeUnpivot,
		Columns: []string{"id", "a", "b"},
		Strings: []string{"variable", "value", "1"},
	})
	if unpivot.Height() != 4 {
		t.Fatalf("unpivot height = %d, want 4", unpivot.Height())
	}

	// Incomplete metadata (<3 strings) -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeMelt, Columns: []string{"id"}, Strings: []string{"v"}})
	// Non-integer id count -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeMelt, Columns: []string{"id", "a"}, Strings: []string{"v", "val", "x"}})
	// Out-of-range id count -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeMelt, Columns: []string{"id"}, Strings: []string{"v", "val", "9"}})
}

// TestExecuteWithRowIdxNode covers NodeWithRowIdx (with offset) and its
// missing-name error.
func TestExecuteWithRowIdxNode(t *testing.T) {
	t.Parallel()

	src := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3)}})

	out := execNode(t, src, logical.Node{Type: logical.NodeWithRowIdx, Strings: []string{"row", "10"}})
	row, ok := out.Series("row")
	if !ok {
		t.Fatal("with_row_index missing row column")
	}
	if row.Value(0) != int64(10) {
		t.Fatalf("row[0] = %v, want 10 (offset)", row.Value(0))
	}

	// No offset string -> defaults to 0.
	out0 := execNode(t, src, logical.Node{Type: logical.NodeWithRowIdx, Strings: []string{"r"}})
	r0, _ := out0.Series("r")
	if r0.Value(0) != int64(0) {
		t.Fatalf("default offset row[0] = %v, want 0", r0.Value(0))
	}

	// Missing name -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeWithRowIdx, Strings: nil})
}

// TestExecuteFillNaNNode covers NodeFillNaN (replacing NaN float values).
func TestExecuteFillNaNNode(t *testing.T) {
	t.Parallel()

	nan := func() float64 { var z float64; return z / z }()
	src := mustFrame(t, frame.SeriesInput{Name: "v", Values: []any{1.0, nan, 3.0}})

	out := execNode(t, src, logical.Node{Type: logical.NodeFillNaN, Strings: []string{"0"}})
	col, _ := out.Series("v")
	if col.Value(1) != 0.0 {
		t.Fatalf("fill_nan v[1] = %v, want 0", col.Value(1))
	}

	// No value string -> defaults to 0.0 (still valid).
	out2 := execNode(t, src, logical.Node{Type: logical.NodeFillNaN})
	col2, _ := out2.Series("v")
	if col2.Value(1) != 0.0 {
		t.Fatalf("default fill_nan v[1] = %v, want 0", col2.Value(1))
	}
}

// TestExecuteInterpolateNode covers NodeInterpolate filling a gap linearly.
func TestExecuteInterpolateNode(t *testing.T) {
	t.Parallel()

	src := mustFrame(t, frame.SeriesInput{Name: "v", Values: []any{1.0, nil, 3.0}})
	out := execNode(t, src, logical.Node{Type: logical.NodeInterpolate, Columns: []string{"v"}})
	col, _ := out.Series("v")
	if col.Value(1) != 2.0 {
		t.Fatalf("interpolate v[1] = %v, want 2.0", col.Value(1))
	}
}

// TestExecuteUpdateNode covers NodeUpdate, which executes a sub-plan against the
// source and merges matching columns.
func TestExecuteUpdateNode(t *testing.T) {
	t.Parallel()

	src := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}},
		frame.SeriesInput{Name: "v", Values: []any{int64(10), int64(20)}},
	)

	// Sub-plan selects v from source and doubles it; Update replaces matching
	// column v in current with the sub-plan's v.
	out := execNode(t, src, logical.Node{
		Type: logical.NodeUpdate,
		Plan: []logical.Node{
			{Type: logical.NodeWithCols, Exprs: []expr.Expr{
				expr.Col("v").Mul(expr.Lit(int64(2))).Alias("v"),
			}},
		},
	})
	col, _ := out.Series("v")
	if col.Value(0) != int64(20) {
		t.Fatalf("update v[0] = %v, want 20", col.Value(0))
	}

	// Empty plan -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeUpdate, Plan: nil})
}

// TestExecutePivotNode covers NodePivot, including its incomplete-columns error.
func TestExecutePivotNode(t *testing.T) {
	t.Parallel()

	src := mustFrame(t,
		frame.SeriesInput{Name: "idx", Values: []any{"r1", "r1", "r2"}},
		frame.SeriesInput{Name: "col", Values: []any{"x", "y", "x"}},
		frame.SeriesInput{Name: "val", Values: []any{int64(1), int64(2), int64(3)}},
	)

	// Columns = [index, columns, values]; Strings[0] = agg.
	out := execNode(t, src, logical.Node{
		Type:    logical.NodePivot,
		Columns: []string{"idx", "col", "val"},
		Strings: []string{"sum"},
	})
	if _, ok := out.Series("idx"); !ok {
		t.Fatal("pivot missing index column idx")
	}
	if out.Height() != 2 {
		t.Fatalf("pivot height = %d, want 2 (r1,r2)", out.Height())
	}

	// Fewer than 3 columns -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodePivot, Columns: []string{"idx", "col"}})
}

// TestExecuteRollingNode covers NodeRolling over a datetime order column and its
// incomplete-metadata and bad-parse error arms.
func TestExecuteRollingNode(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	src := mustFrame(t,
		frame.SeriesInput{Name: "ts", Values: []any{base, base.Add(time.Hour), base.Add(2 * time.Hour)}},
		frame.SeriesInput{Name: "v", Values: []any{1.0, 2.0, 3.0}},
	)

	windowNS := strconv.FormatInt(int64(24*time.Hour), 10)
	// Columns = [by, value, output]; Strings = [windowNS, minRows, closed].
	out := execNode(t, src, logical.Node{
		Type:    logical.NodeRolling,
		Columns: []string{"ts", "v", "roll"},
		Strings: []string{windowNS, "1", "both"},
	})
	if _, ok := out.Series("roll"); !ok {
		t.Fatal("rolling missing output column roll")
	}

	// Incomplete metadata (<3 columns) -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeRolling, Columns: []string{"ts", "v"}, Strings: []string{windowNS, "1"}})
	// Bad window parse -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeRolling, Columns: []string{"ts", "v", "roll"}, Strings: []string{"notint", "1"}})
	// Bad minRows parse -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeRolling, Columns: []string{"ts", "v", "roll"}, Strings: []string{windowNS, "notint"}})
}

// TestExecuteDynamicNode covers NodeDynamic (group_by_dynamic) plus its
// incomplete-metadata and parse error arms.
func TestExecuteDynamicNode(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	src := mustFrame(t,
		frame.SeriesInput{Name: "ts", Values: []any{base, base.Add(30 * time.Minute), base.Add(2 * time.Hour)}},
		frame.SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3)}},
	)

	everyNS := strconv.FormatInt(int64(time.Hour), 10)
	periodNS := everyNS
	offsetNS := "0"
	// Columns = [by, windowColumn]; Strings = [every, period, offset, closed, label].
	out := execNode(t, src, logical.Node{
		Type:    logical.NodeDynamic,
		Columns: []string{"ts", "win"},
		Strings: []string{everyNS, periodNS, offsetNS, "both", "left"},
		Exprs:   []expr.Expr{expr.Sum(expr.Col("v"))},
	})
	if out.Height() == 0 {
		t.Fatal("dynamic produced no groups")
	}

	// Incomplete metadata -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeDynamic, Columns: []string{"ts"}, Strings: []string{everyNS}})
	// Bad every parse -> error.
	execNodeErr(t, src, logical.Node{
		Type:    logical.NodeDynamic,
		Columns: []string{"ts", "win"},
		Strings: []string{"notint", periodNS, offsetNS, "both", "left"},
		Exprs:   []expr.Expr{expr.Sum(expr.Col("v"))},
	})
}

// TestExecuteSetSortedAndDropNaN covers NodeSetSorted (missing column error) and
// NodeDropNans.
func TestExecuteSetSortedAndDropNaN(t *testing.T) {
	t.Parallel()

	src := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}})

	// set_sorted without a column -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeSetSorted, Columns: nil})

	nan := func() float64 { var z float64; return z / z }()
	nanSrc := mustFrame(t, frame.SeriesInput{Name: "v", Values: []any{1.0, nan, 3.0}})
	out := execNode(t, nanSrc, logical.Node{Type: logical.NodeDropNans, Columns: []string{"v"}})
	if out.Height() != 2 {
		t.Fatalf("drop_nans height = %d, want 2", out.Height())
	}
}

// TestExecuteFillNullNoExpr covers the fill_null error arm and the FrameAgg
// missing-aggregation-type error arm.
func TestExecuteFillNullAndFrameAggErrors(t *testing.T) {
	t.Parallel()

	src := mustFrame(t, frame.SeriesInput{Name: "v", Values: []any{int64(1), nil}})

	// fill_null with no value expression -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeFillNull, Exprs: nil})
	// frame_agg with no aggregation type -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeFrameAgg, Strings: nil})
}

// TestApplySetOpDirect covers applySetOp's union / union-all branches and the
// unsupported-operation error arm directly.
func TestApplySetOpDirect(t *testing.T) {
	t.Parallel()

	left := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}})
	right := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(2), int64(3)}})

	union, err := applySetOp(left, right, "UNION")
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	if union.Height() != 3 {
		t.Fatalf("union height = %d, want 3 (deduped)", union.Height())
	}

	all, err := applySetOp(left, right, "  union all  ")
	if err != nil {
		t.Fatalf("union all: %v", err)
	}
	if all.Height() != 4 {
		t.Fatalf("union all height = %d, want 4", all.Height())
	}

	intersect, err := applySetOp(left, right, "intersect")
	if err != nil {
		t.Fatalf("intersect: %v", err)
	}
	if intersect.Height() != 1 {
		t.Fatalf("intersect height = %d, want 1", intersect.Height())
	}

	except, err := applySetOp(left, right, "except")
	if err != nil {
		t.Fatalf("except: %v", err)
	}
	if except.Height() != 1 {
		t.Fatalf("except height = %d, want 1", except.Height())
	}

	if _, err := applySetOp(left, right, "bogus"); err == nil {
		t.Fatal("expected error for unsupported set operation")
	}
}

// TestExecuteSetOpUnionViaEngine drives a NodeSetOp union through the engine and
// the missing-operation error arm.
func TestExecuteSetOpUnionViaEngine(t *testing.T) {
	t.Parallel()

	src := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3)}})

	out, err := New().Execute(context.Background(), src, []logical.Node{
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Le(expr.Lit(int64(1)))}},
		{
			Type:    logical.NodeSetOp,
			Strings: []string{"union all"},
			Plan: []logical.Node{
				{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Ge(expr.Lit(int64(3)))}},
			},
		},
	})
	if err != nil {
		t.Fatalf("union via engine: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("union all via engine height = %d, want 2", out.Height())
	}

	// set_op node with no operation string -> error.
	execNodeErr(t, src, logical.Node{Type: logical.NodeSetOp, Strings: nil})
}

// TestToFloatDirect covers toFloat's int64 / float64 / unsupported branches.
func TestToFloatDirect(t *testing.T) {
	t.Parallel()

	if f, ok := toFloat(int64(7)); !ok || f != 7.0 {
		t.Fatalf("toFloat(int64) = %v,%v want 7,true", f, ok)
	}
	if f, ok := toFloat(2.5); !ok || f != 2.5 {
		t.Fatalf("toFloat(float64) = %v,%v want 2.5,true", f, ok)
	}
	if _, ok := toFloat("nope"); ok {
		t.Fatal("toFloat(string) should be false")
	}
}

// TestCompareForOrderDirect covers compareForOrder's nil, int64, float64, string,
// time.Time, and type-mismatch branches.
func TestCompareForOrderDirect(t *testing.T) {
	t.Parallel()

	if compareForOrder(nil, nil) != 0 {
		t.Fatal("nil,nil should compare equal")
	}
	if compareForOrder(nil, int64(1)) != -1 {
		t.Fatal("nil < value should be -1")
	}
	if compareForOrder(int64(1), nil) != 1 {
		t.Fatal("value > nil should be 1")
	}

	if compareForOrder(int64(1), int64(2)) != -1 || compareForOrder(int64(2), int64(1)) != 1 || compareForOrder(int64(1), int64(1)) != 0 {
		t.Fatal("int64 comparison incorrect")
	}
	if compareForOrder(1.0, 2.0) != -1 || compareForOrder(2.0, 1.0) != 1 {
		t.Fatal("float64 comparison incorrect")
	}
	if compareForOrder("a", "b") != -1 || compareForOrder("b", "a") != 1 || compareForOrder("a", "a") != 0 {
		t.Fatal("string comparison incorrect")
	}

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)
	if compareForOrder(t0, t1) != -1 || compareForOrder(t1, t0) != 1 || compareForOrder(t0, t0) != 0 {
		t.Fatal("time comparison incorrect")
	}

	// Type mismatches return 0 (incomparable).
	if compareForOrder(int64(1), 2.0) != 0 {
		t.Fatal("int64 vs float64 should be 0")
	}
	if compareForOrder(1.0, "x") != 0 {
		t.Fatal("float64 vs string should be 0")
	}
	if compareForOrder("x", int64(1)) != 0 {
		t.Fatal("string vs int64 should be 0")
	}
	if compareForOrder(t0, int64(1)) != 0 {
		t.Fatal("time vs int64 should be 0")
	}
}

// TestComputePartitionAggDirect covers computePartitionAgg's count-with-target,
// numeric-error, float64-sum, min/max, and unsupported-function branches.
func TestComputePartitionAggDirect(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		frame.SeriesInput{Name: "v", Values: []any{1.0, nil, 3.0}},
		frame.SeriesInput{Name: "label", Values: []any{"a", "b", "c"}},
	)
	rows := []int{0, 1, 2}

	// count with a named target excludes nulls.
	c, err := computePartitionAgg(df, rows, "count", "v")
	if err != nil || c != int64(2) {
		t.Fatalf("count target = %v err=%v, want 2", c, err)
	}

	// count missing target column -> error.
	if _, err := computePartitionAgg(df, rows, "count", "nope"); err == nil {
		t.Fatal("expected error for missing count target")
	}

	// sum over float64 returns float64.
	s, err := computePartitionAgg(df, rows, "sum", "v")
	if err != nil || s != 4.0 {
		t.Fatalf("sum = %v err=%v, want 4.0", s, err)
	}

	// sum over non-numeric column -> error.
	if _, err := computePartitionAgg(df, rows, "sum", "label"); err == nil {
		t.Fatal("expected error for non-numeric sum")
	}

	// max over all-null rows returns nil best.
	allNull := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{nil, nil}, DType: dtypes.Float64})
	best, err := computePartitionAgg(allNull, []int{0, 1}, "max", "x")
	if err != nil || best != nil {
		t.Fatalf("max all-null = %v err=%v, want nil", best, err)
	}

	// mean over all-null rows returns nil.
	m, err := computePartitionAgg(allNull, []int{0, 1}, "mean", "x")
	if err != nil || m != nil {
		t.Fatalf("mean all-null = %v err=%v, want nil", m, err)
	}

	// unsupported function -> error.
	if _, err := computePartitionAgg(df, rows, "bogus", "v"); err == nil {
		t.Fatal("expected error for unsupported window function")
	}

	// missing target for non-count -> error.
	if _, err := computePartitionAgg(df, rows, "sum", "missing"); err == nil {
		t.Fatal("expected error for missing sum target")
	}
}

// TestApplyWindowsLastValueAndLead covers last_value and lead window functions
// plus the partition-agg sum over int64 (returns int64).
func TestApplyWindowsLastValueAndLead(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		frame.SeriesInput{Name: "grp", Values: []any{"a", "a", "a"}},
		frame.SeriesInput{Name: "val", Values: []any{int64(10), int64(20), int64(30)}},
	)

	last, err := applyWindows(df, []logical.WindowSpec{{
		Func: "last_value", Target: "val", Alias: "lv", PartitionBy: []string{"grp"}, OrderBy: []string{"val"},
	}})
	if err != nil {
		t.Fatalf("last_value: %v", err)
	}
	lv := colValues(t, last, "lv")
	for i := range lv {
		if lv[i] != int64(30) {
			t.Fatalf("last_value row %d = %v, want 30", i, lv[i])
		}
	}

	lead, err := applyWindows(df, []logical.WindowSpec{{
		Func: "lead", Target: "val", Alias: "next", PartitionBy: []string{"grp"}, OrderBy: []string{"val"}, Offset: 1, Default: int64(-1),
	}})
	if err != nil {
		t.Fatalf("lead: %v", err)
	}
	next := colValues(t, lead, "next")
	want := []any{int64(20), int64(30), int64(-1)}
	for i := range next {
		if next[i] != want[i] {
			t.Fatalf("lead row %d = %v, want %v", i, next[i], want[i])
		}
	}

	// first_value missing target -> error.
	if _, err := applyWindows(df, []logical.WindowSpec{{
		Func: "first_value", Target: "nope", Alias: "fv", PartitionBy: []string{"grp"}, OrderBy: []string{"val"},
	}}); err == nil {
		t.Fatal("expected error for missing first_value target")
	}
}
