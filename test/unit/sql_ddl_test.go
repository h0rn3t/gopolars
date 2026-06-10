package unit

import (
	"context"
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func ddlContext(t *testing.T) polars.SQLContext {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "name", Values: []any{"ann", "bob", "cid", "dee"}},
			{Name: "age", Values: []any{int64(12), int64(20), int64(35), int64(17)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe: %v", err)
	}
	ctx := polars.NewSQLContext()
	if err := ctx.Register("people", df); err != nil {
		t.Fatalf("register: %v", err)
	}
	return ctx
}

func mustExec(t *testing.T, ctx polars.SQLContext, query string) polars.DataFrame {
	t.Helper()
	lf, err := ctx.Execute(context.Background(), query)
	if err != nil {
		t.Fatalf("execute %q: %v", query, err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect %q: %v", query, err)
	}
	return out
}

func TestSQLCreateTableAs(t *testing.T) {
	ctx := ddlContext(t)
	resp := mustExec(t, ctx, "CREATE TABLE adults AS SELECT name FROM people WHERE age >= 18")
	col, ok := resp.Series("response")
	if !ok || col.Len() != 1 || col.Value(0) != "CREATE TABLE" {
		t.Fatalf("response frame: got %v", resp.ToDicts())
	}
	out := mustExec(t, ctx, "SELECT name FROM adults ORDER BY name")
	if out.Height() != 2 {
		t.Fatalf("adults height: got %d, want 2", out.Height())
	}
	names, _ := out.Series("name")
	if names.Value(0) != "bob" || names.Value(1) != "cid" {
		t.Fatalf("adults rows: got %v", out.ToDicts())
	}
	found := false
	for _, n := range ctx.Tables() {
		if n == "adults" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Tables() = %v, want it to include adults", ctx.Tables())
	}
}

func TestSQLCreateTableReplacesExisting(t *testing.T) {
	ctx := ddlContext(t)
	mustExec(t, ctx, "CREATE TABLE t2 AS SELECT name FROM people")
	mustExec(t, ctx, "CREATE TABLE t2 AS SELECT name FROM people WHERE age >= 18")
	out := mustExec(t, ctx, "SELECT name FROM t2")
	if out.Height() != 2 {
		t.Fatalf("t2 height after replace: got %d, want 2", out.Height())
	}
}

func TestSQLCreateTableInvalidQueryLeavesRegistryUntouched(t *testing.T) {
	ctx := ddlContext(t)
	if _, err := ctx.Execute(context.Background(), "CREATE TABLE bad AS SELECT name FROM missing"); err == nil {
		t.Fatalf("expected error for unregistered source table")
	}
	for _, n := range ctx.Tables() {
		if n == "bad" {
			t.Fatalf("registry contains %q after failed CREATE", n)
		}
	}
}

func TestSQLDropTable(t *testing.T) {
	ctx := ddlContext(t)
	mustExec(t, ctx, "DROP TABLE people")
	if len(ctx.Tables()) != 0 {
		t.Fatalf("Tables() = %v, want empty after drop", ctx.Tables())
	}
	if _, err := ctx.Execute(context.Background(), "SELECT * FROM people"); err == nil {
		t.Fatalf("expected query on dropped table to fail")
	}
	if _, err := ctx.Execute(context.Background(), "DROP TABLE missing"); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("drop missing: err = %v, want error naming the table", err)
	}
	mustExec(t, ctx, "DROP TABLE IF EXISTS missing")
}

func TestSQLTruncateTable(t *testing.T) {
	ctx := ddlContext(t)
	mustExec(t, ctx, "TRUNCATE TABLE people")
	out := mustExec(t, ctx, "SELECT * FROM people")
	if out.Height() != 0 {
		t.Fatalf("height after truncate: got %d, want 0", out.Height())
	}
	cols := out.Columns()
	if len(cols) != 2 || cols[0] != "name" || cols[1] != "age" {
		t.Fatalf("columns after truncate: got %v, want [name age]", cols)
	}
	if _, err := ctx.Execute(context.Background(), "TRUNCATE TABLE missing"); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("truncate missing: err = %v, want error naming the table", err)
	}
}

