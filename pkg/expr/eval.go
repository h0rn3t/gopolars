package expr

import (
	"encoding/json"
	"fmt"
	"math"
	"math/bits"
	"regexp"
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
			// Polars/ISO weekday: Monday=1 .. Sunday=7. Go's time.Weekday is
			// Sunday=0 .. Saturday=6, so remap.
			return int64((int(t.Weekday())+6)%7) + 1, nil
		case "dt_minute":
			t, ok := v.(time.Time)
			if !ok {
				return nil, fmt.Errorf("dt_minute expects datetime")
			}
			return int64(t.Minute()), nil
		case "dt_second":
			t, ok := v.(time.Time)
			if !ok {
				return nil, fmt.Errorf("dt_second expects datetime")
			}
			return int64(t.Second()), nil
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
			if strings.HasPrefix(e.Op(), "str_replace_all:") {
				spec := strings.TrimPrefix(e.Op(), "str_replace_all:")
				parts := strings.SplitN(spec, ":", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid str_replace_all configuration")
				}
				s, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("str_replace_all expects string")
				}
				re, err := regexp.Compile(parts[0])
				if err != nil {
					return nil, fmt.Errorf("str_replace_all invalid pattern %q: %w", parts[0], err)
				}
				// Replacement is taken literally (no $-group expansion), matching the
				// common Polars str.replace_all(pattern, value) usage.
				return re.ReplaceAllLiteralString(s, parts[1]), nil
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
				// Polars str.replace replaces only the FIRST match, with the pattern
				// treated as a regex (literal=False default).
				re, err := regexp.Compile(parts[0])
				if err != nil {
					return nil, fmt.Errorf("str_replace invalid pattern %q: %w", parts[0], err)
				}
				loc := re.FindStringIndex(s)
				if loc == nil {
					return s, nil
				}
				return s[:loc[0]] + parts[1] + s[loc[1]:], nil
			}
			if strings.HasPrefix(e.Op(), "str_like:") {
				if v == nil {
					return nil, nil
				}
				s, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("str_like expects string")
				}
				re, err := compileLikePattern(strings.TrimPrefix(e.Op(), "str_like:"))
				if err != nil {
					return nil, err
				}
				return re.MatchString(s), nil
			}
			if strings.HasPrefix(e.Op(), "str_substr:") {
				if v == nil {
					return nil, nil
				}
				s, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("str_substr expects string")
				}
				spec := strings.TrimPrefix(e.Op(), "str_substr:")
				parts := strings.SplitN(spec, ":", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid str_substr configuration")
				}
				start, err1 := strconv.Atoi(parts[0])
				length, err2 := strconv.Atoi(parts[1])
				if err1 != nil || err2 != nil {
					return nil, fmt.Errorf("invalid str_substr configuration")
				}
				return sqlSubstr(s, start, length), nil
			}
			if strings.HasPrefix(e.Op(), "round_dp:") {
				if v == nil {
					return nil, nil
				}
				decimals, err := strconv.Atoi(strings.TrimPrefix(e.Op(), "round_dp:"))
				if err != nil {
					return nil, fmt.Errorf("invalid round_dp configuration")
				}
				switch t := v.(type) {
				case float64:
					scale := math.Pow(10, float64(decimals))
					return math.Round(t*scale) / scale, nil
				case int64:
					return t, nil
				default:
					return nil, fmt.Errorf("round_dp expects numeric")
				}
			}
			if strings.HasPrefix(e.Op(), "struct_field:") {
				key := strings.TrimPrefix(e.Op(), "struct_field:")
				m, ok := v.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("struct_field expects struct")
				}
				return m[key], nil
			}
			if out, handled, err := evalSQLUnary(e.Op(), v); handled {
				return out, err
			}
			return nil, fmt.Errorf("unsupported unary op %s", e.Op())
		}
	case KindTern:
		if e.Op() == "str_pad_start" || e.Op() == "str_pad_end" || e.Op() == "str_split_part" {
			target, err := Eval(*e.Target(), row)
			if err != nil {
				return nil, err
			}
			left, err := Eval(*e.Left(), row)
			if err != nil {
				return nil, err
			}
			right, err := Eval(*e.Right(), row)
			if err != nil {
				return nil, err
			}
			out, _, err := evalSQLTern(e.Op(), target, left, right)
			return out, err
		}
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

func evalBin(op string, left any, right any) (any, error) {
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
		return left == right, nil
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
		return left != right, nil
	case "gt", "ge", "lt", "le":
		if left == nil || right == nil {
			return nil, nil
		}
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
		if out, handled, err := evalSQLBin(op, left, right); handled {
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

// compileLikePattern converts a SQL LIKE pattern into an anchored regular
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

// sqlSubstr implements SQL SUBSTRING semantics: start is 1-based and length is
// the number of characters to take. Out-of-range requests clamp to the string.
func sqlSubstr(s string, start int, length int) string {
	runes := []rune(s)
	if start < 1 {
		start = 1
	}
	from := start - 1
	if from >= len(runes) || length <= 0 {
		return ""
	}
	to := from + length
	if to > len(runes) {
		to = len(runes)
	}
	return string(runes[from:to])
}
