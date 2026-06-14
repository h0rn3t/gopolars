package physical

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// TestBuildWrapsNodes verifies Build turns each logical node into one physical
// operator, in order, preserving the underlying node.
func TestBuildWrapsNodes(t *testing.T) {
	t.Parallel()

	nodes := []logical.Node{
		{Type: logical.NodeScan},
		{Type: logical.NodeFilter},
		{Type: logical.NodeLimit, IntValue: 3},
	}

	plan := Build(nodes)

	if len(plan.Operators) != len(nodes) {
		t.Fatalf("Operators len = %d, want %d", len(plan.Operators), len(nodes))
	}
	for i, op := range plan.Operators {
		if op.Node.Type != nodes[i].Type {
			t.Errorf("operator %d type = %q, want %q", i, op.Node.Type, nodes[i].Type)
		}
	}
	if plan.Operators[2].Node.IntValue != 3 {
		t.Fatalf("limit operator IntValue = %d, want 3", plan.Operators[2].Node.IntValue)
	}
}

// TestBuildEmpty covers the zero-node path: an empty (non-nil) operator slice.
func TestBuildEmpty(t *testing.T) {
	t.Parallel()

	plan := Build(nil)
	if plan.Operators == nil {
		t.Fatal("Operators should be a non-nil empty slice")
	}
	if len(plan.Operators) != 0 {
		t.Fatalf("Operators len = %d, want 0", len(plan.Operators))
	}
}
