package expr

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"math/bits"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

type RowValueGetter interface {
	ValueByName(name string) (any, bool)
}

func Eval(e Expr, row RowValueGetter) (any, error) {
	switch e.Kind() {
	case KindCol:
		v, ok := row.ValueByName(e.ColName())
		if !ok {
			return nil, fmt.Errorf("column %s not found", e.ColName())
		}
		return v, nil
	case KindLit:
		return e.Value(), nil
	case KindCast:
		v, err := Eval(*e.Target(), row)
		if err != nil {
			return nil, err
		}
		return cast(v, e.CastType())
	case KindUnary:
		// struct_pack reads several columns by name and assembles a struct value;
		// it has no single target, so handle it before the target deref below.
		if e.Op() == "struct_pack" {
			m := make(map[string]any, len(e.names))
			for _, name := range e.names {
				val, ok := row.ValueByName(name)
				if !ok {
					return nil, fmt.Errorf("struct field column %s not found", name)
				}
				m[name] = val
			}
			return m, nil
		}
		// fold horizontally reduces several operand exprs per row; no single target.
		if e.Op() == "fold" {
			spec, ok := e.Value().(FoldSpec)
			if !ok {
				return nil, fmt.Errorf("fold missing spec")
			}
			acc := spec.Acc
			for i := range spec.Exprs {
				v, err := Eval(spec.Exprs[i], row)
				if err != nil {
					return nil, err
				}
				acc, err = spec.Fn(acc, v)
				if err != nil {
					return nil, err
				}
			}
			return acc, nil
		}
		v, err := Eval(*e.Target(), row)
		if err != nil {
			return nil, err
		}
		switch e.Op() {
		case "not":
			if v == nil {
				return nil, nil
			}
			b, ok := v.(bool)
			if !ok {
				return nil, fmt.Errorf("not expects bool")
			}
			return !b, nil
		case "is_null":
			return v == nil, nil
		case "is_not_null":
			return v != nil, nil
		case "str_len":
			if v == nil {
				return nil, nil
			}
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("str_len expects string")
			}
			return int64(len(s)), nil
		case "str_lower":
			if v == nil {
				return nil, nil
			}
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("str_lower expects string")
			}
			return strings.ToLower(s), nil
		case "str_upper":
			if v == nil {
				return nil, nil
			}
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("str_upper expects string")
			}
			return strings.ToUpper(s), nil
		case "str_trim":
			if v == nil {
				return nil, nil
			}
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("str_trim expects string")
			}
			return strings.TrimSpace(s), nil
		case "list_len":
			list, ok := v.([]any)
			if !ok {
				return nil, fmt.Errorf("list_len expects list")
			}
			return int64(len(list)), nil
		case "reverse":
			if ra, ok := row.(RasterRowAccess); ok {
				return Eval(*e.Target(), reverseRowCtx{base: ra})
			}
			return nil, fmt.Errorf("reverse requires RasterRowAccess row context")
		case "round":
			switch t := v.(type) {
			case float64:
				return math.Round(t), nil
			case int64:
				return t, nil
			default:
				return nil, fmt.Errorf("round expects numeric")
			}
		case "agg_groups", "approx_n_unique", "arg_max", "arg_min", "arg_sort", "arg_true", "arg_unique", "arr", "backward_fill", "bin", "cat", "count", "cum_count", "cum_max", "cum_min", "cum_prod", "cum_sum", "cumulative_eval", "cut", "deserialize", "diff", "drop_nans", "drop_nulls", "dt", "entropy", "ewm_mean", "ewm_std", "ewm_var", "explode", "ext", "first", "flatten", "forward_fill", "hash", "hist", "implode", "inspect", "interpolate", "is_duplicated", "is_first_distinct", "is_last_distinct", "is_unique", "item", "kurtosis", "last", "list", "lower_bound", "map_batches", "map_elements", "max", "mean", "median", "meta", "min", "mode", "n_unique", "nan_max", "nan_min", "null_count", "pct_change", "peak_max", "peak_min", "pipe", "product", "qcut", "quantile", "rank", "rechunk", "reinterpret", "rle", "rle_id", "shrink_dtype", "skew", "std", "str", "struct", "sum", "to_physical", "truncate", "unique", "unique_counts", "upper_bound", "value_counts", "var":
			return v, nil
		case "all":
			if b, ok := v.(bool); ok {
				return b, nil
			}
			if list, ok := v.([]any); ok {
				for _, item := range list {
					if flag, ok := item.(bool); !ok || !flag {
						return false, nil
					}
				}
				return true, nil
			}
			return false, nil
		case "any":
			if b, ok := v.(bool); ok {
				return b, nil
			}
			if list, ok := v.([]any); ok {
				for _, item := range list {
					if flag, ok := item.(bool); ok && flag {
						return true, nil
					}
				}
			}
			return false, nil
		case "from_json":
			if s, ok := v.(string); ok {
				var out any
				if err := json.Unmarshal([]byte(s), &out); err != nil {
					return nil, err
				}
				return out, nil
			}
			return nil, fmt.Errorf("from_json expects string")
		case "has_nulls":
			if list, ok := v.([]any); ok {
				for _, item := range list {
					if item == nil {
						return true, nil
					}
				}
				return false, nil
			}
			return v == nil, nil
		case "is_finite":
			if f, ok := ToFloat(v); ok {
				return !math.IsNaN(f) && !math.IsInf(f, 0), nil
			}
			return false, nil
		case "is_infinite":
			if f, ok := ToFloat(v); ok {
				return math.IsInf(f, 0), nil
			}
			return false, nil
		case "is_nan":
			if f, ok := ToFloat(v); ok {
				return math.IsNaN(f), nil
			}
			return false, nil
		case "is_not_nan":
			if f, ok := ToFloat(v); ok {
				return !math.IsNaN(f), nil
			}
			return v != nil, nil
		case "neg":
			switch t := v.(type) {
			case int64:
				return -t, nil
			case float64:
				return -t, nil
			}
			return nil, fmt.Errorf("neg expects numeric")
		case "not_":
			if b, ok := v.(bool); ok {
				return !b, nil
			}
			return nil, fmt.Errorf("not_ expects bool")
		case "abs":
			switch t := v.(type) {
			case float64:
				return math.Abs(t), nil
			case int64:
				if t < 0 {
					return -t, nil
				}
				return t, nil
			default:
				return nil, fmt.Errorf("abs expects numeric")
			}
		default:
			if fn, ok := floatUnaryOps[e.Op()]; ok {
				f, ok := ToFloat(v)
				if !ok {
					return nil, fmt.Errorf("%s expects numeric", e.Op())
				}
				return fn(f), nil
			}
			if fn, ok := dtUnaryOps[e.Op()]; ok {
				t, ok := v.(time.Time)
				if !ok {
					return nil, fmt.Errorf("%s expects datetime", e.Op())
				}
				return fn(t), nil
			}
			if fn, ok := bitwiseUnaryOps[e.Op()]; ok {
				i, ok := toInt64(v)
				if !ok {
					return nil, fmt.Errorf("%s expects int", e.Op())
				}
				return int64(fn(uint64(i))), nil
			}
			if name, arg, ok := strings.Cut(e.Op(), ":"); ok {
				if out, handled, err := evalParamUnary(name, arg, v); handled {
					return out, err
				}
			}
			if out, handled, err := evalExtraUnary(e.Op(), v); handled {
				return out, err
			}
			return nil, fmt.Errorf("unsupported unary op %s", e.Op())
		}
	case KindTern:
		op := e.Op()
		// Pass-through ops return the target without evaluating their other operands
		// (such as the `by` column); the parameterized ones carry an argument after ":".
		if name, _, ok := strings.Cut(op, ":"); ok {
			switch name {
			case "bottom_k_by", "top_k_by", "sort_by", "rolling_max_by", "rolling_mean_by", "rolling_min_by", "rolling_sum_by", "rolling_std_by", "rolling_var_by", "rolling_median_by", "rolling_quantile_by", "rolling_rank_by":
				return Eval(*e.Target(), row)
			}
		}
		switch op {
		case "ewm_mean_by", "extend_constant", "max_by", "min_by", "interpolate_by":
			return Eval(*e.Target(), row)
		case "str_pad_start", "str_pad_end", "str_split_part":
			target, left, right, err := evalTernOperands(e, row)
			if err != nil {
				return nil, err
			}
			out, _, err := evalExtraTern(op, target, left, right)
			return out, err
		case "clip":
			current, minValue, maxValue, err := evalTernOperands(e, row)
			if err != nil {
				return nil, err
			}
			cur, cok := ToFloat(current)
			minv, mok := ToFloat(minValue)
			maxv, xok := ToFloat(maxValue)
			if !cok || !mok || !xok {
				return nil, fmt.Errorf("clip expects numeric")
			}
			if cur < minv {
				return minv, nil
			}
			if cur > maxv {
				return maxv, nil
			}
			return cur, nil
		case "replace", "replace_strict":
			current, oldValue, newValue, err := evalTernOperands(e, row)
			if err != nil {
				return nil, err
			}
			if current == oldValue {
				return newValue, nil
			}
			return current, nil
		case "is_between":
			current, lower, upper, err := evalTernOperands(e, row)
			if err != nil {
				return nil, err
			}
			cv, cok := ToFloat(current)
			lv, lok := ToFloat(lower)
			uv, uok := ToFloat(upper)
			if !cok || !lok || !uok {
				return nil, fmt.Errorf("is_between expects numeric")
			}
			return cv >= lv && cv <= uv, nil
		}
		cond, err := Eval(*e.Left(), row)
		if err != nil {
			return nil, err
		}
		c, ok := cond.(bool)
		if !ok {
			return nil, fmt.Errorf("when expects bool condition")
		}
		if c {
			return Eval(*e.Right(), row)
		}
		if e.Extra() == nil {
			return nil, fmt.Errorf("when expects otherwise branch")
		}
		return Eval(*e.Extra(), row)
	case KindBin:
		l, err := Eval(*e.Left(), row)
		if err != nil {
			return nil, err
		}
		r, err := Eval(*e.Right(), row)
		if err != nil {
			return nil, err
		}
		return EvalBin(e.Op(), l, r)
	default:
		return nil, fmt.Errorf("unsupported expr kind %s", e.Kind())
	}
}

