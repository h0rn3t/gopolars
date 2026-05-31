package physical

import "github.com/h0rn3t/gopolars/pkg/plan/logical"

type Plan struct {
	Operators []Operator
}

type Operator struct {
	Node logical.Node
}

func Build(nodes []logical.Node) Plan {
	ops := make([]Operator, 0, len(nodes))
	for _, n := range nodes {
		ops = append(ops, Operator{Node: n})
	}
	return Plan{Operators: ops}
}
