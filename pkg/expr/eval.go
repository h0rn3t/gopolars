package expr

import (
	"fmt"
	"math"
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
		default:
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
	case "and":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, fmt.Errorf("and expects bool")
		}
		return l && r, nil
	case "or":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, fmt.Errorf("or expects bool")
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
