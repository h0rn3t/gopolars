package optimizer

import (
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// ProjectionPruning drops empty selects, which DataFrame.Select treats as a
// no-op, and merges two consecutive selects into the second one when the first
// only passes columns through unchanged and the second reads nothing else.
func ProjectionPruning(nodes []logical.Node) []logical.Node {
	if len(nodes) < 2 {
		return nodes
	}
	out := make([]logical.Node, 0, len(nodes))
	for _, current := range nodes {
		if current.Type == logical.NodeSelect && len(current.Exprs) == 0 {
			continue
		}
		if current.Type == logical.NodeSelect && len(out) > 0 && canMergeSelects(out[len(out)-1], current) {
			out[len(out)-1] = current
			continue
		}
		out = append(out, current)
	}
	return out
}

// canMergeSelects reports whether next run on the output of prev equals next
// run on prev's input: every expression of prev is a pass-through column and
// next reads only columns that prev passes through.
func canMergeSelects(prev logical.Node, next logical.Node) bool {
	if prev.Type != logical.NodeSelect {
		return false
	}
	passed := make(map[string]struct{}, len(prev.Exprs))
	for _, e := range prev.Exprs {
		if !isPassThrough(e) {
			return false
		}
		passed[e.Name()] = struct{}{}
	}
	for _, e := range next.Exprs {
		cols, ok := expr.InputColumns(e)
		if !ok {
			return false
		}
		for _, c := range cols {
			if _, found := passed[c]; !found {
				return false
			}
		}
	}
	return true
}

// isPassThrough reports whether e is a plain reference to one column that keeps
// the column's name: not a selector, not aliased to another name.
func isPassThrough(e expr.Expr) bool {
	cols, ok := expr.InputColumns(e)
	return e.Kind() == expr.KindCol && ok && len(cols) == 1 && cols[0] == e.Name()
}
