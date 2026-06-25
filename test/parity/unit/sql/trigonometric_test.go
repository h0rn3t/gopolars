//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_trigonometric.py (py-1.28.1).
//
// gopolars runs SQL through an embedded DuckDB engine (build -tags "duckdb
// duckdb_arrow"), so behavior is measured against DuckDB's dialect, not polars'
// native polars-sql engine.
//
// DIALECT NOTE: polars exposes degree-input trig functions SIND/COSD/TAND/COTD/
// ASIND/ACOSD/ATAND/ATAN2D. DuckDB does NOT define any of these (Catalog Error:
// "Scalar Function ... does not exist"). DuckDB only ships the radian functions
// plus DEGREES()/RADIANS(). The degree variants are therefore re-expressed here
// as DEGREES(fn(...)) / fn(RADIANS(...)), which is mathematically identical, and
// pinned with a // DISCREPANCY: note. All radian trig matches polars (MATCH).
package sql

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

const trigTol = 1e-9

func trigClose(t *testing.T, got any, want float64, msg string) {
	t.Helper()
	g, ok := got.(float64)
	if !ok {
		t.Fatalf("%s: got %v (%T), want float64", msg, got, got)
	}
	if math.IsNaN(want) {
		if !math.IsNaN(g) {
			t.Fatalf("%s: got %v, want NaN", msg, g)
		}
		return
	}
	if math.IsInf(want, 0) {
		if g != want {
			t.Fatalf("%s: got %v, want %v", msg, g, want)
		}
		return
	}
	if math.Abs(g-want) > trigTol {
		t.Fatalf("%s: got %v, want %v", msg, g, want)
	}
}

// test_arctan2: ATAN2(y,x) (radians) and the degree form. polars writes ATAN2D;
// DuckDB lacks it, so the degree result is produced via DEGREES(ATAN2(...)).
func TestTrigArctan2(t *testing.T) {
	r := math.Sqrt(2) / 2.0
	d := mustFrame(t,
		frame.SeriesInput{Name: "y", Values: []any{r, -r, r, -r}},
		frame.SeriesInput{Name: "x", Values: []any{r, r, -r, -r}},
		frame.SeriesInput{Name: "idx", Values: []any{int64(0), int64(1), int64(2), int64(3)}},
	)
	// DISCREPANCY: ATAN2D(y,x) (polars) -> DEGREES(ATAN2(y,x)) (DuckDB equivalent).
	out := query(t, d, `
		SELECT
		  DEGREES(ATAN2(y,x)) AS "atan2d",
		  ATAN2(y,x) AS "atan2"
		FROM self
		ORDER BY idx`)

	wantDeg := []float64{45.0, -45.0, 135.0, -135.0}
	deg := col(t, out, "atan2d")
	rad := col(t, out, "atan2")
	for i, w := range wantDeg {
		trigClose(t, deg.Value(i), w, "atan2d")
		trigClose(t, rad.Value(i), w*math.Pi/180.0, "atan2")
	}
}

