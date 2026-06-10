package expr

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Aggregation constructors for the SQL catalog (group-by reductions, like
// Sum/Min/Max/Mean/Count/NUnique above in expr.go).
func Std(e Expr) Expr    { return Expr{kind: KindAgg, op: "std", target: &e} }
func Var(e Expr) Expr    { return Expr{kind: KindAgg, op: "var", target: &e} }
func Median(e Expr) Expr { return Expr{kind: KindAgg, op: "median", target: &e} }
func First(e Expr) Expr  { return Expr{kind: KindAgg, op: "first", target: &e} }
func Last(e Expr) Expr   { return Expr{kind: KindAgg, op: "last", target: &e} }

// CountDistinct counts distinct non-null values (SQL COUNT(DISTINCT col)).
func CountDistinct(e Expr) Expr { return Expr{kind: KindAgg, op: "count_distinct", target: &e} }

// Builder methods for SQL-motivated operations (see sql_ops.go for the earlier
// set). NULL inputs propagate as NULL outputs in every kernel below.

func (e Expr) EndsWith(suffix Expr) Expr { return bin("ends_with", e, suffix) }

// StrPos returns the 1-based character position of sub in e, NULL when absent.
func (e Expr) StrPos(sub Expr) Expr { return bin("str_pos", e, sub) }

// StrLeft keeps the first n characters (negative n drops the last |n|).
func (e Expr) StrLeft(n Expr) Expr { return bin("str_left", e, n) }

// StrRight keeps the last n characters (negative n drops the first |n|).
func (e Expr) StrRight(n Expr) Expr { return bin("str_right", e, n) }

// RegexpLike reports whether e matches the regular expression pattern.
func (e Expr) RegexpLike(pattern Expr) Expr { return bin("regexp_like", e, pattern) }

// Atan2 computes atan2(e, x) with e as the y coordinate.
func (e Expr) Atan2(x Expr) Expr { return bin("atan2", e, x) }

// StrConcatWS joins e and other with sep, skipping NULL operands (CONCAT_WS).
func (e Expr) StrConcatWS(sep string, other Expr) Expr {
	return bin("str_concat_ws:"+sep, e, other)
}

func (e Expr) StrLTrim() Expr   { return Expr{kind: KindUnary, op: "str_ltrim", target: &e} }
func (e Expr) StrRTrim() Expr   { return Expr{kind: KindUnary, op: "str_rtrim", target: &e} }
func (e Expr) StrReverse() Expr { return Expr{kind: KindUnary, op: "str_reverse", target: &e} }

// StrToTitle uppercases the first letter of each word and lowercases the rest
// (SQL INITCAP).
func (e Expr) StrToTitle() Expr { return Expr{kind: KindUnary, op: "str_to_title", target: &e} }

// StrCharLen counts characters (runes), unlike StrLen which counts bytes.
func (e Expr) StrCharLen() Expr { return Expr{kind: KindUnary, op: "str_char_len", target: &e} }

// DtOrdinalDay extracts the day of the year (1-366).
func (e Expr) DtOrdinalDay() Expr { return Expr{kind: KindUnary, op: "dt_ordinal_day", target: &e} }

// StrPadStart left-pads e to n characters with fill, truncating past n (LPAD).
func (e Expr) StrPadStart(n Expr, fill Expr) Expr {
	return Expr{kind: KindTern, op: "str_pad_start", target: &e, left: &n, right: &fill}
}

// StrPadEnd right-pads e to n characters with fill, truncating past n (RPAD).
func (e Expr) StrPadEnd(n Expr, fill Expr) Expr {
	return Expr{kind: KindTern, op: "str_pad_end", target: &e, left: &n, right: &fill}
}

// StrSplitPart splits e by sep and returns the 1-based n-th field; out-of-range
// positions yield an empty string (SPLIT_PART).
func (e Expr) StrSplitPart(sep Expr, n Expr) Expr {
	return Expr{kind: KindTern, op: "str_split_part", target: &e, left: &sep, right: &n}
}

// evalSQLUnary evaluates the SQL-motivated unary kernels. handled is false
// when op is not one of them.
func evalSQLUnary(op string, v any) (out any, handled bool, err error) {
	switch op {
	case "str_ltrim":
		if v == nil {
			return nil, true, nil
		}
		s, ok := v.(string)
		if !ok {
			return nil, true, fmt.Errorf("str_ltrim expects string")
		}
		return strings.TrimLeftFunc(s, unicode.IsSpace), true, nil
	case "str_rtrim":
		if v == nil {
			return nil, true, nil
		}
		s, ok := v.(string)
		if !ok {
			return nil, true, fmt.Errorf("str_rtrim expects string")
		}
		return strings.TrimRightFunc(s, unicode.IsSpace), true, nil
	case "str_reverse":
		if v == nil {
			return nil, true, nil
		}
		s, ok := v.(string)
		if !ok {
			return nil, true, fmt.Errorf("str_reverse expects string")
		}
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), true, nil
	case "str_to_title":
		if v == nil {
			return nil, true, nil
		}
		s, ok := v.(string)
		if !ok {
			return nil, true, fmt.Errorf("str_to_title expects string")
		}
		var b strings.Builder
		startOfWord := true
		for _, r := range s {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				if startOfWord {
					b.WriteRune(unicode.ToUpper(r))
				} else {
					b.WriteRune(unicode.ToLower(r))
				}
				startOfWord = false
			} else {
				b.WriteRune(r)
				startOfWord = true
			}
		}
		return b.String(), true, nil
	case "str_char_len":
		if v == nil {
			return nil, true, nil
		}
		s, ok := v.(string)
		if !ok {
			return nil, true, fmt.Errorf("str_char_len expects string")
		}
		return int64(utf8.RuneCountInString(s)), true, nil
	case "dt_ordinal_day":
		if v == nil {
			return nil, true, nil
		}
		t, ok := v.(time.Time)
		if !ok {
			return nil, true, fmt.Errorf("dt_ordinal_day expects datetime")
		}
		return int64(t.YearDay()), true, nil
	}
	return nil, false, nil
}

