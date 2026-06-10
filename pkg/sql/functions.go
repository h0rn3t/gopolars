package sql

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

// fnSpec describes one catalog entry: the canonical (lowercase) name, the
// accepted argument count range, and the builder mapping parsed arguments to a
// gopolars expression. maxArgs == -1 means variadic.
type fnSpec struct {
	name    string
	minArgs int
	maxArgs int
	build   func(args []expr.Expr) (expr.Expr, error)
}

// fnRegistry maps lowercase function names (canonical names and aliases) to
// their specs.
var fnRegistry = map[string]*fnSpec{}

// registerFn adds a function (and its aliases) to the catalog.
func registerFn(name string, minArgs, maxArgs int, build func(args []expr.Expr) (expr.Expr, error), aliases ...string) {
	spec := &fnSpec{name: name, minArgs: minArgs, maxArgs: maxArgs, build: build}
	fnRegistry[name] = spec
	for _, a := range aliases {
		fnRegistry[a] = spec
	}
}

// buildFunction maps a SQL function call (name + already-parsed argument
// expressions) to a gopolars expression. Unknown functions produce a clear
// "unsupported function" error; wrong arity names the function and the
// expected signature.
func buildFunction(name string, args []expr.Expr) (expr.Expr, error) {
	spec, ok := fnRegistry[strings.ToLower(name)]
	if !ok {
		return expr.Expr{}, fmt.Errorf("unsupported function: %s", name)
	}
	if len(args) < spec.minArgs || (spec.maxArgs >= 0 && len(args) > spec.maxArgs) {
		return expr.Expr{}, fmt.Errorf("%s expects %s, got %d", strings.ToUpper(spec.name), arityText(spec.minArgs, spec.maxArgs), len(args))
	}
	return spec.build(args)
}

func arityText(minArgs, maxArgs int) string {
	switch {
	case maxArgs < 0:
		return fmt.Sprintf("at least %d argument(s)", minArgs)
	case minArgs == maxArgs:
		return fmt.Sprintf("%d argument(s)", minArgs)
	default:
		return fmt.Sprintf("between %d and %d arguments", minArgs, maxArgs)
	}
}

func init() {
	// ---- aggregates ----
	registerFn("sum", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.Sum(args[0]), nil })
	registerFn("min", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.Min(args[0]), nil })
	registerFn("max", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.Max(args[0]), nil })
	registerFn("avg", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.Mean(args[0]), nil }, "mean")
	// COUNT(*) and COUNT(col) both count rows.
	registerFn("count", 0, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.Count(), nil })
	registerFn("n_unique", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.NUnique(args[0]), nil })

	// ---- string scalar ----
	registerFn("upper", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrUpper(), nil })
	registerFn("lower", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrLower(), nil })
	registerFn("length", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrLen(), nil },
		"char_length", "character_length")
	registerFn("trim", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrTrim(), nil })
	registerFn("replace", 3, 3, func(args []expr.Expr) (expr.Expr, error) {
		from, ok1 := strLiteral(args[1])
		to, ok2 := strLiteral(args[2])
		if !ok1 || !ok2 {
			return expr.Expr{}, fmt.Errorf("REPLACE requires string literal arguments")
		}
		return args[0].StrReplace(from, to), nil
	})
	registerFn("concat", 1, -1, func(args []expr.Expr) (expr.Expr, error) {
		out := args[0]
		for _, a := range args[1:] {
			out = out.StrConcat(a)
		}
		return out, nil
	})
	registerFn("substr", 2, 3, func(args []expr.Expr) (expr.Expr, error) {
		start, ok := intLiteral(args[1])
		if !ok {
			return expr.Expr{}, fmt.Errorf("SUBSTR requires an integer start position")
		}
		length := 1 << 30
		if len(args) == 3 {
			l, ok := intLiteral(args[2])
			if !ok {
				return expr.Expr{}, fmt.Errorf("SUBSTR requires an integer length")
			}
			length = l
		}
		return args[0].StrSubstr(start, length), nil
	}, "substring")

	// ---- conditional / null ----
	registerFn("coalesce", 1, -1, func(args []expr.Expr) (expr.Expr, error) {
		out := args[0]
		for _, a := range args[1:] {
			out = out.FillNull(a)
		}
		return out, nil
	})
	registerFn("nullif", 2, 2, func(args []expr.Expr) (expr.Expr, error) {
		return expr.When(args[0].Eq(args[1]), expr.Lit(nil), args[0]), nil
	})

	// ---- math scalar ----
	registerFn("abs", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Abs(), nil })
	registerFn("round", 1, 2, func(args []expr.Expr) (expr.Expr, error) {
		decimals := 0
		if len(args) == 2 {
			d, ok := intLiteral(args[1])
			if !ok {
				return expr.Expr{}, fmt.Errorf("ROUND requires an integer number of decimals")
			}
			decimals = d
		}
		return args[0].RoundDP(decimals), nil
	})
	registerFn("ceil", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Ceil(), nil }, "ceiling")
	registerFn("floor", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Floor(), nil })
	registerFn("sqrt", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Sqrt(), nil })
	registerFn("power", 2, 2, func(args []expr.Expr) (expr.Expr, error) { return args[0].Pow(args[1]), nil }, "pow")
	registerFn("mod", 2, 2, func(args []expr.Expr) (expr.Expr, error) { return args[0].Mod(args[1]), nil })
	registerFn("exp", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Exp(), nil })
	registerFn("ln", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Log(), nil })

	// ---- date scalar ----
	registerFn("year", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].DtYear(), nil })
	registerFn("month", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].DtMonth(), nil })
	registerFn("day", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].DtDay(), nil })
	registerFn("hour", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].DtHour(), nil })
	registerFn("minute", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].DtMinute(), nil })
	registerFn("second", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].DtSecond(), nil })

	registerExtendedFns()
}

