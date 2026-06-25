//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_subqueries.py (py-1.28.1).
//
// gopolars runs SQL through an embedded DuckDB engine (build -tags "duckdb
// duckdb_arrow"). Subquery semantics (FROM-subqueries, IN/NOT IN scalar
// subqueries) match polars' results; the only divergence is the error *message*
// for a multi-column IN-subquery, which we assert by error condition only.
// Results are ORDER BY-wrapped for deterministic comparison.
package sql

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_from_subquery (subset): join two FROM-subquery relations on x = y and
// project a single resulting column. Across INNER/LEFT/FULL variants the matched
// keys are exactly {0,1,2,3}.
func TestSubqueriesFromSubquery(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df1": mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{int64(-1), int64(0), int64(3), int64(1), int64(2), int64(-1)}}),
		"df2": mustFrame(t, frame.SeriesInput{Name: "y", Values: []any{int64(0), int64(1), int64(2), int64(3)}}),
	}

	// INNER, project x
	inner := queryCtx(t, `
		SELECT x FROM (SELECT * FROM df1) AS df1
		INNER JOIN (SELECT * FROM df2) AS df2 ON df1.x = df2.y
		ORDER BY x`, tables)
	eqRow(t, vals(t, inner, "x"), []any{int64(0), int64(1), int64(2), int64(3)}, "from-subq inner x")

	// LEFT, project y, constrained to non-null matches.
	left := queryCtx(t, `
		SELECT y FROM (SELECT * FROM df1) AS df1
		LEFT JOIN (SELECT * FROM df2) AS df2 ON df1.x = df2.y
		WHERE y >= 0
		ORDER BY y`, tables)
	eqRow(t, vals(t, left, "y"), []any{int64(0), int64(1), int64(2), int64(3)}, "from-subq left y")

	// FULL, project df1.* (=x) where the join matched (y >= 0).
	full := queryCtx(t, `
		SELECT df1.x AS x FROM (SELECT * FROM df1) AS df1
		FULL JOIN (SELECT * FROM df2) AS df2 ON df1.x = df2.y
		WHERE y >= 0
		ORDER BY x`, tables)
	eqRow(t, vals(t, full, "x"), []any{int64(0), int64(1), int64(2), int64(3)}, "from-subq full x")

	// `* EXCLUDE y` over a LEFT join keeps only x; restricted to matched rows.
	excl := queryCtx(t, `
		SELECT * EXCLUDE y FROM (SELECT * FROM df1) AS df1
		LEFT JOIN (SELECT * FROM df2) AS df2 ON df1.x = df2.y
		WHERE y >= 0
		ORDER BY x`, tables)
	if got := excl.Columns(); len(got) != 1 || got[0] != "x" {
		t.Fatalf("EXCLUDE y columns = %v, want [x]", got)
	}
	eqRow(t, vals(t, excl, "x"), []any{int64(0), int64(1), int64(2), int64(3)}, "from-subq exclude x")
}

func inSubqFrames(t *testing.T) map[string]polars.DataFrame {
	t.Helper()
	return map[string]polars.DataFrame{
		"df": mustFrame(t,
			frame.SeriesInput{Name: "x", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}},
			frame.SeriesInput{Name: "y", Values: []any{int64(2), int64(3), int64(4), int64(5), int64(6), int64(7)}},
		),
		"df_other": mustFrame(t,
			frame.SeriesInput{Name: "w", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}},
			frame.SeriesInput{Name: "z", Values: []any{int64(2), int64(3), int64(4), int64(5), int64(6), int64(7)}},
		),
		"df_chars": mustFrame(t,
			frame.SeriesInput{Name: "one", Values: []any{"a", "b", "c", "d", "e", "f"}},
			frame.SeriesInput{Name: "two", Values: []any{"b", "c", "d", "e", "f", "g"}},
		),
	}
}

// test_in_subquery (subset): WHERE ... IN (SELECT ...) with single & double
// subqueries, expression operands, NOT IN, and a string-column variant.
func TestSubqueriesInSubquery(t *testing.T) {
	tables := inSubqFrames(t)

	// x IN (SELECT y FROM df) -> x in {2..7} ∩ {1..6} = {2,3,4,5,6}
	same := queryCtx(t, `
		SELECT df.x AS x FROM df WHERE x IN (SELECT y FROM df) ORDER BY x`, tables)
	eqRow(t, vals(t, same, "x"), []any{int64(2), int64(3), int64(4), int64(5), int64(6)}, "in same")

	// x IN (SELECT y) AND y IN (SELECT w) -> {2,3,4,5}
	double := queryCtx(t, `
		SELECT df.x AS x FROM df
		WHERE x IN (SELECT y FROM df)
		AND y IN (SELECT w FROM df_other)
		ORDER BY x`, tables)
	eqRow(t, vals(t, double, "x"), []any{int64(2), int64(3), int64(4), int64(5)}, "in double")

	// expression operands: x+1 IN (SELECT y) AND y IN (SELECT w-1) -> {1,2,3,4}
	expr := queryCtx(t, `
		SELECT df.x AS x FROM df
		WHERE x+1 IN (SELECT y FROM df)
		AND y IN (SELECT w-1 FROM df_other)
		ORDER BY x`, tables)
	eqRow(t, vals(t, expr, "x"), []any{int64(1), int64(2), int64(3), int64(4)}, "in expr")

	// NOT IN with shifted subqueries -> {3,4}
	notIn := queryCtx(t, `
		SELECT df.x AS x FROM df
		WHERE x NOT IN (SELECT y-5 FROM df)
		AND y NOT IN (SELECT w+5 FROM df_other)
		ORDER BY x`, tables)
	eqRow(t, vals(t, notIn, "x"), []any{int64(3), int64(4)}, "not in")

	// string column: one IN (SELECT two) -> {b,c,d,e,f}
	chars := queryCtx(t, `
		SELECT df_chars.one AS one FROM df_chars
		WHERE one IN (SELECT two FROM df_chars) ORDER BY one`, tables)
	eqRow(t, vals(t, chars, "one"), []any{"b", "c", "d", "e", "f"}, "in chars")
}

// test_in_subquery (error case): an IN-subquery returning >1 column must error.
// DISCREPANCY: polars raises SQLSyntaxError ("SQL subquery returns more than one
// column"); DuckDB raises a Binder Error ("Subquery returns 2 columns - expected
// 1"). Only the error condition is asserted (messages differ).
func TestSubqueriesInSubqueryMultiColumnErrors(t *testing.T) {
	df := mustFrame(t,
		frame.SeriesInput{Name: "one", Values: []any{"a", "b", "c"}},
		frame.SeriesInput{Name: "two", Values: []any{"b", "c", "d"}},
	)
	_, err := df.SQL(context.Background(), "SELECT one FROM self WHERE one IN (SELECT one, two FROM self)")
	if err == nil {
		t.Fatalf("expected error for multi-column IN-subquery")
	}
}
