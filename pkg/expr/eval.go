package expr

import (
	"encoding/json"
	"fmt"
	"math"
	"math/bits"
	"strconv"
	"strings"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
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
	case KindAlias:
		if e.target != nil {
			return Eval(*e.target, row)
		}
		return nil, fmt.Errorf("alias target is nil")
	case KindCast:
		v, err := Eval(*e.Target(), row)
		if err != nil {
			return nil, err
		}
		return cast(v, e.CastType())
	case KindUnary:
		v, err := Eval(*e.Target(), row)
		if err != nil {
			return nil, err
		}
		switch e.Op() {
		case "not":
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
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("str_len expects string")
			}
			return int64(len(s)), nil
		case "str_lower":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("str_lower expects string")
			}
			return strings.ToLower(s), nil
		case "str_upper":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("str_upper expects string")
			}
			return strings.ToUpper(s), nil
		case "str_trim":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("str_trim expects string")
			}
			return strings.TrimSpace(s), nil
		case "dt_year":
			t, ok := v.(time.Time)
			if !ok {
				return nil, fmt.Errorf("dt_year expects datetime")
			}
			return int64(t.Year()), nil
		case "dt_month":
			t, ok := v.(time.Time)
			if !ok {
				return nil, fmt.Errorf("dt_month expects datetime")
			}
			return int64(t.Month()), nil
		case "dt_day":
			t, ok := v.(time.Time)
			if !ok {
				return nil, fmt.Errorf("dt_day expects datetime")
			}
			return int64(t.Day()), nil
		case "dt_hour":
			t, ok := v.(time.Time)
			if !ok {
				return nil, fmt.Errorf("dt_hour expects datetime")
			}
			return int64(t.Hour()), nil
		case "dt_weekday":
			t, ok := v.(time.Time)
			if !ok {
				return nil, fmt.Errorf("dt_weekday expects datetime")
			}
			return int64(t.Weekday()), nil
		case "list_len":
			list, ok := v.([]any)
			if !ok {
				return nil, fmt.Errorf("list_len expects list")
			}
			return int64(len(list)), nil
		case "cum_sum", "cum_count", "rank":
			return v, nil
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
		case "log":
			switch t := v.(type) {
			case float64:
				return math.Log(t), nil
			case int64:
				return math.Log(float64(t)), nil
			default:
				return nil, fmt.Errorf("log expects numeric")
			}
		case "sqrt":
			switch t := v.(type) {
			case float64:
				return math.Sqrt(t), nil
			case int64:
				return math.Sqrt(float64(t)), nil
			default:
				return nil, fmt.Errorf("sqrt expects numeric")
			}
		case "dt", "str", "list", "struct":
			return v, nil
		case "agg_groups", "approx_n_unique", "arg_max", "arg_min", "arg_sort", "arg_true", "arg_unique", "arr", "backward_fill", "bin", "cat", "count", "cum_max", "cum_min", "cum_prod", "cumulative_eval", "cut", "deserialize", "diff", "drop_nans", "drop_nulls", "entropy", "ewm_mean", "ewm_std", "ewm_var", "explode", "ext", "first", "flatten", "forward_fill", "hash", "hist", "implode", "inspect", "interpolate", "is_duplicated", "is_first_distinct", "is_last_distinct", "is_unique", "item", "kurtosis", "last", "lower_bound", "map_batches", "map_elements", "max", "mean", "median", "meta", "min", "mode", "n_unique", "nan_max", "nan_min", "null_count", "pct_change", "peak_max", "peak_min", "pipe", "product", "qcut", "quantile", "rechunk", "reinterpret", "rle", "rle_id", "shrink_dtype", "skew", "std", "sum", "to_physical", "truncate", "unique", "unique_counts", "upper_bound", "value_counts", "var":
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
		case "arccos":
			if f, ok := toFloat(v); ok {
				return math.Acos(f), nil
			}
			return nil, fmt.Errorf("arccos expects numeric")
		case "arccosh":
			if f, ok := toFloat(v); ok {
				return math.Acosh(f), nil
			}
			return nil, fmt.Errorf("arccosh expects numeric")
		case "arcsin":
			if f, ok := toFloat(v); ok {
				return math.Asin(f), nil
			}
			return nil, fmt.Errorf("arcsin expects numeric")
		case "arcsinh":
			if f, ok := toFloat(v); ok {
				return math.Asinh(f), nil
			}
			return nil, fmt.Errorf("arcsinh expects numeric")
		case "arctan":
			if f, ok := toFloat(v); ok {
				return math.Atan(f), nil
			}
			return nil, fmt.Errorf("arctan expects numeric")
		case "arctanh":
			if f, ok := toFloat(v); ok {
				return math.Atanh(f), nil
			}
			return nil, fmt.Errorf("arctanh expects numeric")
		case "bitwise_count_ones":
			i, ok := toInt64(v)
			if !ok {
				return nil, fmt.Errorf("bitwise_count_ones expects int")
			}
			return int64(bits.OnesCount64(uint64(i))), nil
		case "bitwise_count_zeros":
			i, ok := toInt64(v)
			if !ok {
				return nil, fmt.Errorf("bitwise_count_zeros expects int")
			}
			return int64(bits.LeadingZeros64(uint64(i)) + bits.TrailingZeros64(uint64(i))), nil
		case "bitwise_leading_ones":
			i, ok := toInt64(v)
			if !ok {
				return nil, fmt.Errorf("bitwise_leading_ones expects int")
			}
			return int64(bits.LeadingZeros64(^uint64(i))), nil
		case "bitwise_leading_zeros":
			i, ok := toInt64(v)
			if !ok {
				return nil, fmt.Errorf("bitwise_leading_zeros expects int")
			}
			return int64(bits.LeadingZeros64(uint64(i))), nil
		case "bitwise_trailing_ones":
			i, ok := toInt64(v)
			if !ok {
				return nil, fmt.Errorf("bitwise_trailing_ones expects int")
			}
			return int64(bits.TrailingZeros64(^uint64(i))), nil
		case "bitwise_trailing_zeros":
			i, ok := toInt64(v)
			if !ok {
				return nil, fmt.Errorf("bitwise_trailing_zeros expects int")
			}
			return int64(bits.TrailingZeros64(uint64(i))), nil
		case "cbrt":
			if f, ok := toFloat(v); ok {
				return math.Cbrt(f), nil
			}
			return nil, fmt.Errorf("cbrt expects numeric")
		case "ceil":
			if f, ok := toFloat(v); ok {
				return math.Ceil(f), nil
			}
			return nil, fmt.Errorf("ceil expects numeric")
		case "cos":
			if f, ok := toFloat(v); ok {
				return math.Cos(f), nil
			}
			return nil, fmt.Errorf("cos expects numeric")
		case "sin":
			if f, ok := toFloat(v); ok {
				return math.Sin(f), nil
			}
			return nil, fmt.Errorf("sin expects numeric")
		case "cosh":
			if f, ok := toFloat(v); ok {
				return math.Cosh(f), nil
			}
			return nil, fmt.Errorf("cosh expects numeric")
		case "sinh":
			if f, ok := toFloat(v); ok {
				return math.Sinh(f), nil
			}
			return nil, fmt.Errorf("sinh expects numeric")
		case "cot":
			if f, ok := toFloat(v); ok {
				return 1 / math.Tan(f), nil
			}
			return nil, fmt.Errorf("cot expects numeric")
		case "tan":
			if f, ok := toFloat(v); ok {
				return math.Tan(f), nil
			}
			return nil, fmt.Errorf("tan expects numeric")
		case "tanh":
			if f, ok := toFloat(v); ok {
				return math.Tanh(f), nil
			}
			return nil, fmt.Errorf("tanh expects numeric")
		case "degrees":
			if f, ok := toFloat(v); ok {
				return f * 180 / math.Pi, nil
			}
			return nil, fmt.Errorf("degrees expects numeric")
		case "radians":
			if f, ok := toFloat(v); ok {
				return f * math.Pi / 180, nil
			}
			return nil, fmt.Errorf("radians expects numeric")
		case "floor":
			if f, ok := toFloat(v); ok {
				return math.Floor(f), nil
			}
			return nil, fmt.Errorf("floor expects numeric")
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
			if f, ok := toFloat(v); ok {
				return !math.IsNaN(f) && !math.IsInf(f, 0), nil
			}
			return false, nil
		case "is_infinite":
			if f, ok := toFloat(v); ok {
				return math.IsInf(f, 0), nil
			}
			return false, nil
		case "is_nan":
			if f, ok := toFloat(v); ok {
				return math.IsNaN(f), nil
			}
			return false, nil
		case "is_not_nan":
			if f, ok := toFloat(v); ok {
				return !math.IsNaN(f), nil
			}
			return v != nil, nil
		case "log10":
			if f, ok := toFloat(v); ok {
				return math.Log10(f), nil
			}
			return nil, fmt.Errorf("log10 expects numeric")
		case "log1p":
			if f, ok := toFloat(v); ok {
				return math.Log1p(f), nil
			}
			return nil, fmt.Errorf("log1p expects numeric")
		case "sign":
			if f, ok := toFloat(v); ok {
				switch {
				case f < 0:
					return float64(-1), nil
				case f > 0:
					return float64(1), nil
				default:
					return float64(0), nil
				}
			}
			return nil, fmt.Errorf("sign expects numeric")
		case "neg":
			if f, ok := toFloat(v); ok {
				return -f, nil
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
		case "exp":
			switch t := v.(type) {
			case float64:
				return math.Exp(t), nil
			case int64:
				return math.Exp(float64(t)), nil
			default:
				return nil, fmt.Errorf("exp expects numeric")
			}
		default:
			if strings.HasPrefix(e.Op(), "over:") {
				return v, nil
			}
			if strings.HasPrefix(e.Op(), "bottom_k:") {
				return v, nil
			}
			if strings.HasPrefix(e.Op(), "exclude:") || strings.HasPrefix(e.Op(), "gather_every:") || strings.HasPrefix(e.Op(), "head:") || strings.HasPrefix(e.Op(), "limit:") || strings.HasPrefix(e.Op(), "repeat_by:") || strings.HasPrefix(e.Op(), "sample:") || strings.HasPrefix(e.Op(), "set_sorted:") || strings.HasPrefix(e.Op(), "shift:") || strings.HasPrefix(e.Op(), "shuffle:") || strings.HasPrefix(e.Op(), "slice:") || strings.HasPrefix(e.Op(), "sort:") || strings.HasPrefix(e.Op(), "tail:") || strings.HasPrefix(e.Op(), "top_k:") || strings.HasPrefix(e.Op(), "reshape:") {
				return v, nil
			}
			if strings.HasPrefix(e.Op(), "rolling_min:") || strings.HasPrefix(e.Op(), "rolling_max:") || strings.HasPrefix(e.Op(), "rolling_mean:") || strings.HasPrefix(e.Op(), "rolling_sum:") || strings.HasPrefix(e.Op(), "rolling_std:") || strings.HasPrefix(e.Op(), "rolling_var:") || strings.HasPrefix(e.Op(), "rolling_median:") || strings.HasPrefix(e.Op(), "rolling_quantile:") || strings.HasPrefix(e.Op(), "rolling_skew:") || strings.HasPrefix(e.Op(), "rolling_kurtosis:") || strings.HasPrefix(e.Op(), "rolling_map:") || strings.HasPrefix(e.Op(), "rolling:") || strings.HasPrefix(e.Op(), "rolling_rank:") {
				return v, nil
			}
			if strings.HasPrefix(e.Op(), "round_sig_figs:") {
				f, ok := toFloat(v)
				if !ok {
					return nil, fmt.Errorf("round_sig_figs expects numeric")
				}
				digits := 3
				if parsed, err := strconv.Atoi(strings.TrimPrefix(e.Op(), "round_sig_figs:")); err == nil && parsed > 0 {
					digits = parsed
				}
				return roundToSigFigs(f, digits), nil
			}
			if strings.HasPrefix(e.Op(), "str_replace:") {
				spec := strings.TrimPrefix(e.Op(), "str_replace:")
				parts := strings.SplitN(spec, ":", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid str_replace configuration")
				}
				s, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("str_replace expects string")
				}
				return strings.ReplaceAll(s, parts[0], parts[1]), nil
			}
			if strings.HasPrefix(e.Op(), "struct_field:") {
				key := strings.TrimPrefix(e.Op(), "struct_field:")
				m, ok := v.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("struct_field expects struct")
				}
				return m[key], nil
			}
			return nil, fmt.Errorf("unsupported unary op %s", e.Op())
		}
	case KindTern:
		if e.Op() == "clip" {
			current, err := Eval(*e.Target(), row)
			if err != nil {
				return nil, err
			}
			minValue, err := Eval(*e.Left(), row)
			if err != nil {
				return nil, err
			}
			maxValue, err := Eval(*e.Right(), row)
			if err != nil {
				return nil, err
			}
			cur, cok := toFloat(current)
			minv, mok := toFloat(minValue)
			maxv, xok := toFloat(maxValue)
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
		}
		if e.Op() == "replace" || e.Op() == "replace_strict" {
			current, err := Eval(*e.Target(), row)
			if err != nil {
				return nil, err
			}
			oldValue, err := Eval(*e.Left(), row)
			if err != nil {
				return nil, err
			}
			newValue, err := Eval(*e.Right(), row)
			if err != nil {
				return nil, err
			}
			if current == oldValue {
				return newValue, nil
			}
			return current, nil
		}
		if strings.HasPrefix(e.Op(), "bottom_k_by:") || strings.HasPrefix(e.Op(), "top_k_by:") || strings.HasPrefix(e.Op(), "sort_by:") {
			return Eval(*e.Target(), row)
		}
		if e.Op() == "is_between" {
			current, err := Eval(*e.Target(), row)
			if err != nil {
				return nil, err
			}
			lower, err := Eval(*e.Left(), row)
			if err != nil {
				return nil, err
			}
			upper, err := Eval(*e.Right(), row)
			if err != nil {
				return nil, err
			}
			cv, cok := toFloat(current)
			lv, lok := toFloat(lower)
			uv, uok := toFloat(upper)
			if !cok || !lok || !uok {
				return nil, fmt.Errorf("is_between expects numeric")
			}
			return cv >= lv && cv <= uv, nil
		}
		if e.Op() == "ewm_mean_by" || e.Op() == "extend_constant" || e.Op() == "max_by" || e.Op() == "min_by" {
			return Eval(*e.Target(), row)
		}
		if e.Op() == "interpolate_by" {
			return Eval(*e.Target(), row)
		}
		if strings.HasPrefix(e.Op(), "rolling_max_by:") || strings.HasPrefix(e.Op(), "rolling_mean_by:") || strings.HasPrefix(e.Op(), "rolling_min_by:") || strings.HasPrefix(e.Op(), "rolling_sum_by:") || strings.HasPrefix(e.Op(), "rolling_std_by:") || strings.HasPrefix(e.Op(), "rolling_var_by:") || strings.HasPrefix(e.Op(), "rolling_median_by:") || strings.HasPrefix(e.Op(), "rolling_quantile_by:") || strings.HasPrefix(e.Op(), "rolling_rank_by:") {
			return Eval(*e.Target(), row)
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
		return evalBin(e.Op(), l, r)
	default:
		return nil, fmt.Errorf("unsupported expr kind %s", e.Kind())
	}
}

