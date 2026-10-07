package optimizer

import (
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

func TestConstantFoldingRemovesTrueFilter(t *testing.T) {
	nodes := []logical.Node{
		{Type: logical.NodeScan},
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Lit(true)}},
		{Type: logical.NodeLimit, IntValue: 5},
	}
	got := ConstantFolding(nodes)
	if len(got) != 2 {
		t.Fatalf("очікували 2 вузли після згортання true-фільтра, отримали %d", len(got))
	}
	if got[0].Type != logical.NodeScan || got[1].Type != logical.NodeLimit {
		t.Fatalf("неочікувана послідовність вузлів")
	}
}

func TestConstantFoldingFalseFilterBecomesEmptyLimit(t *testing.T) {
	nodes := []logical.Node{
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Lit(false)}},
	}
	got := ConstantFolding(nodes)
	if len(got) != 1 || got[0].Type != logical.NodeLimit || got[0].IntValue != 0 {
		t.Fatalf("очікували limit=0 для false-фільтра, отримали %+v", got)
	}
}

func TestPredicatePushdownMovesFilterBeforeScan(t *testing.T) {
	nodes := []logical.Node{
		{Type: logical.NodeScan},
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Gt(expr.Lit(int64(0)))}},
	}
	got := PredicatePushdown(nodes)
	if got[0].Type != logical.NodeFilter || got[1].Type != logical.NodeScan {
		t.Fatalf("фільтр має бути перед scan")
	}
}

// planString renders nodes as "type(name,...) -> ..." using each expression's
// output name, so a test failure shows the order and shape of a plan.
func planString(nodes []logical.Node) string {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		names := make([]string, 0, len(n.Exprs))
		for _, e := range n.Exprs {
			names = append(names, e.Name())
		}
		parts = append(parts, string(n.Type)+"("+strings.Join(names, ",")+")")
	}
	return strings.Join(parts, " -> ")
}

func selectNode(exprs ...expr.Expr) logical.Node {
	return logical.Node{Type: logical.NodeSelect, Exprs: exprs}
}

func TestProjectionPruning(t *testing.T) {
	a, b := expr.Col("a"), expr.Col("b")
	tests := []struct {
		name  string
		nodes []logical.Node
		want  string
	}{
		{"pass-through then subset merges", []logical.Node{selectNode(a, b), selectNode(b)}, "select(b)"},
		{"pass-through then computed merges", []logical.Node{selectNode(a, b), selectNode(a.Add(b).Alias("s"))}, "select(s)"},
		{"empty select dropped", []logical.Node{selectNode(), selectNode(a)}, "select(a)"},
		{"empty second select dropped", []logical.Node{selectNode(a, b), selectNode()}, "select(a,b)"},
		{"alias kept", []logical.Node{selectNode(a.Alias("x")), selectNode(expr.Col("x"))}, "select(x) -> select(x)"},
		{"computed kept", []logical.Node{selectNode(a.Add(expr.Lit(int64(1))).Alias("a")), selectNode(a)}, "select(a) -> select(a)"},
		{"aggregate kept", []logical.Node{selectNode(expr.Sum(a)), selectNode(expr.Col("sum_a"))}, "select(sum_a) -> select(sum_a)"},
		{"missing column kept", []logical.Node{selectNode(a), selectNode(b)}, "select(a) -> select(b)"},
		{"selector first kept", []logical.Node{selectNode(expr.All()), selectNode(a)}, "select(*) -> select(a)"},
		{"selector second kept", []logical.Node{selectNode(a, b), selectNode(expr.All())}, "select(a,b) -> select(*)"},
		{"over partition outside first kept", []logical.Node{selectNode(a), selectNode(a.Sum().Over("p"))}, "select(a) -> select(expr)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := planString(ProjectionPruning(tt.nodes)); got != tt.want {
				t.Errorf("ProjectionPruning(%s) = %s, want %s", planString(tt.nodes), got, tt.want)
			}
		})
	}
}

func TestAdaptivePlanning(t *testing.T) {
	byP := logical.WindowSpec{Func: "sum", Target: "v", Alias: "s", PartitionBy: []string{"p"}}
	byPQ := logical.WindowSpec{Func: "row_number", Alias: "rn", PartitionBy: []string{"p", "q"}, OrderBy: []string{"id"}}
	global := logical.WindowSpec{Func: "row_number", Alias: "rn", OrderBy: []string{"id"}}
	tests := []struct {
		name    string
		windows []logical.WindowSpec
		filter  expr.Expr
		want    bool // filter moved before the window
	}{
		{"partition key", []logical.WindowSpec{byP}, expr.Col("p").Eq(expr.Lit("x")), true},
		{"subset of partition keys", []logical.WindowSpec{byPQ}, expr.Col("q").Gt(expr.Lit(int64(1))), true},
		{"partition key of every spec", []logical.WindowSpec{byP, byPQ}, expr.Col("p").IsNotNull(), true},
		{"no partition", []logical.WindowSpec{global}, expr.Col("id").Gt(expr.Lit(int64(1))), false},
		{"one spec without partition", []logical.WindowSpec{byP, global}, expr.Col("p").Eq(expr.Lit("x")), false},
		{"non-partition column", []logical.WindowSpec{byP}, expr.Col("v").Gt(expr.Lit(int64(1))), false},
		{"partition and other column", []logical.WindowSpec{byP}, expr.Col("p").Eq(expr.Col("v")), false},
		{"window alias", []logical.WindowSpec{byP}, expr.Col("s").Gt(expr.Lit(int64(1))), false},
		{"alias shadowing a partition key", []logical.WindowSpec{{Func: "sum", Target: "v", Alias: "p", PartitionBy: []string{"p"}}}, expr.Col("p").Eq(expr.Lit("x")), false},
		{"constant without partition", []logical.WindowSpec{global}, expr.Lit(true).And(expr.Lit(false)), false},
		{"not elementwise", []logical.WindowSpec{byP}, expr.Col("p").IsFirstDistinct(), false},
		{"selector", []logical.WindowSpec{byP}, expr.All().IsNotNull(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := []logical.Node{
				{Type: logical.NodeWindow, Windows: tt.windows},
				{Type: logical.NodeFilter, Exprs: []expr.Expr{tt.filter}},
			}
			got := AdaptivePlanning(nodes)
			if moved := got[0].Type == logical.NodeFilter; moved != tt.want {
				t.Errorf("AdaptivePlanning(window -> filter %s) = %s, want filter moved = %t", tt.name, planString(got), tt.want)
			}
		})
	}
}

func TestAdaptivePlanningKeepsFilterAfterWindowWhenAliasReferenced(t *testing.T) {
	nodes := []logical.Node{
		{Type: logical.NodeWindow, Windows: []logical.WindowSpec{{Alias: "roll", Target: "v"}}},
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("roll").Gt(expr.Lit(int64(0)))}},
	}
	got := AdaptivePlanning(nodes)
	if got[0].Type != logical.NodeWindow || got[1].Type != logical.NodeFilter {
		t.Fatalf("фільтр на window alias не повинен переставлятися")
	}
}
