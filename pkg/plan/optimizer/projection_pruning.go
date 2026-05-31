package optimizer

import "github.com/h0rn3t/gopolars/pkg/plan/logical"

func ProjectionPruning(nodes []logical.Node) []logical.Node {
	if len(nodes) < 2 {
		return nodes
	}
	out := make([]logical.Node, 0, len(nodes))
	for i := 0; i < len(nodes); i++ {
		current := nodes[i]
		if current.Type == logical.NodeSelect && len(out) > 0 && out[len(out)-1].Type == logical.NodeSelect {
			out[len(out)-1] = current
			continue
		}
		if current.Type == logical.NodeSelect && len(current.Exprs) == 0 {
			continue
		}
		out = append(out, current)
	}
	return out
}
