//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_functions.py (py-1.28.1).
//
// The upstream module has a single test, test_sql_expr, exercising polars'
// `pl.sql_expr([...])` helper: it PARSES bare SQL expression strings into polars
// Expr objects and applies them via `df.select(*exprs)` (NOT a full SELECT query).
// gopolars exposes no `sql_expr` / SQL-string-to-Expr parser, so that API surface
// is a GAP (TestFunctionsSqlExpr).
//
// To still give the underlying scalar/aggregate functions real parity coverage,
// the same functions (MIN, POWER, SUBSTR) are exercised through the DuckDB SELECT
// engine instead. DuckDB's function catalog differs from polars-sql's:
//   - POWER/POW return DOUBLE (float64) even for integer args, whereas polars'
//     sql_expr preserves the integer dtype -> DISCREPANCY on result dtype.
//   - SUBSTR(s, start, len) is 1-indexed and NULL-preserving in both -> MATCH.
package sql

import (
	"context"
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// test_sql_expr: pl.sql_expr(["MIN(a)", "POWER(a,a) AS aa", "SUBSTR(b,2,2) AS b2"])
// applied via df.select. This is the polars expression-string parser, which
// gopolars does not provide -> GAP. The polars-only error case (sql_expr("xyz.*")
// raising 'unable to parse ... as Expr') is likewise inapplicable.
func TestFunctionsSqlExpr(t *testing.T) {
	t.Skip("GAP: pl.sql_expr (SQL-string-to-Expr parser) has no gopolars equivalent")
}

// TestFunctionsScalarsViaSelect ports the *functions* under test in test_sql_expr
// (POWER, SUBSTR) to the DuckDB SELECT engine.
//
//   - SUBSTR(b, 2, 2): 1-indexed substring, NULL-preserving. MATCH with polars
//     (["yz", "bc", NULL]).
//   - POWER(a, a): DISCREPANCY: DuckDB returns DOUBLE (1.0, 4.0, 27.0); polars'
//     sql_expr keeps Int64 (1, 4, 27). Same numeric values, different dtype; we
//     assert the DuckDB float result.
func TestFunctionsScalarsViaSelect(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "b", Values: []any{"xyz", "abcde", nil}},
	)
	out := query(t, d, `
		SELECT
		  POWER(a, a)   AS aa,
		  SUBSTR(b, 2, 2) AS b2
		FROM self
		ORDER BY a
	`)

	// DISCREPANCY: POWER returns float64 in DuckDB.
	aa := col(t, out, "aa")
	wantAA := []float64{1, 4, 27}
	for i, w := range wantAA {
		if got := toFloat(aa.Value(i)); math.Abs(got-w) > 1e-9 {
			t.Fatalf("POWER aa[%d] = %v, want %v", i, aa.Value(i), w)
		}
	}

	// MATCH: SUBSTR 1-indexed, NULL-preserving.
	eqRow(t, vals(t, out, "b2"), []any{"yz", "bc", nil}, "SUBSTR(b,2,2)")
}

// TestFunctionsMinAggregate covers the MIN(a) aggregate from test_sql_expr.
// MATCH: integer MIN stays int64.
func TestFunctionsMinAggregate(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
	)
	out := query(t, d, `SELECT MIN(a) AS m FROM self`)
	if got := col(t, out, "m").Value(0); got != int64(1) {
		t.Fatalf("MIN(a) = %v (%T), want int64(1)", got, got)
	}
}

// TestFunctionsSqlExprWildcardRejected mirrors the polars-only error arm of
// test_sql_expr (pl.sql_expr("xyz.*") -> SQLInterfaceError). gopolars has no
// expression-string parser, but the DuckDB SELECT engine likewise rejects a
// table-qualified wildcard against a missing table -> assert error condition only
// (DISCREPANCY: different message).
func TestFunctionsSqlExprWildcardRejected(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1)}})
	// xyz is not a registered table -> binder/catalog error.
	lf, err := d.SQL(context.Background(), `SELECT xyz.* FROM self`)
	if err == nil && lf != nil {
		if _, cerr := lf.Collect(context.Background()); cerr == nil {
			t.Fatalf("expected SELECT xyz.* (unknown table) to be rejected")
		}
	}
}
