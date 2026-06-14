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
