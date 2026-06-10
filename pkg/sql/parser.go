package sql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

type ParsedQuery struct {
	Raw      string
	Table    string
	From     FromClause
	Select   []SelectItem
	Distinct bool
	Where    *expr.Expr
	GroupBy  []string
	OrderBy  []OrderItem
	Having   *expr.Expr
	Limit    *int
	Offset   *int
	HasWhere bool
	CTEName  string
	CTE      *ParsedQuery
	Subquery *ParsedQuery
	SetOp    string
	SetRight *ParsedQuery
}

type SelectItem struct {
	Expr   expr.Expr
	Window *WindowSpec
}

type OrderItem struct {
	Column string
	Desc   bool
}

type WindowSpec struct {
	Func        string
	Target      string
	Alias       string
	PartitionBy []string
	OrderBy     []OrderItem
	// Offset and Default parameterize LAG/LEAD.
	Offset  int
	Default any
}

// unsupportedStatements maps a leading keyword of an out-of-scope statement to a
// descriptive error message.
var unsupportedStatements = map[string]string{
	"create":   "CREATE statements are not supported",
	"drop":     "DROP statements are not supported",
	"alter":    "ALTER statements are not supported",
	"insert":   "INSERT statements are not supported",
	"update":   "UPDATE statements are not supported",
	"delete":   "DELETE statements are not supported",
	"truncate": "TRUNCATE statements are not supported",
	"explain":  "EXPLAIN statements are not supported",
	"show":     "SHOW statements are not supported",
	"describe": "DESCRIBE statements are not supported",
	"use":      "USE statements are not supported",
}

// firstKeyword returns the leading identifier word of a (lowercased) query.
func firstKeyword(lower string) string {
	i := 0
	for i < len(lower) && (lower[i] == ' ' || lower[i] == '\t' || lower[i] == '\n' || lower[i] == '\r' || lower[i] == '(') {
		i++
	}
	start := i
	for i < len(lower) && lower[i] >= 'a' && lower[i] <= 'z' {
		i++
	}
	return lower[start:i]
}

func Parse(query string) (ParsedQuery, error) {
	raw := strings.TrimSpace(query)
	if raw == "" {
		return ParsedQuery{}, fmt.Errorf("query is empty")
	}
	lower := strings.ToLower(raw)
	if first := firstKeyword(lower); first != "" {
		if msg, ok := unsupportedStatements[first]; ok {
			return ParsedQuery{}, fmt.Errorf("unsupported SQL: %s", msg)
		}
	}
	if strings.HasPrefix(lower, "with ") {
		return parseWithQuery(raw)
	}
	if left, op, right, ok := splitTopLevelSetOp(raw); ok {
		lq, err := parseSelectQuery(left)
		if err != nil {
			return ParsedQuery{}, err
		}
		rq, err := parseSelectQuery(right)
		if err != nil {
			return ParsedQuery{}, err
		}
		lq.SetOp = op
		lq.SetRight = &rq
		return lq, nil
	}
	return parseSelectQuery(raw)
}

func parseWithQuery(raw string) (ParsedQuery, error) {
	lower := strings.ToLower(raw)
	asIdx := strings.Index(lower, " as ")
	if asIdx == -1 {
		return ParsedQuery{}, fmt.Errorf("invalid WITH clause")
	}
	name := strings.TrimSpace(raw[len("with "):asIdx])
	rest := strings.TrimSpace(raw[asIdx+len(" as "):])
	if !strings.HasPrefix(rest, "(") {
		return ParsedQuery{}, fmt.Errorf("invalid WITH subquery")
	}
	depth := 0
	end := -1
	for i, ch := range rest {
		if ch == '(' {
			depth++
		}
		if ch == ')' {
			depth--
			if depth == 0 {
				end = i
				break
			}
		}
	}
	if end == -1 {
		return ParsedQuery{}, fmt.Errorf("unterminated WITH subquery")
	}
	innerRaw := strings.TrimSpace(rest[1:end])
	outerRaw := strings.TrimSpace(rest[end+1:])
	inner, err := parseSelectQuery(innerRaw)
	if err != nil {
		return ParsedQuery{}, err
	}
	outer, err := parseSelectQuery(outerRaw)
	if err != nil {
		return ParsedQuery{}, err
	}
	outer.CTEName = name
	outer.CTE = &inner
	return outer, nil
}