// evalTernOperands evaluates a ternary expression's target, left and right
// operands, in that order, stopping at the first error.
func evalTernOperands(e Expr, row RowValueGetter) (target, left, right any, err error) {
	if target, err = Eval(*e.Target(), row); err != nil {
		return nil, nil, nil, err
	}
	if left, err = Eval(*e.Left(), row); err != nil {
		return nil, nil, nil, err
	}
	if right, err = Eval(*e.Right(), row); err != nil {
		return nil, nil, nil, err
	}
	return target, left, right, nil
}

// floatUnaryOps are the unary ops that convert an int64 or float64 operand to
// float64 and apply one function to it. Any other operand, null included, is
// an "<op> expects numeric" error.
var floatUnaryOps = map[string]func(float64) float64{
	"arccos":  math.Acos,
	"arccosh": math.Acosh,
	"arcsin":  math.Asin,
	"arcsinh": math.Asinh,
	"arctan":  math.Atan,
	"arctanh": math.Atanh,
	"cbrt":    math.Cbrt,
	"ceil":    math.Ceil,
	"cos":     math.Cos,
	"sin":     math.Sin,
	"cosh":    math.Cosh,
	"sinh":    math.Sinh,
	"cot":     func(f float64) float64 { return 1 / math.Tan(f) },
	"tan":     math.Tan,
	"tanh":    math.Tanh,
	"degrees": func(f float64) float64 { return f * 180 / math.Pi },
	"radians": func(f float64) float64 { return f * math.Pi / 180 },
	"floor":   math.Floor,
	"log":     math.Log,
	"log10":   math.Log10,
	"log1p":   math.Log1p,
	"sqrt":    math.Sqrt,
	"exp":     math.Exp,
	"sign": func(f float64) float64 {
		switch {
		case f < 0:
			return -1
		case f > 0:
			return 1
		default:
			return 0
		}
	},
}

