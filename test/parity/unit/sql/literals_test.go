//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_literals.py (py-1.28.1).
//
// DuckDB-backed (build -tags "duckdb duckdb_arrow"). Divergence zones:
//
//   - Binary/BLOB literals (b'...', x'...'): polars surfaces a Binary dtype.
//     gopolars' Arrow bridge handling of BLOB is probed below; a BLOB *result*
//     column is GAP'd if it cannot be read back, but filtering on a BLOB column
//     and selecting a non-binary result MATCHes.
//   - INTERVAL literals produce a Duration result (Arrow interval/duration) which
//     gopolars cannot read back → GAP. Interval *comparisons* yield a bool and
//     are pinned to DuckDB's behavior.
//   - polars-specific SQLSyntaxError/SQLInterfaceError message assertions → GAP.
//   - There is no Date dtype, so date-returning interval offsets are GAP'd.
package sql

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_bit_hex_literals: polars surfaces b'...'/x'...' as BLOB result columns; they
// now read back as gopolars Binary ([]byte).
//
// DISCREPANCY: DuckDB has no b'...'/x'...' literal in a SELECT projection (it lexes
// `x'FF'` as identifier+string). The equivalent DuckDB blob literal '\xNN'::BLOB
// produces the same bytes — a polars bit-string b'1001' is the byte its bits encode
// (0x09), and x'4142' is "AB". MATCH on the resulting bytes.
func TestLiteralsBitHexLiterals(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1)}})
	out := query(t, d, `
		SELECT
		  ''::BLOB               AS b0,
		  '\x09'::BLOB           AS b1,
		  '\xeb'::BLOB           AS b2,
		  '\xfd\x32'::BLOB       AS b3,
		  ''::BLOB               AS x0,
		  '\xFF'::BLOB           AS x1,
		  '\x41\x42'::BLOB       AS x2,
		  '\xDE\xAD\xBE\xEF'::BLOB AS x3
		FROM self`)
	eqRow(t, vals(t, out, "b0"), []any{[]byte{}}, "b0 (b'')")
	eqRow(t, vals(t, out, "b1"), []any{[]byte{0x09}}, "b1 (b'1001')")
	eqRow(t, vals(t, out, "b2"), []any{[]byte{0xeb}}, "b2 (b'11101011')")
	eqRow(t, vals(t, out, "b3"), []any{[]byte{0xfd, 0x32}}, "b3 (b'1111110100110010')")
	eqRow(t, vals(t, out, "x0"), []any{[]byte{}}, "x0 (x'')")
	eqRow(t, vals(t, out, "x1"), []any{[]byte{0xff}}, "x1 (x'FF')")
	eqRow(t, vals(t, out, "x2"), []any{[]byte("AB")}, "x2 (x'4142')")
	eqRow(t, vals(t, out, "x3"), []any{[]byte{0xde, 0xad, 0xbe, 0xef}}, "x3 (x'DeadBeef')")
}

// test_bit_hex_filter: filter a BLOB column with hex/bit literals, projecting a
// non-binary `val` column. gopolars has no Binary dtype, so the BLOB column is
// built inside DuckDB via a VALUES clause (over a `self` stub); the result is the
// int `val`, which crosses the Arrow bridge fine.
//
//	MATCH: `bin > x'02'` keeps 0x03, 0x04 → val [7, 6].
//	DISCREPANCY: polars' `b'00000010'` bit-string equals the byte 0x02, but DuckDB
//	parses b'...' as a BIT type whose comparison to a BLOB does not match the byte;
//	`bin > b'00000010'` returns ALL rows. Pinned to DuckDB's actual behavior.
func TestLiteralsBitHexFilter(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "dummy", Values: []any{int64(0)}})
	const tbl = `(VALUES (x'01',9),(x'02',8),(x'03',7),(x'04',6)) AS t(bin,val)`

	// MATCH: hex literal x'02'.
	out := query(t, d, `SELECT val FROM `+tbl+` WHERE bin > x'02' ORDER BY val DESC`)
	eqRow(t, vals(t, out, "val"), []any{int64(7), int64(6)}, "bin > x'02'")

	// DISCREPANCY: bit-string b'00000010' is a BIT (not BLOB) in DuckDB → all rows.
	outB := query(t, d, `SELECT val FROM `+tbl+` WHERE bin > b'00000010' ORDER BY val DESC`)
	eqRow(t, vals(t, outB, "val"), []any{int64(9), int64(8), int64(7), int64(6)}, "bin > b'...' (BIT, not BLOB)")
}

