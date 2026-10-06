package exec

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// nodesSource builds a frame with groups and a null for the null-handling nodes:
//
//	id:  1 2 3 4
//	grp: a a b b
//	val: 10 <nil> 30 40
func nodesSource(t *testing.T) frame.DataFrame {
	t.Helper()
	return mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		frame.SeriesInput{Name: "grp", Values: []any{"a", "a", "b", "b"}},
		frame.SeriesInput{Name: "val", Values: []any{int64(10), nil, int64(30), int64(40)}},
	)
}

// TestExecuteNodeCoverage drives a broad set of logical node types through the
// engine, asserting a basic correctness property for each. Together these cover
// the bulk of the executeOptimized switch.
func TestExecuteNodeCoverage(t *testing.T) {
	t.Parallel()

	src := nodesSource(t)
	engine := New()
	ctx := context.Background()

	run := func(nodes ...logical.Node) frame.DataFrame {
		out, err := engine.Execute(ctx, src, nodes)
		if err != nil {
			t.Fatalf("execute %v: %v", nodes[0].Type, err)
		}
		return out
	}

	t.Run("with_columns", func(t *testing.T) {
		out := run(logical.Node{Type: logical.NodeWithCols, Exprs: []expr.Expr{
			expr.Col("id").Mul(expr.Lit(int64(2))).Alias("id2"),
		}})
		col, ok := out.Series("id2")
		if !ok || col.Value(0) != int64(2) {
			t.Fatalf("with_columns id2 = %v ok=%v", col.Value(0), ok)
		}
	})

	t.Run("rename", func(t *testing.T) {
		out := run(logical.Node{Type: logical.NodeRename, Strings: []string{"id", "ident"}})
		if _, ok := out.Series("ident"); !ok {
			t.Fatal("rename: column ident missing")
		}
	})

	t.Run("drop", func(t *testing.T) {
		out := run(logical.Node{Type: logical.NodeDrop, Columns: []string{"val"}})
		if _, ok := out.Series("val"); ok {
			t.Fatal("drop: val should be gone")
		}
	})

	t.Run("drop_nulls", func(t *testing.T) {
		out := run(logical.Node{Type: logical.NodeDropNulls, Columns: []string{"val"}})
		if out.Height() != 3 {
			t.Fatalf("drop_nulls height = %d, want 3", out.Height())
		}
	})

	t.Run("fill_null", func(t *testing.T) {
		out := run(logical.Node{Type: logical.NodeFillNull, Exprs: []expr.Expr{expr.Lit(int64(0))}})
		col, _ := out.Series("val")
		if col.Value(1) != int64(0) {
			t.Fatalf("fill_null val[1] = %v, want 0", col.Value(1))
		}
	})

	t.Run("unique", func(t *testing.T) {
		out := run(logical.Node{Type: logical.NodeUnique, Columns: []string{"grp"}})
		if out.Height() != 2 {
			t.Fatalf("unique(grp) height = %d, want 2", out.Height())
		}
	})

	t.Run("cast", func(t *testing.T) {
		out := run(logical.Node{Type: logical.NodeCast, Strings: []string{"id", string(dtypes.Float64)}})
		col, _ := out.Series("id")
		if col.DataType() != dtypes.Float64 {
			t.Fatalf("cast id dtype = %v, want float64", col.DataType())
		}
	})

	t.Run("shift", func(t *testing.T) {
		out := run(logical.Node{Type: logical.NodeShift, IntValue: 1})
		if out.Height() != src.Height() {
			t.Fatalf("shift height = %d, want %d", out.Height(), src.Height())
		}
	})

	t.Run("set_sorted", func(t *testing.T) {
		out := run(logical.Node{Type: logical.NodeSetSorted, Columns: []string{"id"}})
		if out.Height() != src.Height() {
			t.Fatalf("set_sorted height = %d", out.Height())
		}
	})

	t.Run("aggregate", func(t *testing.T) {
		out := run(logical.Node{
			Type:    logical.NodeAggregate,
			Columns: []string{"grp"},
			Exprs:   []expr.Expr{expr.Sum(expr.Col("id"))},
		})
		if out.Height() != 2 {
			t.Fatalf("aggregate height = %d, want 2 groups", out.Height())
		}
	})
}

// TestExecuteJoin covers the NodeJoin switch arm end to end.
func TestExecuteJoin(t *testing.T) {
	t.Parallel()

	left := mustFrame(t,
		frame.SeriesInput{Name: "key", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "lv", Values: []any{"a", "b", "c"}},
	)
	right := mustFrame(t,
		frame.SeriesInput{Name: "key", Values: []any{int64(2), int64(3), int64(4)}},
		frame.SeriesInput{Name: "rv", Values: []any{"x", "y", "z"}},
	)

	engine := New()
	out, err := engine.Execute(context.Background(), left, []logical.Node{{
		Type: logical.NodeJoin,
		Join: &logical.JoinSpec{
			Other:   right,
			LeftOn:  []string{"key"},
			RightOn: []string{"key"},
			How:     frame.JoinTypeInner,
		},
	}})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("inner join height = %d, want 2 (keys 2,3)", out.Height())
	}
}

// TestExecuteUnsupportedAndErrors covers error arms: an unknown node type and a
// filter with no expression.
func TestExecuteUnsupportedAndErrors(t *testing.T) {
	t.Parallel()

	engine := New()
	src := nodesSource(t)

	if _, err := engine.Execute(context.Background(), src, []logical.Node{
		{Type: logical.NodeType("nonsense")},
	}); err == nil {
		t.Fatal("expected error for unsupported node type")
	}

	if _, err := engine.Execute(context.Background(), src, []logical.Node{
		{Type: logical.NodeFilter},
	}); err == nil {
		t.Fatal("expected error for filter with no expression")
	}
}