// dtUnaryOps are the unary ops that extract one field of a datetime operand as
// int64. Any other operand, null included, is an "<op> expects datetime" error.
var dtUnaryOps = map[string]func(time.Time) int64{
	"dt_year":  func(t time.Time) int64 { return int64(t.Year()) },
	"dt_month": func(t time.Time) int64 { return int64(t.Month()) },
	"dt_day":   func(t time.Time) int64 { return int64(t.Day()) },
	"dt_hour":  func(t time.Time) int64 { return int64(t.Hour()) },
	// Polars/ISO weekday: Monday=1 .. Sunday=7. Go's time.Weekday is
	// Sunday=0 .. Saturday=6, so remap.
	"dt_weekday": func(t time.Time) int64 { return int64((int(t.Weekday())+6)%7) + 1 },
	"dt_minute":  func(t time.Time) int64 { return int64(t.Minute()) },
	"dt_second":  func(t time.Time) int64 { return int64(t.Second()) },
}

// bitwiseUnaryOps are the unary ops that count bits of an operand truncated to
// int64. Any other operand, null included, is an "<op> expects int" error.
var bitwiseUnaryOps = map[string]func(uint64) int{
	"bitwise_count_ones":     bits.OnesCount64,
	"bitwise_count_zeros":    func(u uint64) int { return 64 - bits.OnesCount64(u) },
	"bitwise_leading_ones":   func(u uint64) int { return bits.LeadingZeros64(^u) },
	"bitwise_leading_zeros":  bits.LeadingZeros64,
	"bitwise_trailing_ones":  func(u uint64) int { return bits.TrailingZeros64(^u) },
	"bitwise_trailing_zeros": bits.TrailingZeros64,
}

