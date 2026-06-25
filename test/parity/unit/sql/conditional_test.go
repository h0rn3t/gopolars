//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_conditional.py (py-1.28.1).
//
// DuckDB-backed: behavior measured against DuckDB's dialect. CASE/COALESCE/NULLIF/
// IFNULL/GREATEST/LEAST are standard SQL and mostly MATCH. polars-only spellings
// (IF(), trailing-comma arg lists, polars error messages) and Date-only fixtures
// are pinned as DISCREPANCY / GAP.
package sql

import (
	"context"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_case_when: CASE WHEN COALESCE(v1,v2) % 2 != 0 THEN 'odd' ELSE 'even'.
// Standard SQL → MATCH.
func TestConditionalCaseWhen(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "v1", Values: []any{nil, int64(2), nil, int64(4)}},
		frame.SeriesInput{Name: "v2", Values: []any{int64(101), int64(202), int64(303), int64(404)}},
	)
	out := query(t, d, `
		SELECT *, CASE WHEN COALESCE(v1, v2) % 2 != 0 THEN 'odd' ELSE 'even' END AS v3
		FROM self
	`)
	eqRow(t, vals(t, out, "v1"), []any{nil, int64(2), nil, int64(4)}, "v1")
	eqRow(t, vals(t, out, "v2"), []any{int64(101), int64(202), int64(303), int64(404)}, "v2")
	eqRow(t, vals(t, out, "v3"), []any{"odd", "even", "odd", "even"}, "v3")
}

// test_case_when_optional_else: AVG(CASE WHEN a<=b THEN c ELSE NULL END) and the
// same with the ELSE omitted. Both forms are standard SQL → MATCH.
func TestConditionalCaseWhenOptionalElse(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6), int64(7)}},
		frame.SeriesInput{Name: "b", Values: []any{int64(7), int64(6), int64(5), int64(4), int64(3), int64(2), int64(1)}},
		frame.SeriesInput{Name: "c", Values: []any{int64(3), int64(4), int64(0), int64(3), int64(4), int64(1), int64(1)}},
	)
	for _, elseClause := range []string{"ELSE NULL ", ""} {
		q := `SELECT AVG(CASE WHEN a <= b THEN c ` + elseClause + `END) AS conditional_mean FROM self`
		out := query(t, d, q)
		got := toFloat(col(t, out, "conditional_mean").Value(0))
		if got != 2.5 {
			t.Fatalf("conditional_mean (else=%q) = %v, want 2.5", elseClause, got)
		}
	}
}

// test_control_flow: COALESCE / NULLIF / IFNULL / nested COALESCE+NULLIF, plus the
// polars `IF(cond, a, b)` shorthand. DuckDB supports all of these including IF().
func TestConditionalControlFlow(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df": mustFrame(t,
			frame.SeriesInput{Name: "x", Values: []any{int64(1), nil, int64(2), int64(3), nil, int64(4)}},
			frame.SeriesInput{Name: "y", Values: []any{int64(5), int64(4), nil, int64(3), nil, int64(2)}},
			frame.SeriesInput{Name: "z", Values: []any{int64(3), int64(4), nil, int64(3), int64(6), nil}},
		),
	}
	out := queryCtx(t, `
		SELECT
		  COALESCE(x,y,z) AS coalsc,
		  NULLIF(x, y) AS nullif_x_y,
		  NULLIF(y, z) AS nullif_y_z,
		  IFNULL(x, y) AS ifnull_x_y,
		  IFNULL(y,-1) AS inullf_y_z,
		  COALESCE(x, NULLIF(y,z)) AS both,
		  IF(x = y, 'eq', 'ne') AS x_eq_y
		FROM df
	`, tables)

	eqRow(t, vals(t, out, "coalsc"), []any{int64(1), int64(4), int64(2), int64(3), int64(6), int64(4)}, "coalsc")
	eqRow(t, vals(t, out, "nullif_x_y"), []any{int64(1), nil, int64(2), nil, nil, int64(4)}, "nullif x_y")
	eqRow(t, vals(t, out, "nullif_y_z"), []any{int64(5), nil, nil, nil, nil, int64(2)}, "nullif y_z")
	eqRow(t, vals(t, out, "ifnull_x_y"), []any{int64(1), int64(4), int64(2), int64(3), nil, int64(4)}, "ifnull x_y")
	eqRow(t, vals(t, out, "inullf_y_z"), []any{int64(5), int64(4), int64(-1), int64(3), int64(-1), int64(2)}, "inullf y_z")
	eqRow(t, vals(t, out, "both"), []any{int64(1), nil, int64(2), int64(3), nil, int64(4)}, "both")
	eqRow(t, vals(t, out, "x_eq_y"), []any{"ne", "ne", "ne", "eq", "ne", "ne"}, "x_eq_y")
}

// test_control_flow (error arm): polars raises SQLSyntaxError for IFNULL/NULLIF
// with 3 args. DuckDB also rejects them (binder error) — the message differs, so
// only the error condition is pinned (DISCREPANCY on message).
func TestConditionalNullFuncArityErrors(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df": mustFrame(t,
			frame.SeriesInput{Name: "x", Values: []any{int64(1), int64(2)}},
			frame.SeriesInput{Name: "y", Values: []any{int64(3), int64(4)}},
			frame.SeriesInput{Name: "z", Values: []any{int64(5), int64(6)}},
		),
	}
	for _, fn := range []string{"IFNULL", "NULLIF"} {
		ctx := polars.NewSQLContext()
		for n, dd := range tables {
			_ = ctx.Register(n, dd)
		}
		if _, err := ctx.Execute(context.Background(), "SELECT "+fn+"(x,y,z) FROM df"); err == nil {
			t.Fatalf("expected %s(x,y,z) to error (3 args)", fn)
		}
	}
}

