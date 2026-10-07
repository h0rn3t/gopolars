package optimizer

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

func TestSimplifyLimitsKeepsSmallestConsecutiveLimit(t *testing.T) {
	nodes := []logical.Node{
		{Type: logical.NodeFilter},
		{Type: logical.NodeLimit, IntValue: 100},
		{Type: logical.NodeLimit, IntValue: 10},
		{Type: logical.NodeSelect},
	}
	got := SimplifyLimits(nodes)
	if len(got) != 3 {
		t.Fatalf("unexpected node count: %d", len(got))
	}
	if got[1].Type != logical.NodeLimit || got[1].IntValue != 10 {
		t.Fatalf("expected merged limit=10")
	}
}

// TestOptimizeKeepsLimitBeforeSort pins that a limit followed by a sort is not
// reordered: sorting the first n rows differs from taking the first n sorted
// rows.
func TestOptimizeKeepsLimitBeforeSort(t *testing.T) {
	nodes := []logical.Node{
		{Type: logical.NodeLimit, IntValue: 2},
		{Type: logical.NodeSort, Columns: []string{"v"}},
	}
	got := Optimize(nodes)
	if len(got) != 2 || got[0].Type != logical.NodeLimit || got[1].Type != logical.NodeSort {
		t.Errorf("Optimize(limit -> sort) = %s, want limit -> sort", planString(got))
	}
}
