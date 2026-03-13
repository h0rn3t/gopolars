package optimizer

import "github.com/eugeneshershen/gopolars/pkg/plan/logical"

func Optimize(nodes []logical.Node) []logical.Node {
	current := make([]logical.Node, len(nodes))
	copy(current, nodes)
	current = ConstantFolding(current)
	current = PredicatePushdown(current)
	current = ProjectionPruning(current)
	current = NormalizeSortLimit(current)
	current = SimplifyLimits(current)
	current = AdaptivePlanning(current)
	return current
}
