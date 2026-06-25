package expr

import "fmt"

// This file holds expression builders for the scalar string, datetime, and
// numeric operations. They follow the same op-string convention as the builders
// in expr.go and are evaluated in eval.go.

// StrLike builds a LIKE-style match. The pattern uses wildcards: '%' matches
// any run of characters and '_' matches a single character.
func (e Expr) StrLike(pattern string) Expr {
	return Expr{kind: KindUnary, op: "str_like:" + pattern, target: &e}
}

// StrConcat concatenates two string expressions.
func (e Expr) StrConcat(other Expr) Expr {
	return bin("str_concat", e, other)
}

// StrSubstr extracts a substring. start is 1-based; length is the number of
// characters to take.
func (e Expr) StrSubstr(start int, length int) Expr {
	return Expr{kind: KindUnary, op: fmt.Sprintf("str_substr:%d:%d", start, length), target: &e}
}

// RoundDP rounds a numeric value to the given number of decimal places.
func (e Expr) RoundDP(decimals int) Expr {
	return Expr{kind: KindUnary, op: fmt.Sprintf("round_dp:%d", decimals), target: &e}
}

// DtMinute extracts the minute component of a datetime.
func (e Expr) DtMinute() Expr {
	return Expr{kind: KindUnary, op: "dt_minute", target: &e}
}

// DtSecond extracts the second component of a datetime.
func (e Expr) DtSecond() Expr {
	return Expr{kind: KindUnary, op: "dt_second", target: &e}
}
