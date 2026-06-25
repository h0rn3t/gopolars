//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_cast.py (py-1.28.1).
//
// gopolars runs SQL through an embedded DuckDB engine (-tags "duckdb
// duckdb_arrow"), so cast semantics are DuckDB's, not polars'. The big divergence
// zones here:
//   - DTYPE: polars preserves the precise cast target dtype (Int8/UInt16/Float32/
//     Int128/Binary/...). gopolars normalizes every DuckDB integer width (and the
//     unsigned/hugeint variants) to int64, every float width to float64, so the
//     fine-grained dtype assertions can't be reproduced (we assert the normalized
//     value instead, with a // DISCREPANCY note).
//   - ROUNDING: casting a float to an integer truncates in polars but ROUNDS in
//     DuckDB (b=5.5 -> 6, not 5). DISCREPANCY, pinned to DuckDB's actual output.
//   - SYNTAX: polars accepts `uint1/uint2/uint4/uint8` and `<INT> UNSIGNED`; DuckDB
//     uses `utinyint/usmallint/uinteger/ubigint` and rejects the `UNSIGNED`
//     postfix and `FORMAT '...'` with a parser error.
//   - BINARY/DATE/TIME: ::blob/::bytes/::VARBINARY and the date/time arrow result
//     types are unsupported by the gopolars arrow bridge -> GAP.
package sql

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func castBaseFrame(t *testing.T) polars.DataFrame {
	t.Helper()
	return mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
		frame.SeriesInput{Name: "b", Values: []any{1.1, 2.2, 3.3, 4.4, 5.5}},
		frame.SeriesInput{Name: "c", Values: []any{"a", "b", "c", "d", "e"}},
		frame.SeriesInput{Name: "d", Values: []any{true, false, true, false, true}},
		frame.SeriesInput{Name: "e", Values: []any{int64(-1), int64(0), nil, int64(1), int64(2)}},
	)
}

// test_cast (float arm): DOUBLE/DOUBLE PRECISION/real/float4/float8/float(24)/
// float(25) variants. gopolars normalizes every float width to float64, so the
// polars Float32-vs-Float64 dtype split is lost (DISCREPANCY on dtype). The
// VALUES still round-trip; float(24) yields f32-precision values (b stored at
// single precision) which we pin.
func TestCastFloatVariants(t *testing.T) {
	d := castBaseFrame(t)

	// DOUBLE / DOUBLE PRECISION / float8 keep full f64 precision.
	out := query(t, d, `
		SELECT
		  CAST(a AS DOUBLE PRECISION) AS a_f64,
		  CAST(b AS DOUBLE) AS b_f64,
		  e::float8 AS e_f64,
		  b::float(25) AS b_f64b
		FROM self
	`)
	floatRowCast(t, out, "a_f64", []float64{1, 2, 3, 4, 5})
	floatRowCast(t, out, "b_f64", []float64{1.1, 2.2, 3.3, 4.4, 5.5})
	floatRowCast(t, out, "b_f64b", []float64{1.1, 2.2, 3.3, 4.4, 5.5})
	// e has a NULL in slot 2.
	if col(t, out, "e_f64").Value(2) != nil {
		t.Fatalf("e_f64[2] should be NULL")
	}

	// real / float4 / float(24) round-trip at single precision: gopolars surfaces
	// them as float64 but holding the f32-rounded value (DISCREPANCY: dtype is
	// Float32 in polars).
	out32 := query(t, d, `
		SELECT
		  a::real AS a_f32,
		  a::float4 AS a_f32b,
		  b::float(24) AS b_f32
		FROM self
	`)
	floatRowCast(t, out32, "a_f32", []float64{1, 2, 3, 4, 5})
	floatRowCast(t, out32, "a_f32b", []float64{1, 2, 3, 4, 5})
	// f32 rounding of [1.1, 2.2, 3.3, 4.4, 5.5].
	wantF32 := []float64{
		float64(float32(1.1)), float64(float32(2.2)), float64(float32(3.3)),
		float64(float32(4.4)), float64(float32(5.5)),
	}
	floatRowCast(t, out32, "b_f32", wantF32)
}

