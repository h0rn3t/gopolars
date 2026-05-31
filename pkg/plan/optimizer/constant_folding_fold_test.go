package optimizer

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

func TestConstantFoldingPartialBinaryFold(t *testing.T) {
	nodes := []logical.Node{
		{
			Type:  logical.NodeFilter,
			Exprs: []expr.Expr{expr.Lit(int64(1)).Add(expr.Col("missing"))},
		},
	}
	got := ConstantFolding(nodes)
	if len(got) != 1 || got[0].Type != logical.NodeFilter {
		t.Fatalf("очікували збережений фільтр, отримали %+v", got)
	}
}

func TestConstantFoldingUnaryNotFold(t *testing.T) {
	nodes := []logical.Node{
		{
			Type:  logical.NodeFilter,
			Exprs: []expr.Expr{expr.Col("x").Not()},
		},
	}
	got := ConstantFolding(nodes)
	if len(got) != 1 {
		t.Fatalf("not на колонці не згортається: %+v", got)
	}
}

func TestFoldExprFullyEvaluatesLiteralComparison(t *testing.T) {
	e := expr.Lit(int64(2)).Gt(expr.Lit(int64(1)))
	folded, ok, val := foldExpr(e)
	if !ok || val != true {
		t.Fatalf("fold gt: ok=%v val=%v expr=%+v", ok, val, folded)
	}
}

func TestLeftBinaryAndUnaryExprBuilders(t *testing.T) {
	lit := expr.Lit(int64(1))
	col := expr.Col("x")
	_ = leftBinary("sub", lit, col)
	_ = leftBinary("mul", lit, col)
	_ = leftBinary("div", lit, col)
	_ = leftBinary("and", lit, col)
	_ = leftBinary("or", lit, col)
	_ = leftBinary("unknown", lit, col)
	_ = unaryExpr("not", col)
	_ = unaryExpr("is_null", col)
	_ = unaryExpr("is_not_null", col)
	_ = unaryExpr("other", col)
	var row staticRow
	if _, ok := row.ValueByName("x"); ok {
		t.Fatal("staticRow не має повертати значення")
	}
}
