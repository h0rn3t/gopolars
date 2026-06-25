//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_numeric.py (py-1.28.1).
//
// DuckDB-backed: behavior measured against DuckDB's dialect. DIV/MOD/ROUND and the
// std/var aggregate aliases are standard SQL and MATCH on values. The polars
// Decimal-cast dtype assertions and polars-specific ROUND error messages diverge
// (DISCREPANCY / GAP).
package sql

import (
	"context"
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_div: polars exposes a DIV(a,b) integer floor-division function returning
// a_div_b = [3, NULL, 0, NULL, 0] and b_div_a = [0, NULL, 2, NULL, 2].
//
// DISCREPANCY: gopolars/DuckDB has neither the DIV() function nor floor-dividing
// `//` for floats:
//   - DIV() is polars-only — DuckDB rejects it (suggests `fdiv`).
//   - DuckDB's `//` over DOUBLE operands is plain float division (NOT floor), so
//     a // b == a / b. Pin DuckDB's actual float output.
func TestNumericDiv(t *testing.T) {
	// df.SQL requires the `self` table to exist; the data comes from an inline
	// VALUES clause, so `self` is just a single-row stub that is never referenced.
	d := mustFrame(t, frame.SeriesInput{Name: "dummy", Values: []any{int64(0)}})

	// DISCREPANCY: DIV() is polars-only — DuckDB rejects it.
	if _, err := d.SQL(context.Background(), `SELECT DIV(1, 2) FROM self`); err == nil {
		t.Fatalf("expected DuckDB to reject DIV() (polars-only function)")
	}

	// DISCREPANCY: `//` on float operands is plain float division in DuckDB.
	out := query(t, d, `
		SELECT label, a // b AS a_div_b, b // a AS b_div_a
		FROM (
		  VALUES
		    ('a', 20.5, 6),
		    ('b', NULL, 12),
		    ('c', 10.0, 24),
		    ('d', 5.0, NULL),
		    ('e', 2.5, 5)
		) AS tbl(label, a, b)
		ORDER BY label
	`)
	eqRow(t, vals(t, out, "label"), []any{"a", "b", "c", "d", "e"}, "label")
	// rows b and d are NULL (NULL operand); the rest are float a/b.
	if col(t, out, "a_div_b").Value(1) != nil || col(t, out, "b_div_a").Value(1) != nil {
		t.Fatalf("row b should be NULL on both")
	}
	if col(t, out, "a_div_b").Value(3) != nil || col(t, out, "b_div_a").Value(3) != nil {
		t.Fatalf("row d should be NULL on both")
	}
	checkFloat(t, col(t, out, "a_div_b").Value(0), 20.5/6, "a_div_b[a]")
	checkFloat(t, col(t, out, "a_div_b").Value(2), 10.0/24, "a_div_b[c]")
	checkFloat(t, col(t, out, "a_div_b").Value(4), 2.5/5, "a_div_b[e]")
	checkFloat(t, col(t, out, "b_div_a").Value(0), 6.0/20.5, "b_div_a[a]")
	checkFloat(t, col(t, out, "b_div_a").Value(2), 24.0/10.0, "b_div_a[c]")
	checkFloat(t, col(t, out, "b_div_a").Value(4), 5.0/2.5, "b_div_a[e]")
}

// checkFloat asserts a single value equals want within tolerance.
func checkFloat(t *testing.T, got any, want float64, msg string) {
	t.Helper()
	if d := toFloat(got) - want; d > 1e-9 || d < -1e-9 {
		t.Fatalf("%s = %v, want %v", msg, got, want)
	}
}

// test_modulo: `%` and MOD() over int and float operands, including NULL passthru.
// Standard SQL → MATCH.
func TestNumericModulo(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{1.5, nil, 3.0, 13.0 / 3.0, 5.0}},
		frame.SeriesInput{Name: "b", Values: []any{int64(6), int64(7), int64(8), int64(9), int64(10)}},
		frame.SeriesInput{Name: "c", Values: []any{int64(11), int64(12), int64(13), int64(14), int64(15)}},
		frame.SeriesInput{Name: "d", Values: []any{16.5, 17.0, 18.5, nil, 20.0}},
	)
	out := query(t, d, `
		SELECT
		  a % 2 AS a2,
		  b % 3 AS b3,
		  MOD(c, 4) AS c4,
		  MOD(d, 5.5) AS d55
		FROM self
	`)
	// a % 2 (float): 1.5, NULL, 1.0, (13/3)%2=1/3, 1.0
	a2 := floats(t, out, "a2")
	wantA2 := []float64{1.5, math.NaN(), 1.0, 1.0 / 3.0, 1.0}
	for i, w := range wantA2 {
		if i == 1 {
			if col(t, out, "a2").Value(1) != nil {
				t.Fatalf("a2[1] = %v, want NULL", col(t, out, "a2").Value(1))
			}
			continue
		}
		if math.Abs(a2[i]-w) > 1e-9 {
			t.Fatalf("a2[%d] = %v, want %v", i, a2[i], w)
		}
	}
	eqRow(t, ints(t, out, "b3"), []any{int64(0), int64(1), int64(2), int64(0), int64(1)}, "b3")
	eqRow(t, ints(t, out, "c4"), []any{int64(3), int64(0), int64(1), int64(2), int64(3)}, "c4")
	// d % 5.5 (float): 0.0, 0.5, 2.0, NULL, 3.5
	d55 := floats(t, out, "d55")
	wantD55 := []float64{0.0, 0.5, 2.0, math.NaN(), 3.5}
	for i, w := range wantD55 {
		if i == 3 {
			if col(t, out, "d55").Value(3) != nil {
				t.Fatalf("d55[3] = %v, want NULL", col(t, out, "d55").Value(3))
			}
			continue
		}
		if math.Abs(d55[i]-w) > 1e-9 {
			t.Fatalf("d55[%d] = %v, want %v", i, d55[i], w)
		}
	}
}