// evalSQLBin evaluates the SQL-motivated binary kernels. handled is false when
// op is not one of them.
func evalSQLBin(op string, left any, right any) (out any, handled bool, err error) {
	if sep, ok := strings.CutPrefix(op, "str_concat_ws:"); ok {
		// NULL operands are skipped (CONCAT_WS semantics); both NULL stays NULL.
		if left == nil && right == nil {
			return nil, true, nil
		}
		if left == nil {
			return right, true, nil
		}
		if right == nil {
			return left, true, nil
		}
		l, lok := left.(string)
		r, rok := right.(string)
		if !lok || !rok {
			return nil, true, fmt.Errorf("concat_ws expects strings")
		}
		return l + sep + r, true, nil
	}
	switch op {
	case "ends_with":
		if left == nil || right == nil {
			return nil, true, nil
		}
		l, lok := left.(string)
		r, rok := right.(string)
		if !lok || !rok {
			return nil, true, fmt.Errorf("ends_with expects strings")
		}
		return strings.HasSuffix(l, r), true, nil
	case "str_pos":
		if left == nil || right == nil {
			return nil, true, nil
		}
		l, lok := left.(string)
		r, rok := right.(string)
		if !lok || !rok {
			return nil, true, fmt.Errorf("str_pos expects strings")
		}
		idx := strings.Index(l, r)
		if idx < 0 {
			return nil, true, nil
		}
		return int64(utf8.RuneCountInString(l[:idx])) + 1, true, nil
	case "str_left", "str_right":
		if left == nil || right == nil {
			return nil, true, nil
		}
		s, sok := left.(string)
		n, nok := toInt64(right)
		if !sok || !nok {
			return nil, true, fmt.Errorf("%s expects a string and an integer count", op)
		}
		runes := []rune(s)
		count := int(n)
		if count < 0 {
			count = len(runes) + count
		}
		if count < 0 {
			count = 0
		}
		if count > len(runes) {
			count = len(runes)
		}
		if op == "str_left" {
			return string(runes[:count]), true, nil
		}
		return string(runes[len(runes)-count:]), true, nil
	case "regexp_like":
		if left == nil || right == nil {
			return nil, true, nil
		}
		l, lok := left.(string)
		r, rok := right.(string)
		if !lok || !rok {
			return nil, true, fmt.Errorf("regexp_like expects strings")
		}
		re, err := regexp.Compile(r)
		if err != nil {
			return nil, true, fmt.Errorf("regexp_like invalid pattern %q: %w", r, err)
		}
		return re.MatchString(l), true, nil
	case "atan2":
		if left == nil || right == nil {
			return nil, true, nil
		}
		y, yok := toFloat(left)
		x, xok := toFloat(right)
		if !yok || !xok {
			return nil, true, fmt.Errorf("atan2 expects numeric")
		}
		return math.Atan2(y, x), true, nil
	}
	return nil, false, nil
}

// evalSQLTern evaluates the SQL-motivated ternary kernels (already-evaluated
// operands). handled is false when op is not one of them.
func evalSQLTern(op string, target any, left any, right any) (out any, handled bool, err error) {
	switch op {
	case "str_pad_start", "str_pad_end":
		if target == nil {
			return nil, true, nil
		}
		s, sok := target.(string)
		n, nok := toInt64(left)
		if !sok || !nok {
			return nil, true, fmt.Errorf("%s expects a string and an integer length", op)
		}
		fill := " "
		if right != nil {
			f, fok := right.(string)
			if !fok {
				return nil, true, fmt.Errorf("%s expects a string fill", op)
			}
			fill = f
		}
		runes := []rune(s)
		width := int(n)
		if width < 0 {
			width = 0
		}
		if len(runes) >= width {
			return string(runes[:width]), true, nil
		}
		if fill == "" {
			return s, true, nil
		}
		fillRunes := []rune(fill)
		pad := make([]rune, 0, width-len(runes))
		for i := 0; i < width-len(runes); i++ {
			pad = append(pad, fillRunes[i%len(fillRunes)])
		}
		if op == "str_pad_start" {
			return string(pad) + s, true, nil
		}
		return s + string(pad), true, nil
	case "str_split_part":
		if target == nil {
			return nil, true, nil
		}
		s, sok := target.(string)
		sep, pok := left.(string)
		n, nok := toInt64(right)
		if !sok || !pok || !nok {
			return nil, true, fmt.Errorf("str_split_part expects a string, a string separator, and an integer field")
		}
		if n <= 0 {
			return nil, true, fmt.Errorf("str_split_part field position must be greater than zero")
		}
		parts := strings.Split(s, sep)
		if int(n) > len(parts) {
			return "", true, nil
		}
		return parts[n-1], true, nil
	}
	return nil, false, nil
}
