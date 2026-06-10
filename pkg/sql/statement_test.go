package sql

import (
	"strings"
	"testing"
)

func TestParseStatementSelect(t *testing.T) {
	stmt, err := ParseStatement("SELECT a FROM t WHERE a > 1")
	if err != nil {
		t.Fatalf("ParseStatement: %v", err)
	}
	sel, ok := stmt.(SelectStatement)
	if !ok {
		t.Fatalf("statement type = %T, want SelectStatement", stmt)
	}
	if sel.Query.Table != "t" {
		t.Fatalf("table = %q, want t", sel.Query.Table)
	}
}

func TestParseStatementCreateTableAs(t *testing.T) {
	stmt, err := ParseStatement("CREATE TABLE adults AS SELECT a FROM people WHERE a >= 18")
	if err != nil {
		t.Fatalf("ParseStatement: %v", err)
	}
	ct, ok := stmt.(CreateTableAs)
	if !ok {
		t.Fatalf("statement type = %T, want CreateTableAs", stmt)
	}
	if ct.Name != "adults" {
		t.Fatalf("name = %q, want adults", ct.Name)
	}
	if ct.Query.Table != "people" {
		t.Fatalf("inner table = %q, want people", ct.Query.Table)
	}
}

func TestParseStatementCreateRejectsColumnList(t *testing.T) {
	_, err := ParseStatement("CREATE TABLE t (a INT)")
	if err == nil || !strings.Contains(err.Error(), "AS <select>") {
		t.Fatalf("err = %v, want error pointing at CREATE TABLE ... AS <select>", err)
	}
	if _, err := ParseStatement("CREATE VIEW v AS SELECT * FROM t"); err == nil {
		t.Fatalf("expected CREATE VIEW to be rejected")
	}
}

func TestParseStatementDropTable(t *testing.T) {
	stmt, err := ParseStatement("DROP TABLE t1")
	if err != nil {
		t.Fatalf("ParseStatement: %v", err)
	}
	dt, ok := stmt.(DropTable)
	if !ok || dt.Name != "t1" || dt.IfExists {
		t.Fatalf("got %#v, want DropTable{Name:t1, IfExists:false}", stmt)
	}
	stmt, err = ParseStatement("drop table if exists t2")
	if err != nil {
		t.Fatalf("ParseStatement: %v", err)
	}
	dt, ok = stmt.(DropTable)
	if !ok || dt.Name != "t2" || !dt.IfExists {
		t.Fatalf("got %#v, want DropTable{Name:t2, IfExists:true}", stmt)
	}
}

func TestParseStatementTruncateAndShow(t *testing.T) {
	stmt, err := ParseStatement("TRUNCATE TABLE t")
	if err != nil {
		t.Fatalf("ParseStatement: %v", err)
	}
	if tt, ok := stmt.(TruncateTable); !ok || tt.Name != "t" {
		t.Fatalf("got %#v, want TruncateTable{Name:t}", stmt)
	}
	stmt, err = ParseStatement("SHOW TABLES;")
	if err != nil {
		t.Fatalf("ParseStatement: %v", err)
	}
	if _, ok := stmt.(ShowTables); !ok {
		t.Fatalf("got %#v, want ShowTables", stmt)
	}
	if _, err := ParseStatement("SHOW DATABASES"); err == nil {
		t.Fatalf("expected SHOW DATABASES to be rejected")
	}
}

func TestParseStatementExplain(t *testing.T) {
	stmt, err := ParseStatement("EXPLAIN SELECT a FROM t WHERE a > 1")
	if err != nil {
		t.Fatalf("ParseStatement: %v", err)
	}
	ex, ok := stmt.(ExplainSelect)
	if !ok || ex.Query.Table != "t" {
		t.Fatalf("got %#v, want ExplainSelect over table t", stmt)
	}
	if _, err := ParseStatement("EXPLAIN INSERT INTO t VALUES (1)"); err == nil {
		t.Fatalf("expected EXPLAIN over non-SELECT to be rejected")
	}
}
