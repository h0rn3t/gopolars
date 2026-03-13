package polars

import (
	"github.com/eugeneshershen/gopolars/pkg/expr"
	"github.com/eugeneshershen/gopolars/pkg/frame"
)

type groupBy struct {
	value frame.GroupBy
}

func (g groupBy) Agg(exprs ...Expr) (DataFrame, error) {
	internal := make([]expr.Expr, len(exprs))
	copy(internal, exprs)
	next, err := g.value.Agg(internal...)
	return fromFrame(next, err)
}