// evalParamUnary evaluates the unary ops encoded as "name:arg". handled is
// false when name is not one of them.
func evalParamUnary(name, arg string, v any) (out any, handled bool, err error) {
	switch name {
	case "over", "bottom_k",
		"exclude", "gather_every", "head", "limit", "repeat_by", "sample", "set_sorted",
		"shift", "shuffle", "slice", "sort", "tail", "top_k", "reshape",
		"rolling_min", "rolling_max", "rolling_mean", "rolling_sum", "rolling_std",
		"rolling_var", "rolling_median", "rolling_quantile", "rolling_skew",
		"rolling_kurtosis", "rolling_map", "rolling", "rolling_rank":
		return v, true, nil
	case "round_sig_figs":
		f, ok := ToFloat(v)
		if !ok {
			return nil, true, fmt.Errorf("round_sig_figs expects numeric")
		}
		digits := 3
		if parsed, err := strconv.Atoi(arg); err == nil && parsed > 0 {
			digits = parsed
		}
		return roundToSigFigs(f, digits), true, nil
	case "str_replace_all":
		pattern, repl, ok := strings.Cut(arg, ":")
		if !ok {
			return nil, true, fmt.Errorf("invalid str_replace_all configuration")
		}
		s, ok := v.(string)
		if !ok {
			return nil, true, fmt.Errorf("str_replace_all expects string")
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, true, fmt.Errorf("str_replace_all invalid pattern %q: %w", pattern, err)
		}
		// Replacement is taken literally (no $-group expansion), matching the
		// common Polars str.replace_all(pattern, value) usage.
		return re.ReplaceAllLiteralString(s, repl), true, nil
	case "str_replace":
		pattern, repl, ok := strings.Cut(arg, ":")
		if !ok {
			return nil, true, fmt.Errorf("invalid str_replace configuration")
		}
		s, ok := v.(string)
		if !ok {
			return nil, true, fmt.Errorf("str_replace expects string")
		}
		// Polars str.replace replaces only the FIRST match, with the pattern
		// treated as a regex (literal=False default).
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, true, fmt.Errorf("str_replace invalid pattern %q: %w", pattern, err)
		}
		loc := re.FindStringIndex(s)
		if loc == nil {
			return s, true, nil
		}
		return s[:loc[0]] + repl + s[loc[1]:], true, nil
	case "str_like":
		if v == nil {
			return nil, true, nil
		}
		s, ok := v.(string)
		if !ok {
			return nil, true, fmt.Errorf("str_like expects string")
		}
		re, err := compileLikePattern(arg)
		if err != nil {
			return nil, true, err
		}
		return re.MatchString(s), true, nil
	case "str_substr":
		if v == nil {
			return nil, true, nil
		}
		s, ok := v.(string)
		if !ok {
			return nil, true, fmt.Errorf("str_substr expects string")
		}
		startArg, lengthArg, ok := strings.Cut(arg, ":")
		if !ok {
			return nil, true, fmt.Errorf("invalid str_substr configuration")
		}
		start, err1 := strconv.Atoi(startArg)
		length, err2 := strconv.Atoi(lengthArg)
		if err1 != nil || err2 != nil {
			return nil, true, fmt.Errorf("invalid str_substr configuration")
		}
		return substrKernel(s, start, length), true, nil
	case "round_dp":
		if v == nil {
			return nil, true, nil
		}
		decimals, err := strconv.Atoi(arg)
		if err != nil {
			return nil, true, fmt.Errorf("invalid round_dp configuration")
		}
		switch t := v.(type) {
		case float64:
			scale := math.Pow(10, float64(decimals))
			return math.Round(t*scale) / scale, true, nil
		case int64:
			return t, true, nil
		default:
			return nil, true, fmt.Errorf("round_dp expects numeric")
		}
	case "struct_field":
		m, ok := v.(map[string]any)
		if !ok {
			return nil, true, fmt.Errorf("struct_field expects struct")
		}
		return m[arg], true, nil
	}
	return nil, false, nil
}

