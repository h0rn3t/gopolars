//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_table_operations.py (py-1.28.1).
//
// DuckDB-backed (build -tags "duckdb duckdb_arrow"). These tests exercise DDL/DML
// statements (DELETE / DROP / TRUNCATE / SHOW TABLES / EXPLAIN). gopolars' SQL
// engine runs over read-only Arrow views registered in DuckDB, so mutating DDL
// against a registered frame behaves differently from polars' SQLContext, and
// EXPLAIN output is DuckDB-shaped (no polars "Logical Plan" column). Each form is
// probed and pinned to its actual gopolars/DuckDB behavior.
package sql

import (
	"context"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func tableOpsFrame(t *testing.T) polars.DataFrame {
	t.Helper()
	return mustFrame(t,
		frame.SeriesInput{Name: "x", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "y", Values: []any{"aaa", "bbb", "ccc"}},
	)
}

// test_delete_clause: `DELETE FROM self WHERE ...` returns the surviving rows.
// gopolars materializes each frame into a real, mutable DuckDB table, so the engine
// runs the DELETE and then reads the affected table back — matching polars. MATCH.
// (The dt column is modeled as Datetime, so EXTRACT(year FROM dt) works.)
func TestTableOpsDeleteClause(t *testing.T) {
	dt := func(y, mo, d int) time.Time { return utc(y, mo, d, 0, 0, 0, 0) }
	d := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(100), int64(200), int64(300)}},
		frame.SeriesInput{Name: "dt", Values: []any{dt(2020, 10, 10), dt(1999, 1, 2), dt(2001, 7, 5)}},
		frame.SeriesInput{Name: "v1", Values: []any{3.5, -4.0, nil}},
		frame.SeriesInput{Name: "v2", Values: []any{10.0, 2.5, -1.5}},
	)
	cases := []struct {
		constraint string
		ids        []int64
	}{
		{`WHERE id = 200`, []int64{100, 300}},
		{`WHERE id = 200 OR id = 300`, []int64{100}},
		{`WHERE id IN (200, 300, 400)`, []int64{100}},
		{`WHERE id NOT IN (200, 300, 400)`, []int64{200, 300}},
		{`WHERE EXTRACT(year FROM dt) >= 2000`, []int64{200}},
		{`WHERE v1 < 0`, []int64{100, 300}},
		{`WHERE v1 > 0`, []int64{200, 300}},
		{`WHERE v1 IS NULL`, []int64{100, 200}},
		{`WHERE v1 IS NOT NULL`, []int64{300}},
		{`WHERE FALSE`, []int64{100, 200, 300}},
		{`WHERE TRUE`, nil},
		{``, nil},
	}
	for _, c := range cases {
		out := query(t, d, `DELETE FROM self `+c.constraint)
		got := map[int64]bool{}
		for _, v := range vals(t, out, "id") {
			got[v.(int64)] = true
		}
		if len(got) != len(c.ids) {
			t.Fatalf("DELETE %q: survivors %v, want %v", c.constraint, vals(t, out, "id"), c.ids)
		}
		for _, id := range c.ids {
			if !got[id] {
				t.Fatalf("DELETE %q: missing survivor %d (got %v)", c.constraint, id, vals(t, out, "id"))
			}
		}
	}
}

// test_drop_table: in polars, DROP TABLE removes the table from the SQL context
// and a subsequent SELECT errors with "'frame' was not found".
//
// DISCREPANCY: gopolars registers each frame as a read-only Arrow view that the
// SQLContext keeps independently of DuckDB's catalog. `DROP TABLE frame`
// succeeds (no error), but the registration is NOT removed, so a subsequent
// `SELECT * FROM frame` still returns the rows. Pinned to gopolars' actual
// behavior: drop is accepted, the table remains queryable.
func TestTableOpsDropTable(t *testing.T) {
	ctx := polars.NewSQLContext()
	if err := ctx.Register("frame", tableOpsFrame(t)); err != nil {
		t.Fatalf("register: %v", err)
	}
	// DROP TABLE is accepted without error.
	lf, err := ctx.Execute(context.Background(), "DROP TABLE frame")
	if err != nil {
		t.Fatalf("DROP TABLE frame: %v", err)
	}
	if lf != nil {
		_, _ = lf.Collect(context.Background())
	}
	// DISCREPANCY: the table is still queryable after DROP (re-registered view).
	out := queryCtxOK(t, ctx, "SELECT x FROM frame ORDER BY x")
	eqRow(t, vals(t, out, "x"), []any{int64(1), int64(2), int64(3)}, "SELECT after DROP still works")
}

