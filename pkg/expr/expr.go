package expr

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

type Kind string

const (
	KindCol   Kind = "col"
	KindLit   Kind = "lit"
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
	names  []string // multi-column selectors (Cols, Struct, Fold) and field lists
}

func Col(name string) Expr {
	return Expr{kind: KindCol, name: name}
}

// selectorAll is the column-name sentinel for the all-columns selector (pl.all()).
const selectorAll = "*"

// All selects every column (pl.all()). It is a multi-column selector expanded by
// the Select/WithColumns projection layer.
func All() Expr {
	return Expr{kind: KindCol, name: selectorAll}
}

// Cols selects the named columns (pl.col("a","b")). It expands, in the given
// order, to one column per name.
func Cols(names ...string) Expr {
	return Expr{kind: KindCol, op: "selector_cols", names: append([]string(nil), names...)}
}

// Exclude selects every column except the named ones (pl.exclude("a")).
func Exclude(names ...string) Expr {
	return All().Exclude(names...)
}

// StructCols packs the named columns into a single Struct column (pl.struct([...])).
// The default output name is "struct"; use Alias to rename.
func StructCols(names ...string) Expr {
	return Expr{kind: KindUnary, op: "struct_pack", name: "struct", names: append([]string(nil), names...)}
}

// FoldSpec carries the accumulator, reduce function, and operand exprs for Fold.
type FoldSpec struct {
	Acc   any
	Fn    func(acc any, next any) (any, error)
	Exprs []Expr
}

// Fold horizontally reduces the operand exprs left-to-right per row, starting from
// acc (pl.fold(acc, function, exprs)). fn is applied as fn(acc, next_value).
func Fold(acc any, fn func(acc any, next any) (any, error), exprs ...Expr) Expr {
	return Expr{kind: KindUnary, op: "fold", name: "fold", value: FoldSpec{Acc: acc, Fn: fn, Exprs: exprs}}
}

// SelectorColumns resolves a multi-column selector against the available column
// names (in schema order). ok is false for a normal single-column/value expr that
// the caller should evaluate as-is.
func SelectorColumns(e Expr, available []string) (cols []string, ok bool) {
	switch e.kind {
	case KindCol:
		switch {
		case e.op == "selector_cols":
			return append([]string(nil), e.names...), true
		case e.name == selectorAll:
			return append([]string(nil), available...), true
		case isRegexName(e.name):
			re, err := regexp.Compile(e.name)
			if err != nil {
				return nil, false
			}
			out := make([]string, 0, len(available))
			for _, n := range available {
				if re.MatchString(n) {
					out = append(out, n)
				}
			}
			return out, true
		}
		return nil, false
	case KindUnary:
		if strings.HasPrefix(e.op, "exclude:") && e.target != nil {
			base, isSel := SelectorColumns(*e.target, available)
			if !isSel {
				return nil, false
			}
			excluded := map[string]struct{}{}
			for _, n := range strings.Split(strings.TrimPrefix(e.op, "exclude:"), ",") {
				if n != "" {
					excluded[n] = struct{}{}
				}
			}
			out := make([]string, 0, len(base))
			for _, n := range base {
				if _, drop := excluded[n]; !drop {
					out = append(out, n)
				}
			}
			return out, true
		}
	}
	return nil, false
}

