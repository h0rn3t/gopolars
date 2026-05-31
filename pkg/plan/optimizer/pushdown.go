package optimizer

import (
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

func PredicatePushdown(nodes []logical.Node) []logical.Node {
	out := make([]logical.Node, len(nodes))
	copy(out, nodes)
	for i := 1; i < len(out); i++ {
		if out[i].Type != logical.NodeFilter {
			continue
		}
		j := i
		for j > 0 && canSwapFilter(out[j], out[j-1]) {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
	return out
}

func canSwapFilter(filter logical.Node, prev logical.Node) bool {
	if filter.Type != logical.NodeFilter {
		return false
	}
	if prev.Type == logical.NodeScan {
		return true
	}
	if prev.Type == logical.NodeWithCols {
		return false
	}
	if prev.Type == logical.NodeSelect {
		refs := referencedColumns(filter.Exprs[0])
		selected := map[string]struct{}{}
		for _, ex := range prev.Exprs {
			if ex.Kind() == expr.KindCol {
				selected[ex.ColName()] = struct{}{}
			}
		}
		for _, c := range refs {
			if _, ok := selected[c]; !ok {
				return false
			}
		}
		return true
	}
	return false
}

func referencedColumns(e expr.Expr) []string {
	set := map[string]struct{}{}
	var walk func(v expr.Expr)
	walk = func(v expr.Expr) {
		switch v.Kind() {
		case expr.KindCol:
			if v.ColName() != "*" {
				set[v.ColName()] = struct{}{}
			}
		case expr.KindBin:
			walk(*v.Left())
			walk(*v.Right())
		case expr.KindUnary, expr.KindCast, expr.KindAgg, expr.KindAlias:
			if v.Target() != nil {
				walk(*v.Target())
			}
		}
	}
	walk(e)
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}
