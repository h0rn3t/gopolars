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

// canSwapFilter reports whether filter may run before prev. Past a Select that
// is only safe when the Select keeps every row (all its expressions are
// elementwise) and passes each column the filter reads through unchanged, so
// the filter sees the same values on either side.
func canSwapFilter(filter logical.Node, prev logical.Node) bool {
	if filter.Type != logical.NodeFilter || len(filter.Exprs) == 0 {
		return false
	}
	switch prev.Type {
	case logical.NodeScan:
		return true
	case logical.NodeSelect:
		refs, ok := expr.InputColumns(filter.Exprs[0])
		if !ok {
			return false
		}
		passed := map[string]struct{}{}
		for _, e := range prev.Exprs {
			if !expr.IsElementwise(e) {
				return false
			}
			if isPassThrough(e) {
				passed[e.Name()] = struct{}{}
			}
		}
		for _, c := range refs {
			if _, found := passed[c]; !found {
				return false
			}
		}
		return true
	}
	return false
}