// isRegexName reports whether a column selector follows Polars' regex convention
// (anchored with ^...$), e.g. pl.col("^foo.*$").
func isRegexName(name string) bool {
	return len(name) >= 2 && name[0] == '^' && name[len(name)-1] == '$'
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

func (e Expr) StrReplaceAll(old string, new string) Expr {
	return Expr{kind: KindUnary, op: "str_replace_all:" + old + ":" + new, target: &e}
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

func (e Expr) CumSum() Expr {
	return Expr{kind: KindUnary, op: "cum_sum", target: &e}
}

func (e Expr) CumCount() Expr {
	return Expr{kind: KindUnary, op: "cum_count", target: &e}
}

func (e Expr) Rank() Expr {
	return Expr{kind: KindUnary, op: "rank", target: &e}
}

func (e Expr) Over(partitionBy ...string) Expr {
	return Expr{kind: KindUnary, op: "over:" + strings.Join(partitionBy, ","), target: &e}
}

func (e Expr) Replace(old Expr, new Expr) Expr {
	oldCopy := old
	newCopy := new
	return Expr{kind: KindTern, op: "replace", target: &e, left: &oldCopy, right: &newCopy}
}

func (e Expr) FillNull(value Expr) Expr {
	return bin("fill_null_expr", e, value)
}

func (e Expr) FillNaN(value Expr) Expr {
	return bin("fill_nan_expr", e, value)
}

func (e Expr) RollingMin(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_min:" + strconv.Itoa(window), target: &e}
}

func (e Expr) RollingMax(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_max:" + strconv.Itoa(window), target: &e}
}

func (e Expr) RollingMean(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_mean:" + strconv.Itoa(window), target: &e}
}

func (e Expr) RollingSum(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_sum:" + strconv.Itoa(window), target: &e}
}

func (e Expr) RollingStd(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_std:" + strconv.Itoa(window), target: &e}
}

func (e Expr) RollingVar(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_var:" + strconv.Itoa(window), target: &e}
}

func (e Expr) Clip(min Expr, max Expr) Expr {
	return Expr{kind: KindTern, op: "clip", target: &e, left: &min, right: &max}
}

func (e Expr) Round() Expr {
	return Expr{kind: KindUnary, op: "round", target: &e}
}

func (e Expr) Pow(power Expr) Expr {
	return bin("pow", e, power)
}

func (e Expr) Log() Expr {
	return Expr{kind: KindUnary, op: "log", target: &e}
}

func (e Expr) Sqrt() Expr {
	return Expr{kind: KindUnary, op: "sqrt", target: &e}
}

func (e Expr) Dt() Expr {
	return Expr{kind: KindUnary, op: "dt", target: &e}
}

func (e Expr) Str() Expr {
	return Expr{kind: KindUnary, op: "str", target: &e}
}

func (e Expr) List() Expr {
	return Expr{kind: KindUnary, op: "list", target: &e}
}

func (e Expr) Struct() Expr {
	return Expr{kind: KindUnary, op: "struct", target: &e}
}

func (e Expr) Abs() Expr {
	return Expr{kind: KindUnary, op: "abs", target: &e}
}

func (e Expr) Exp() Expr {
	return Expr{kind: KindUnary, op: "exp", target: &e}
}

func (e Expr) AggGroups() Expr {
	return Expr{kind: KindUnary, op: "agg_groups", target: &e}
}

func (e Expr) All() Expr {
	return Expr{kind: KindUnary, op: "all", target: &e}
}

func (e Expr) And_(other Expr) Expr {
	return bin("and_", e, other)
}

func (e Expr) Any() Expr {
	return Expr{kind: KindUnary, op: "any", target: &e}
}

func (e Expr) Append(other Expr) Expr {
	return bin("append", e, other)
}

func (e Expr) ApproxNUnique() Expr {
	return Expr{kind: KindUnary, op: "approx_n_unique", target: &e}
}

func (e Expr) Arccos() Expr {
	return Expr{kind: KindUnary, op: "arccos", target: &e}
}

func (e Expr) Arccosh() Expr {
	return Expr{kind: KindUnary, op: "arccosh", target: &e}
}

func (e Expr) Arcsin() Expr {
	return Expr{kind: KindUnary, op: "arcsin", target: &e}
}

func (e Expr) Arcsinh() Expr {
	return Expr{kind: KindUnary, op: "arcsinh", target: &e}
}

func (e Expr) Arctan() Expr {
	return Expr{kind: KindUnary, op: "arctan", target: &e}
}

func (e Expr) Arctanh() Expr {
	return Expr{kind: KindUnary, op: "arctanh", target: &e}
}

func (e Expr) ArgMax() Expr {
	return Expr{kind: KindUnary, op: "arg_max", target: &e}
}

func (e Expr) ArgMin() Expr {
	return Expr{kind: KindUnary, op: "arg_min", target: &e}
}

func (e Expr) ArgSort() Expr {
	return Expr{kind: KindUnary, op: "arg_sort", target: &e}
}

func (e Expr) ArgTrue() Expr {
	return Expr{kind: KindUnary, op: "arg_true", target: &e}
}

func (e Expr) ArgUnique() Expr {
	return Expr{kind: KindUnary, op: "arg_unique", target: &e}
}

func (e Expr) Arr() Expr {
	return Expr{kind: KindUnary, op: "arr", target: &e}
}

func (e Expr) BackwardFill() Expr {
	return Expr{kind: KindUnary, op: "backward_fill", target: &e}
}

func (e Expr) Bin() Expr {
	return Expr{kind: KindUnary, op: "bin", target: &e}
}

func (e Expr) BitwiseAnd(other Expr) Expr {
	return bin("bitwise_and", e, other)
}

func (e Expr) BitwiseCountOnes() Expr {
	return Expr{kind: KindUnary, op: "bitwise_count_ones", target: &e}
}

func (e Expr) BitwiseCountZeros() Expr {
	return Expr{kind: KindUnary, op: "bitwise_count_zeros", target: &e}
}

func (e Expr) BitwiseLeadingOnes() Expr {
	return Expr{kind: KindUnary, op: "bitwise_leading_ones", target: &e}
}

func (e Expr) BitwiseLeadingZeros() Expr {
	return Expr{kind: KindUnary, op: "bitwise_leading_zeros", target: &e}
}

func (e Expr) BitwiseTrailingOnes() Expr {
	return Expr{kind: KindUnary, op: "bitwise_trailing_ones", target: &e}
}

func (e Expr) BitwiseTrailingZeros() Expr {
	return Expr{kind: KindUnary, op: "bitwise_trailing_zeros", target: &e}
}

func (e Expr) BitwiseOr(other Expr) Expr {
	return bin("bitwise_or", e, other)
}

func (e Expr) BitwiseXor(other Expr) Expr {
	return bin("bitwise_xor", e, other)
}

func (e Expr) BottomK(k int) Expr {
	return Expr{kind: KindUnary, op: "bottom_k:" + strconv.Itoa(k), target: &e}
}

func (e Expr) BottomKBy(by Expr, k int) Expr {
	return Expr{kind: KindTern, op: "bottom_k_by:" + strconv.Itoa(k), target: &e, left: &by}
}

func (e Expr) Cat() Expr {
	return Expr{kind: KindUnary, op: "cat", target: &e}
}

func (e Expr) Cbrt() Expr {
	return Expr{kind: KindUnary, op: "cbrt", target: &e}
}

func (e Expr) Ceil() Expr {
	return Expr{kind: KindUnary, op: "ceil", target: &e}
}

func (e Expr) Cos() Expr {
	return Expr{kind: KindUnary, op: "cos", target: &e}
}

func (e Expr) Sin() Expr {
	return Expr{kind: KindUnary, op: "sin", target: &e}
}

func (e Expr) Cosh() Expr {
	return Expr{kind: KindUnary, op: "cosh", target: &e}
}

func (e Expr) Sinh() Expr {
	return Expr{kind: KindUnary, op: "sinh", target: &e}
}

func (e Expr) Tan() Expr {
	return Expr{kind: KindUnary, op: "tan", target: &e}
}

func (e Expr) Tanh() Expr {
	return Expr{kind: KindUnary, op: "tanh", target: &e}
}

func (e Expr) Cot() Expr {
	return Expr{kind: KindUnary, op: "cot", target: &e}
}

func (e Expr) Count() Expr {
	return Expr{kind: KindUnary, op: "count", target: &e}
}

func (e Expr) CumMax() Expr {
	return Expr{kind: KindUnary, op: "cum_max", target: &e}
}

func (e Expr) CumMin() Expr {
	return Expr{kind: KindUnary, op: "cum_min", target: &e}
}

func (e Expr) CumProd() Expr {
	return Expr{kind: KindUnary, op: "cum_prod", target: &e}
}

func (e Expr) CumulativeEval() Expr {
	return Expr{kind: KindUnary, op: "cumulative_eval", target: &e}
}

func (e Expr) Cut() Expr {
	return Expr{kind: KindUnary, op: "cut", target: &e}
}

func (e Expr) Degrees() Expr {
	return Expr{kind: KindUnary, op: "degrees", target: &e}
}

func (e Expr) Deserialize() Expr {
	return Expr{kind: KindUnary, op: "deserialize", target: &e}
}

func (e Expr) Diff() Expr {
	return Expr{kind: KindUnary, op: "diff", target: &e}
}

func (e Expr) Dot(other Expr) Expr {
	return bin("dot", e, other)
}

func (e Expr) DropNans() Expr {
	return Expr{kind: KindUnary, op: "drop_nans", target: &e}
}

func (e Expr) DropNulls() Expr {
	return Expr{kind: KindUnary, op: "drop_nulls", target: &e}
}

func (e Expr) Entropy() Expr {
	return Expr{kind: KindUnary, op: "entropy", target: &e}
}

func (e Expr) EqMissing(other Expr) Expr {
	return bin("eq_missing", e, other)
}

func (e Expr) EwmMean() Expr {
	return Expr{kind: KindUnary, op: "ewm_mean", target: &e}
}

func (e Expr) EwmMeanBy(by Expr) Expr {
	return Expr{kind: KindTern, op: "ewm_mean_by", target: &e, left: &by}
}

func (e Expr) EwmStd() Expr {
	return Expr{kind: KindUnary, op: "ewm_std", target: &e}
}

func (e Expr) EwmVar() Expr {
	return Expr{kind: KindUnary, op: "ewm_var", target: &e}
}

func (e Expr) Exclude(columns ...string) Expr {
	return Expr{kind: KindUnary, op: "exclude:" + strings.Join(columns, ","), target: &e}
}

func (e Expr) Explode() Expr {
	return Expr{kind: KindUnary, op: "explode", target: &e}
}

func (e Expr) Ext() Expr {
	return Expr{kind: KindUnary, op: "ext", target: &e}
}

func (e Expr) ExtendConstant(value Expr) Expr {
	return Expr{kind: KindTern, op: "extend_constant", target: &e, left: &value}
}

func (e Expr) Filter(mask Expr) Expr {
	return bin("filter_expr", e, mask)
}

func (e Expr) First() Expr {
	return Expr{kind: KindUnary, op: "first", target: &e}
}

func (e Expr) Flatten() Expr {
	return Expr{kind: KindUnary, op: "flatten", target: &e}
}

func (e Expr) Floor() Expr {
	return Expr{kind: KindUnary, op: "floor", target: &e}
}

func (e Expr) FloorDiv(other Expr) Expr {
	return bin("floordiv", e, other)
}

func (e Expr) Floordiv(other Expr) Expr {
	return e.FloorDiv(other)
}

func (e Expr) ForwardFill() Expr {
	return Expr{kind: KindUnary, op: "forward_fill", target: &e}
}

func (e Expr) FromJSON() Expr {
	return Expr{kind: KindUnary, op: "from_json", target: &e}
}

func (e Expr) FromJson() Expr {
	return e.FromJSON()
}

func (e Expr) Gather(index Expr) Expr {
	return bin("gather", e, index)
}

func (e Expr) GatherEvery(step int) Expr {
	return Expr{kind: KindUnary, op: "gather_every:" + strconv.Itoa(step), target: &e}
}

func (e Expr) Get(index Expr) Expr {
	return bin("get", e, index)
}

func (e Expr) HasNulls() Expr {
	return Expr{kind: KindUnary, op: "has_nulls", target: &e}
}

func (e Expr) Hash() Expr {
	return Expr{kind: KindUnary, op: "hash", target: &e}
}

func (e Expr) Head(n int) Expr {
	return Expr{kind: KindUnary, op: "head:" + strconv.Itoa(n), target: &e}
}

func (e Expr) Hist() Expr {
	return Expr{kind: KindUnary, op: "hist", target: &e}
}

func (e Expr) Implode() Expr {
	return Expr{kind: KindUnary, op: "implode", target: &e}
}

func (e Expr) IndexOf(item Expr) Expr {
	return bin("index_of", e, item)
}

func (e Expr) Inspect() Expr {
	return Expr{kind: KindUnary, op: "inspect", target: &e}
}

func (e Expr) Interpolate() Expr {
	return Expr{kind: KindUnary, op: "interpolate", target: &e}
}

func (e Expr) InterpolateBy(by Expr) Expr {
	return Expr{kind: KindTern, op: "interpolate_by", target: &e, left: &by}
}

func (e Expr) IsBetween(lower Expr, upper Expr) Expr {
	return Expr{kind: KindTern, op: "is_between", target: &e, left: &lower, right: &upper}
}

func (e Expr) IsClose(other Expr) Expr {
	return bin("is_close", e, other)
}

func (e Expr) IsDuplicated() Expr {
	return Expr{kind: KindUnary, op: "is_duplicated", target: &e}
}

func (e Expr) IsFinite() Expr {
	return Expr{kind: KindUnary, op: "is_finite", target: &e}
}

func (e Expr) IsFirstDistinct() Expr {
	return Expr{kind: KindUnary, op: "is_first_distinct", target: &e}
}

func (e Expr) IsIn(values Expr) Expr {
	return bin("is_in", e, values)
}

func (e Expr) IsInfinite() Expr {
	return Expr{kind: KindUnary, op: "is_infinite", target: &e}
}

func (e Expr) IsLastDistinct() Expr {
	return Expr{kind: KindUnary, op: "is_last_distinct", target: &e}
}

func (e Expr) IsNan() Expr {
	return Expr{kind: KindUnary, op: "is_nan", target: &e}
}

func (e Expr) IsNotNan() Expr {
	return Expr{kind: KindUnary, op: "is_not_nan", target: &e}
}

func (e Expr) IsUnique() Expr {
	return Expr{kind: KindUnary, op: "is_unique", target: &e}
}

func (e Expr) Item() Expr {
	return Expr{kind: KindUnary, op: "item", target: &e}
}

func (e Expr) Kurtosis() Expr {
	return Expr{kind: KindUnary, op: "kurtosis", target: &e}
}

func (e Expr) Last() Expr {
	return Expr{kind: KindUnary, op: "last", target: &e}
}

func (e Expr) Limit(n int) Expr {
	return Expr{kind: KindUnary, op: "limit:" + strconv.Itoa(n), target: &e}
}

func (e Expr) Log10() Expr {
	return Expr{kind: KindUnary, op: "log10", target: &e}
}

func (e Expr) Log1p() Expr {
	return Expr{kind: KindUnary, op: "log1p", target: &e}
}

func (e Expr) LowerBound() Expr {
	return Expr{kind: KindUnary, op: "lower_bound", target: &e}
}

func (e Expr) MapBatches() Expr {
	return Expr{kind: KindUnary, op: "map_batches", target: &e}
}

func (e Expr) MapElements() Expr {
	return Expr{kind: KindUnary, op: "map_elements", target: &e}
}

func (e Expr) Max() Expr {
	return Expr{kind: KindUnary, op: "max", target: &e}
}

func (e Expr) MaxBy(by Expr) Expr {
	return Expr{kind: KindTern, op: "max_by", target: &e, left: &by}
}

func (e Expr) Mean() Expr {
	return Expr{kind: KindUnary, op: "mean", target: &e}
}

func (e Expr) Median() Expr {
	return Expr{kind: KindUnary, op: "median", target: &e}
}

func (e Expr) Meta() Expr {
	return Expr{kind: KindUnary, op: "meta", target: &e}
}

func (e Expr) Min() Expr {
	return Expr{kind: KindUnary, op: "min", target: &e}
}

func (e Expr) MinBy(by Expr) Expr {
	return Expr{kind: KindTern, op: "min_by", target: &e, left: &by}
}

func (e Expr) Mod(other Expr) Expr {
	return bin("mod", e, other)
}

func (e Expr) Mode() Expr {
	return Expr{kind: KindUnary, op: "mode", target: &e}
}

func (e Expr) NUnique() Expr {
	return Expr{kind: KindUnary, op: "n_unique", target: &e}
}

func (e Expr) NanMax() Expr {
	return Expr{kind: KindUnary, op: "nan_max", target: &e}
}

func (e Expr) NanMin() Expr {
	return Expr{kind: KindUnary, op: "nan_min", target: &e}
}

func (e Expr) NeMissing(other Expr) Expr {
	return bin("ne_missing", e, other)
}

func (e Expr) Neg() Expr {
	return Expr{kind: KindUnary, op: "neg", target: &e}
}

func (e Expr) Not_() Expr {
	return Expr{kind: KindUnary, op: "not_", target: &e}
}

func (e Expr) NullCount() Expr {
	return Expr{kind: KindUnary, op: "null_count", target: &e}
}

func (e Expr) Or_(other Expr) Expr {
	return bin("or_", e, other)
}

func (e Expr) PctChange() Expr {
	return Expr{kind: KindUnary, op: "pct_change", target: &e}
}

func (e Expr) PeakMax() Expr {
	return Expr{kind: KindUnary, op: "peak_max", target: &e}
}

func (e Expr) PeakMin() Expr {
	return Expr{kind: KindUnary, op: "peak_min", target: &e}
}

func (e Expr) Pipe() Expr {
	return Expr{kind: KindUnary, op: "pipe", target: &e}
}

func (e Expr) Product() Expr {
	return Expr{kind: KindUnary, op: "product", target: &e}
}

func (e Expr) QCut() Expr {
	return Expr{kind: KindUnary, op: "qcut", target: &e}
}

func (e Expr) Quantile() Expr {
	return Expr{kind: KindUnary, op: "quantile", target: &e}
}

func (e Expr) Sign() Expr {
	return Expr{kind: KindUnary, op: "sign", target: &e}
}

func (e Expr) Radians() Expr {
	return Expr{kind: KindUnary, op: "radians", target: &e}
}

func (e Expr) RoundSigFigs(digits int) Expr {
	return Expr{kind: KindUnary, op: "round_sig_figs:" + strconv.Itoa(digits), target: &e}
}

func (e Expr) RollingMaxBy(by Expr, window int) Expr {
	return Expr{kind: KindTern, op: "rolling_max_by:" + strconv.Itoa(window), target: &e, left: &by}
}

func (e Expr) RollingMeanBy(by Expr, window int) Expr {
	return Expr{kind: KindTern, op: "rolling_mean_by:" + strconv.Itoa(window), target: &e, left: &by}
}

func (e Expr) RollingMinBy(by Expr, window int) Expr {
	return Expr{kind: KindTern, op: "rolling_min_by:" + strconv.Itoa(window), target: &e, left: &by}
}

func (e Expr) RollingSumBy(by Expr, window int) Expr {
	return Expr{kind: KindTern, op: "rolling_sum_by:" + strconv.Itoa(window), target: &e, left: &by}
}

func (e Expr) RollingStdBy(by Expr, window int) Expr {
	return Expr{kind: KindTern, op: "rolling_std_by:" + strconv.Itoa(window), target: &e, left: &by}
}

func (e Expr) RollingVarBy(by Expr, window int) Expr {
	return Expr{kind: KindTern, op: "rolling_var_by:" + strconv.Itoa(window), target: &e, left: &by}
}

func (e Expr) RollingMedian(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_median:" + strconv.Itoa(window), target: &e}
}

func (e Expr) RollingMedianBy(by Expr, window int) Expr {
	return Expr{kind: KindTern, op: "rolling_median_by:" + strconv.Itoa(window), target: &e, left: &by}
}

func (e Expr) RollingQuantile(window int, q float64) Expr {
	return Expr{kind: KindUnary, op: "rolling_quantile:" + strconv.Itoa(window) + ":" + strconv.FormatFloat(q, 'f', -1, 64), target: &e}
}

func (e Expr) RollingQuantileBy(by Expr, window int, q float64) Expr {
	return Expr{kind: KindTern, op: "rolling_quantile_by:" + strconv.Itoa(window) + ":" + strconv.FormatFloat(q, 'f', -1, 64), target: &e, left: &by}
}

func (e Expr) RollingSkew(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_skew:" + strconv.Itoa(window), target: &e}
}

func (e Expr) RollingKurtosis(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_kurtosis:" + strconv.Itoa(window), target: &e}
}

func (e Expr) RollingMap(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_map:" + strconv.Itoa(window), target: &e}
}

func (e Expr) Rolling(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling:" + strconv.Itoa(window), target: &e}
}

func (e Expr) RollingRank(window int) Expr {
	return Expr{kind: KindUnary, op: "rolling_rank:" + strconv.Itoa(window), target: &e}
}

func (e Expr) RollingRankBy(by Expr, window int) Expr {
	return Expr{kind: KindTern, op: "rolling_rank_by:" + strconv.Itoa(window), target: &e, left: &by}
}

func (e Expr) Sort(descending bool) Expr {
	return Expr{kind: KindUnary, op: "sort:" + strconv.FormatBool(descending), target: &e}
}

func (e Expr) SortBy(by Expr, descending bool) Expr {
	return Expr{kind: KindTern, op: "sort_by:" + strconv.FormatBool(descending), target: &e, left: &by}
}

func (e Expr) Slice(offset int, length int) Expr {
	return Expr{kind: KindUnary, op: "slice:" + strconv.Itoa(offset) + ":" + strconv.Itoa(length), target: &e}
}

func (e Expr) Tail(n int) Expr {
	return Expr{kind: KindUnary, op: "tail:" + strconv.Itoa(n), target: &e}
}

func (e Expr) Unique() Expr {
	return Expr{kind: KindUnary, op: "unique", target: &e}
}

func (e Expr) UniqueCounts() Expr {
	return Expr{kind: KindUnary, op: "unique_counts", target: &e}
}

func (e Expr) ValueCounts() Expr {
	return Expr{kind: KindUnary, op: "value_counts", target: &e}
}

func (e Expr) Rechunk() Expr {
	return Expr{kind: KindUnary, op: "rechunk", target: &e}
}

func (e Expr) Reinterpret() Expr {
	return Expr{kind: KindUnary, op: "reinterpret", target: &e}
}

func (e Expr) RepeatBy(n int) Expr {
	return Expr{kind: KindUnary, op: "repeat_by:" + strconv.Itoa(n), target: &e}
}

func (e Expr) ReplaceStrict(old Expr, new Expr) Expr {
	oldCopy := old
	newCopy := new
	return Expr{kind: KindTern, op: "replace_strict", target: &e, left: &oldCopy, right: &newCopy}
}

func (e Expr) Reshape(dims ...int) Expr {
	parts := make([]string, 0, len(dims))
	for _, dim := range dims {
		parts = append(parts, strconv.Itoa(dim))
	}
	return Expr{kind: KindUnary, op: "reshape:" + strings.Join(parts, ","), target: &e}
}

func (e Expr) Rle() Expr {
	return Expr{kind: KindUnary, op: "rle", target: &e}
}

func (e Expr) RleId() Expr {
	return Expr{kind: KindUnary, op: "rle_id", target: &e}
}

func (e Expr) Sample(n int, seed int64) Expr {
	return Expr{kind: KindUnary, op: "sample:" + strconv.Itoa(n) + ":" + strconv.FormatInt(seed, 10), target: &e}
}

func (e Expr) SearchSorted(value Expr) Expr {
	return bin("search_sorted", e, value)
}

func (e Expr) SetSorted(descending bool) Expr {
	return Expr{kind: KindUnary, op: "set_sorted:" + strconv.FormatBool(descending), target: &e}
}

func (e Expr) Shift(periods int) Expr {
	return Expr{kind: KindUnary, op: "shift:" + strconv.Itoa(periods), target: &e}
}

func (e Expr) Reverse() Expr {
	return Expr{kind: KindUnary, op: "reverse", target: &e}
}

func (e Expr) ShrinkDtype() Expr {
	return Expr{kind: KindUnary, op: "shrink_dtype", target: &e}
}

func (e Expr) Shuffle(seed int64) Expr {
	return Expr{kind: KindUnary, op: "shuffle:" + strconv.FormatInt(seed, 10), target: &e}
}

func (e Expr) Skew() Expr {
	return Expr{kind: KindUnary, op: "skew", target: &e}
}

func (e Expr) Std() Expr {
	return Expr{kind: KindUnary, op: "std", target: &e}
}

func (e Expr) Sum() Expr {
	return Expr{kind: KindUnary, op: "sum", target: &e}
}

func (e Expr) Var() Expr {
	return Expr{kind: KindUnary, op: "var", target: &e}
}

func (e Expr) Where(mask Expr) Expr {
	return bin("where", e, mask)
}

func (e Expr) Xor(other Expr) Expr {
	return bin("xor", e, other)
}

func (e Expr) ToPhysical() Expr {
	return Expr{kind: KindUnary, op: "to_physical", target: &e}
}

func (e Expr) TopK(k int) Expr {
	return Expr{kind: KindUnary, op: "top_k:" + strconv.Itoa(k), target: &e}
}

func (e Expr) TopKBy(by Expr, k int) Expr {
	return Expr{kind: KindTern, op: "top_k_by:" + strconv.Itoa(k), target: &e, left: &by}
}

func (e Expr) Truncate() Expr {
	return Expr{kind: KindUnary, op: "truncate", target: &e}
}

func (e Expr) TrueDiv(other Expr) Expr {
	return bin("true_div", e, other)
}

func (e Expr) UpperBound() Expr {
	return Expr{kind: KindUnary, op: "upper_bound", target: &e}
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
	if e.name != "" {
		return e.name
	}
	return "expr"
}

// Names returns the column-name list carried by multi-column exprs (Cols,
// StructCols, Fold). It is nil for ordinary exprs.
func (e Expr) Names() []string {
	return e.names
}

// MapAggregates returns a copy of e with every aggregation sub-expression replaced
// by a literal scalar. fn classifies a node: it returns (scalar, true) for an
// aggregation to fold to a constant, or (_, false) to recurse into the node's
// children. This lets the eager evaluator support aggregations nested inside
// row-wise expressions (e.g. pl.col("b") + pl.col("c").first()) by precomputing
// the reductions and broadcasting them.
func MapAggregates(e Expr, fn func(Expr) (any, bool, error)) (Expr, error) {
	scalar, isAgg, err := fn(e)
	if err != nil {
		return Expr{}, err
	}
	if isAgg {
		lit := Lit(scalar)
		if e.alias != "" {
			lit = lit.Alias(e.alias)
		}
		return lit, nil
	}
	// A window/group context (.over()) evaluates its own aggregation per group; do
	// not fold aggregations nested under it into a global scalar.
	if strings.HasPrefix(e.op, "over:") {
		return e, nil
	}
	out := e
	for _, slot := range []**Expr{&out.target, &out.left, &out.right, &out.extra} {
		if *slot != nil {
			child, err := MapAggregates(**slot, fn)
			if err != nil {
				return Expr{}, err
			}
			*slot = &child
		}
	}
	return out, nil
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
