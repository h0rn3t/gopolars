package sql

import (
	"fmt"
	"strings"
)

// Statement is a parsed top-level SQL statement. SELECT queries keep flowing
// through Parse/Bind/Plan; the DDL statements are executed directly against a
// SQLContext's table registry.
type Statement interface {
	isStatement()
}

// SelectStatement wraps a parsed SELECT (including WITH and set operations).
type SelectStatement struct {
	Query ParsedQuery
}

// CreateTableAs is CREATE TABLE <name> AS <select>.
type CreateTableAs struct {
	Name  string
	Query ParsedQuery
}

// DropTable is DROP TABLE [IF EXISTS] <name>.
type DropTable struct {
	Name     string
	IfExists bool
}

// TruncateTable is TRUNCATE TABLE <name>.
type TruncateTable struct {
	Name string
}

// ShowTables is SHOW TABLES.
type ShowTables struct{}

// ExplainSelect is EXPLAIN <select>: plan the query without executing it.
type ExplainSelect struct {
	Query ParsedQuery
}

func (SelectStatement) isStatement() {}
func (CreateTableAs) isStatement()   {}
func (DropTable) isStatement()       {}
func (TruncateTable) isStatement()   {}
func (ShowTables) isStatement()      {}
func (ExplainSelect) isStatement()   {}

// ParseStatement parses a top-level SQL statement: SELECT (delegated to Parse)
// or one of the supported session DDL statements. DML (INSERT/UPDATE/DELETE)
// and ALTER stay unsupported, matching Polars SQL.
func ParseStatement(query string) (Statement, error) {
	raw := strings.TrimSpace(query)
	raw = strings.TrimSpace(strings.TrimSuffix(raw, ";"))
	if raw == "" {
		return nil, fmt.Errorf("query is empty")
	}
	switch firstKeyword(strings.ToLower(raw)) {
	case "create":
		return parseCreateTableAs(raw)
	case "drop":
		return parseDropTable(raw)
	case "truncate":
		return parseTruncateTable(raw)
	case "show":
		return parseShowTables(raw)
	case "explain":
		rest, _ := takeKeyword(raw, "explain")
		q, err := Parse(rest)
		if err != nil {
			return nil, err
		}
		return ExplainSelect{Query: q}, nil
	default:
		q, err := Parse(raw)
		if err != nil {
			return nil, err
		}
		return SelectStatement{Query: q}, nil
	}
}

func parseCreateTableAs(raw string) (Statement, error) {
	rest, _ := takeKeyword(raw, "create")
	rest, ok := takeKeyword(rest, "table")
	if !ok {
		return nil, fmt.Errorf("unsupported SQL: only CREATE TABLE <name> AS <select> is supported")
	}
	name, rest, err := takeIdent(rest)
	if err != nil {
		return nil, fmt.Errorf("invalid CREATE TABLE: %w", err)
	}
	rest, ok = takeKeyword(rest, "as")
	if !ok {
		return nil, fmt.Errorf("unsupported SQL: only CREATE TABLE <name> AS <select> is supported (column definitions are not)")
	}
	q, err := Parse(rest)
	if err != nil {
		return nil, err
	}
	return CreateTableAs{Name: name, Query: q}, nil
}

func parseDropTable(raw string) (Statement, error) {
	rest, _ := takeKeyword(raw, "drop")
	rest, ok := takeKeyword(rest, "table")
	if !ok {
		return nil, fmt.Errorf("unsupported SQL: only DROP TABLE [IF EXISTS] <name> is supported")
	}
	ifExists := false
	if r, ok := takeKeyword(rest, "if"); ok {
		r, ok = takeKeyword(r, "exists")
		if !ok {
			return nil, fmt.Errorf("invalid DROP TABLE: expected EXISTS after IF")
		}
		ifExists = true
		rest = r
	}
	name, rest, err := takeIdent(rest)
	if err != nil {
		return nil, fmt.Errorf("invalid DROP TABLE: %w", err)
	}
	if rest != "" {
		return nil, fmt.Errorf("invalid DROP TABLE: unexpected %q", rest)
	}
	return DropTable{Name: name, IfExists: ifExists}, nil
}

func parseTruncateTable(raw string) (Statement, error) {
	rest, _ := takeKeyword(raw, "truncate")
	rest, ok := takeKeyword(rest, "table")
	if !ok {
		return nil, fmt.Errorf("unsupported SQL: only TRUNCATE TABLE <name> is supported")
	}
	name, rest, err := takeIdent(rest)
	if err != nil {
		return nil, fmt.Errorf("invalid TRUNCATE TABLE: %w", err)
	}
	if rest != "" {
		return nil, fmt.Errorf("invalid TRUNCATE TABLE: unexpected %q", rest)
	}
	return TruncateTable{Name: name}, nil
}

func parseShowTables(raw string) (Statement, error) {
	rest, _ := takeKeyword(raw, "show")
	rest, ok := takeKeyword(rest, "tables")
	if !ok || rest != "" {
		return nil, fmt.Errorf("unsupported SQL: only SHOW TABLES is supported")
	}
	return ShowTables{}, nil
}

// takeKeyword consumes a leading keyword (case-insensitive, at a word
// boundary) and returns the trimmed remainder.
func takeKeyword(s string, kw string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) < len(kw) || !strings.EqualFold(s[:len(kw)], kw) {
		return s, false
	}
	rest := s[len(kw):]
	if rest != "" && isIdentPart(rune(rest[0])) {
		return s, false
	}
	return strings.TrimSpace(rest), true
}

// takeIdent consumes a leading identifier (bare or double-quoted) and returns
// it with the trimmed remainder.
func takeIdent(s string) (string, string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", fmt.Errorf("expected identifier")
	}
	if s[0] == '"' {
		runes := []rune(s)
		name, next, err := lexQuotedIdent(runes, 0)
		if err != nil {
			return "", "", err
		}
		return name, strings.TrimSpace(string(runes[next:])), nil
	}
	if !isIdentStart(rune(s[0])) {
		return "", "", fmt.Errorf("expected identifier near %q", s)
	}
	i := 1
	for i < len(s) && isIdentPart(rune(s[i])) {
		i++
	}
	return s[:i], strings.TrimSpace(s[i:]), nil
}