// test_bit_hex_errors: polars rejects malformed bit/hex literals.
//
// DISCREPANCY: polars rejects malformed b'..'/x'..' with SQLSyntaxError; the
// equivalent malformed DuckDB blob literals — '\xGG'::BLOB (invalid hex) and
// '\xF'::BLOB (odd nibble) — raise a Conversion Error. Only the error condition is
// asserted (message text differs).
func TestLiteralsBitHexErrors(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1)}})
	for _, lit := range []string{`'\xGG'::BLOB`, `'\xF'::BLOB`} {
		if runErr(t, d, `SELECT `+lit+` AS bad FROM self`) == nil {
			t.Fatalf("expected malformed blob %s to be rejected", lit)
		}
	}
}

// test_bit_hex_membership: `x IN (lit, lit)` over a BLOB column, projecting the
// non-binary `y`. The BLOB column is built inline via VALUES (no Binary dtype in
// gopolars).
//
//	MATCH: hex literals — `x IN (x'05', x'0b')` → y [1, 4].
//	DISCREPANCY: polars accepts the b'...' bit-string form as binary, but DuckDB's
//	b'...' is a BIT type, so `x IN (b'00000101', b'00001011')` matches nothing
//	(empty). Pinned to DuckDB's actual behavior.
func TestLiteralsBitHexMembership(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "dummy", Values: []any{int64(0)}})
	const tbl = `(VALUES (x'05',1),(x'ff',2),(x'cc',3),(x'0b',4)) AS t(x,y)`

	// MATCH: hex literals.
	out := query(t, d, `SELECT y FROM `+tbl+` WHERE x IN (x'05', x'0b') ORDER BY y`)
	eqRow(t, vals(t, out, "y"), []any{int64(1), int64(4)}, "x IN (x'05', x'0b')")

	// DISCREPANCY: b'...' is BIT, not BLOB → no rows match.
	outB := query(t, d, `SELECT y FROM `+tbl+` WHERE x IN (b'00000101', b'00001011') ORDER BY y`)
	if outB.Height() != 0 {
		t.Fatalf("x IN (b'...') matched %d rows, want 0 (BIT vs BLOB)", outB.Height())
	}
}

// test_dollar_quoted_literals: $$...$$, $tag$...$tag$ dollar-quoted string
// literals. DuckDB supports dollar quoting → MATCH on the string values.
func TestLiteralsDollarQuoted(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df": mustFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1)}}),
	}
	out := queryCtx(t, `
		SELECT
		  $$xyz$$ AS dq1,
		  $q$xyz$q$ AS dq2,
		  $tag$xyz$tag$ AS dq3,
		  $QUOTE$xyz$QUOTE$ AS dq4
		FROM df
	`, tables)
	eqRow(t, vals(t, out, "dq1"), []any{"xyz"}, "dq1")
	eqRow(t, vals(t, out, "dq2"), []any{"xyz"}, "dq2")
	eqRow(t, vals(t, out, "dq3"), []any{"xyz"}, "dq3")
	eqRow(t, vals(t, out, "dq4"), []any{"xyz"}, "dq4")

	out2 := queryCtx(t, `SELECT $$x$z$$ AS dq FROM df`, tables)
	eqRow(t, vals(t, out2, "dq"), []any{"x$z"}, "embedded dollar")
}