// registerExtendedFns adds the full-SQL-surface catalog: extended string,
// math, conditional, temporal, and aggregate functions.
func registerExtendedFns() {
	// ---- string scalar ----
	registerFn("starts_with", 2, 2, func(args []expr.Expr) (expr.Expr, error) { return args[0].StartsWith(args[1]), nil })
	registerFn("ends_with", 2, 2, func(args []expr.Expr) (expr.Expr, error) { return args[0].EndsWith(args[1]), nil })
	registerFn("left", 2, 2, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrLeft(args[1]), nil })
	registerFn("right", 2, 2, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrRight(args[1]), nil })
	registerFn("ltrim", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrLTrim(), nil })
	registerFn("rtrim", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrRTrim(), nil })
	registerFn("reverse", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrReverse(), nil })
	registerFn("initcap", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrToTitle(), nil })
	registerFn("lpad", 2, 3, func(args []expr.Expr) (expr.Expr, error) {
		fill := expr.Lit(" ")
		if len(args) == 3 {
			fill = args[2]
		}
		return args[0].StrPadStart(args[1], fill), nil
	})
	registerFn("rpad", 2, 3, func(args []expr.Expr) (expr.Expr, error) {
		fill := expr.Lit(" ")
		if len(args) == 3 {
			fill = args[2]
		}
		return args[0].StrPadEnd(args[1], fill), nil
	})
	registerFn("split_part", 3, 3, func(args []expr.Expr) (expr.Expr, error) {
		return args[0].StrSplitPart(args[1], args[2]), nil
	})
	registerFn("strpos", 2, 2, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrPos(args[1]), nil },
		"position")
	registerFn("octet_length", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].StrLen(), nil })
	registerFn("bit_length", 1, 1, func(args []expr.Expr) (expr.Expr, error) {
		return args[0].StrLen().Mul(expr.Lit(int64(8))), nil
	})
	registerFn("regexp_like", 2, 2, func(args []expr.Expr) (expr.Expr, error) { return args[0].RegexpLike(args[1]), nil })
	registerFn("concat_ws", 2, -1, func(args []expr.Expr) (expr.Expr, error) {
		sep, ok := strLiteral(args[0])
		if !ok {
			return expr.Expr{}, fmt.Errorf("CONCAT_WS requires a string literal separator")
		}
		out := args[1]
		for _, a := range args[2:] {
			out = out.StrConcatWS(sep, a)
		}
		return out, nil
	})

	// ---- math scalar ----
	registerFn("sign", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Sign(), nil })
	registerFn("cbrt", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Cbrt(), nil })
	// LOG(x) is the natural log (alias of LN); LOG(base, x) = ln(x)/ln(base).
	registerFn("log", 1, 2, func(args []expr.Expr) (expr.Expr, error) {
		if len(args) == 1 {
			return args[0].Log(), nil
		}
		return args[1].Log().Div(args[0].Log()), nil
	})
	registerFn("log2", 1, 1, func(args []expr.Expr) (expr.Expr, error) {
		return args[0].Log().Div(expr.Lit(math.Ln2)), nil
	})
	registerFn("log10", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Log10(), nil })
	registerFn("log1p", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Log1p(), nil })
	registerFn("pi", 0, 0, func(args []expr.Expr) (expr.Expr, error) { return expr.Lit(math.Pi), nil })
	registerFn("degrees", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Degrees(), nil })
	registerFn("radians", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Radians(), nil })
	registerFn("sin", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Sin(), nil })
	registerFn("cos", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Cos(), nil })
	registerFn("tan", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Tan(), nil })
	registerFn("cot", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Cot(), nil })
	registerFn("asin", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Arcsin(), nil })
	registerFn("acos", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Arccos(), nil })
	registerFn("atan", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].Arctan(), nil })
	registerFn("atan2", 2, 2, func(args []expr.Expr) (expr.Expr, error) { return args[0].Atan2(args[1]), nil })

	// ---- conditional / null ----
	registerFn("ifnull", 2, 2, func(args []expr.Expr) (expr.Expr, error) { return args[0].FillNull(args[1]), nil })
	registerFn("if", 3, 3, func(args []expr.Expr) (expr.Expr, error) {
		return expr.When(args[0], args[1], args[2]), nil
	})
	registerFn("greatest", 1, -1, buildExtremeFn("greatest"))
	registerFn("least", 1, -1, buildExtremeFn("least"))

	// ---- temporal ----
	registerFn("date_part", 2, 2, func(args []expr.Expr) (expr.Expr, error) {
		part, ok := strLiteral(args[0])
		if !ok {
			return expr.Expr{}, fmt.Errorf("DATE_PART requires a string literal part")
		}
		return buildDatePart(part, args[1])
	})
	registerFn("dayofweek", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].DtWeekday(), nil },
		"weekday")
	registerFn("dayofyear", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return args[0].DtOrdinalDay(), nil },
		"ordinal_day")

	// ---- aggregates ----
	registerFn("stddev", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.Std(args[0]), nil },
		"stdev", "stddev_samp")
	registerFn("variance", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.Var(args[0]), nil },
		"var", "var_samp")
	registerFn("median", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.Median(args[0]), nil })
	registerFn("first", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.First(args[0]), nil })
	registerFn("last", 1, 1, func(args []expr.Expr) (expr.Expr, error) { return expr.Last(args[0]), nil })
}