func roundToSigFigs(v float64, digits int) float64 {
	if v == 0 || digits <= 0 {
		return 0
	}
	scale := math.Pow(10, float64(digits)-math.Ceil(math.Log10(math.Abs(v))))
	return math.Round(v*scale) / scale
}

// kleeneBool interprets a value as a three-valued boolean: ok=false when it is
// neither a bool nor null; isNull=true for a null (nil) operand.
func kleeneBool(v any) (b bool, isNull bool, ok bool) {
	if v == nil {
		return false, true, true
	}
	if bb, isb := v.(bool); isb {
		return bb, false, true
	}
	return false, false, false
}

// CompareValues orders left and right and returns -1, 0 or +1. It compares
// the values that == cannot: lists ([]any) element by element, with a list
// that is a prefix of the other first; structs (map[string]any) field by field
// in ascending field-name order, as a boxed struct carries no field order; and
// binary values ([]byte) bytewise. Their elements order as follows: nil after
// every value; NaN after every other float64 and equal to another NaN; int64,
// float64, string, bool, time.Time and time.Duration by value; values of
// different types by type name, so values of different types never compare
// equal.
func CompareValues(left, right any) int {
	switch {
	case left == nil && right == nil:
		return 0
	case left == nil:
		return 1
	case right == nil:
		return -1
	}
	switch l := left.(type) {
	case []any:
		if r, ok := right.([]any); ok {
			for i := range min(len(l), len(r)) {
				if c := CompareValues(l[i], r[i]); c != 0 {
					return c
				}
			}
			return cmp.Compare(len(l), len(r))
		}
	case map[string]any:
		if r, ok := right.(map[string]any); ok {
			lk, rk := slices.Sorted(maps.Keys(l)), slices.Sorted(maps.Keys(r))
			for i := range min(len(lk), len(rk)) {
				if c := cmp.Compare(lk[i], rk[i]); c != 0 {
					return c
				}
				if c := CompareValues(l[lk[i]], r[rk[i]]); c != 0 {
					return c
				}
			}
			return cmp.Compare(len(lk), len(rk))
		}
	case []byte:
		if r, ok := right.([]byte); ok {
			return bytes.Compare(l, r)
		}
	case float64:
		if r, ok := right.(float64); ok {
			if lNaN, rNaN := math.IsNaN(l), math.IsNaN(r); lNaN != rNaN {
				if lNaN {
					return 1
				}
				return -1
			}
			return cmp.Compare(l, r)
		}
	case int64:
		if r, ok := right.(int64); ok {
			return cmp.Compare(l, r)
		}
	case string:
		if r, ok := right.(string); ok {
			return cmp.Compare(l, r)
		}
	case bool:
		if r, ok := right.(bool); ok {
			switch {
			case l == r:
				return 0
			case r:
				return -1
			}
			return 1
		}
	case time.Time:
		if r, ok := right.(time.Time); ok {
			return l.Compare(r)
		}
	case time.Duration:
		if r, ok := right.(time.Duration); ok {
			return cmp.Compare(l, r)
		}
	}
	if lt, rt := fmt.Sprintf("%T", left), fmt.Sprintf("%T", right); lt != rt {
		return cmp.Compare(lt, rt)
	}
	return cmp.Compare(fmt.Sprint(left), fmt.Sprint(right))
}

