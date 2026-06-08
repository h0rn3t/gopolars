package sql

import (
	"fmt"
	"strings"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

// buildFunction maps a SQL function call (name + already-parsed argument
// expressions) to a gopolars expression. Unknown functions produce a clear
// "unsupported function" error.
func buildFunction(name string, args []expr.Expr) (expr.Expr, error) {
	lname := strings.ToLower(name)
	switch lname {
	// ---- aggregates ----
	case "sum":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return expr.Sum(args[0]), nil
	case "min":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return expr.Min(args[0]), nil
	case "max":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return expr.Max(args[0]), nil
	case "avg", "mean":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return expr.Mean(args[0]), nil
	case "count":
		// COUNT(*) and COUNT(col) both count rows.
		return expr.Count(), nil
	case "n_unique":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return expr.NUnique(args[0]), nil

	// ---- string scalar ----
	case "upper":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].StrUpper(), nil
	case "lower":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].StrLower(), nil
	case "length", "char_length", "character_length":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].StrLen(), nil
	case "trim":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].StrTrim(), nil
	case "replace":
		if err := wantArgs(lname, args, 3); err != nil {
			return expr.Expr{}, err
		}
		from, ok1 := strLiteral(args[1])
		to, ok2 := strLiteral(args[2])
		if !ok1 || !ok2 {
			return expr.Expr{}, fmt.Errorf("REPLACE requires string literal arguments")
		}
		return args[0].StrReplace(from, to), nil
	case "concat":
		if len(args) == 0 {
			return expr.Expr{}, fmt.Errorf("CONCAT requires at least one argument")
		}
		out := args[0]
		for _, a := range args[1:] {
			out = out.StrConcat(a)
		}
		return out, nil
	case "substr", "substring":
		if len(args) < 2 || len(args) > 3 {
			return expr.Expr{}, fmt.Errorf("%s requires 2 or 3 arguments", lname)
		}
		start, ok := intLiteral(args[1])
		if !ok {
			return expr.Expr{}, fmt.Errorf("%s requires an integer start position", lname)
		}
		length := 1 << 30
		if len(args) == 3 {
			l, ok := intLiteral(args[2])
			if !ok {
				return expr.Expr{}, fmt.Errorf("%s requires an integer length", lname)
			}
			length = l
		}
		return args[0].StrSubstr(start, length), nil

	// ---- conditional / null ----
	case "coalesce":
		if len(args) == 0 {
			return expr.Expr{}, fmt.Errorf("COALESCE requires at least one argument")
		}
		out := args[0]
		for _, a := range args[1:] {
			out = out.FillNull(a)
		}
		return out, nil
	case "nullif":
		if err := wantArgs(lname, args, 2); err != nil {
			return expr.Expr{}, err
		}
		return expr.When(args[0].Eq(args[1]), expr.Lit(nil), args[0]), nil

	// ---- math scalar ----
	case "abs":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].Abs(), nil
	case "round":
		if len(args) < 1 || len(args) > 2 {
			return expr.Expr{}, fmt.Errorf("ROUND requires 1 or 2 arguments")
		}
		decimals := 0
		if len(args) == 2 {
			d, ok := intLiteral(args[1])
			if !ok {
				return expr.Expr{}, fmt.Errorf("ROUND requires an integer number of decimals")
			}
			decimals = d
		}
		return args[0].RoundDP(decimals), nil
	case "ceil", "ceiling":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].Ceil(), nil
	case "floor":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].Floor(), nil
	case "sqrt":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].Sqrt(), nil
	case "power", "pow":
		if err := wantArgs(lname, args, 2); err != nil {
			return expr.Expr{}, err
		}
		return args[0].Pow(args[1]), nil
	case "mod":
		if err := wantArgs(lname, args, 2); err != nil {
			return expr.Expr{}, err
		}
		return args[0].Mod(args[1]), nil
	case "exp":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].Exp(), nil
	case "ln":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].Log(), nil

	// ---- date scalar ----
	case "year":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].DtYear(), nil
	case "month":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].DtMonth(), nil
	case "day":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].DtDay(), nil
	case "hour":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].DtHour(), nil
	case "minute":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].DtMinute(), nil
	case "second":
		if err := wantArgs(lname, args, 1); err != nil {
			return expr.Expr{}, err
		}
		return args[0].DtSecond(), nil

	default:
		return expr.Expr{}, fmt.Errorf("unsupported function: %s", name)
	}
}

func wantArgs(name string, args []expr.Expr, n int) error {
	if len(args) != n {
		return fmt.Errorf("%s expects %d argument(s), got %d", strings.ToUpper(name), n, len(args))
	}
	return nil
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
