package logical

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestNodeConstruction pins the field layout of a logical Node: the executor and
// optimizer read these fields directly, so a struct round-trip is the contract.
func TestNodeConstruction(t *testing.T) {
	t.Parallel()

	n := Node{
		Type:       NodeFilter,
		Exprs:      []expr.Expr{expr.Col("a").Gt(expr.Lit(int64(0)))},
		Columns:    []string{"a", "b"},
		IntValue:   7,
		Strings:    []string{"x", "y"},
		Descending: []bool{true, false},
		Prefix:     "p_",
	}

	if n.Type != NodeFilter {
		t.Fatalf("Type = %q, want %q", n.Type, NodeFilter)
	}
	if len(n.Exprs) != 1 {
		t.Fatalf("Exprs len = %d, want 1", len(n.Exprs))
	}
	if got := n.Exprs[0].Op(); got != "gt" {
		t.Fatalf("Exprs[0].Op() = %q, want gt", got)
	}
	if n.IntValue != 7 {
		t.Fatalf("IntValue = %d, want 7", n.IntValue)
	}
	if len(n.Columns) != 2 || n.Columns[1] != "b" {
		t.Fatalf("Columns = %v, want [a b]", n.Columns)
	}
	if len(n.Descending) != 2 || !n.Descending[0] || n.Descending[1] {
		t.Fatalf("Descending = %v, want [true false]", n.Descending)
	}
	if n.Prefix != "p_" {
		t.Fatalf("Prefix = %q, want p_", n.Prefix)
	}
}

// TestNodeTypeConstants guards against accidental string-value drift: these
// values are matched verbatim by the executor's switch and serialized plans.
func TestNodeTypeConstants(t *testing.T) {
	t.Parallel()

	cases := map[NodeType]string{
		NodeScan:      "scan",
		NodeSelect:    "select",
		NodeFilter:    "filter",
		NodeWithCols:  "with_columns",
		NodeJoin:      "join",
		NodeGroupBy:   "group_by",
		NodeSort:      "sort",
		NodeLimit:     "limit",
		NodeAggregate: "aggregate",
		NodeSetOp:     "set_op",
		NodeWindow:    "window",
		NodeRolling:   "rolling_mean",
		NodeDynamic:   "group_by_dynamic",
	}
	for nt, want := range cases {
		if string(nt) != want {
			t.Errorf("NodeType %v = %q, want %q", nt, string(nt), want)
		}
	}
}

// TestJoinSpec exercises the nested join payload carried by a NodeJoin.
func TestJoinSpec(t *testing.T) {
	t.Parallel()

	other, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "key", Values: []any{int64(1), int64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}

	spec := JoinSpec{
		Other:         other,
		LeftOn:        []string{"key"},
		RightOn:       []string{"key"},
		How:           frame.JoinTypeInner,
		Suffix:        "_r",
		AsofDirection: "backward",
		AsofTolerance: 1000,
	}
	node := Node{Type: NodeJoin, Join: &spec}

	if node.Join == nil {
		t.Fatal("Join payload is nil")
	}
	if node.Join.How != frame.JoinTypeInner {
		t.Fatalf("How = %v, want inner", node.Join.How)
	}
	if node.Join.Other.Height() != 2 {
		t.Fatalf("Other height = %d, want 2", node.Join.Other.Height())
	}
	if node.Join.AsofTolerance != 1000 || node.Join.AsofDirection != "backward" {
		t.Fatalf("asof fields = %d/%q", node.Join.AsofTolerance, node.Join.AsofDirection)
	}
}

// TestWindowSpec exercises the window descriptor used by NodeWindow.
func TestWindowSpec(t *testing.T) {
	t.Parallel()

	w := WindowSpec{
		Func:        "lag",
		Target:      "value",
		Alias:       "value_lag",
		PartitionBy: []string{"grp"},
		OrderBy:     []string{"ts"},
		Descending:  []bool{false},
		Offset:      2,
		Default:     int64(-1),
	}
	node := Node{Type: NodeWindow, Windows: []WindowSpec{w}}

	if len(node.Windows) != 1 {
		t.Fatalf("Windows len = %d, want 1", len(node.Windows))
	}
	got := node.Windows[0]
	if got.Func != "lag" || got.Target != "value" || got.Alias != "value_lag" {
		t.Fatalf("window descriptor mismatch: %+v", got)
	}
	if got.Offset != 2 || got.Default != int64(-1) {
		t.Fatalf("offset/default mismatch: %d / %v", got.Offset, got.Default)
	}
	if len(got.PartitionBy) != 1 || got.PartitionBy[0] != "grp" {
		t.Fatalf("PartitionBy = %v", got.PartitionBy)
	}
}

// TestNodeNestedPlan covers the Plan field used by set-op and update nodes that
// carry a sub-pipeline.
func TestNodeNestedPlan(t *testing.T) {
	t.Parallel()

	node := Node{
		Type:    NodeSetOp,
		Strings: []string{"union"},
		Plan: []Node{
			{Type: NodeFilter, Exprs: []expr.Expr{expr.Col("id").Le(expr.Lit(int64(5)))}},
		},
	}
	if len(node.Plan) != 1 {
		t.Fatalf("nested plan len = %d, want 1", len(node.Plan))
	}
	if node.Plan[0].Type != NodeFilter {
		t.Fatalf("nested node type = %q, want filter", node.Plan[0].Type)
	}
}