// equalValues reports whether left and right are equal, comparing lists,
// structs and binary values through CompareValues, since == panics on them.
func equalValues(left, right any) bool {
	switch left.(type) {
	case []any, map[string]any, []byte:
		return CompareValues(left, right) == 0
	}
	return left == right
}

// EvalBin applies the binary operator op to two already-evaluated operands,
// with the null, NaN and type rules Eval uses for a KindBin expression.
func EvalBin(op string, left any, right any) (any, error) {
	switch op {
	case "eq":
		// A comparison with null yields null (Polars semantics). Null-aware
		// equality lives in eq_missing/ne_missing.
		if left == nil || right == nil {
			return nil, nil
		}
		if lf, ok := left.(float64); ok && math.IsNaN(lf) {
			return false, nil
		}
		if rf, ok := right.(float64); ok && math.IsNaN(rf) {
			return false, nil
		}
		return equalValues(left, right), nil
	case "ne":
		if left == nil || right == nil {
			return nil, nil
		}
		if lf, ok := left.(float64); ok && math.IsNaN(lf) {
			return true, nil
		}
		if rf, ok := right.(float64); ok && math.IsNaN(rf) {
			return true, nil
		}
		return !equalValues(left, right), nil
	case "gt", "ge", "lt", "le":
		if left == nil || right == nil {
			return nil, nil
		}
		return compare(op, left, right)
	case "add", "sub", "mul", "div":
		return arith(op, left, right)
	case "pow":
		lf, lok := ToFloat(left)
		rf, rok := ToFloat(right)
		if !lok || !rok {
			return nil, fmt.Errorf("pow expects numeric")
		}
		return math.Pow(lf, rf), nil
	case "and", "and_":
		// Three-valued (Kleene) AND: a definite false wins; otherwise any null
		// makes the result null; otherwise true.
		lb, ln, lok := kleeneBool(left)
		rb, rn, rok := kleeneBool(right)
		if !lok || !rok {
			return nil, fmt.Errorf("and expects bool")
		}
		if (!ln && !lb) || (!rn && !rb) {
			return false, nil
		}
		if ln || rn {
			return nil, nil
		}
		return true, nil
	case "or", "or_":
		// Three-valued (Kleene) OR: a definite true wins; otherwise any null
		// makes the result null; otherwise false.
		lb, ln, lok := kleeneBool(left)
		rb, rn, rok := kleeneBool(right)
		if !lok || !rok {
			return nil, fmt.Errorf("or expects bool")
		}
		if (!ln && lb) || (!rn && rb) {
			return true, nil
		}
		if ln || rn {
			return nil, nil
		}
		return false, nil
	case "contains":
		l, lok := left.(string)
		r, rok := right.(string)
		if !lok || !rok {
			return nil, fmt.Errorf("contains expects string")
		}
		return strings.Contains(l, r), nil
	case "starts_with":
		l, lok := left.(string)
		r, rok := right.(string)
		if !lok || !rok {
			return nil, fmt.Errorf("starts_with expects strings")
		}
		return strings.HasPrefix(l, r), nil
	case "str_concat":
		if left == nil || right == nil {
			return nil, nil
		}
		l, lok := left.(string)
		r, rok := right.(string)
		if !lok || !rok {
			return nil, fmt.Errorf("str_concat expects strings")
		}
		return l + r, nil
	case "list_contains":
		l, ok := left.([]any)
		if !ok {
			return nil, fmt.Errorf("list_contains expects list")
		}
		return slices.Contains(l, right), nil
	case "list_get":
		list, ok := left.([]any)
		if !ok {
			return nil, fmt.Errorf("list_get expects list")
		}
		idx, ok := toInt64(right)
		if !ok {
			return nil, fmt.Errorf("list_get expects numeric index")
		}
		if idx < 0 || int(idx) >= len(list) {
			return nil, nil
		}
		return list[idx], nil
	case "append":
		if list, ok := left.([]any); ok {
			return append(append([]any{}, list...), right), nil
		}
		if list, ok := right.([]any); ok {
			return append([]any{left}, list...), nil
		}
		return []any{left, right}, nil
	case "bitwise_and":
		li, lok := toInt64(left)
		ri, rok := toInt64(right)
		if !lok || !rok {
			return nil, fmt.Errorf("bitwise_and expects ints")
		}
		return li & ri, nil
	case "bitwise_or":
		li, lok := toInt64(left)
		ri, rok := toInt64(right)
		if !lok || !rok {
			return nil, fmt.Errorf("bitwise_or expects ints")
		}
		return li | ri, nil
	case "bitwise_xor":
		li, lok := toInt64(left)
		ri, rok := toInt64(right)
		if !lok || !rok {
			return nil, fmt.Errorf("bitwise_xor expects ints")
		}
		return li ^ ri, nil
	case "dot":
		if lf, lok := ToFloat(left); lok {
			if rf, rok := ToFloat(right); rok {
				return lf * rf, nil
			}
		}
		return nil, fmt.Errorf("dot expects numeric")
	case "eq_missing":
		return equalValues(left, right), nil
	case "ne_missing":
		return !equalValues(left, right), nil
	case "floordiv":
		lf, lok := ToFloat(left)
		rf, rok := ToFloat(right)
		if !lok || !rok || rf == 0 {
			return nil, fmt.Errorf("floordiv expects numeric non-zero divisor")
		}
		return math.Floor(lf / rf), nil
	case "mod":
		lf, lok := ToFloat(left)
		rf, rok := ToFloat(right)
		if !lok || !rok || rf == 0 {
			return nil, fmt.Errorf("mod expects numeric non-zero divisor")
		}
		return math.Mod(lf, rf), nil
	case "gather", "get":
		if list, ok := left.([]any); ok {
			idx, ok := toInt64(right)
			if !ok {
				return nil, fmt.Errorf("%s expects numeric index", op)
			}
			if idx < 0 || int(idx) >= len(list) {
				return nil, nil
			}
			return list[idx], nil
		}
		return left, nil
	case "filter_expr", "where":
		if keep, ok := right.(bool); ok {
			if keep {
				return left, nil
			}
			return nil, nil
		}
		return left, nil
	case "is_close":
		lf, lok := ToFloat(left)
		rf, rok := ToFloat(right)
		if !lok || !rok {
			return nil, fmt.Errorf("is_close expects numeric")
		}
		return math.Abs(lf-rf) <= 1e-9, nil
	case "is_in":
		if list, ok := right.([]any); ok {
			return slices.Contains(list, left), nil
		}
		return left == right, nil
	case "index_of":
		if list, ok := left.([]any); ok {
			return int64(slices.Index(list, right)), nil
		}
		return int64(-1), nil
	case "fill_null_expr":
		if left == nil {
			return right, nil
		}
		return left, nil
	case "fill_nan_expr":
		if lf, ok := left.(float64); ok && math.IsNaN(lf) {
			return right, nil
		}
		return left, nil
	case "xor":
		if left == nil || right == nil {
			return nil, nil
		}
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, fmt.Errorf("xor expects bool")
		}
		return l != r, nil
	case "true_div":
		return arith("div", left, right)
	case "search_sorted":
		return left, nil
	default:
		if out, handled, err := evalExtraBin(op, left, right); handled {
			return out, err
		}
		return nil, fmt.Errorf("unsupported binary op %s", op)
	}
}