// test_cast (integer arm): polars truncates float->int; DuckDB ROUNDS.
// b = [1.1, 2.2, 3.3, 4.4, 5.5] -> [1, 2, 3, 4, 6] (5.5 rounds up).
// Booleans cast to integer 0/1. gopolars normalizes every signed width
// (TINYINT/SMALLINT/INT/BIGINT/int1..int8/hugeint) to int64.
//
// DISCREPANCY: polars expects b->5 (truncation); DuckDB rounds 5.5->6.
func TestCastIntegerVariants(t *testing.T) {
	d := castBaseFrame(t)
	out := query(t, d, `
		SELECT
		  CAST(b AS TINYINT)  AS b_i8,
		  CAST(b AS SMALLINT) AS b_i16,
		  b::bigint           AS b_i64,
		  b::int4             AS b_i32,
		  d::tinyint          AS d_i8,
		  d::hugeint          AS d_i128,
		  a::int1             AS a_i8,
		  a::int2             AS a_i16,
		  a::int4             AS a_i32,
		  a::int8             AS a_i64
		FROM self
	`)
	// DISCREPANCY: 5.5 -> 6 (DuckDB rounds, polars truncates to 5).
	wantB := []any{int64(1), int64(2), int64(3), int64(4), int64(6)}
	eqRow(t, vals(t, out, "b_i8"), wantB, "b_i8 (DuckDB rounds)")
	eqRow(t, vals(t, out, "b_i16"), wantB, "b_i16")
	eqRow(t, vals(t, out, "b_i64"), wantB, "b_i64")
	eqRow(t, vals(t, out, "b_i32"), wantB, "b_i32")

	// booleans -> 0/1.
	wantD := []any{int64(1), int64(0), int64(1), int64(0), int64(1)}
	eqRow(t, vals(t, out, "d_i8"), wantD, "d_i8")
	eqRow(t, vals(t, out, "d_i128"), wantD, "d_i128 (hugeint normalized to int64)")

	wantA := []any{int64(1), int64(2), int64(3), int64(4), int64(5)}
	eqRow(t, vals(t, out, "a_i8"), wantA, "a_i8")
	eqRow(t, vals(t, out, "a_i16"), wantA, "a_i16")
	eqRow(t, vals(t, out, "a_i32"), wantA, "a_i32")
	eqRow(t, vals(t, out, "a_i64"), wantA, "a_i64")
}

// test_cast (unsigned-integer arm).
//
// DISCREPANCY: polars spells unsigned casts `uint1/uint2/uint4/uint8` and
// `<INT> UNSIGNED`; DuckDB uses `utinyint/usmallint/uinteger/ubigint` and rejects
// both polars spellings with a parser/catalog error. We assert the DuckDB spellings
// (all normalized to int64) and pin the rejections of the polars-only syntax.
func TestCastUnsignedVariants(t *testing.T) {
	d := castBaseFrame(t)

	// DuckDB unsigned type names work; results normalize to int64.
	out := query(t, d, `
		SELECT
		  d::utinyint  AS d_u8,
		  a::usmallint AS a_u16,
		  a::uinteger  AS a_u32,
		  d::ubigint   AS d_u64
		FROM self
	`)
	eqRow(t, vals(t, out, "d_u8"), []any{int64(1), int64(0), int64(1), int64(0), int64(1)}, "d_u8")
	eqRow(t, vals(t, out, "a_u16"), []any{int64(1), int64(2), int64(3), int64(4), int64(5)}, "a_u16")
	eqRow(t, vals(t, out, "a_u32"), []any{int64(1), int64(2), int64(3), int64(4), int64(5)}, "a_u32")
	eqRow(t, vals(t, out, "d_u64"), []any{int64(1), int64(0), int64(1), int64(0), int64(1)}, "d_u64")

	// b (float) -> unsigned int rounds, matching the signed arm (5.5 -> 6).
	outB := query(t, d, `SELECT b::usmallint AS b_u16, b::ubigint AS b_u64 FROM self`)
	wantB := []any{int64(1), int64(2), int64(3), int64(4), int64(6)}
	eqRow(t, vals(t, outB, "b_u16"), wantB, "b_u16 (DuckDB rounds)")
	eqRow(t, vals(t, outB, "b_u64"), wantB, "b_u64")

	// DISCREPANCY: polars-only spellings rejected by DuckDB.
	for _, q := range []string{
		`SELECT a::uint2 AS x FROM self`,                    // catalog: no type uint2
		`SELECT CAST(a AS TINYINT UNSIGNED) AS x FROM self`, // parser: UNSIGNED postfix
		`SELECT CAST(a AS BIGINT UNSIGNED) AS x FROM self`,
	} {
		if _, err := d.SQL(context.Background(), q); err == nil {
			t.Fatalf("expected DuckDB to reject polars-only unsigned syntax: %q", q)
		}
	}
}

