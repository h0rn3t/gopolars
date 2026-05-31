package optimizer

import (
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

func AdaptivePlanning(nodes []logical.Node) []logical.Node {
	out := make([]logical.Node, len(nodes))
	copy(out, nodes)
	for i := 1; i < len(out); i++ {
		if out[i].Type != logical.NodeFilter || len(out[i].Exprs) == 0 {
			continue
		}
		if out[i-1].Type != logical.NodeWindow {
			continue
		}
		if hasWindowAliasRef(out[i], out[i-1].Windows) {
			continue
		}
		out[i-1], out[i] = out[i], out[i-1]
	}
	return out
}

func hasWindowAliasRef(filter logical.Node, windows []logical.WindowSpec) bool {
	if len(filter.Exprs) == 0 {
		return false
	}
	aliases := map[string]struct{}{}
	for _, w := range windows {
		aliases[w.Alias] = struct{}{}
	}
	cols := collectColumns(filter.Exprs[0])
	for _, c := range cols {
		if _, ok := aliases[c]; ok {
			return true
		}
	}
	return false
}

func collectColumns(e expr.Expr) []string {
	out := map[string]struct{}{}
	var walk func(x expr.Expr)
	walk = func(x expr.Expr) {
		if x.Kind() == expr.KindCol {
			out[x.ColName()] = struct{}{}
		}
		if x.Left() != nil {
			walk(*x.Left())
		}
		if x.Right() != nil {
			walk(*x.Right())
		}
		if x.Target() != nil {
			walk(*x.Target())
		}
		if x.Extra() != nil {
			walk(*x.Extra())
		}
	}
	walk(e)
	values := make([]string, 0, len(out))
	for c := range out {
		values = append(values, c)
	}
	return values
}
