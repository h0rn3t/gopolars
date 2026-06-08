package polars

import (
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	e "github.com/h0rn3t/gopolars/pkg/expr"
)

type Expr = e.Expr

func Col(name string) Expr {
	return e.Col(name)
}

// All selects every column (pl.all()).
func All() Expr {
	return e.All()
}

// Cols selects the named columns in order (pl.col("a","b")).
func Cols(names ...string) Expr {
	return e.Cols(names...)
}

// Exclude selects every column except the named ones (pl.exclude("a")).
func Exclude(names ...string) Expr {
	return e.Exclude(names...)
}

// StructCols packs the named columns into a single Struct column (pl.struct([...])).
func StructCols(names ...string) Expr {
	return e.StructCols(names...)
}

// Fold horizontally reduces exprs left-to-right per row from acc (pl.fold).
func Fold(acc any, fn func(acc any, next any) (any, error), exprs ...Expr) Expr {
	return e.Fold(acc, fn, exprs...)
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
