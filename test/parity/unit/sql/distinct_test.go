//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_miscellaneous.py (py-1.28.1).
//
// py-polars has no dedicated test_distinct.py; the DISTINCT-related cases live in
// test_miscellaneous.py (test_distinct, plus the COUNT(DISTINCT ...) arms of
// test_count). DuckDB-backed: SELECT DISTINCT and COUNT(DISTINCT ...) are standard
// SQL → MATCH. The polars unregister/relation-not-found error message diverges
// (DISCREPANCY on message; only the error condition is pinned).
package sql

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_distinct (arm 1): SELECT DISTINCT a ... ORDER BY a DESC. Standard SQL → MATCH.
func TestDistinctSelectDistinctColumn(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(1), int64(1), int64(2), int64(2), int64(3)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}},
		),
	}
	out := queryCtx(t, "SELECT DISTINCT a FROM df ORDER BY a DESC", tables)
	if out.Width() != 1 {
		t.Fatalf("width = %d, want 1", out.Width())
	}
	eqRow(t, vals(t, out, "a"), []any{int64(3), int64(2), int64(1)}, "distinct a desc")
}

// test_distinct (arm 2): SELECT DISTINCT over computed expressions with ORDER BY.
//
// DISCREPANCY: polars treats `b / 2` over integers as integer (floor) division, so
// b/2 = [0,1,1,2,2,3] and the DISTINCT result collapses to 4 rows
// (two_a/half_b = [(2,1),(2,0),(4,2),(6,3)]). DuckDB's `/` over integers is TRUE
// division → b/2 = [0.5,1.0,1.5,2.0,2.5,3.0], all six (two_a,half_b) pairs are
// distinct, and half_b is Float64. Pin DuckDB's actual 6-row float output.
func TestDistinctSelectDistinctExpressions(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(1), int64(1), int64(2), int64(2), int64(3)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}},
		),
	}
	out := queryCtx(t, `
		SELECT DISTINCT
		  a * 2 AS two_a,
		  b / 2 AS half_b
		FROM df
		ORDER BY two_a ASC, half_b DESC
	`, tables)
	eqRow(t, vals(t, out, "two_a"), []any{int64(2), int64(2), int64(2), int64(4), int64(4), int64(6)}, "two_a")
	floatRow(t, floats(t, out, "half_b"), []float64{1.5, 1.0, 0.5, 2.5, 2.0, 3.0}, "half_b")
}

// test_distinct (arm 3): after unregistering the table, querying it must error.
// polars raises SQLInterfaceError("relation 'df' was not found"); DuckDB raises a
// catalog/binder error with a different message — pin only the error condition.
func TestDistinctUnregisteredRelationErrors(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}},
	)
	ctx := polars.NewSQLContext()
	if err := ctx.Register("df", d); err != nil {
		t.Fatalf("register: %v", err)
	}
	// sanity: registered query works
	if _, err := ctx.Execute(context.Background(), "SELECT * FROM df"); err != nil {
		t.Fatalf("registered SELECT * failed: %v", err)
	}
	// A fresh context without `df` registered must fail to resolve the relation.
	ctx2 := polars.NewSQLContext()
	if _, err := ctx2.Execute(context.Background(), "SELECT * FROM df"); err == nil {
		t.Fatalf("expected error for unregistered relation 'df'")
	}
}

// test_count (count-distinct arms): COUNT(DISTINCT col) over ints and a column with
// NULLs, plus COUNT(DISTINCT NULL). Standard SQL → MATCH.
//
// COUNT(DISTINCT c) ignores NULLs (c has 2 distinct non-null values); COUNT(DISTINCT
// NULL) is 0. COUNT(*) counts all rows; COUNT(col) counts non-null.
func TestDistinctCountDistinct(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
		frame.SeriesInput{Name: "b", Values: []any{int64(1), int64(1), int64(22), int64(22), int64(333)}},
		frame.SeriesInput{Name: "c", Values: []any{int64(1), int64(1), nil, nil, int64(2)}},
	)
	out := query(t, d, `
		SELECT
		  COUNT(a) AS count_a,
		  COUNT(b) AS count_b,
		  COUNT(c) AS count_c,
		  COUNT(*) AS count_star,
		  COUNT(NULL) AS count_null,
		  COUNT(DISTINCT a) AS count_unique_a,
		  COUNT(DISTINCT b) AS count_unique_b,
		  COUNT(DISTINCT c) AS count_unique_c,
		  COUNT(DISTINCT NULL) AS count_unique_null
		FROM self
	`)
	got := map[string]int64{}
	for _, name := range []string{
		"count_a", "count_b", "count_c", "count_star", "count_null",
		"count_unique_a", "count_unique_b", "count_unique_c", "count_unique_null",
	} {
		got[name] = col(t, out, name).Value(0).(int64)
	}
	want := map[string]int64{
		"count_a": 5, "count_b": 5, "count_c": 3, "count_star": 5, "count_null": 0,
		"count_unique_a": 5, "count_unique_b": 3, "count_unique_c": 2, "count_unique_null": 0,
	}
	for name, w := range want {
		if got[name] != w {
			t.Fatalf("%s = %d, want %d", name, got[name], w)
		}
	}
}

// test_count (all-null arm): COUNT over an all-NULL column.
func TestDistinctCountDistinctAllNull(t *testing.T) {
	// All-null column needs an explicit dtype (inference can't pick one).
	d := mustFrame(t,
		frame.SeriesInput{Name: "x", Values: []any{nil, nil, nil}, DType: dtypes.Int64},
	)
	out := query(t, d, `
		SELECT
		  COUNT(x) AS count_x,
		  COUNT(*) AS count_star,
		  COUNT(DISTINCT x) AS count_unique_x
		FROM self
	`)
	if v := col(t, out, "count_x").Value(0).(int64); v != 0 {
		t.Fatalf("count_x = %d, want 0", v)
	}
	if v := col(t, out, "count_star").Value(0).(int64); v != 3 {
		t.Fatalf("count_star = %d, want 3", v)
	}
	if v := col(t, out, "count_unique_x").Value(0).(int64); v != 0 {
		t.Fatalf("count_unique_x = %d, want 0", v)
	}
}