func roundToSigFigs(v float64, digits int) float64 {
	if v == 0 || digits <= 0 {
		return 0
	}
	scale := math.Pow(10, float64(digits)-math.Ceil(math.Log10(math.Abs(v))))
	return math.Round(v*scale) / scale
}

func evalBin(op string, left any, right any) (any, error) {
	switch op {
	case "eq":
		if left == nil || right == nil {
			return left == nil && right == nil, nil
		}
		if lf, ok := left.(float64); ok && math.IsNaN(lf) {
			return false, nil
		}
		if rf, ok := right.(float64); ok && math.IsNaN(rf) {
			return false, nil
		}
		return left == right, nil
	case "ne":
		if left == nil || right == nil {
			return !(left == nil && right == nil), nil
		}
		if lf, ok := left.(float64); ok && math.IsNaN(lf) {
			return true, nil
		}
		if rf, ok := right.(float64); ok && math.IsNaN(rf) {
			return true, nil
		}
		return left != right, nil
	case "gt", "ge", "lt", "le":
		return compare(op, left, right)
	case "add", "sub", "mul", "div":
		return arith(op, left, right)
	case "pow":
		lf, lok := toFloat(left)
		rf, rok := toFloat(right)
		if !lok || !rok {
			return nil, fmt.Errorf("pow expects numeric")
		}
		return math.Pow(lf, rf), nil
	case "and":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, fmt.Errorf("and expects bool")
		}
		return l && r, nil
	case "and_":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, fmt.Errorf("and_ expects bool")
		}
		return l && r, nil
	case "or":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, fmt.Errorf("or expects bool")
		}
		return l || r, nil
	case "or_":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, fmt.Errorf("or_ expects bool")
		}
		return l || r, nil
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
	case "list_contains":
		l, ok := left.([]any)
		if !ok {
			return nil, fmt.Errorf("list_contains expects list")
		}
		for _, v := range l {
			if v == right {
				return true, nil
			}
		}
		return false, nil
	case "list_get":
		list, ok := left.([]any)
		if !ok {
			return nil, fmt.Errorf("list_get expects list")
		}
		var idx int64
		switch i := right.(type) {
		case int64:
			idx = i
		case float64:
			idx = int64(i)
		default:
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
		if lf, lok := toFloat(left); lok {
			if rf, rok := toFloat(right); rok {
				return lf * rf, nil
			}
		}
		return nil, fmt.Errorf("dot expects numeric")
	case "eq_missing":
		if left == nil || right == nil {
			return left == right, nil
		}
		return left == right, nil
	case "ne_missing":
		if left == nil || right == nil {
			return left != right, nil
		}
		return left != right, nil
	case "floordiv":
		lf, lok := toFloat(left)
		rf, rok := toFloat(right)
		if !lok || !rok || rf == 0 {
			return nil, fmt.Errorf("floordiv expects numeric non-zero divisor")
		}
		return math.Floor(lf / rf), nil
	case "mod":
		lf, lok := toFloat(left)
		rf, rok := toFloat(right)
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
	case "filter_expr":
		if keep, ok := right.(bool); ok {
			if keep {
				return left, nil
			}
			return nil, nil
		}
		return left, nil
	case "is_close":
		lf, lok := toFloat(left)
		rf, rok := toFloat(right)
		if !lok || !rok {
			return nil, fmt.Errorf("is_close expects numeric")
		}
		return math.Abs(lf-rf) <= 1e-9, nil
	case "is_in":
		if list, ok := right.([]any); ok {
			for _, item := range list {
				if item == left {
					return true, nil
				}
			}
			return false, nil
		}
		return left == right, nil
	case "index_of":
		if list, ok := left.([]any); ok {
			for i, item := range list {
				if item == right {
					return int64(i), nil
				}
			}
			return int64(-1), nil
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
	case "where":
		if keep, ok := right.(bool); ok {
			if keep {
				return left, nil
			}
			return nil, nil
		}
		return left, nil
	case "xor":
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
		return cmpInts(op, l, r), nil
	case float64:
		r, ok := right.(float64)
		if !ok {
			return false, fmt.Errorf("compare type mismatch")
		}
		if math.IsNaN(l) || math.IsNaN(r) {
			return false, nil
		}
		return cmpFloats(op, l, r), nil
	case string:
		r, ok := right.(string)
		if !ok {
			return false, fmt.Errorf("compare type mismatch")
		}
		return cmpStrings(op, l, r), nil
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
		}
	case dtypes.Float64:
		switch t := v.(type) {
		case float64:
			return t, nil
		case int64:
			return float64(t), nil
		}
	case dtypes.String:
		return fmt.Sprintf("%v", v), nil
	case dtypes.Boolean:
		b, ok := v.(bool)
		if ok {
			return b, nil
		}
	case dtypes.Datetime:
		t, ok := v.(time.Time)
		if ok {
			return t, nil
		}
	}
	return nil, fmt.Errorf("cannot cast value")
}

func toFloat(v any) (float64, bool) {
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

func cmpInts(op string, l int64, r int64) bool {
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

func cmpFloats(op string, l float64, r float64) bool {
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

func cmpStrings(op string, l string, r string) bool {
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
