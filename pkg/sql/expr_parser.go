package sql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
)

// exprParser is a recursive-descent parser over a token stream that produces a
// gopolars expression. Precedence (low to high): OR < AND < NOT < comparison
// (= <> < <= > >=, IS [NOT] NULL, [NOT] IN, [NOT] BETWEEN, [NOT] LIKE) <
// additive (+ -) < multiplicative (* / %) < unary (- +) < postfix (::) <
// primary (literal, column, function call, CASE, parenthesized expression).
type exprParser struct {
	tokens []token
	pos    int
}

func newExprParser(tokens []token) *exprParser {
	return &exprParser{tokens: tokens}
}

// parseExpression tokenizes and fully parses a standalone SQL expression.
func parseExpression(s string) (expr.Expr, error) {
	toks, err := lex(s)
	if err != nil {
		return expr.Expr{}, err
	}
	p := newExprParser(toks)
	e, err := p.parse()
	if err != nil {
		return expr.Expr{}, err
	}
	if p.peek().kind != tokEOF {
		return expr.Expr{}, fmt.Errorf("unexpected token %q in SQL expression", p.peek().text)
	}
	return e, nil
}

func (p *exprParser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{kind: tokEOF}
	}
	return p.tokens[p.pos]
}

func (p *exprParser) next() token {
	t := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return t
}

// keyword reports whether t is an unquoted identifier equal (case-insensitively) to kw.
func keyword(t token, kw string) bool {
	return t.kind == tokIdent && !t.quoted && strings.EqualFold(t.text, kw)
}

func (p *exprParser) parse() (expr.Expr, error) {
	return p.parseOr()
}

func (p *exprParser) parseOr() (expr.Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return expr.Expr{}, err
	}
	for keyword(p.peek(), "or") {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return expr.Expr{}, err
		}
		left = left.Or(right)
	}
	return left, nil
}

func (p *exprParser) parseAnd() (expr.Expr, error) {
	left, err := p.parseNot()
	if err != nil {
		return expr.Expr{}, err
	}
	for keyword(p.peek(), "and") {
		p.next()
		right, err := p.parseNot()
		if err != nil {
			return expr.Expr{}, err
		}
		left = left.And(right)
	}
	return left, nil
}

func (p *exprParser) parseNot() (expr.Expr, error) {
	if keyword(p.peek(), "not") {
		p.next()
		inner, err := p.parseNot()
		if err != nil {
			return expr.Expr{}, err
		}
		return inner.Not(), nil
	}
	return p.parseComparison()
}

func (p *exprParser) parseComparison() (expr.Expr, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return expr.Expr{}, err
	}
	// IS [NOT] NULL
	if keyword(p.peek(), "is") {
		p.next()
		negate := false
		if keyword(p.peek(), "not") {
			p.next()
			negate = true
		}
		if !keyword(p.peek(), "null") {
			return expr.Expr{}, fmt.Errorf("expected NULL after IS")
		}
		p.next()
		if negate {
			return left.IsNotNull(), nil
		}
		return left.IsNull(), nil
	}
	// [NOT] IN | BETWEEN | LIKE
	if keyword(p.peek(), "not") {
		// look ahead for IN/BETWEEN/LIKE
		save := p.pos
		p.next()
		switch {
		case keyword(p.peek(), "in"):
			e, err := p.parseIn(left)
			if err != nil {
				return expr.Expr{}, err
			}
			return e.Not(), nil
		case keyword(p.peek(), "between"):
			e, err := p.parseBetween(left)
			if err != nil {
				return expr.Expr{}, err
			}
			return e.Not(), nil
		case keyword(p.peek(), "like"):
			e, err := p.parseLike(left)
			if err != nil {
				return expr.Expr{}, err
			}
			return e.Not(), nil
		default:
			// not part of a predicate suffix; rewind so the outer NOT handling is unaffected
			p.pos = save
			return left, nil
		}
	}
	if keyword(p.peek(), "in") {
		return p.parseIn(left)
	}
	if keyword(p.peek(), "between") {
		return p.parseBetween(left)
	}
	if keyword(p.peek(), "like") {
		return p.parseLike(left)
	}
	// comparison operators
	if t := p.peek(); t.kind == tokOp && isComparisonOp(t.text) {
		p.next()
		right, err := p.parseAdditive()
		if err != nil {
			return expr.Expr{}, err
		}
		return buildComparison(t.text, left, right), nil
	}
	return left, nil
}

func (p *exprParser) parseIn(left expr.Expr) (expr.Expr, error) {
	p.next() // consume IN
	if p.peek().kind != tokLParen {
		return expr.Expr{}, fmt.Errorf("expected '(' after IN")
	}
	p.next()
	values := make([]any, 0)
	for {
		e, err := p.parse()
		if err != nil {
			return expr.Expr{}, err
		}
		if e.Kind() != expr.KindLit {
			return expr.Expr{}, fmt.Errorf("IN list supports literal values only")
		}
		values = append(values, e.Value())
		if p.peek().kind == tokComma {
			p.next()
			continue
		}
		break
	}
	if p.peek().kind != tokRParen {
		return expr.Expr{}, fmt.Errorf("expected ')' to close IN list")
	}
	p.next()
	return left.IsIn(expr.Lit(values)), nil
}