// test_cast (string arm): CAST AS CHAR / VARCHAR / CHARACTER VARYING -> string.
// Floats stringify as "1.1"; booleans as "true"/"false". MATCH.
func TestCastStringVariants(t *testing.T) {
	d := castBaseFrame(t)
	out := query(t, d, `
		SELECT
		  CAST(a AS CHAR)             AS a_char,
		  CAST(b AS VARCHAR)          AS b_varchar,
		  CAST(d AS CHARACTER VARYING) AS d_charvar
		FROM self
	`)
	eqRow(t, vals(t, out, "a_char"), []any{"1", "2", "3", "4", "5"}, "a_char")
	eqRow(t, vals(t, out, "b_varchar"), []any{"1.1", "2.2", "3.3", "4.4", "5.5"}, "b_varchar")
	eqRow(t, vals(t, out, "d_charvar"), []any{"true", "false", "true", "false", "true"}, "d_charvar")
}

// test_cast (boolean arm): integer e -> bool (nonzero=true, 0=false, NULL passes
// through). MATCH.
func TestCastBooleanVariants(t *testing.T) {
	d := castBaseFrame(t)
	out := query(t, d, `SELECT e::bool AS e_bool, e::boolean AS e_boolean FROM self`)
	want := []any{true, false, nil, true, true}
	eqRow(t, vals(t, out, "e_bool"), want, "e_bool")
	eqRow(t, vals(t, out, "e_boolean"), want, "e_boolean")
}

// test_cast (binary arm): casting a string to a binary type now reads back as a
// gopolars Binary column ([]byte). MATCH.
//
// DISCREPANCY: polars' `::bytes` spelling is not a DuckDB type (Catalog Error,
// "did you mean bytea?"); DuckDB's BLOB aliases are blob/bytea/VARBINARY.
func TestCastBinaryVariants(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "c", Values: []any{"a", "b", "c", "d", "e"}})
	out := query(t, d, `SELECT c::blob AS c_blob, c::bytea AS c_bytea, c::VARBINARY AS c_varbinary FROM self`)
	want := []any{[]byte("a"), []byte("b"), []byte("c"), []byte("d"), []byte("e")}
	for _, name := range []string{"c_blob", "c_bytea", "c_varbinary"} {
		eqRow(t, vals(t, out, name), want, name)
	}
}

// test_cast (FORMAT arm): polars raises SQLInterfaceError 'use of FORMAT is not
// currently supported in CAST'. DuckDB also rejects `CAST(... AS STRING FORMAT
// 'HEX')`, but with a parser syntax error rather than the polars message. We only
// assert the error condition (DISCREPANCY: message differs).
func TestCastFormatRejected(t *testing.T) {
	d := castBaseFrame(t)
	if _, err := d.SQL(context.Background(), `SELECT CAST(a AS STRING FORMAT 'HEX') FROM self`); err == nil {
		t.Fatalf("expected CAST ... FORMAT to be rejected")
	}
}

// test_cast_json: parse a JSON string into a Struct with nested-list/scalar fields.
//
// DISCREPANCY: polars' `txt::json` / CAST AS JSON auto-infers the struct schema; in
// DuckDB `::json` yields VARCHAR, so the JSON is cast to an explicit STRUCT(...) target
// type, which reads back as a gopolars Struct. MATCH on the parsed fields.
func TestCastJSON(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "txt", Values: []any{
		`{"a":[1,2,3],"b":["x","y","z"],"c":5.0}`,
	}})
	out := query(t, d, `SELECT txt::STRUCT(a INTEGER[], b VARCHAR[], c DOUBLE) AS j FROM self`)
	j, ok := col(t, out, "j").Value(0).(map[string]any)
	if !ok {
		t.Fatalf("j is not a struct: %#v", col(t, out, "j").Value(0))
	}
	if !reflect.DeepEqual(j["a"], []any{int64(1), int64(2), int64(3)}) {
		t.Fatalf("j.a = %#v, want [1 2 3]", j["a"])
	}
	if !reflect.DeepEqual(j["b"], []any{"x", "y", "z"}) {
		t.Fatalf("j.b = %#v, want [x y z]", j["b"])
	}
	if j["c"] != float64(5) {
		t.Fatalf("j.c = %#v, want 5.0", j["c"])
	}
}