// test_trig: the full radian trig battery plus the degree variants.
//   - Radian funcs (cos/cot/sin/tan/acos/asin/atan) MATCH polars exactly.
//   - cosd/cotd/sind/tand are re-expressed as fn(RADIANS(x)); acosd/asind/atand
//     as DEGREES(fn(x)) (DuckDB lacks the *D names). // DISCREPANCY: name only.
//
// Expected values are computed in Go from the same formulas the py test uses, so
// they are derived, not transcribed.
//
// DISCREPANCY: the py test includes a=0.0, for which ASIN(1.0)/a is +Inf and
// polars' trig returns NaN. DuckDB instead raises "Out of Range Error: input
// value inf is out of range for numeric function" and fails the whole query, so
// a=0.0 is excluded here and covered separately by TestTrigInfInputErrors.
func TestTrigBattery(t *testing.T) {
	a := []float64{-4.0, -3.0, -2.0, -1.00001, 1.00001, 2.0, 3.0, 4.0}
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: toAny(a)},
		frame.SeriesInput{Name: "idx", Values: idxAny(len(a))},
	)
	out := query(t, d, `
		SELECT
		  ASIN(1.0)/a AS "pi_values",
		  COS(ASIN(1.0)/a) AS "cos",
		  COT(ASIN(1.0)/a) AS "cot",
		  SIN(ASIN(1.0)/a) AS "sin",
		  TAN(ASIN(1.0)/a) AS "tan",

		  COS(RADIANS(DEGREES(ASIN(1.0))/a)) AS "cosd",
		  COT(RADIANS(DEGREES(ASIN(1.0))/a)) AS "cotd",
		  SIN(RADIANS(DEGREES(ASIN(1.0))/a)) AS "sind",
		  TAN(RADIANS(DEGREES(ASIN(1.0))/a)) AS "tand",

		  1.0/a AS "inv_pi_values",
		  ACOS(1.0/a) AS "acos",
		  ASIN(1.0/a) AS "asin",
		  ATAN(1.0/a) AS "atan",

		  DEGREES(ACOS(1.0/a)) AS "acosd",
		  DEGREES(ASIN(1.0/a)) AS "asind",
		  DEGREES(ATAN(1.0/a)) AS "atand"
		FROM self
		ORDER BY idx`)

	asin1 := math.Asin(1.0) // pi/2
	for i, av := range a {
		piV := asin1 / av
		invPi := 1.0 / av

		trigClose(t, col(t, out, "pi_values").Value(i), piV, "pi_values")
		trigClose(t, col(t, out, "cos").Value(i), math.Cos(piV), "cos")
		trigClose(t, col(t, out, "cot").Value(i), 1.0/math.Tan(piV), "cot")
		trigClose(t, col(t, out, "sin").Value(i), math.Sin(piV), "sin")
		trigClose(t, col(t, out, "tan").Value(i), math.Tan(piV), "tan")

		// Degree variants are mathematically identical to the radian ones here
		// (degrees->radians round-trips), so they share the same expected values.
		trigClose(t, col(t, out, "cosd").Value(i), math.Cos(piV), "cosd")
		trigClose(t, col(t, out, "cotd").Value(i), 1.0/math.Tan(piV), "cotd")
		trigClose(t, col(t, out, "sind").Value(i), math.Sin(piV), "sind")
		trigClose(t, col(t, out, "tand").Value(i), math.Tan(piV), "tand")

		trigClose(t, col(t, out, "inv_pi_values").Value(i), invPi, "inv_pi_values")
		trigClose(t, col(t, out, "acos").Value(i), math.Acos(invPi), "acos")
		trigClose(t, col(t, out, "asin").Value(i), math.Asin(invPi), "asin")
		trigClose(t, col(t, out, "atan").Value(i), math.Atan(invPi), "atan")

		trigClose(t, col(t, out, "acosd").Value(i), math.Acos(invPi)*180.0/math.Pi, "acosd")
		trigClose(t, col(t, out, "asind").Value(i), math.Asin(invPi)*180.0/math.Pi, "asind")
		trigClose(t, col(t, out, "atand").Value(i), math.Atan(invPi)*180.0/math.Pi, "atand")
	}
}

// TestTrigInfInputErrors pins the a=0.0 row dropped from TestTrigBattery:
// polars evaluates trig(inf) to NaN, whereas DuckDB raises an Out of Range error
// for infinite trig input.
func TestTrigInfInputErrors(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "a", Values: []any{0.0}})
	q := `SELECT COS(ASIN(1.0)/a) AS r FROM self`
	if _, err := d.SQL(t.Context(), q); err == nil {
		t.Fatalf("trig(inf): expected Out of Range error, got nil")
	}
}

// TestTrigDegreeFunctionsAbsent documents the dialect gap: polars' *D trig
// function names are unknown to DuckDB and raise a Catalog Error rather than
// computing a degree-based result.
func TestTrigDegreeFunctionsAbsent(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{0.5}})
	for _, fn := range []string{"SIND", "COSD", "TAND", "COTD", "ASIND", "ACOSD", "ATAND", "ATAN2D"} {
		q := "SELECT " + fn + "(x) AS r FROM self"
		if _, err := d.SQL(t.Context(), q); err == nil {
			t.Fatalf("%s: expected Catalog Error (DuckDB lacks degree trig fn), got nil", fn)
		}
	}
}

func toAny(xs []float64) []any {
	out := make([]any, len(xs))
	for i, x := range xs {
		out[i] = x
	}
	return out
}

func idxAny(n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = int64(i)
	}
	return out
}