func compare(op string, left any, right any) (bool, error) {
	if left == nil || right == nil {
		return false, nil
	}
	switch l := left.(type) {
	case int64:
		r, ok := right.(int64)
		if !ok {
			return false, fmt.Errorf("compare type mismatch")
		}
		return cmpOrdered(op, l, r), nil
	case float64:
		r, ok := right.(float64)
		if !ok {
			return false, fmt.Errorf("compare type mismatch")
		}
		if math.IsNaN(l) || math.IsNaN(r) {
			return false, nil
		}
		return cmpOrdered(op, l, r), nil
	case string:
		r, ok := right.(string)
		if !ok {
			return false, fmt.Errorf("compare type mismatch")
		}
		return cmpOrdered(op, l, r), nil
	case time.Time:
		r, ok := right.(time.Time)
		if !ok {
			return false, fmt.Errorf("compare type mismatch")
		}
		switch op {
		case "gt":
			return l.After(r), nil
		case "ge":
			return l.After(r) || l.Equal(r), nil
		case "lt":
			return l.Before(r), nil
		case "le":
			return l.Before(r) || l.Equal(r), nil
		}
	}
	return false, fmt.Errorf("unsupported compare types")
}

func arith(op string, left any, right any) (any, error) {
	if left == nil || right == nil {
		return nil, nil
	}
	switch l := left.(type) {
	case int64:
		r, ok := right.(int64)
		if !ok {
			return nil, fmt.Errorf("arith type mismatch")
		}
		switch op {
		case "add":
			return l + r, nil
		case "sub":
			return l - r, nil
		case "mul":
			return l * r, nil
		case "div":
			if r == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return l / r, nil
		}
	case float64:
		r, ok := right.(float64)
		if !ok {
			return nil, fmt.Errorf("arith type mismatch")
		}
		switch op {
		case "add":
			return l + r, nil
		case "sub":
			return l - r, nil
		case "mul":
			return l * r, nil
		case "div":
			if r == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return l / r, nil
		}
	case string:
		// String "+" concatenates (matching Polars); other ops are unsupported.
		r, ok := right.(string)
		if !ok {
			return nil, fmt.Errorf("arith type mismatch")
		}
		if op == "add" {
			return l + r, nil
		}
		return nil, fmt.Errorf("unsupported string arithmetic %q", op)
	}
	return nil, fmt.Errorf("unsupported arithmetic types")
}