func parseSelectQuery(raw string) (ParsedQuery, error) {
	normalized := strings.Join(strings.Fields(raw), " ")
	lower := strings.ToLower(normalized)
	if !strings.HasPrefix(lower, "select ") {
		return ParsedQuery{}, fmt.Errorf("only select queries are supported")
	}
	// Paren- and quote-aware: a FROM inside EXTRACT(... FROM ...) or a string
	// literal is not the clause boundary.
	fromIdx := topLevelIndexFold(normalized, " from ")
	if fromIdx == -1 {
		return ParsedQuery{}, fmt.Errorf("missing FROM clause")
	}
	selectPart := strings.TrimSpace(normalized[len("select "):fromIdx])
	rest := strings.TrimSpace(normalized[fromIdx+len(" from "):])

	distinct := false
	if strings.HasPrefix(strings.ToLower(selectPart), "distinct ") {
		distinct = true
		selectPart = strings.TrimSpace(selectPart[len("distinct "):])
	}

	fromSeg, afterFrom := splitFromAndClauses(rest)
	from, err := parseFromClause(fromSeg)
	if err != nil {
		return ParsedQuery{}, err
	}
	q := ParsedQuery{
		Raw:      strings.TrimSpace(raw),
		Table:    from.Primary.Name,
		From:     from,
		Select:   make([]SelectItem, 0),
		Distinct: distinct,
		Subquery: from.Primary.Subquery,
	}
	items, err := parseSelectList(selectPart)
	if err != nil {
		return ParsedQuery{}, err
	}
	q.Select = items

	remaining := strings.TrimSpace(afterFrom)
	for remaining != "" {
		l := strings.ToLower(remaining)
		switch {
		case strings.HasPrefix(l, "where "):
			clause, next := splitClause(remaining[len("where "):])
			w, err := parseCondition(clause)
			if err != nil {
				return ParsedQuery{}, err
			}
			q.Where = &w
			q.HasWhere = true
			remaining = strings.TrimSpace(next)
		case strings.HasPrefix(l, "group by "):
			clause, next := splitClause(remaining[len("group by "):])
			parts := strings.Split(clause, ",")
			for _, p := range parts {
				col := strings.TrimSpace(p)
				if col != "" {
					q.GroupBy = append(q.GroupBy, col)
				}
			}
			remaining = strings.TrimSpace(next)
		case strings.HasPrefix(l, "having "):
			clause, next := splitClause(remaining[len("having "):])
			h, err := parseCondition(clause)
			if err != nil {
				return ParsedQuery{}, err
			}
			q.Having = &h
			remaining = strings.TrimSpace(next)
		case strings.HasPrefix(l, "order by "):
			clause, next := splitClause(remaining[len("order by "):])
			orderItems := strings.Split(clause, ",")
			for _, item := range orderItems {
				tokens := strings.Fields(strings.TrimSpace(item))
				if len(tokens) == 0 {
					continue
				}
				order := OrderItem{Column: tokens[0]}
				if len(tokens) > 1 && strings.EqualFold(tokens[1], "desc") {
					order.Desc = true
				}
				q.OrderBy = append(q.OrderBy, order)
			}
			remaining = strings.TrimSpace(next)
		case strings.HasPrefix(l, "limit "):
			token, next := readFirstToken(remaining[len("limit "):])
			n, err := strconv.Atoi(strings.TrimSpace(token))
			if err != nil {
				return ParsedQuery{}, fmt.Errorf("invalid LIMIT value")
			}
			q.Limit = &n
			remaining = strings.TrimSpace(next)
		case strings.HasPrefix(l, "offset "):
			token, next := readFirstToken(remaining[len("offset "):])
			m, err := strconv.Atoi(strings.TrimSpace(token))
			if err != nil {
				return ParsedQuery{}, fmt.Errorf("invalid OFFSET value")
			}
			q.Offset = &m
			remaining = strings.TrimSpace(next)
		default:
			return ParsedQuery{}, fmt.Errorf("cannot parse SQL near: %s", remaining)
		}
	}
	return q, nil
}

func parseSelectList(input string) ([]SelectItem, error) {
	parts := splitTopLevel(input, ',')
	out := make([]SelectItem, 0, len(parts))
	for _, p := range parts {
		item := strings.TrimSpace(p)
		if item == "" {
			continue
		}
		ex, err := parseSelectExpr(item)
		if err != nil {
			return nil, err
		}
		out = append(out, ex)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty select list")
	}
	return out, nil
}