// test_cast_errors: out-of-range / unparseable casts.
//
// MATCH (condition): an invalid hard CAST raises an error, while TRY_CAST returns
// NULL for the offending value. polars and DuckDB agree on the CONDITION (error
// vs null); only the error message text differs (DISCREPANCY on message, not
// asserted). The date/time TRY_CAST cases additionally hit the arrow date/time
// bridge GAP, so they're covered separately below.
func TestCastErrors(t *testing.T) {
	type tc struct {
		values  []any
		typ     string // DuckDB type name for the ::cast
		errCast string // expression that must hard-error
	}
	cases := []tc{
		// f64 -1.0 -> u8 out of range.
		{values: []any{1.0, -1.0}, typ: "utinyint", errCast: "values::utinyint"},
		// i64 -1 -> u32 out of range.
		{values: []any{int64(10), int64(0), int64(-1)}, typ: "uinteger", errCast: "values::uinteger"},
		// i64 1e8 -> i8 out of range.
		{values: []any{int64(100000000)}, typ: "int1", errCast: "values::int1"},
		// str -> i32 unparseable.
		{values: []any{"a", "b"}, typ: "int4", errCast: "values::int4"},
	}
	for _, c := range cases {
		d := mustFrame(t, frame.SeriesInput{Name: "values", Values: c.values})

		// hard cast errors.
		if _, err := d.SQL(context.Background(), `SELECT `+c.errCast+` AS x FROM self`); err == nil {
			// Some engines defer to collect; treat a clean LazyFrame as a failure
			// only if collect also succeeds.
			lf, _ := d.SQL(context.Background(), `SELECT `+c.errCast+` AS x FROM self`)
			if lf != nil {
				if _, cerr := lf.Collect(context.Background()); cerr == nil {
					t.Fatalf("expected hard cast %q to error", c.errCast)
				}
			}
		}

		// TRY_CAST returns NULL for the bad value(s).
		out := query(t, d, `SELECT TRY_CAST(values AS `+c.typ+`) AS cast_values FROM self`)
		s := col(t, out, "cast_values")
		sawNull := false
		for i := 0; i < s.Len(); i++ {
			if s.Value(i) == nil {
				sawNull = true
				break
			}
		}
		if !sawNull {
			t.Fatalf("TRY_CAST(values AS %s) expected at least one NULL", c.typ)
		}
	}
}

// test_cast_errors (date/time arm): str->date and str->time. The hard CAST errors
// (MATCH on condition), and TRY_CAST of the unparseable strings now yields NULLs —
// the date32/time64 result column reads back (as Datetime; see DISCREPANCY notes on
// TestTemporalDateResult / TestTemporalDatetimeToTime).
func TestCastErrorsDateTime(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "values", Values: []any{"a", "b"}})

	for _, expr := range []string{`values::date`, `values::time`} {
		lf, err := d.SQL(context.Background(), `SELECT `+expr+` AS x FROM self`)
		if err == nil && lf != nil {
			if _, cerr := lf.Collect(context.Background()); cerr == nil {
				t.Fatalf("expected hard cast %q to error", expr)
			}
		}
	}
	for _, typ := range []string{"date", "time"} {
		out := query(t, d, `SELECT TRY_CAST(values AS `+typ+`) AS x FROM self`)
		eqRow(t, vals(t, out, "x"), []any{nil, nil}, "TRY_CAST AS "+typ)
	}
}

// floatRowCast compares a float column against want within tolerance, allowing
// NULLs in want via NaN sentinel handling at the call site.
func floatRowCast(t *testing.T, d polars.DataFrame, name string, want []float64) {
	t.Helper()
	s := col(t, d, name)
	if s.Len() != len(want) {
		t.Fatalf("%s len = %d, want %d", name, s.Len(), len(want))
	}
	for i, w := range want {
		got := toFloat(s.Value(i))
		if math.Abs(got-w) > 1e-6 {
			t.Fatalf("%s[%d] = %v, want %v", name, i, got, w)
		}
	}
}