// test_numeric_decimal_type: casts a value to NUMERIC/DECIMAL(p,s) and asserts the
// resulting polars Decimal dtype + exact decimal value. gopolars normalizes DuckDB
// DECIMAL/HUGEINT to float64/int64 (no Decimal dtype), so the precise dtype/value
// semantics can't be reproduced → GAP.
func TestNumericDecimalType(t *testing.T) {
	// ::NUMERIC/::DECIMAL(p,s) now reads back as a gopolars Decimal column. The bridge
	// preserves explicit decimal casts (precision < 38, or scale > 0) and still widens
	// DuckDB's HUGEINT / integer aggregates — Decimal128(38,0) — to int64.
	//
	// DISCREPANCY: (1) DuckDB ROUNDS on a scale-0 cast (512.5→513, -1024.75→-1025);
	// polars truncates (512, -1024). (2) bare `::dec` is DECIMAL(18,3) in DuckDB vs
	// polars' Decimal(38,9), so the scale differs. Values are pinned to DuckDB's output.
	cases := []struct {
		sqltype string
		val     float64
		want    string
	}{
		{"numeric(3,1)", 64.5, "64.5"},
		{"decimal(4,1)", 512.5, "512.5"},
		{"numeric(4,0)", 512.5, "513"},
		{"decimal(10,0)", -1024.75, "-1025"},
		{"numeric(10)", -1024.75, "-1025"},
		{"dec", -1024.75, "-1024.750"},
	}
	for _, c := range cases {
		d := mustFrame(t, frame.SeriesInput{Name: "n", Values: []any{c.val}})
		out := query(t, d, `SELECT n::`+c.sqltype+` AS "dec" FROM self`)
		s := col(t, out, "dec")
		if s.DataType() != dtypes.Decimal {
			t.Fatalf("n::%s: dtype = %s, want decimal", c.sqltype, s.DataType())
		}
		if got := s.Value(0); got != dtypes.DecimalValue(c.want) {
			t.Fatalf("n::%s = %#v, want %q", c.sqltype, got, c.want)
		}
	}
}

// test_round_ndigits: ROUND(n) and ROUND(n, k) for k in 0..4. Standard SQL banker-
// vs half-away rounding can differ; we assert DuckDB's actual output.
//
// DISCREPANCY (rounding mode): polars rounds half away from zero. DuckDB ROUND uses
// round-half-away-from-zero as well for these inputs, so values agree with the
// polars expectations — pinned per decimals below.
func TestNumericRoundNdigits(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "n", Values: []any{
		-8192.499, -3.9550, -1.54321, 2.45678, 3.59901, 8192.5001,
	}})

	// decimals == 0 has two spellings: ROUND(n) and ROUND(n,0). Both must agree.
	out0 := query(t, d, `SELECT ROUND(n) AS n FROM self`)
	floatRow(t, floats(t, out0, "n"), []float64{-8192.0, -4.0, -2.0, 2.0, 4.0, 8193.0}, "round0 (no arg)")

	cases := []struct {
		dec  int
		want []float64
	}{
		{0, []float64{-8192.0, -4.0, -2.0, 2.0, 4.0, 8193.0}},
		{1, []float64{-8192.5, -4.0, -1.5, 2.5, 3.6, 8192.5}},
		{2, []float64{-8192.5, -3.96, -1.54, 2.46, 3.6, 8192.5}},
		{3, []float64{-8192.499, -3.955, -1.543, 2.457, 3.599, 8192.5}},
		{4, []float64{-8192.499, -3.955, -1.5432, 2.4568, 3.599, 8192.5001}},
	}
	for _, tc := range cases {
		q := `SELECT ROUND("n",` + itoa(tc.dec) + `) AS n FROM self`
		out := query(t, d, q)
		got := floats(t, out, "n")
		for i, w := range tc.want {
			if math.Abs(got[i]-w) > 1e-6 {
				t.Fatalf("ROUND(n,%d)[%d] = %v, want %v", tc.dec, i, got[i], w)
			}
		}
	}
}

