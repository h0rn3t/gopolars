package optimizer

import (
	"github.com/eugeneshershen/gopolars/pkg/expr"
	"github.com/eugeneshershen/gopolars/pkg/plan/logical"
)

func ConstantFolding(nodes []logical.Node) []logical.Node {
	out := make([]logical.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Type != logical.NodeFilter || len(n.Exprs) == 0 {
			out = append(out, n)
			continue
		}
		folded, ok, value := foldExpr(n.Exprs[0])
		if ok {
			if b, bok := value.(bool); bok {
				if b {
					continue
				}
				out = append(out, logical.Node{Type: logical.NodeLimit, IntValue: 0})
				continue
			}
			n.Exprs[0] = folded
			out = append(out, n)
			continue
		}
		out = append(out, n)
	}
	return out
}

func foldExpr(e expr.Expr) (expr.Expr, bool, any) {
	switch e.Kind() {
	case expr.KindLit:
		return e, true, e.Value()
	case expr.KindBin:
		lExpr, lok, lv := foldExpr(*e.Left())
		rExpr, rok, rv := foldExpr(*e.Right())
		if lok && rok {
			v, err := expr.Eval(e, staticRow{})
			if err == nil {
				return expr.Lit(v), true, v
			}
			return e, false, nil
		}
		if lok {
			left := expr.Lit(lv)
			e = leftBinary(e.Op(), left, rExpr)
		}
		if rok {
			right := expr.Lit(rv)
			e = leftBinary(e.Op(), lExpr, right)
		}
		return e, false, nil
	case expr.KindUnary:
		t, tok, _ := foldExpr(*e.Target())
		if tok {
			v, err := expr.Eval(unaryExpr(e.Op(), t), staticRow{})
			if err == nil {
				return expr.Lit(v), true, v
			}
		}
		return unaryExpr(e.Op(), t), false, nil
	default:
		return e, false, nil
	}
}

type staticRow struct{}

func (staticRow) ValueByName(name string) (any, bool) {
	_ = name
	return nil, false
}

func leftBinary(op string, left expr.Expr, right expr.Expr) expr.Expr {
	switch op {
	case "eq":
		return left.Eq(right)
	case "ne":
		return left.Ne(right)
	case "gt":
		return left.Gt(right)
	case "ge":
		return left.Ge(right)
	case "lt":
		return left.Lt(right)
	case "le":
		return left.Le(right)
	case "add":
		return left.Add(right)
	case "sub":
		return left.Sub(right)
	case "mul":
		return left.Mul(right)
	case "div":
		return left.Div(right)
	case "and":
		return left.And(right)
	case "or":
		return left.Or(right)
	default:
		return left.Eq(right)
	}
}

func unaryExpr(op string, t expr.Expr) expr.Expr {
	switch op {
	case "not":
		return t.Not()
	case "is_null":
		return t.IsNull()
	case "is_not_null":
		return t.IsNotNull()
	default:
		return t
	}
}
