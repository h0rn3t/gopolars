//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_set_ops.py (py-1.28.1).
//
// DuckDB-backed. Set ops are wrapped in `ORDER BY` for deterministic comparison
// (the py tests sort rows). polars-only error-message assertions are omitted;
// notably `EXCEPT ALL`/`INTERSECT ALL` are unsupported in polars but DuckDB
// supports them (see TestSetOpsExceptAllSupported — a DISCREPANCY).
package sql

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func queryCtx(t *testing.T, q string, tables map[string]polars.DataFrame) polars.DataFrame {
	t.Helper()
	ctx := polars.NewSQLContext()
	for n, d := range tables {
		if err := ctx.Register(n, d); err != nil {
			t.Fatalf("register %q: %v", n, err)
		}
	}
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

func setOpFrames(t *testing.T) map[string]polars.DataFrame {
	return map[string]polars.DataFrame{
		"df1": mustFrame(t,
			frame.SeriesInput{Name: "x", Values: []any{int64(1), int64(9), int64(1), int64(1)}},
			frame.SeriesInput{Name: "y", Values: []any{int64(2), int64(3), int64(4), int64(4)}},
			frame.SeriesInput{Name: "z", Values: []any{int64(5), int64(5), int64(5), int64(5)}},
		),
		"df2": mustFrame(t,
			frame.SeriesInput{Name: "x", Values: []any{int64(1), int64(9), int64(1)}},
			frame.SeriesInput{Name: "y", Values: []any{int64(2), nil, int64(4)}},
			frame.SeriesInput{Name: "z", Values: []any{int64(7), int64(6), int64(5)}},
		),
	}
}

// test_except_intersect (basic): EXCEPT and INTERSECT over two frames.
func TestSetOpsExceptIntersect(t *testing.T) {
	tables := setOpFrames(t)
	e := queryCtx(t, "SELECT * FROM (SELECT x, y, z FROM df1 EXCEPT SELECT * FROM df2) ORDER BY x", tables)
	eqRow(t, vals(t, e, "x"), []any{int64(1), int64(9)}, "except x")
	eqRow(t, vals(t, e, "y"), []any{int64(2), int64(3)}, "except y")

	i := queryCtx(t, "SELECT * FROM (SELECT * FROM df1 INTERSECT SELECT x, y, z FROM df2) ORDER BY x", tables)
	eqRow(t, vals(t, i, "x"), []any{int64(1)}, "intersect x")
	eqRow(t, vals(t, i, "y"), []any{int64(4)}, "intersect y")
}

// test_union (subset): UNION (distinct) and UNION ALL.
func TestSetOpsUnion(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"frame1": mustFrame(t,
			frame.SeriesInput{Name: "c1", Values: []any{int64(1), int64(2)}},
			frame.SeriesInput{Name: "c2", Values: []any{"zz", "yy"}},
		),
		"frame2": mustFrame(t,
			frame.SeriesInput{Name: "c1", Values: []any{int64(2), int64(3)}},
			frame.SeriesInput{Name: "c2", Values: []any{"yy", "xx"}},
		),
	}
	u := queryCtx(t, "SELECT * FROM (SELECT * FROM frame1 UNION SELECT * FROM frame2) ORDER BY c1", tables)
	eqRow(t, vals(t, u, "c1"), []any{int64(1), int64(2), int64(3)}, "union c1")
	eqRow(t, vals(t, u, "c2"), []any{"zz", "yy", "xx"}, "union c2")

	ua := queryCtx(t, "SELECT * FROM (SELECT * FROM frame1 UNION ALL SELECT * FROM frame2) ORDER BY c1, c2", tables)
	eqRow(t, vals(t, ua, "c1"), []any{int64(1), int64(2), int64(2), int64(3)}, "union all c1")
}

// test_except_intersect_by_name — DISCREPANCY: polars supports `EXCEPT BY NAME`
// (align columns by name); DuckDB rejects the `BY NAME` modifier for EXCEPT/
// INTERSECT (it only accepts `UNION [ALL] BY NAME`). Pin DuckDB's rejection.
func TestSetOpsExceptByNameUnsupported(t *testing.T) {
	tables := setOpFrames(t)
	ctx := polars.NewSQLContext()
	for n, d := range tables {
		_ = ctx.Register(n, d)
	}
	if _, err := ctx.Execute(context.Background(), "SELECT x, y, z FROM df1 EXCEPT BY NAME SELECT * FROM df2"); err == nil {
		t.Fatalf("expected DuckDB to reject EXCEPT BY NAME")
	}
}

// DISCREPANCY: polars rejects `EXCEPT ALL` ("not supported"); DuckDB supports it
// (multiset difference). Pin DuckDB's behavior.
func TestSetOpsExceptAllSupported(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df1": mustFrame(t, frame.SeriesInput{Name: "n", Values: []any{int64(1), int64(1), int64(1), int64(2), int64(2), int64(2), int64(3)}}),
		"df2": mustFrame(t, frame.SeriesInput{Name: "n", Values: []any{int64(1), int64(1), int64(2), int64(2)}}),
	}
	out := queryCtx(t, "SELECT * FROM (SELECT * FROM df1 EXCEPT ALL SELECT * FROM df2) ORDER BY n", tables)
	// 1:3-2=1, 2:3-2=1, 3:1-0=1 → one of each.
	eqRow(t, vals(t, out, "n"), []any{int64(1), int64(2), int64(3)}, "except all n")
}

// test_except_intersect_errors: mismatched column counts must error (DuckDB's
// message differs from polars — only the error condition is asserted).
func TestSetOpsColumnCountMismatchErrors(t *testing.T) {
	tables := setOpFrames(t)
	ctx := polars.NewSQLContext()
	for n, d := range tables {
		_ = ctx.Register(n, d)
	}
	if _, err := ctx.Execute(context.Background(), "SELECT x FROM df1 EXCEPT SELECT x, y FROM df2"); err == nil {
		t.Fatalf("expected error for mismatched column counts")
	}
}