func (p *exprParser) parseBetween(left expr.Expr) (expr.Expr, error) {
	p.next() // consume BETWEEN
	lower, err := p.parseAdditive()
	if err != nil {
		return expr.Expr{}, err
	}
	if !keyword(p.peek(), "and") {
		return expr.Expr{}, fmt.Errorf("expected AND in BETWEEN")
	}
	p.next()
	upper, err := p.parseAdditive()
	if err != nil {
		return expr.Expr{}, err
	}
	return left.IsBetween(lower, upper), nil
}

func (p *exprParser) parseLike(left expr.Expr) (expr.Expr, error) {
	p.next() // consume LIKE
	pat, err := p.parseAdditive()
	if err != nil {
		return expr.Expr{}, err
	}
	s, ok := strLiteral(pat)
	if !ok {
		return expr.Expr{}, fmt.Errorf("LIKE requires a string pattern literal")
	}
	return left.StrLike(s), nil
}

func (p *exprParser) parseAdditive() (expr.Expr, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return expr.Expr{}, err
	}
	for {
		t := p.peek()
		if t.kind != tokOp || (t.text != "+" && t.text != "-") {
			break
		}
		p.next()
		right, err := p.parseMultiplicative()
		if err != nil {
			return expr.Expr{}, err
		}
		if t.text == "+" {
			left = left.Add(right)
		} else {
			left = left.Sub(right)
		}
	}
	return left, nil
}

func (p *exprParser) parseMultiplicative() (expr.Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return expr.Expr{}, err
	}
	for {
		t := p.peek()
		if t.kind != tokOp || (t.text != "*" && t.text != "/" && t.text != "%") {
			break
		}
		p.next()
		right, err := p.parseUnary()
		if err != nil {
			return expr.Expr{}, err
		}
		switch t.text {
		case "*":
			left = left.Mul(right)
		case "/":
			left = left.Div(right)
		case "%":
			left = left.Mod(right)
		}
	}
	return left, nil
}

func (p *exprParser) parseUnary() (expr.Expr, error) {
	t := p.peek()
	if t.kind == tokOp && (t.text == "-" || t.text == "+") {
		p.next()
		operand, err := p.parseUnary()
		if err != nil {
			return expr.Expr{}, err
		}
		if t.text == "+" {
			return operand, nil
		}
		// fold negation into numeric literals so types are preserved
		if operand.Kind() == expr.KindLit {
			switch v := operand.Value().(type) {
			case int64:
				return expr.Lit(-v), nil
			case float64:
				return expr.Lit(-v), nil
			}
		}
		return operand.Neg(), nil
	}
	return p.parsePostfix()
}

func (p *exprParser) parsePostfix() (expr.Expr, error) {
	e, err := p.parsePrimary()
	if err != nil {
		return expr.Expr{}, err
	}
	for p.peek().kind == tokOp && p.peek().text == "::" {
		p.next()
		name := p.peek()
		if name.kind != tokIdent {
			return expr.Expr{}, fmt.Errorf("expected type name after '::'")
		}
		p.next()
		dt, err := sqlType(name.text)
		if err != nil {
			return expr.Expr{}, err
		}
		e = e.Cast(dt)
	}
	return e, nil
}

func (p *exprParser) parsePrimary() (expr.Expr, error) {
	t := p.peek()
	switch {
	case t.kind == tokOp && t.text == "*":
		p.next()
		return expr.Col("*"), nil
	case t.kind == tokNumber:
		p.next()
		return numberLiteral(t.text)
	case t.kind == tokString:
		p.next()
		return expr.Lit(t.text), nil
	case t.kind == tokLParen:
		p.next()
		e, err := p.parse()
		if err != nil {
			return expr.Expr{}, err
		}
		if p.peek().kind != tokRParen {
			return expr.Expr{}, fmt.Errorf("expected ')'")
		}
		p.next()
		return e, nil
	case keyword(t, "case"):
		return p.parseCase()
	case keyword(t, "cast"):
		return p.parseCast()
	case keyword(t, "null"):
		p.next()
		return expr.Lit(nil), nil
	case keyword(t, "true"):
		p.next()
		return expr.Lit(true), nil
	case keyword(t, "false"):
		p.next()
		return expr.Lit(false), nil
	case t.kind == tokIdent:
		return p.parseIdentifier()
	default:
		return expr.Expr{}, fmt.Errorf("unexpected token %q in SQL expression", t.text)
	}
}

