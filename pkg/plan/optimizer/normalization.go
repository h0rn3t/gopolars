package optimizer

import "github.com/h0rn3t/gopolars/pkg/plan/logical"

func SimplifyLimits(nodes []logical.Node) []logical.Node {
	if len(nodes) == 0 {
		return nodes
	}
	out := make([]logical.Node, 0, len(nodes))
	for _, n := range nodes {
		if len(out) > 0 && n.Type == logical.NodeLimit && out[len(out)-1].Type == logical.NodeLimit {
			if n.IntValue < out[len(out)-1].IntValue {
				out[len(out)-1] = n
			}
			continue
		}
		out = append(out, n)
	}
	return out
}
