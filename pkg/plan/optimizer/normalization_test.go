package optimizer

import (
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/plan/logical"
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

func TestNormalizeSortLimitSwapsLimitBeforeSort(t *testing.T) {
	nodes := []logical.Node{
		{Type: logical.NodeFilter},
		{Type: logical.NodeLimit, IntValue: 5},
		{Type: logical.NodeSort, Columns: []string{"v"}},
	}
	got := NormalizeSortLimit(nodes)
	if got[1].Type != logical.NodeSort || got[2].Type != logical.NodeLimit {
		t.Fatalf("expected sort before limit after normalization")
	}
}
