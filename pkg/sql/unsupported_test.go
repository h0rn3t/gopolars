package sql

import (
	"strings"
	"testing"
)

func TestParseRejectsUnsupportedStatements(t *testing.T) {
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

func TestParseRejectsTableFunction(t *testing.T) {
	if _, err := Parse("SELECT * FROM read_csv('x.csv')"); err == nil {
		t.Fatalf("expected error for table function in FROM")
	}
}