// test_fixed_intervals: INTERVAL '...' literals produce a Duration result, now read
// back as a gopolars Duration column (time.Duration).
//
// DISCREPANCY: (1) polars' compact ('1w2h3m4s') and comma-separated interval syntax
// is rewritten to DuckDB's space-separated form; (2) gopolars' Duration has no
// calendar component, so the Arrow month_day_nano interval is flattened to a fixed
// time.Duration (days→24h). The durations MATCH polars. The polars-specific error
// arms (negative/unary/months-in-fixed) are not reproduced: DuckDB accepts months and
// has different diagnostics, so they are out of scope.
func TestLiteralsFixedIntervals(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1)}})
	out := query(t, d, `
		SELECT
		  INTERVAL '1 week 2 hours 3 minutes 4 seconds' AS i1,
		  INTERVAL '100 milliseconds 100 microseconds'  AS i2,
		  INTERVAL '1 week 2 hours 3 minutes 4 seconds' AS i3
		FROM self`)
	want1 := 7*24*time.Hour + 2*time.Hour + 3*time.Minute + 4*time.Second
	want2 := 100*time.Millisecond + 100*time.Microsecond
	if got := col(t, out, "i1").Value(0); got != want1 {
		t.Fatalf("i1 = %v, want %v", got, want1)
	}
	if got := col(t, out, "i2").Value(0); got != want2 {
		t.Fatalf("i2 = %v, want %v", got, want2)
	}
	if got := col(t, out, "i3").Value(0); got != want1 {
		t.Fatalf("i3 = %v, want %v", got, want1)
	}
}

// test_interval_offsets: datetime/date +/- INTERVAL arithmetic.
//
// DISCREPANCY: (1) gopolars has no Date dtype, so the `dt` column is modeled as
// Datetime (midnight UTC) and the DATE +/- INTERVAL results read back as Datetime;
// (2) DuckDB rejects comma-separated unit lists inside one INTERVAL literal, so
// '2 months, 30 minutes' is written as a sum of two INTERVAL literals. The instants
// MATCH polars.
func TestLiteralsIntervalOffsets(t *testing.T) {
	dt := func(y, mo, d, h, mi, s int) time.Time { return time.Date(y, time.Month(mo), d, h, mi, s, 0, time.UTC) }
	d := mustFrame(t,
		frame.SeriesInput{Name: "dtm", Values: []any{
			dt(1899, 12, 31, 8, 0, 0), dt(1999, 6, 8, 10, 30, 0), dt(2010, 5, 7, 20, 20, 20),
		}},
		frame.SeriesInput{Name: "dt", Values: []any{
			dt(1950, 4, 10, 0, 0, 0), dt(2048, 1, 20, 0, 0, 0), dt(2026, 8, 5, 0, 0, 0),
		}},
	)
	out := query(t, d, `
		SELECT
		  dtm + INTERVAL '2 months' + INTERVAL '30 minutes' AS dtm_plus_2mo30m,
		  dt + INTERVAL '100 years' AS dt_plus_100y,
		  dt - INTERVAL '1 quarter' AS dt_minus_1q
		FROM self
		ORDER BY 1`)
	eqTimes(t, vals(t, out, "dtm_plus_2mo30m"),
		[]any{dt(1900, 2, 28, 8, 30, 0), dt(1999, 8, 8, 11, 0, 0), dt(2010, 7, 7, 20, 50, 20)}, "dtm_plus_2mo30m")
	eqTimes(t, vals(t, out, "dt_plus_100y"),
		[]any{dt(2050, 4, 10, 0, 0, 0), dt(2148, 1, 20, 0, 0, 0), dt(2126, 8, 5, 0, 0, 0)}, "dt_plus_100y")
	eqTimes(t, vals(t, out, "dt_minus_1q"),
		[]any{dt(1950, 1, 10, 0, 0, 0), dt(2047, 10, 20, 0, 0, 0), dt(2026, 5, 5, 0, 0, 0)}, "dt_minus_1q")
}