// parseIdentifier handles a column reference (optionally qualified as a.b) or a
// function call (ident '(' args ')').
func (p *exprParser) parseIdentifier() (expr.Expr, error) {
	name := p.next().text
	// qualified column: a.b
	if p.peek().kind == tokDot {
		p.next()
		part := p.peek()
		if part.kind != tokIdent {
			return expr.Expr{}, fmt.Errorf("expected identifier after '.'")
		}
		p.next()
		return expr.Col(name + "." + part.text), nil
	}
	// function call
	if p.peek().kind == tokLParen {
		p.next()
		args := make([]expr.Expr, 0)
		if p.peek().kind != tokRParen {
			for {
				a, err := p.parse()
				if err != nil {
					return expr.Expr{}, err
				}
				args = append(args, a)
				if p.peek().kind == tokComma {
					p.next()
					continue
				}
				break
			}
		}
		if p.peek().kind != tokRParen {
			return expr.Expr{}, fmt.Errorf("expected ')' to close call to %s", name)
		}
		p.next()
		return buildFunction(name, args)
	}
	return expr.Col(name), nil
}

func (p *exprParser) parseCase() (expr.Expr, error) {
	p.next() // consume CASE
	// optional operand for simple CASE
	var operand *expr.Expr
	if !keyword(p.peek(), "when") {
		op, err := p.parse()
		if err != nil {
			return expr.Expr{}, err
		}
		operand = &op
	}
	type branch struct {
		cond   expr.Expr
		result expr.Expr
	}
	branches := make([]branch, 0)
	for keyword(p.peek(), "when") {
		p.next()
		cond, err := p.parse()
		if err != nil {
			return expr.Expr{}, err
		}
		if !keyword(p.peek(), "then") {
			return expr.Expr{}, fmt.Errorf("expected THEN in CASE expression")
		}
		p.next()
		result, err := p.parse()
		if err != nil {
			return expr.Expr{}, err
		}
		if operand != nil {
			cond = operand.Eq(cond)
		}
		branches = append(branches, branch{cond: cond, result: result})
	}
	if len(branches) == 0 {
		return expr.Expr{}, fmt.Errorf("CASE requires at least one WHEN branch")
	}
	elseExpr := expr.Lit(nil)
	if keyword(p.peek(), "else") {
		p.next()
		e, err := p.parse()
		if err != nil {
			return expr.Expr{}, err
		}
		elseExpr = e
	}
	if !keyword(p.peek(), "end") {
		return expr.Expr{}, fmt.Errorf("expected END to close CASE expression")
	}
	p.next()
	acc := elseExpr
	for i := len(branches) - 1; i >= 0; i-- {
		acc = expr.When(branches[i].cond, branches[i].result, acc)
	}
	return acc, nil
}

func (p *exprParser) parseCast() (expr.Expr, error) {
	p.next() // consume CAST
	if p.peek().kind != tokLParen {
		return expr.Expr{}, fmt.Errorf("expected '(' after CAST")
	}
	p.next()
	inner, err := p.parse()
	if err != nil {
		return expr.Expr{}, err
	}
	if !keyword(p.peek(), "as") {
		return expr.Expr{}, fmt.Errorf("expected AS in CAST expression")
	}
	p.next()
	name := p.peek()
	if name.kind != tokIdent {
		return expr.Expr{}, fmt.Errorf("expected type name in CAST")
	}
	p.next()
	dt, err := sqlType(name.text)
	if err != nil {
		return expr.Expr{}, err
	}
	if p.peek().kind != tokRParen {
		return expr.Expr{}, fmt.Errorf("expected ')' to close CAST")
	}
	p.next()
	return inner.Cast(dt), nil
}

func isComparisonOp(op string) bool {
	switch op {
	case "=", "!=", "<", "<=", ">", ">=":
		return true
	}
	return false
}

func buildComparison(op string, left expr.Expr, right expr.Expr) expr.Expr {
	switch op {
	case "=":
		return left.Eq(right)
	case "!=":
		return left.Ne(right)
	case "<":
		return left.Lt(right)
	case "<=":
		return left.Le(right)
	case ">":
		return left.Gt(right)
	case ">=":
		return left.Ge(right)
	}
	return left
}

func numberLiteral(text string) (expr.Expr, error) {
	if !strings.ContainsAny(text, ".eE") {
		if i, err := strconv.ParseInt(text, 10, 64); err == nil {
			return expr.Lit(i), nil
		}
	}
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return expr.Expr{}, fmt.Errorf("invalid numeric literal %q", text)
	}
	return expr.Lit(f), nil
}

// sqlType maps a SQL type name to a gopolars dtype.
func sqlType(name string) (dtypes.DataType, error) {
	switch strings.ToLower(name) {
	case "int", "integer", "bigint", "smallint", "tinyint":
		return dtypes.Int64, nil
	case "float", "double", "real", "decimal", "numeric":
		return dtypes.Float64, nil
	case "varchar", "text", "string", "char":
		return dtypes.String, nil
	case "boolean", "bool":
		return dtypes.Boolean, nil
	case "date", "datetime", "timestamp":
		return dtypes.Datetime, nil
	default:
		return "", fmt.Errorf("unsupported SQL type: %s", name)
	}
}