func parseSelectExpr(item string) (SelectItem, error) {
	trimmed := strings.TrimSpace(item)
	// Window functions are handled by the dedicated window parser. The alias (if
	// any) is the top-level " AS " after the OVER(...) clause.
	if topLevelIndexFold(trimmed, " over ") >= 0 {
		rawExpr := trimmed
		alias := ""
		if idx := topLevelIndexFold(trimmed, " as "); idx >= 0 {
			rawExpr = strings.TrimSpace(trimmed[:idx])
			alias = strings.TrimSpace(trimmed[idx+len(" as "):])
		}
		w, err := parseWindowExpr(rawExpr, alias)
		if err != nil {
			return SelectItem{}, err
		}
		return SelectItem{Window: &w, Expr: expr.Col(w.Alias)}, nil
	}
	// Detect a top-level alias (" AS name"); an AS inside parentheses (e.g.
	// CAST(x AS INT)) is at depth > 0 and is left for the expression parser.
	rawExpr := trimmed
	alias := ""
	if idx := topLevelIndexFold(trimmed, " as "); idx >= 0 {
		rawExpr = strings.TrimSpace(trimmed[:idx])
		alias = strings.TrimSpace(trimmed[idx+len(" as "):])
	}
	e, err := parseExpression(rawExpr)
	if err != nil {
		return SelectItem{}, err
	}
	if alias != "" {
		e = e.Alias(alias)
	}
	return SelectItem{Expr: e}, nil
}

func parseWindowExpr(raw string, alias string) (WindowSpec, error) {
	lower := strings.ToLower(raw)
	overIdx := strings.Index(lower, " over ")
	if overIdx == -1 {
		return WindowSpec{}, fmt.Errorf("invalid window expression")
	}
	fnPart := strings.TrimSpace(raw[:overIdx])
	overPart := strings.TrimSpace(raw[overIdx+len(" over "):])
	if !strings.HasPrefix(overPart, "(") || !strings.HasSuffix(overPart, ")") {
		return WindowSpec{}, fmt.Errorf("invalid OVER clause")
	}
	inside := strings.TrimSpace(overPart[1 : len(overPart)-1])
	lp := strings.Index(fnPart, "(")
	if lp == -1 || !strings.HasSuffix(fnPart, ")") {
		return WindowSpec{}, fmt.Errorf("invalid window expression")
	}
	fnName := strings.ToLower(strings.TrimSpace(fnPart[:lp]))
	argsRaw := strings.TrimSpace(fnPart[lp+1 : len(fnPart)-1])
	args := make([]string, 0, 3)
	if argsRaw != "" {
		for _, a := range splitTopLevel(argsRaw, ',') {
			args = append(args, strings.TrimSpace(a))
		}
	}
	wantWindowArgs := func(n int) error {
		if len(args) != n {
			return fmt.Errorf("window function %s expects %d argument(s), got %d", fnName, n, len(args))
		}
		return nil
	}
	spec := WindowSpec{}
	switch fnName {
	case "sum", "min", "max", "count", "first_value", "last_value":
		if err := wantWindowArgs(1); err != nil {
			return WindowSpec{}, err
		}
		spec.Func = fnName
		spec.Target = args[0]
	case "mean", "avg", "rolling_mean":
		if err := wantWindowArgs(1); err != nil {
			return WindowSpec{}, err
		}
		spec.Func = "mean"
		spec.Target = args[0]
	case "row_number", "rank", "dense_rank":
		if err := wantWindowArgs(0); err != nil {
			return WindowSpec{}, err
		}
		spec.Func = fnName
	case "lag", "lead":
		if len(args) < 1 || len(args) > 3 {
			return WindowSpec{}, fmt.Errorf("window function %s expects between 1 and 3 arguments, got %d", fnName, len(args))
		}
		spec.Func = fnName
		spec.Target = args[0]
		spec.Offset = 1
		if len(args) >= 2 {
			off, err := strconv.Atoi(args[1])
			if err != nil {
				return WindowSpec{}, fmt.Errorf("window function %s requires an integer offset", fnName)
			}
			spec.Offset = off
		}
		if len(args) == 3 {
			def, err := parseExpression(args[2])
			if err != nil {
				return WindowSpec{}, err
			}
			if def.Kind() != expr.KindLit {
				return WindowSpec{}, fmt.Errorf("window function %s requires a literal default", fnName)
			}
			spec.Default = def.Value()
		}
	default:
		return WindowSpec{}, fmt.Errorf("unsupported window function: %s", fnName)
	}
	if alias == "" {
		alias = fmt.Sprintf("%s_window", spec.Func)
	}
	spec.Alias = alias
	spec.PartitionBy, spec.OrderBy = parseWindowOver(inside)
	return spec, nil
}

