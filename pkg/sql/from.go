package sql

import (
	"fmt"
	"strings"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

// TableRef is a table reference in a FROM/JOIN clause: a named table, a
// parenthesized subquery, or a file-reading table function, with an optional
// alias.
type TableRef struct {
	Name     string
	Alias    string
	Subquery *ParsedQuery
	Fn       *TableFn
}

// TableFn is a file-reading table function reference in table position, e.g.
// read_csv('data.csv'). Name is the canonical lowercase function name. The
// reference is resolved to a DataFrame by the executing layer before binding.
type TableFn struct {
	Name string
	Path string
}

// tableFnNames are the supported FROM-clause table functions.
var tableFnNames = map[string]bool{
	"read_csv":     true,
	"read_parquet": true,
	"read_json":    true,
	"read_ipc":     true,
}

// key returns the identifier used to qualify columns of this table reference:
// the alias if present, otherwise the table name.
func (t TableRef) key() string {
	if t.Alias != "" {
		return t.Alias
	}
	return t.Name
}

// JoinClause is a single join applied to the FROM clause. Type is one of
// "inner", "left", "right", "full", "cross". On is the join predicate (nil for
// cross joins).
type JoinClause struct {
	Type  string
	Table TableRef
	On    *expr.Expr
}

// FromClause is the parsed FROM clause: a primary table reference plus an
// ordered list of joins (left-deep).
type FromClause struct {
	Primary TableRef
	Joins   []JoinClause
}

// parseFromClause parses the text between FROM and the first trailing clause
// keyword (WHERE/GROUP BY/...). It supports table aliases, comma cross joins,
// and INNER/LEFT/RIGHT/FULL/CROSS joins with ON predicates.
func parseFromClause(seg string) (FromClause, error) {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return FromClause{}, fmt.Errorf("missing FROM clause")
	}
	words, err := splitFromWords(seg)
	if err != nil {
		return FromClause{}, err
	}
	if len(words) == 0 {
		return FromClause{}, fmt.Errorf("missing table in FROM clause")
	}
	primary, pos, err := readTableRef(words, 0)
	if err != nil {
		return FromClause{}, err
	}
	fc := FromClause{Primary: primary}
	for pos < len(words) {
		if words[pos] == "," {
			tr, np, err := readTableRef(words, pos+1)
			if err != nil {
				return FromClause{}, err
			}
			pos = np
			fc.Joins = append(fc.Joins, JoinClause{Type: "cross", Table: tr})
			continue
		}
		jtype, consumed := matchJoinType(words, pos)
		if jtype == "" {
			return FromClause{}, fmt.Errorf("cannot parse FROM clause near %q", words[pos])
		}
		pos += consumed
		tr, np, err := readTableRef(words, pos)
		if err != nil {
			return FromClause{}, err
		}
		pos = np
		jc := JoinClause{Type: jtype, Table: tr}
		if jtype != "cross" {
			if pos >= len(words) || !strings.EqualFold(words[pos], "on") {
				return FromClause{}, fmt.Errorf("expected ON for %s join", jtype)
			}
			pos++
			start := pos
			for pos < len(words) && words[pos] != "," {
				if jt, _ := matchJoinType(words, pos); jt != "" {
					break
				}
				pos++
			}
			if start == pos {
				return FromClause{}, fmt.Errorf("missing ON predicate in %s join", jtype)
			}
			onExpr, err := parseExpression(strings.Join(words[start:pos], " "))
			if err != nil {
				return FromClause{}, err
			}
			jc.On = &onExpr
		}
		fc.Joins = append(fc.Joins, jc)
	}
	return fc, nil
}

