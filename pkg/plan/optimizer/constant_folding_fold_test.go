package optimizer

import (
	"reflect"
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

func TestUnaryExprBuildersAndStaticRow(t *testing.T) {
	col := expr.Col("x")
	_ = unaryExpr("not", col)
	_ = unaryExpr("is_null", col)
	_ = unaryExpr("is_not_null", col)
	_ = unaryExpr("other", col)
	var row staticRow
	if _, ok := row.ValueByName("x"); ok {
		t.Fatal("staticRow не має повертати значення")
	}
}

// TestConstantFoldingOutcomes pins each ConstantFolding outcome: a true constant
// drops the filter, a false one becomes Limit(0), a non-bool constant replaces
// the predicate, and a predicate that fails to evaluate or reads a column keeps
// the filter unchanged.
func TestConstantFoldingOutcomes(t *testing.T) {
	t.Parallel()

	filter := func(e expr.Expr) []logical.Node {
		return []logical.Node{{Type: logical.NodeFilter, Exprs: []expr.Expr{e}}}
	}

	if got := ConstantFolding(filter(expr.Lit(int64(2)).Gt(expr.Lit(int64(1))))); len(got) != 0 {
		t.Fatalf("true predicate: got %+v, want the filter dropped", got)
	}
	if got := ConstantFolding(filter(expr.Lit(true).Not())); len(got) != 1 || got[0].Type != logical.NodeLimit || got[0].IntValue != 0 {
		t.Fatalf("false predicate: got %+v, want Limit(0)", got)
	}
	got := ConstantFolding(filter(expr.Lit(int64(1)).Add(expr.Lit(int64(2)))))
	if len(got) != 1 || got[0].Type != logical.NodeFilter || got[0].Exprs[0].Kind() != expr.KindLit || got[0].Exprs[0].Value() != int64(3) {
		t.Fatalf("non-bool constant: got %+v, want a filter on Lit(3)", got)
	}

	for _, e := range []expr.Expr{
		expr.Lit("a").Gt(expr.Lit(int64(1))),
		expr.Lit(int64(1)).Not(),
		expr.Col("x").Gt(expr.Lit(int64(1)).Add(expr.Lit(int64(2)))),
		expr.Lit(int64(1)).Add(expr.Lit(int64(2))).Lt(expr.Col("x")),
		expr.Col("x").Not(),
	} {
		got := ConstantFolding(filter(e))
		if len(got) != 1 || got[0].Type != logical.NodeFilter || !reflect.DeepEqual(got[0].Exprs[0], e) {
			t.Errorf("unfoldable %s predicate: got %+v, want the filter unchanged", e.Op(), got)
		}
	}
}
