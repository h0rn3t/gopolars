package sql

import (
	"strings"
	"testing"
)

// Parse is the SELECT-only entry point (DataFrame.SQL / LazyFrame.SQL); it
// keeps rejecting every non-SELECT statement.
func TestParseRejectsNonSelectStatements(t *testing.T) {
	cases := []string{
		"CREATE TABLE x AS SELECT * FROM t",
		"DROP TABLE t",
		"ALTER TABLE t ADD COLUMN c INT",
		"INSERT INTO t VALUES (1)",
		"UPDATE t SET a = 1",
		"DELETE FROM t",
		"EXPLAIN SELECT * FROM t",
		"SHOW TABLES",
	}
	for _, q := range cases {
		_, err := Parse(q)
		if err == nil {
			t.Fatalf("Parse(%q) returned nil error, want unsupported error", q)
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("Parse(%q) error = %v, want it to mention 'unsupported'", q, err)
		}
	}
}

// ParseStatement keeps DML and ALTER unsupported (matching Polars SQL).
func TestParseStatementRejectsDML(t *testing.T) {
	cases := []string{
		"INSERT INTO t VALUES (1)",
		"UPDATE t SET a = 1",
		"DELETE FROM t",
		"ALTER TABLE t ADD COLUMN c INT",
		"DESCRIBE t",
		"USE db",
	}
	for _, q := range cases {
		_, err := ParseStatement(q)
		if err == nil {
			t.Fatalf("ParseStatement(%q) returned nil error, want unsupported error", q)
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("ParseStatement(%q) error = %v, want it to mention 'unsupported'", q, err)
		}
	}
}

func TestParseRejectsUnknownTableFunction(t *testing.T) {
	if _, err := Parse("SELECT * FROM read_xlsx('x.xlsx')"); err == nil {
		t.Fatalf("expected error for unknown table function in FROM")
	}
}