// test_round_ndigits_errors: polars raises specific SQLSyntaxError/SQLInterfaceError
// messages for ROUND with a string ndigits, negative ndigits, or too many args.
// DuckDB's handling differs:
//   - ROUND(n,'!!')  → DuckDB errors (cannot cast '!!' to integer).
//   - ROUND(n,-1)    → DuckDB accepts negative ndigits (rounds to tens) — no error.
//   - ROUND(a,b,c,d) → DuckDB errors (no 4-arg ROUND).
//
// Pin DuckDB's actual behavior (DISCREPANCY on the negative-ndigits arm).
func TestNumericRoundNdigitsErrors(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "n", Values: []any{99.999}})

	if _, err := d.SQL(context.Background(), "SELECT ROUND(n,'!!') AS n FROM self"); err == nil {
		t.Fatalf("expected ROUND(n,'!!') to error")
	}

	// DISCREPANCY: polars rejects negative decimals; DuckDB accepts ROUND(99.999,-1)
	// → 100.0 (round to nearest ten).
	out := query(t, d, "SELECT ROUND(n,-1) AS n FROM self")
	if got := toFloat(col(t, out, "n").Value(0)); math.Abs(got-100.0) > 1e-9 {
		t.Fatalf("ROUND(99.999,-1) = %v, want 100.0 (DuckDB accepts negative ndigits)", got)
	}

	if _, err := d.SQL(context.Background(), "SELECT ROUND(1.2345,6,7,8) AS n FROM self"); err == nil {
		t.Fatalf("expected 4-arg ROUND to error")
	}
}

// test_stddev_variance: STDEV/STDDEV/STDEV_SAMP/STDDEV_SAMP and VAR/VARIANCE/
// VAR_SAMP — all sample (n-1) std/var aliases. DuckDB supports these aliases and
// uses sample stats → MATCH on values.
func TestNumericStddevVariance(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df": mustFrame(t,
			frame.SeriesInput{Name: "v1", Values: []any{-1.0, 0.0, 1.0}},
			frame.SeriesInput{Name: "v2", Values: []any{5.5, 0.0, 3.0}},
			frame.SeriesInput{Name: "v3", Values: []any{int64(-10), nil, int64(10)}},
			frame.SeriesInput{Name: "v4", Values: []any{-100.0, 0.0, -50.0}},
		),
	}
	out := queryCtx(t, `
		SELECT
		  STDDEV(v1) AS v1_std,
		  STDDEV(v2) AS v2_std,
		  STDDEV_SAMP(v3) AS v3_std,
		  STDDEV_SAMP(v4) AS v4_std,
		  VARIANCE(v1) AS v1_var,
		  VARIANCE(v2) AS v2_var,
		  VARIANCE(v3) AS v3_var,
		  VAR_SAMP(v4) AS v4_var
		FROM df
	`, tables)

	want := map[string]float64{
		"v1_std": 1.0,
		"v2_std": 2.7537852736431,
		"v3_std": 14.142135623731,
		"v4_std": 50.0,
		"v1_var": 1.0,
		"v2_var": 7.5833333333333,
		"v3_var": 200.0,
		"v4_var": 2500.0,
	}
	for name, w := range want {
		got := toFloat(col(t, out, name).Value(0))
		if math.Abs(got-w) > 1e-9 {
			t.Fatalf("%s = %v, want %v", name, got, w)
		}
	}
}

// ints reads a column whose entries are int64 or NULL, asserting each is int64/nil.
func ints(t *testing.T, d polars.DataFrame, name string) []any {
	t.Helper()
	return vals(t, d, name)
}

// itoa is a tiny base-10 int formatter (avoids importing strconv just for this).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