func cast(v any, dt dtypes.DataType) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch dt {
	case dtypes.Int64:
		switch t := v.(type) {
		case int64:
			return t, nil
		case float64:
			return int64(t), nil
		case bool:
			if t {
				return int64(1), nil
			}
			return int64(0), nil
		case string:
			if i, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil {
				return i, nil
			}
			if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
				return int64(f), nil
			}
		}
	case dtypes.Float64:
		switch t := v.(type) {
		case float64:
			return t, nil
		case int64:
			return float64(t), nil
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
				return f, nil
			}
		}
	case dtypes.String:
		return fmt.Sprintf("%v", v), nil
	case dtypes.Boolean:
		switch t := v.(type) {
		case bool:
			return t, nil
		case string:
			if b, err := strconv.ParseBool(strings.TrimSpace(t)); err == nil {
				return b, nil
			}
		}
	case dtypes.Datetime:
		t, ok := v.(time.Time)
		if ok {
			return t, nil
		}
	}
	return nil, fmt.Errorf("cannot cast value")
}

// ToFloat converts an int64 or float64 value to float64. ok is false for any
// other value, including nil.
func ToFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case int64:
		return float64(t), true
	case float64:
		return t, true
	default:
		return 0, false
	}
}

func toInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case float64:
		return int64(t), true
	default:
		return 0, false
	}
}

// cmpOrdered applies one of the four ordering operators. An op outside the set
// (equality is handled by the caller) is false, not an error.
func cmpOrdered[T cmp.Ordered](op string, l T, r T) bool {
	switch op {
	case "gt":
		return l > r
	case "ge":
		return l >= r
	case "lt":
		return l < r
	case "le":
		return l <= r
	}
	return false
}

// compileLikePattern converts a LIKE pattern into an anchored regular
// expression. '%' matches any run of characters, '_' matches exactly one, and
// every other character (including regex metacharacters) is matched literally.
func compileLikePattern(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for _, ch := range pattern {
		switch ch {
		case '%':
			b.WriteString(".*")
		case '_':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// substrKernel implements SUBSTRING semantics: start is 1-based and length is
// the number of characters to take. Out-of-range requests clamp to the string.
func substrKernel(s string, start int, length int) string {
	runes := []rune(s)
	if start < 1 {
		start = 1
	}
	from := start - 1
	if from >= len(runes) || length <= 0 {
		return ""
	}
	to := min(from+length, len(runes))
	return string(runes[from:to])
}
