package sql

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

type mapRow map[string]any

func (m mapRow) ValueByName(name string) (any, bool) {
	v, ok := m[name]
	return v, ok
}

func evalStr(t *testing.T, src string, row mapRow) any {
	t.Helper()
	e, err := parseExpression(src)
	if err != nil {
		t.Fatalf("parseExpression(%q): %v", src, err)
	}
	v, err := expr.Eval(e, row)
	if err != nil {
		t.Fatalf("Eval(%q): %v", src, err)
	}
	return v
}

func TestExprParserBooleanPrecedence(t *testing.T) {
	row := mapRow{"a": int64(2), "b": int64(3), "c": int64(0)}
	// (a > 1 AND b < 5) OR c = 0  => true
	if got := evalStr(t, "a > 1 AND b < 5 OR c = 0", row); got != true {
		t.Fatalf("precedence got %v, want true", got)
	}
	// a > 1 AND (b < 5 OR c = 9): a>1 true, (b<5 true) => true
	if got := evalStr(t, "a > 1 AND (b < 5 OR c = 9)", row); got != true {
		t.Fatalf("paren got %v, want true", got)
	}
	// NOT (a = 1) => true
	if got := evalStr(t, "NOT (a = 1)", row); got != true {
		t.Fatalf("not got %v, want true", got)
	}
}

func TestExprParserArithmetic(t *testing.T) {
	row := mapRow{"a": int64(1), "b": int64(4)}
	// a + b * 2 = 1 + 8 = 9
	if got := evalStr(t, "a + b * 2", row); got != int64(9) {
		t.Fatalf("arith got %v, want 9", got)
	}
	// (a + b) * 2 = 10
	if got := evalStr(t, "(a + b) * 2", row); got != int64(10) {
		t.Fatalf("paren arith got %v, want 10", got)
	}
}

func TestExprParserPredicates(t *testing.T) {
	row := mapRow{"x": int64(2), "v": int64(15), "name": "Alpha", "n": nil}
	if got := evalStr(t, "x IN (1, 2, 3)", row); got != true {
		t.Fatalf("IN got %v, want true", got)
	}
	if got := evalStr(t, "x NOT IN (5, 6)", row); got != true {
		t.Fatalf("NOT IN got %v, want true", got)
	}
	if got := evalStr(t, "v BETWEEN 10 AND 20", row); got != true {
		t.Fatalf("BETWEEN got %v, want true", got)
	}
	if got := evalStr(t, "name LIKE 'A%'", row); got != true {
		t.Fatalf("LIKE prefix got %v, want true", got)
	}
	if got := evalStr(t, "name LIKE '_lpha'", row); got != true {
		t.Fatalf("LIKE underscore got %v, want true", got)
	}
	if got := evalStr(t, "name LIKE 'z%'", row); got != false {
		t.Fatalf("LIKE no-match got %v, want false", got)
	}
	if got := evalStr(t, "n IS NULL", row); got != true {
		t.Fatalf("IS NULL got %v, want true", got)
	}
	if got := evalStr(t, "x IS NOT NULL", row); got != true {
		t.Fatalf("IS NOT NULL got %v, want true", got)
	}
}

func TestExprParserCase(t *testing.T) {
	pos := mapRow{"x": int64(5)}
	neg := mapRow{"x": int64(-5)}
	if got := evalStr(t, "CASE WHEN x > 0 THEN 'pos' WHEN x < 0 THEN 'neg' ELSE 'zero' END", pos); got != "pos" {
		t.Fatalf("CASE pos got %v", got)
	}
	if got := evalStr(t, "CASE WHEN x > 0 THEN 'pos' WHEN x < 0 THEN 'neg' ELSE 'zero' END", neg); got != "neg" {
		t.Fatalf("CASE neg got %v", got)
	}
	// no ELSE => null
	if got := evalStr(t, "CASE WHEN x > 0 THEN 1 END", neg); got != nil {
		t.Fatalf("CASE no-else got %v, want nil", got)
	}
}

func TestExprParserFunctions(t *testing.T) {
	row := mapRow{"s": "abc", "v": 3.14159, "a": nil, "b": int64(7)}
	if got := evalStr(t, "UPPER(s)", row); got != "ABC" {
		t.Fatalf("UPPER got %v", got)
	}
	if got := evalStr(t, "LENGTH(s)", row); got != int64(3) {
		t.Fatalf("LENGTH got %v", got)
	}
	if got := evalStr(t, "ROUND(v, 2)", row); got != 3.14 {
		t.Fatalf("ROUND got %v, want 3.14", got)
	}
	if got := evalStr(t, "COALESCE(a, b, 0)", row); got != int64(7) {
		t.Fatalf("COALESCE got %v, want 7", got)
	}
	if got := evalStr(t, "ABS(0 - 5)", row); got != int64(5) {
		t.Fatalf("ABS got %v, want 5", got)
	}
}

func TestExprParserCast(t *testing.T) {
	row := mapRow{"s": "123", "v": int64(9)}
	if got := evalStr(t, "CAST(s AS INTEGER)", row); got != int64(123) {
		t.Fatalf("CAST string->int got %v, want 123", got)
	}
	if got := evalStr(t, "v::FLOAT", row); got != float64(9) {
		t.Fatalf(":: cast got %v, want 9.0", got)
	}
}

func TestExprParserUnknownFunction(t *testing.T) {
	if _, err := parseExpression("BOGUS(x)"); err == nil {
		t.Fatalf("expected error for unknown function")
	}
}

func TestExprParserAggregateKind(t *testing.T) {
	e, err := parseExpression("sum(value)")
	if err != nil {
		t.Fatalf("parse sum: %v", err)
	}
	if e.Kind() != expr.KindAgg {
		t.Fatalf("sum(value) kind = %v, want agg", e.Kind())
	}
}
