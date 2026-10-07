package optimizer

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// TestOptimizeRunsAllPasses drives the top-level Optimize entry point with a
// plan that exercises constant folding, predicate pushdown, and projection
// pruning together.
func TestOptimizeRunsAllPasses(t *testing.T) {
	t.Parallel()

	nodes := []logical.Node{
		{Type: logical.NodeScan},
		{Type: logical.NodeSelect, Exprs: []expr.Expr{expr.Col("a"), expr.Col("b")}},
		// Filter referencing only a selected column -> can be pushed before Select.
		{Type: logical.NodeFilter, Exprs: []expr.Expr{
			// 1 + 2 folds to a constant; comparison keeps the filter shape.
			expr.Col("a").Gt(expr.Lit(int64(1)).Add(expr.Lit(int64(2)))),
		}},
	}

	out := Optimize(nodes)
	if len(out) == 0 {
		t.Fatal("Optimize returned no nodes")
	}
	// The plan must still contain the scan and the filter after optimization.
	var hasScan, hasFilter bool
	for _, n := range out {
		switch n.Type {
		case logical.NodeScan:
			hasScan = true
		case logical.NodeFilter:
			hasFilter = true
		}
	}
	if !hasScan || !hasFilter {
		t.Fatalf("Optimize dropped nodes: scan=%v filter=%v", hasScan, hasFilter)
	}
}

// TestPredicatePushdownReordersBeforeScan verifies a filter is pushed ahead of a
// reorderable predecessor (Select whose columns cover the filter refs).
func TestPredicatePushdownReordersBeforeScan(t *testing.T) {
	t.Parallel()

	nodes := []logical.Node{
		{Type: logical.NodeScan},
		{Type: logical.NodeSelect, Exprs: []expr.Expr{expr.Col("a"), expr.Col("b")}},
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("a").Gt(expr.Lit(int64(0)))}},
	}
	out := PredicatePushdown(nodes)

	// Filter references only column "a", which Select carries, so it moves before
	// the Select (to just after the Scan).
	filterIdx, selectIdx := -1, -1
	for i, n := range out {
		switch n.Type {
		case logical.NodeFilter:
			filterIdx = i
		case logical.NodeSelect:
			selectIdx = i
		}
	}
	if filterIdx >= selectIdx {
		t.Fatalf("filter (idx %d) should be pushed before select (idx %d)", filterIdx, selectIdx)
	}
}

// TestCanSwapFilterRules covers the swap-eligibility rules directly.
func TestCanSwapFilterRules(t *testing.T) {
	t.Parallel()

	a, b, v := expr.Col("a"), expr.Col("b"), expr.Col("v")
	aPositive := a.Gt(expr.Lit(int64(0)))
	tests := []struct {
		name   string
		filter expr.Expr
		prev   logical.Node
		want   bool
	}{
		{"past scan", aPositive, logical.Node{Type: logical.NodeScan}, true},
		{"past with_columns", aPositive, logical.Node{Type: logical.NodeWithCols, Exprs: []expr.Expr{b}}, false},
		{"past sort", aPositive, logical.Node{Type: logical.NodeSort, Columns: []string{"a"}}, false},
		{"past covering select", aPositive, selectNode(a, b), true},
		{"past select with unread computed column", aPositive, selectNode(a, b.Mul(expr.Lit(int64(2))).Alias("c")), true},
		{"past non-covering select", aPositive, selectNode(b), false},
		{"past renaming select", aPositive, selectNode(a.Alias("b"), b.Alias("a")), false},
		{"past computed column of the same name", aPositive, selectNode(a.Add(expr.Lit(int64(1))).Alias("a")), false},
		{"past aggregating select", v.Gt(expr.Lit(int64(1))), selectNode(v, expr.Sum(v).Alias("s")), false},
		{"past select with window expression", aPositive, selectNode(a, b.CumSum().Alias("c")), false},
		{"filter with selector", expr.All().IsNotNull(), selectNode(a, b), false},
		{"filter reading over partition", a.Sum().Over("p").Gt(expr.Lit(int64(0))), selectNode(a, b), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filter := logical.Node{Type: logical.NodeFilter, Exprs: []expr.Expr{tt.filter}}
			if got := canSwapFilter(filter, tt.prev); got != tt.want {
				t.Errorf("canSwapFilter(%s) = %t, want %t", tt.name, got, tt.want)
			}
		})
	}

	// A non-filter node is never swap-eligible.
	if canSwapFilter(logical.Node{Type: logical.NodeSort}, logical.Node{Type: logical.NodeScan}) {
		t.Error("canSwapFilter(sort, scan) = true, want false")
	}
}