// buildDatePart maps a DATE_PART/EXTRACT part name to the matching temporal
// extraction. Unsupported parts name the part in the error.
func buildDatePart(part string, e expr.Expr) (expr.Expr, error) {
	switch strings.ToLower(part) {
	case "year":
		return e.DtYear(), nil
	case "month":
		return e.DtMonth(), nil
	case "day":
		return e.DtDay(), nil
	case "hour":
		return e.DtHour(), nil
	case "minute":
		return e.DtMinute(), nil
	case "second":
		return e.DtSecond(), nil
	case "dow", "weekday", "dayofweek", "isodow":
		return e.DtWeekday(), nil
	case "doy", "dayofyear", "ordinal_day":
		return e.DtOrdinalDay(), nil
	default:
		return expr.Expr{}, fmt.Errorf("unsupported date part: %s", part)
	}
}

// buildExtremeFn returns the builder for GREATEST/LEAST: a row-wise fold over
// the arguments that skips NULL operands.
func buildExtremeFn(name string) func(args []expr.Expr) (expr.Expr, error) {
	greatest := name == "greatest"
	return func(args []expr.Expr) (expr.Expr, error) {
		fold := expr.Fold(nil, func(acc any, next any) (any, error) {
			if next == nil {
				return acc, nil
			}
			if acc == nil {
				return next, nil
			}
			cmp, err := compareScalars(acc, next)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", strings.ToUpper(name), err)
			}
			if (greatest && cmp < 0) || (!greatest && cmp > 0) {
				return next, nil
			}
			return acc, nil
		}, args...)
		return fold.Alias(name), nil
	}
}

// compareScalars orders two non-nil scalar values of comparable types.
func compareScalars(a any, b any) (int, error) {
	if af, aok := toFloatScalar(a); aok {
		bf, bok := toFloatScalar(b)
		if !bok {
			return 0, fmt.Errorf("cannot compare %T with %T", a, b)
		}
		switch {
		case af < bf:
			return -1, nil
		case af > bf:
			return 1, nil
		default:
			return 0, nil
		}
	}
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return strings.Compare(as, bs), nil
	}
	at, aok := a.(time.Time)
	bt, bok := b.(time.Time)
	if aok && bok {
		switch {
		case at.Before(bt):
			return -1, nil
		case at.After(bt):
			return 1, nil
		default:
			return 0, nil
		}
	}
	return 0, fmt.Errorf("cannot compare %T with %T", a, b)
}

func toFloatScalar(v any) (float64, bool) {
	switch t := v.(type) {
	case int64:
		return float64(t), true
	case int:
		return float64(t), true
	case float64:
		return t, true
	default:
		return 0, false
	}
}

// strLiteral returns the string value of a literal expression, if it is one.
func strLiteral(e expr.Expr) (string, bool) {
	if e.Kind() != expr.KindLit {
		return "", false
	}
	s, ok := e.Value().(string)
	return s, ok
}

// intLiteral returns the int value of an integer literal expression, if it is one.
func intLiteral(e expr.Expr) (int, bool) {
	if e.Kind() != expr.KindLit {
		return 0, false
	}
	switch v := e.Value().(type) {
	case int64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}