// TestExecuteNodeErrorsReturnEmptyFrame pins that a failing node step stops
// execution and yields the zero DataFrame alongside its error.
func TestExecuteNodeErrorsReturnEmptyFrame(t *testing.T) {
	t.Parallel()

	src := nodesSource(t)
	missing := expr.Col("nope")
	failingPlan := []logical.Node{{Type: logical.NodeSelect, Exprs: []expr.Expr{missing}}}
	cases := []struct {
		name string
		node logical.Node
	}{
		{"select", logical.Node{Type: logical.NodeSelect, Exprs: []expr.Expr{missing}}},
		{"filter", logical.Node{Type: logical.NodeFilter, Exprs: []expr.Expr{missing.Gt(expr.Lit(int64(1)))}}},
		{"with_columns", logical.Node{Type: logical.NodeWithCols, Exprs: []expr.Expr{missing.Alias("x")}}},
		{"sort", logical.Node{Type: logical.NodeSort, Columns: []string{"nope"}}},
		{"unique", logical.Node{Type: logical.NodeUnique, Columns: []string{"nope"}}},
		{"fill_null_mismatch", logical.Node{Type: logical.NodeFillNull, Exprs: []expr.Expr{expr.Lit("x")}}},
		{"window", logical.Node{Type: logical.NodeWindow, Windows: []logical.WindowSpec{{Func: "row_number", PartitionBy: []string{"nope"}, Alias: "rn"}}}},
		{"unnest", logical.Node{Type: logical.NodeUnnest, Columns: []string{"nope"}}},
		{"melt", logical.Node{Type: logical.NodeMelt, Columns: []string{"nope", "val"}, Strings: []string{"variable", "value", "1"}}},
		{"unpivot", logical.Node{Type: logical.NodeUnpivot, Columns: []string{"nope", "val"}, Strings: []string{"variable", "value", "1"}}},
		{"melt_bad_count", logical.Node{Type: logical.NodeMelt, Columns: []string{"id"}, Strings: []string{"variable", "value", "x"}}},
		{"cast", logical.Node{Type: logical.NodeCast, Strings: []string{"nope", string(dtypes.Int64)}}},
		{"interpolate", logical.Node{Type: logical.NodeInterpolate, Columns: []string{"nope"}}},
		{"frame_agg_fused", logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"sum"}, Exprs: []expr.Expr{missing.Gt(expr.Lit(int64(1)))}}},
		{"update_plan", logical.Node{Type: logical.NodeUpdate, Plan: failingPlan}},
		{"pivot", logical.Node{Type: logical.NodePivot, Columns: []string{"nope", "grp", "val"}}},
		{"rolling", logical.Node{Type: logical.NodeRolling, Columns: []string{"nope", "val", "out"}, Strings: []string{"1000", "1"}}},
		{"rolling_bad_rows", logical.Node{Type: logical.NodeRolling, Columns: []string{"id", "val", "out"}, Strings: []string{"1000", "x"}}},
		{"dynamic_bad_period", logical.Node{Type: logical.NodeDynamic, Columns: []string{"id", "val"}, Strings: []string{"1", "x", "0", "left", "left"}, Exprs: []expr.Expr{expr.Col("val").Sum()}}},
		{"dynamic_bad_offset", logical.Node{Type: logical.NodeDynamic, Columns: []string{"id", "val"}, Strings: []string{"1", "1", "x", "left", "left"}, Exprs: []expr.Expr{expr.Col("val").Sum()}}},
		{"dynamic", logical.Node{Type: logical.NodeDynamic, Columns: []string{"nope", "val"}, Strings: []string{"1", "1", "0", "left", "left"}, Exprs: []expr.Expr{expr.Col("val").Sum()}}},
		{"aggregate", logical.Node{Type: logical.NodeAggregate, Columns: []string{"nope"}, Exprs: []expr.Expr{expr.Col("val").Sum()}}},
		{"join_missing_payload", logical.Node{Type: logical.NodeJoin}},
		{"join", logical.Node{Type: logical.NodeJoin, Join: &logical.JoinSpec{Other: src, LeftOn: []string{"nope"}, RightOn: []string{"id"}, How: frame.JoinTypeInner}}},
		{"set_op_plan", logical.Node{Type: logical.NodeSetOp, Strings: []string{"union"}, Plan: failingPlan}},
		{"set_op_kind", logical.Node{Type: logical.NodeSetOp, Strings: []string{"bogus"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := New().Execute(context.Background(), src, []logical.Node{tc.node})
			if err == nil {
				t.Fatal("expected an error")
			}
			if out.Width() != 0 || out.Height() != 0 {
				t.Fatalf("got a %dx%d frame with the error, want the zero frame", out.Height(), out.Width())
			}
		})
	}
}

// TestExecuteScanAndWindowNodes runs a scan and a descending-ordered window step
// through the engine.
func TestExecuteScanAndWindowNodes(t *testing.T) {
	t.Parallel()

	src := nodesSource(t)
	out, err := New().Execute(context.Background(), src, []logical.Node{
		{Type: logical.NodeScan},
		{Type: logical.NodeWindow, Windows: []logical.WindowSpec{{
			Func: "row_number", PartitionBy: []string{"grp"}, OrderBy: []string{"id"}, Descending: []bool{true}, Alias: "rn",
		}}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	rn, ok := out.Series("rn")
	if !ok {
		t.Fatalf("missing rn column in %v", out.Columns())
	}
	want := []any{int64(2), int64(1), int64(2), int64(1)}
	for i, w := range want {
		if rn.Value(i) != w {
			t.Fatalf("rn[%d] = %v, want %v", i, rn.Value(i), w)
		}
	}
}