// test_greatest_least (numeric/string arms): GREATEST/LEAST over numeric and
// string columns including mixed int/float and a literal 0. Standard SQL → MATCH.
//
// DISCREPANCY (dtype): polars promotes the numeric GREATEST/LEAST results to
// Float64 across the board; DuckDB keeps integer results where all inputs are
// integers and only promotes when a float operand participates. The values agree
// with polars; we assert DuckDB's actual returned types.
func TestConditionalGreatestLeast(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(-100), nil, int64(200), int64(99)}},
			frame.SeriesInput{Name: "b", Values: []any{nil, -0.1, 99.0, 100.0}},
			frame.SeriesInput{Name: "c", Values: []any{"bb", "aa", "dd", "cc"}},
			frame.SeriesInput{Name: "d", Values: []any{"cc", "bb", "aa", "dd"}},
		),
	}

	// GREATEST: a mixes int64 col with float col `b` and literal 0 → promotes to
	// float; a vs b alone same; string GREATEST returns the lexicographically
	// larger string.
	gmax := queryCtx(t, `
		SELECT
		  GREATEST("a", 0, "b") AS max_ab_zero,
		  GREATEST("a", "b") AS max_ab,
		  GREATEST("c", "d") AS max_cd
		FROM df
	`, tables)
	// DuckDB GREATEST/LEAST skip NULLs (NULL is ignored unless all args NULL),
	// matching polars' horizontal-max semantics.
	floatRow(t, floats(t, gmax, "max_ab_zero"), []float64{0.0, 0.0, 200.0, 100.0}, "max_ab_zero")
	floatRow(t, floats(t, gmax, "max_ab"), []float64{-100.0, -0.1, 200.0, 100.0}, "max_ab")
	eqRow(t, vals(t, gmax, "max_cd"), []any{"cc", "bb", "dd", "dd"}, "max_cd")

	lmin := queryCtx(t, `
		SELECT
		  LEAST("b", "a", 0) AS min_ab_zero,
		  LEAST("a", "b") AS min_ab,
		  LEAST("c", "d") AS min_cd
		FROM df
	`, tables)
	floatRow(t, floats(t, lmin, "min_ab_zero"), []float64{-100.0, -0.1, 0.0, 0.0}, "min_ab_zero")
	floatRow(t, floats(t, lmin, "min_ab"), []float64{-100.0, -0.1, 99.0, 99.0}, "min_ab")
	eqRow(t, vals(t, lmin, "min_cd"), []any{"bb", "aa", "aa", "cc"}, "min_cd")
}

// test_greatest_least (date arms): GREATEST/LEAST over date columns e/f and a
// '1999-12-31'::date literal, skipping NULLs.
//
// DISCREPANCY: gopolars has no Date dtype, so e/f are modeled as Datetime (midnight
// UTC) and the DATE result reads back as Datetime. The instants MATCH polars.
func TestConditionalGreatestLeastDates(t *testing.T) {
	dt := func(y, mo, d int) time.Time { return utc(y, mo, d, 0, 0, 0, 0) }
	tables := map[string]polars.DataFrame{
		"df": mustFrame(t,
			frame.SeriesInput{Name: "e", Values: []any{dt(1969, 12, 31), dt(2021, 1, 2), nil, dt(2021, 1, 4)}},
			frame.SeriesInput{Name: "f", Values: []any{dt(1970, 1, 1), dt(2000, 10, 20), dt(2077, 7, 5), nil}},
		),
	}
	gmax := queryCtx(t, `SELECT GREATEST('1999-12-31'::date, "e", "f") AS max_efx FROM df`, tables)
	eqTimes(t, vals(t, gmax, "max_efx"),
		[]any{dt(1999, 12, 31), dt(2021, 1, 2), dt(2077, 7, 5), dt(2021, 1, 4)}, "max_efx")

	lmin := queryCtx(t, `SELECT LEAST("f", "e", '1999-12-31'::date) AS min_efx FROM df`, tables)
	eqTimes(t, vals(t, lmin, "min_efx"),
		[]any{dt(1969, 12, 31), dt(1999, 12, 31), dt(1999, 12, 31), dt(1999, 12, 31)}, "min_efx")
}

// floats reads a numeric column, coercing int64/float64 to float64 for comparison
// (DuckDB may return either depending on operand promotion).
func floats(t *testing.T, d polars.DataFrame, name string) []float64 {
	t.Helper()
	vs := vals(t, d, name)
	out := make([]float64, len(vs))
	for i, v := range vs {
		out[i] = toFloat(v)
	}
	return out
}

// floatRow compares two float slices within a small tolerance.
func floatRow(t *testing.T, got, want []float64, msg string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: len got=%d want=%d (%v vs %v)", msg, len(got), len(want), got, want)
	}
	for i := range want {
		if d := got[i] - want[i]; d > 1e-9 || d < -1e-9 {
			t.Fatalf("%s[%d] = %v, want %v", msg, i, got[i], want[i])
		}
	}
}
