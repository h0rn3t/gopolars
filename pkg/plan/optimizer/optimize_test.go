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

	filter := logical.Node{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("a").Gt(expr.Lit(int64(0)))}}

	// Past a Scan: always swappable.
	if !canSwapFilter(filter, logical.Node{Type: logical.NodeScan}) {
		t.Error("filter should swap past scan")
	}
	// Past WithColumns: never (the new column may feed the predicate).
	if canSwapFilter(filter, logical.Node{Type: logical.NodeWithCols}) {
		t.Error("filter should not swap past with_columns")
	}
	// Past a Select that covers the referenced column: swappable.
	covering := logical.Node{Type: logical.NodeSelect, Exprs: []expr.Expr{expr.Col("a"), expr.Col("b")}}
	if !canSwapFilter(filter, covering) {
		t.Error("filter should swap past a covering select")
	}
	// Past a Select that does NOT cover the referenced column: not swappable.
	noncovering := logical.Node{Type: logical.NodeSelect, Exprs: []expr.Expr{expr.Col("b")}}
	if canSwapFilter(filter, noncovering) {
		t.Error("filter should not swap past a non-covering select")
	}
	// A non-filter node is never swap-eligible.
	if canSwapFilter(logical.Node{Type: logical.NodeSort}, logical.Node{Type: logical.NodeScan}) {
		t.Error("non-filter node should not be swappable")
	}
}

// TestReferencedColumns covers column extraction across binary/unary/agg exprs.
func TestReferencedColumns(t *testing.T) {
	t.Parallel()

	// (a + b) compared, plus an aggregate over c, plus a wildcard (ignored).
	e := expr.Col("a").Add(expr.Col("b")).Gt(expr.Sum(expr.Col("c")))
	refs := referencedColumns(e)

	want := map[string]bool{"a": true, "b": true, "c": true}
	if len(refs) != 3 {
		t.Fatalf("referencedColumns = %v, want 3 distinct", refs)
	}
	for _, r := range refs {
		if !want[r] {
			t.Errorf("unexpected referenced column %q", r)
		}
	}

	// Wildcard is not reported.
	if got := referencedColumns(expr.All()); len(got) != 0 {
		t.Errorf("wildcard refs = %v, want none", got)
	}
}
