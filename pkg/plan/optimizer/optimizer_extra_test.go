package optimizer

import (
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/expr"
	"github.com/eugeneshershen/gopolars/pkg/plan/logical"
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

func TestProjectionPruningMergesConsecutiveSelects(t *testing.T) {
	nodes := []logical.Node{
		{Type: logical.NodeSelect, Exprs: []expr.Expr{expr.Col("a")}},
		{Type: logical.NodeSelect, Exprs: []expr.Expr{expr.Col("b")}},
	}
	got := ProjectionPruning(nodes)
	if len(got) != 1 || got[0].Exprs[0].ColName() != "b" {
		t.Fatalf("другий select має замінити перший")
	}
}

func TestAdaptivePlanningSwapsFilterBeforeWindow(t *testing.T) {
	nodes := []logical.Node{
		{Type: logical.NodeWindow, Windows: []logical.WindowSpec{{Alias: "roll", Target: "v"}}},
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Gt(expr.Lit(int64(0)))}},
	}
	got := AdaptivePlanning(nodes)
	if got[0].Type != logical.NodeFilter || got[1].Type != logical.NodeWindow {
		t.Fatalf("фільтр без посилання на window alias має піднятися вище")
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
