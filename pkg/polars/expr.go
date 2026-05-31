package polars

import (
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	e "github.com/h0rn3t/gopolars/pkg/expr"
)

type Expr = e.Expr

func Col(name string) Expr {
	return e.Col(name)
}

func Lit(v any) Expr {
	return e.Lit(v)
}

func Sum(v Expr) Expr {
	return e.Sum(v)
}

func Min(v Expr) Expr {
	return e.Min(v)
}

func Max(v Expr) Expr {
	return e.Max(v)
}

func Mean(v Expr) Expr {
	return e.Mean(v)
}

func Count() Expr {
	return e.Count()
}

func NUnique(v Expr) Expr {
	return e.NUnique(v)
}

func When(cond Expr, thenExpr Expr, otherwise Expr) Expr {
	return e.When(cond, thenExpr, otherwise)
}

var (
	Int64       = dtypes.Int64
	Float64     = dtypes.Float64
	String      = dtypes.String
	Boolean     = dtypes.Boolean
	Datetime    = dtypes.Datetime
	Decimal     = dtypes.Decimal
	Categorical = dtypes.Categorical
	Enum        = dtypes.Enum
	List        = dtypes.List
	Struct      = dtypes.Struct
)