// test_interval_comparisons: comparisons between two INTERVAL literals yield a
// bool. The interval expressions run over a `self` stub via df.SQL.
//
// DISCREPANCY: DuckDB does not accept comma-separated unit lists inside a single
// INTERVAL literal (e.g. INTERVAL '3 days, 1 microsecond'), so that py case is
// rewritten as a sum of INTERVAL literals (`INTERVAL '3 days' + INTERVAL '1
// microsecond'`). DuckDB also rejects the `<=>` and `==` polars spellings —
// `==` is replaced with the standard `=`. With those adaptations, DuckDB's
// calendar-vs-fixed comparison results MATCH polars on these cases.
func TestLiteralsIntervalComparisons(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "dummy", Values: []any{int64(0)}})
	cases := []struct {
		expr string
		want bool
	}{
		{`INTERVAL '3 days' <= (INTERVAL '3 days' + INTERVAL '1 microsecond')`, true},
		{`(INTERVAL '3 days' + INTERVAL '1 microsecond') <= INTERVAL '3 days'`, false},
		{`INTERVAL '3 months' >= INTERVAL '3 months'`, true},
		{`INTERVAL '2 quarters' < INTERVAL '2 quarters'`, false},
		{`INTERVAL '2 quarters' > INTERVAL '2 quarters'`, false},
		{`INTERVAL '3 years' = INTERVAL '1008 weeks'`, false},
		{`INTERVAL '8 weeks' != INTERVAL '2 months'`, true},
		{`INTERVAL '8 weeks' = INTERVAL '2 months'`, false},
		{`INTERVAL '1 year' != INTERVAL '365 days'`, true},
		{`INTERVAL '1 year' = INTERVAL '1 year'`, true},
	}
	for _, tc := range cases {
		out := query(t, d, `SELECT (`+tc.expr+`) AS res FROM self`)
		eqRow(t, vals(t, out, "res"), []any{tc.want}, tc.expr)
	}
}

// test_select_literals_no_table: SELECT 1, '2', 3.0 with no real table. df.SQL
// needs a `self` table, so the literals are projected over a single-row stub.
// MATCH on values (int64 / string / float64).
func TestLiteralsSelectNoTable(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "dummy", Values: []any{int64(0)}})
	// DISCREPANCY: a decimal literal like 3.0 is FLOAT in polars but DECIMAL(2,1) in
	// DuckDB (now surfaced as a gopolars Decimal — see TestNumericDecimalType); cast to
	// DOUBLE to assert polars' float semantics.
	out := query(t, d, `SELECT 1 AS one, '2' AS two, 3.0::DOUBLE AS three FROM self`)
	eqRow(t, vals(t, out, "one"), []any{int64(1)}, "one")
	eqRow(t, vals(t, out, "two"), []any{"2"}, "two")
	eqRow(t, vals(t, out, "three"), []any{3.0}, "three")
}

// test_select_from_table_with_reserved_names: a table and columns named with SQL
// reserved words ("select", "from") accessed via double-quoting. DuckDB handles
// quoted identifiers like polars → MATCH ((5, 2)).
func TestLiteralsReservedNames(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"select": mustFrame(t,
			frame.SeriesInput{Name: "select", Values: []any{int64(1), int64(2), int64(3)}},
			frame.SeriesInput{Name: "from", Values: []any{int64(4), int64(5), int64(6)}},
		),
	}
	out := queryCtx(t, `
		SELECT "from", "select"
		  FROM "select"
		  WHERE "from" >= 5 AND "select" % 2 != 1
	`, tables)
	eqRow(t, vals(t, out, "from"), []any{int64(5)}, "from")
	eqRow(t, vals(t, out, "select"), []any{int64(2)}, "select")
}