func parseWindowOver(inside string) ([]string, []OrderItem) {
	lower := strings.ToLower(inside)
	part := []string{}
	order := []OrderItem{}
	if idx := strings.Index(lower, "partition by "); idx >= 0 {
		segment := inside[idx+len("partition by "):]
		if oi := strings.Index(strings.ToLower(segment), " order by "); oi >= 0 {
			segment = segment[:oi]
		}
		for _, p := range splitTopLevel(segment, ',') {
			c := strings.TrimSpace(p)
			if c != "" {
				part = append(part, c)
			}
		}
	}
	if idx := strings.Index(lower, "order by "); idx >= 0 {
		segment := inside[idx+len("order by "):]
		for _, item := range splitTopLevel(segment, ',') {
			tokens := strings.Fields(strings.TrimSpace(item))
			if len(tokens) == 0 {
				continue
			}
			o := OrderItem{Column: tokens[0]}
			if len(tokens) > 1 && strings.EqualFold(tokens[1], "desc") {
				o.Desc = true
			}
			order = append(order, o)
		}
	}
	return part, order
}

func splitTopLevel(input string, sep rune) []string {
	parts := make([]string, 0)
	depth := 0
	start := 0
	for i, ch := range input {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
		} else if ch == sep && depth == 0 {
			parts = append(parts, strings.TrimSpace(input[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, strings.TrimSpace(input[start:]))
	return parts
}

func splitTopLevelSetOp(raw string) (string, string, string, bool) {
	normalized := strings.Join(strings.Fields(raw), " ")
	lower := strings.ToLower(normalized)
	depth := 0
	for i := 0; i < len(lower); i++ {
		ch := lower[i]
		if ch == '(' {
			depth++
			continue
		}
		if ch == ')' {
			depth--
			continue
		}
		if depth != 0 {
			continue
		}
		for _, op := range []string{" union ", " intersect ", " except "} {
			if i+len(op) <= len(lower) && lower[i:i+len(op)] == op {
				left := strings.TrimSpace(normalized[:i])
				right := strings.TrimSpace(normalized[i+len(op):])
				if left == "" || right == "" {
					return "", "", "", false
				}
				return left, strings.TrimSpace(op), right, true
			}
		}
	}
	return "", "", "", false
}

// parseCondition parses a SQL predicate (WHERE/HAVING/ON) into an expression.
func parseCondition(input string) (expr.Expr, error) {
	return parseExpression(input)
}

// topLevelIndexFold returns the byte index of sub (already lowercase) within s,
// matched case-insensitively, ignoring matches inside single-quoted string
// literals or inside parentheses. Returns -1 if not found at the top level.
func topLevelIndexFold(s string, sub string) int {
	lower := strings.ToLower(s)
	depth := 0
	inSingle := false
	for i := 0; i+len(sub) <= len(lower); i++ {
		c := lower[i]
		switch c {
		case '\'':
			inSingle = !inSingle
		case '(':
			if !inSingle {
				depth++
			}
		case ')':
			if !inSingle {
				depth--
			}
		}
		if !inSingle && depth == 0 && lower[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func readFirstToken(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return "", ""
	}
	first := fields[0]
	idx := strings.Index(s, first)
	return first, strings.TrimSpace(s[idx+len(first):])
}

func splitClause(s string) (string, string) {
	l := strings.ToLower(s)
	markers := []string{" where ", " group by ", " having ", " order by ", " limit ", " offset "}
	pos := -1
	for _, marker := range markers {
		if idx := strings.Index(l, marker); idx >= 0 {
			if pos == -1 || idx < pos {
				pos = idx
			}
		}
	}
	if pos == -1 {
		return strings.TrimSpace(s), ""
	}
	return strings.TrimSpace(s[:pos]), strings.TrimSpace(s[pos+1:])
}

// splitFromAndClauses splits the text following FROM into the FROM segment (the
// table list with joins) and the remaining trailing clauses, splitting at the
// first top-level clause keyword.
func splitFromAndClauses(rest string) (string, string) {
	markers := []string{" where ", " group by ", " having ", " order by ", " limit ", " offset "}
	best := -1
	for _, m := range markers {
		if idx := topLevelIndexFold(rest, m); idx >= 0 {
			if best == -1 || idx < best {
				best = idx
			}
		}
	}
	if best == -1 {
		return strings.TrimSpace(rest), ""
	}
	return strings.TrimSpace(rest[:best]), strings.TrimSpace(rest[best:])
}