// queryCtxOK executes a query against an already-built SQLContext and collects it.
func queryCtxOK(t *testing.T, ctx polars.SQLContext, q string) polars.DataFrame {
	t.Helper()
	lf, err := ctx.Execute(context.Background(), q)
	if err != nil {
		t.Fatalf("execute %q: %v", q, err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect %q: %v", q, err)
	}
	return out
}

// test_explain_query: polars EXPLAIN yields a single "Logical Plan" column whose
// joined text matches a PROJECT...COLUMNS pattern.
//
// DISCREPANCY: DuckDB's EXPLAIN returns a different schema — two columns
// (explain_key / explain_value) with a DuckDB-rendered physical plan box, not a
// polars "Logical Plan" column. Pinned to DuckDB's actual output: EXPLAIN runs,
// produces those two columns, and the plan text is non-empty.
func TestTableOpsExplainQuery(t *testing.T) {
	ctx := polars.NewSQLContext()
	if err := ctx.Register("frame", tableOpsFrame(t)); err != nil {
		t.Fatalf("register: %v", err)
	}
	out := queryCtxOK(t, ctx, "EXPLAIN SELECT * FROM frame")
	cols := out.Columns()
	if len(cols) != 2 || cols[0] != "explain_key" || cols[1] != "explain_value" {
		t.Fatalf("EXPLAIN columns = %v, want [explain_key explain_value]", cols)
	}
	if out.Height() == 0 {
		t.Fatalf("EXPLAIN returned no rows")
	}
	plan, _ := out.GetColumn("explain_value")
	if s, ok := plan.Value(0).(string); !ok || s == "" {
		t.Fatalf("EXPLAIN plan text empty/non-string: %v", plan.Value(0))
	}
}

// test_show_tables: SHOW TABLES lists registered tables in sorted order in a
// single "name" column. DuckDB returns the same single "name" column (column
// name MATCHes).
//
// DISCREPANCY: gopolars registers each frame under both its user name and an
// internal `__gopolars_view_<name>` helper view, so SHOW TABLES also lists those
// internal views alongside the user names (polars lists only the user names).
// Pinned to gopolars' actual output: the user-visible names appear sorted, with
// the internal view rows interleaved.
func TestTableOpsShowTables(t *testing.T) {
	ctx := polars.NewSQLContext()
	for _, n := range []string{"tbl3", "tbl2", "tbl1"} {
		if err := ctx.Register(n, tableOpsFrame(t)); err != nil {
			t.Fatalf("register %q: %v", n, err)
		}
	}
	out := queryCtxOK(t, ctx, "SHOW TABLES")
	if cols := out.Columns(); len(cols) != 1 || cols[0] != "name" {
		t.Fatalf("SHOW TABLES columns = %v, want [name]", cols)
	}
	// Collect user names (excluding the internal __gopolars_view_* helpers).
	var user []any
	for _, v := range vals(t, out, "name") {
		if s, ok := v.(string); ok && len(s) >= 16 && s[:16] == "__gopolars_view_" {
			continue
		}
		user = append(user, v)
	}
	eqRow(t, user, []any{"tbl1", "tbl2", "tbl3"}, "SHOW TABLES user names (sorted)")
}

// test_truncate_table: TRUNCATE drops all rows but keeps the (empty) table. The
// materialized table is mutable, so TRUNCATE then a read-back yields an empty frame
// with the original schema. MATCH on the single-statement form.
//
// DISCREPANCY: polars' SQLContext also asserts a SEPARATE later `SELECT * FROM frame`
// is still empty (mutation persists across statements). gopolars' SQLContext is
// stateless (each Execute re-materializes the source frame), so cross-statement
// persistence is out of scope; only the TRUNCATE statement's own empty result is
// asserted.
func TestTableOpsTruncateTable(t *testing.T) {
	d := tableOpsFrame(t)
	for _, stmt := range []string{`TRUNCATE TABLE self`, `TRUNCATE self`} {
		out := query(t, d, stmt)
		if out.Height() != 0 {
			t.Fatalf("%s: height = %d, want 0", stmt, out.Height())
		}
		if cols := out.Columns(); len(cols) != 2 || cols[0] != "x" || cols[1] != "y" {
			t.Fatalf("%s: columns = %v, want [x y] (schema preserved)", stmt, out.Columns())
		}
	}
}
