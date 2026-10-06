package polars

import "github.com/h0rn3t/gopolars/pkg/frame"

type groupBy struct {
	value frame.GroupBy
}

func (g groupBy) Agg(exprs ...Expr) (DataFrame, error) {
	return fromFrame(g.value.Agg(exprs...))
}
