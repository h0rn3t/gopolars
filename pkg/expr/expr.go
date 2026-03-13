package expr

import (
	"fmt"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
)

type Kind string

const (
	KindCol   Kind = "col"
	KindLit   Kind = "lit"
	KindAlias Kind = "alias"
	KindCast  Kind = "cast"
	KindBin   Kind = "bin"
	KindUnary Kind = "unary"
	KindTern  Kind = "ternary"
	KindAgg   Kind = "agg"
)

type Expr struct {
	kind   Kind
	name   string
	value  any
	left   *Expr
	right  *Expr
	extra  *Expr
	alias  string
	dtype  dtypes.DataType
	op     string
	target *Expr
}

func Col(name string) Expr {
	return Expr{kind: KindCol, name: name}
}

func Lit(v any) Expr {
	return Expr{kind: KindLit, value: v}
}

func Sum(e Expr) Expr { return Expr{kind: KindAgg, op: "sum", target: &e} }
func Min(e Expr) Expr { return Expr{kind: KindAgg, op: "min", target: &e} }
func Max(e Expr) Expr { return Expr{kind: KindAgg, op: "max", target: &e} }
func Mean(e Expr) Expr {
	return Expr{kind: KindAgg, op: "mean", target: &e}
}
func Count() Expr { return Expr{kind: KindAgg, op: "count"} }
func NUnique(e Expr) Expr {
	return Expr{kind: KindAgg, op: "n_unique", target: &e}
}

func (e Expr) Alias(name string) Expr {
	e.alias = name
	return e
}

func (e Expr) Cast(dt dtypes.DataType) Expr {
	return Expr{kind: KindCast, target: &e, dtype: dt}
}

func (e Expr) Eq(other Expr) Expr { return bin("eq", e, other) }
func (e Expr) Ne(other Expr) Expr { return bin("ne", e, other) }
func (e Expr) Gt(other Expr) Expr { return bin("gt", e, other) }
func (e Expr) Ge(other Expr) Expr { return bin("ge", e, other) }
func (e Expr) Lt(other Expr) Expr { return bin("lt", e, other) }
func (e Expr) Le(other Expr) Expr { return bin("le", e, other) }
func (e Expr) Add(other Expr) Expr {
	return bin("add", e, other)
}
func (e Expr) Sub(other Expr) Expr {
	return bin("sub", e, other)
}
func (e Expr) Mul(other Expr) Expr {
	return bin("mul", e, other)
}
func (e Expr) Div(other Expr) Expr {
	return bin("div", e, other)
}
func (e Expr) And(other Expr) Expr {
	return bin("and", e, other)
}
func (e Expr) Or(other Expr) Expr {
	return bin("or", e, other)
}

func (e Expr) Not() Expr {
	return Expr{kind: KindUnary, op: "not", target: &e}
}

func (e Expr) IsNull() Expr {
	return Expr{kind: KindUnary, op: "is_null", target: &e}
}

func (e Expr) IsNotNull() Expr {
	return Expr{kind: KindUnary, op: "is_not_null", target: &e}
}

func (e Expr) StrLen() Expr {
	return Expr{kind: KindUnary, op: "str_len", target: &e}
}

func (e Expr) StrLower() Expr {
	return Expr{kind: KindUnary, op: "str_lower", target: &e}
}

func (e Expr) StrUpper() Expr {
	return Expr{kind: KindUnary, op: "str_upper", target: &e}
}

func (e Expr) StrReplace(old string, new string) Expr {
	return Expr{kind: KindUnary, op: "str_replace:" + old + ":" + new, target: &e}
}

func (e Expr) StrTrim() Expr {
	return Expr{kind: KindUnary, op: "str_trim", target: &e}
}

func (e Expr) DtYear() Expr {
	return Expr{kind: KindUnary, op: "dt_year", target: &e}
}

func (e Expr) DtMonth() Expr {
	return Expr{kind: KindUnary, op: "dt_month", target: &e}
}

func (e Expr) DtDay() Expr {
	return Expr{kind: KindUnary, op: "dt_day", target: &e}
}

func (e Expr) DtHour() Expr {
	return Expr{kind: KindUnary, op: "dt_hour", target: &e}
}

func (e Expr) DtWeekday() Expr {
	return Expr{kind: KindUnary, op: "dt_weekday", target: &e}
}

func (e Expr) ListLen() Expr {
	return Expr{kind: KindUnary, op: "list_len", target: &e}
}

func (e Expr) StructField(name string) Expr {
	return Expr{kind: KindUnary, op: "struct_field:" + name, target: &e}
}

func (e Expr) ListContains(other Expr) Expr {
	return bin("list_contains", e, other)
}

func (e Expr) ListGet(index Expr) Expr {
	return bin("list_get", e, index)
}

func (e Expr) StartsWith(prefix Expr) Expr {
	return bin("starts_with", e, prefix)
}

func (e Expr) Contains(substr Expr) Expr {
	return bin("contains", e, substr)
}

func When(cond Expr, thenExpr Expr, otherwise Expr) Expr {
	return Expr{kind: KindTern, op: "when", left: &cond, right: &thenExpr, extra: &otherwise}
}

func (e Expr) Name() string {
	if e.alias != "" {
		return e.alias
	}
	if e.kind == KindCol {
		return e.name
	}
	if e.kind == KindAgg && e.target != nil {
		return fmt.Sprintf("%s_%s", e.op, e.target.Name())
	}
	return "expr"
}

func (e Expr) Kind() Kind {
	return e.kind
}

func (e Expr) Left() *Expr {
	return e.left
}

func (e Expr) Right() *Expr {
	return e.right
}

func (e Expr) Value() any {
	return e.value
}

func (e Expr) Op() string {
	return e.op
}

func (e Expr) ColName() string {
	return e.name
}

func (e Expr) Target() *Expr {
	return e.target
}

func (e Expr) Extra() *Expr {
	return e.extra
}

func (e Expr) CastType() dtypes.DataType {
	return e.dtype
}

func bin(op string, left Expr, right Expr) Expr {
	return Expr{kind: KindBin, op: op, left: &left, right: &right}
}
