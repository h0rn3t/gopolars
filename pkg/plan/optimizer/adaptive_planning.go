package optimizer

import (
	"slices"

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
		if !filtersWholePartitions(out[i].Exprs[0], out[i-1].Windows) {
			continue
		}
		out[i-1], out[i] = out[i], out[i-1]
	}
	return out
}

// filtersWholePartitions reports whether predicate keeps or drops whole
// partitions of every window spec, so running it before the windows leaves
// each kept row's window values unchanged: predicate is elementwise, reads no
// window alias and reads only columns that are partition keys of every spec.
// A spec without partition keys spans all rows and never qualifies.
func filtersWholePartitions(predicate expr.Expr, windows []logical.WindowSpec) bool {
	if len(windows) == 0 || !expr.IsElementwise(predicate) {
		return false
	}
	cols, ok := expr.InputColumns(predicate)
	if !ok {
		return false
	}
	for _, w := range windows {
		if len(w.PartitionBy) == 0 || slices.Contains(cols, w.Alias) {
			return false
		}
		for _, c := range cols {
			if !slices.Contains(w.PartitionBy, c) {
				return false
			}
		}
	}
	return true
}