// readTableRef reads a table reference (optionally aliased) starting at pos and
// returns the new position.
func readTableRef(words []string, pos int) (TableRef, int, error) {
	if pos >= len(words) {
		return TableRef{}, pos, fmt.Errorf("expected table reference")
	}
	w := words[pos]
	if w == "," {
		return TableRef{}, pos, fmt.Errorf("expected table name")
	}
	var tr TableRef
	if strings.HasPrefix(w, "(") {
		inner := strings.TrimSpace(w[1 : len(w)-1])
		sub, err := parseSelectQuery(inner)
		if err != nil {
			return TableRef{}, pos, err
		}
		tr.Subquery = &sub
	} else {
		tr.Name = w
	}
	pos++
	// table function: an identifier immediately followed by a parenthesized
	// argument list, e.g. read_csv('data.csv').
	if tr.Subquery == nil && pos < len(words) && strings.HasPrefix(words[pos], "(") {
		fnName := strings.ToLower(tr.Name)
		if !tableFnNames[fnName] {
			return TableRef{}, pos, fmt.Errorf("unknown table function %s", tr.Name)
		}
		path, err := parseTableFnArg(fnName, words[pos])
		if err != nil {
			return TableRef{}, pos, err
		}
		tr.Fn = &TableFn{Name: fnName, Path: path}
		pos++
	}
	// optional alias: "AS name" or a bare identifier that is not a keyword
	if pos < len(words) {
		nxt := words[pos]
		if strings.EqualFold(nxt, "as") {
			pos++
			if pos >= len(words) {
				return TableRef{}, pos, fmt.Errorf("expected alias after AS")
			}
			tr.Alias = words[pos]
			pos++
		} else if nxt != "," && !strings.EqualFold(nxt, "on") && !strings.HasPrefix(nxt, "(") {
			if jt, _ := matchJoinType(words, pos); jt == "" {
				tr.Alias = nxt
				pos++
			}
		}
	}
	if tr.Subquery != nil && tr.Name == "" {
		if tr.Alias != "" {
			tr.Name = tr.Alias
		} else {
			tr.Name = "_subquery"
		}
	}
	return tr, pos, nil
}

// parseTableFnArg validates a table function's argument list at parse time:
// exactly one single-quoted string literal (the file path).
func parseTableFnArg(fnName string, group string) (string, error) {
	inner := strings.TrimSpace(group[1 : len(group)-1])
	toks, err := lex(inner)
	if err != nil {
		return "", fmt.Errorf("invalid %s argument: %w", fnName, err)
	}
	for _, t := range toks {
		if t.kind == tokComma {
			return "", fmt.Errorf("%s expects exactly one argument", fnName)
		}
	}
	if len(toks) != 2 || toks[0].kind != tokString {
		return "", fmt.Errorf("%s requires a string literal path", fnName)
	}
	return toks[0].text, nil
}

// matchJoinType recognizes a join keyword sequence at pos and returns its
// canonical type and the number of words it spans (including JOIN).
func matchJoinType(words []string, pos int) (string, int) {
	eq := func(i int, s string) bool { return i < len(words) && strings.EqualFold(words[i], s) }
	switch {
	case eq(pos, "join"):
		return "inner", 1
	case eq(pos, "inner") && eq(pos+1, "join"):
		return "inner", 2
	case eq(pos, "cross") && eq(pos+1, "join"):
		return "cross", 2
	case eq(pos, "left") && eq(pos+1, "outer") && eq(pos+2, "join"):
		return "left", 3
	case eq(pos, "left") && eq(pos+1, "join"):
		return "left", 2
	case eq(pos, "right") && eq(pos+1, "outer") && eq(pos+2, "join"):
		return "right", 3
	case eq(pos, "right") && eq(pos+1, "join"):
		return "right", 2
	case eq(pos, "full") && eq(pos+1, "outer") && eq(pos+2, "join"):
		return "full", 3
	case eq(pos, "full") && eq(pos+1, "join"):
		return "full", 2
	}
	return "", 0
}

// splitFromWords splits a FROM segment into words, keeping a parenthesized group
// (subquery or grouped predicate) as a single token and commas as their own
// token.
func splitFromWords(seg string) ([]string, error) {
	words := make([]string, 0)
	runes := []rune(seg)
	i, n := 0, len(runes)
	for i < n {
		c := runes[i]
		switch c {
		case ' ', '\t', '\n', '\r':
			i++
		case ',':
			words = append(words, ",")
			i++
		case '(':
			depth := 0
			start := i
			for i < n {
				if runes[i] == '(' {
					depth++
				} else if runes[i] == ')' {
					depth--
					if depth == 0 {
						i++
						break
					}
				}
				i++
			}
			if depth != 0 {
				return nil, fmt.Errorf("unbalanced parenthesis in FROM clause")
			}
			words = append(words, string(runes[start:i]))
		default:
			start := i
			for i < n {
				r := runes[i]
				if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == ',' || r == '(' {
					break
				}
				i++
			}
			words = append(words, string(runes[start:i]))
		}
	}
	return words, nil
}