func TestSQLShowTables(t *testing.T) {
	ctx := ddlContext(t)
	mustExec(t, ctx, "CREATE TABLE a_first AS SELECT name FROM people")
	out := mustExec(t, ctx, "SHOW TABLES")
	names, ok := out.Series("name")
	if !ok || names.Len() != 2 {
		t.Fatalf("SHOW TABLES: got %v", out.ToDicts())
	}
	if names.Value(0) != "a_first" || names.Value(1) != "people" {
		t.Fatalf("SHOW TABLES order: got %v, want sorted [a_first people]", out.ToDicts())
	}
	empty := polars.NewSQLContext()
	lf, err := empty.Execute(context.Background(), "SHOW TABLES")
	if err != nil {
		t.Fatalf("show tables on empty context: %v", err)
	}
	res, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if res.Height() != 0 {
		t.Fatalf("empty context SHOW TABLES height: got %d, want 0", res.Height())
	}
}

func TestSQLExplain(t *testing.T) {
	ctx := ddlContext(t)
	out := mustExec(t, ctx, "EXPLAIN SELECT name FROM people WHERE age > 1")
	plan, ok := out.Series("plan")
	if !ok || plan.Len() != 1 {
		t.Fatalf("EXPLAIN result: got %v", out.ToDicts())
	}
	text, _ := plan.Value(0).(string)
	if !strings.Contains(text, "filter") || !strings.Contains(text, "select") {
		t.Fatalf("plan text %q, want it to mention filter and select", text)
	}
	if _, err := ctx.Execute(context.Background(), "EXPLAIN SELECT nope FROM missing"); err == nil {
		t.Fatalf("expected EXPLAIN of invalid query to fail")
	}
}

func TestSQLDDLIsContextLocal(t *testing.T) {
	ctxA := ddlContext(t)
	ctxB := polars.NewSQLContext()
	mustExec(t, ctxA, "CREATE TABLE local_t AS SELECT name FROM people")
	if _, err := ctxB.Execute(context.Background(), "SELECT * FROM local_t"); err == nil {
		t.Fatalf("expected local_t to be unknown in context B")
	}
	if len(ctxB.Tables()) != 0 {
		t.Fatalf("context B tables = %v, want empty", ctxB.Tables())
	}
}

// Full DDL round trip: CREATE -> SELECT -> TRUNCATE -> DROP -> SHOW TABLES.
func TestSQLDDLRoundTrip(t *testing.T) {
	ctx := ddlContext(t)
	mustExec(t, ctx, "CREATE TABLE adults AS SELECT name, age FROM people WHERE age >= 18")
	out := mustExec(t, ctx, "SELECT COUNT(*) AS n FROM adults")
	if n, _ := out.Series("n"); n.Value(0) != int64(2) {
		t.Fatalf("created table count: got %v, want 2", out.ToDicts())
	}
	mustExec(t, ctx, "TRUNCATE TABLE adults")
	out = mustExec(t, ctx, "SELECT * FROM adults")
	if out.Height() != 0 || len(out.Columns()) != 2 {
		t.Fatalf("after truncate: got %v rows, columns %v", out.Height(), out.Columns())
	}
	mustExec(t, ctx, "DROP TABLE adults")
	out = mustExec(t, ctx, "SHOW TABLES")
	names, _ := out.Series("name")
	if names.Len() != 1 || names.Value(0) != "people" {
		t.Fatalf("after drop: got %v, want [people]", out.ToDicts())
	}
}

func TestSQLInsertRemainsUnsupported(t *testing.T) {
	ctx := ddlContext(t)
	if _, err := ctx.Execute(context.Background(), "INSERT INTO people VALUES (1)"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("INSERT: err = %v, want unsupported error", err)
	}
}
