//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_bitwise.py (py-1.28.1).
//
// DuckDB-backed (build -tags "duckdb duckdb_arrow"). The bitwise operators &, |
// and the xor() function are row-wise in DuckDB and MATCH polars on values.
//
// DISCREPANCY: polars exposes 2-arg row-wise BIT_AND(a,b)/BIT_OR(a,b)/BIT_XOR(a,b)
// plus the short BITAND/BITOR/BITXOR aliases and a 2-arg-friendly count. In DuckDB
// BIT_AND/BIT_OR/BIT_XOR are *aggregate* functions taking a single column, so the
// polars 2-arg calls are rejected — pinned via the error assertions below. The
// `XOR` keyword is not a DuckDB operator either; DuckDB spells bitwise xor as the
// xor(a,b) function. BIT_COUNT is a 1-arg scalar in both and MATCHes for the
// non-negative column.
package sql

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func bitwiseFrame(t *testing.T) polars.DataFrame {
	t.Helper()
	return mustFrame(t,
		frame.SeriesInput{Name: "x", Values: []any{int64(20), int64(32), int64(50), int64(88), int64(128)}},
		frame.SeriesInput{Name: "y", Values: []any{int64(-128), int64(0), int64(10), int64(-1), nil}},
		frame.SeriesInput{Name: "idx", Values: []any{int64(0), int64(1), int64(2), int64(3), int64(4)}},
	)
}

// test_bitwise_and: `x & y` operator. MATCH on values.
// DISCREPANCY: the polars 2-arg BIT_AND(x,y)/BITAND(x,y) calls are rejected by
// DuckDB (BIT_AND there is a single-column aggregate).
func TestBitwiseAnd(t *testing.T) {
	d := bitwiseFrame(t)
	out := query(t, d, `SELECT x & y AS x_bitand_op_y FROM self ORDER BY idx`)
	eqRow(t, vals(t, out, "x_bitand_op_y"),
		[]any{int64(0), int64(0), int64(2), int64(88), nil}, "x & y")

	// DISCREPANCY: DuckDB has no 2-arg BIT_AND / BITAND row-wise function.
	if _, err := d.SQL(context.Background(), `SELECT BIT_AND(x, y) FROM self`); err == nil {
		t.Fatalf("expected DuckDB to reject 2-arg BIT_AND(x, y)")
	}
	if _, err := d.SQL(context.Background(), `SELECT BITAND(y, x) FROM self`); err == nil {
		t.Fatalf("expected DuckDB to reject polars-only BITAND() alias")
	}
}

// test_bitwise_count: BIT_COUNT(col) is a 1-arg scalar in DuckDB (matches the
// polars BITCOUNT/BIT_COUNT shape). For the non-negative x column, MATCH.
//
// DISCREPANCY: for negative/NULL y, DuckDB's BIT_COUNT counts set bits of the
// two's-complement 64-bit representation; values pinned to DuckDB's actual
// output. The short BITCOUNT() alias is polars-only and rejected.
func TestBitwiseCount(t *testing.T) {
	d := bitwiseFrame(t)
	out := query(t, d, `
		SELECT
		  BIT_COUNT(x) AS x_bits_set,
		  BIT_COUNT(y) AS y_bits_set
		FROM self
		ORDER BY idx
	`)
	// x non-negative → MATCH polars [2, 1, 3, 3, 1].
	eqRow(t, vals(t, out, "x_bits_set"),
		[]any{int64(2), int64(1), int64(3), int64(3), int64(1)}, "BIT_COUNT(x)")
	// y discovered from DuckDB (two's-complement 64-bit popcount; NULL passthru).
	eqRow(t, vals(t, out, "y_bits_set"),
		[]any{int64(57), int64(0), int64(2), int64(64), nil}, "BIT_COUNT(y)")

	// DISCREPANCY: BITCOUNT() (no underscore) is polars-only — DuckDB rejects it.
	if _, err := d.SQL(context.Background(), `SELECT BITCOUNT(x) FROM self`); err == nil {
		t.Fatalf("expected DuckDB to reject polars-only BITCOUNT() alias")
	}
}

// test_bitwise_or: `x | y` operator. MATCH on values.
// DISCREPANCY: 2-arg BIT_OR(x,y)/BITOR(x,y) rejected (BIT_OR is an aggregate).
func TestBitwiseOr(t *testing.T) {
	d := bitwiseFrame(t)
	out := query(t, d, `SELECT x | y AS x_bitor_op_y FROM self ORDER BY idx`)
	eqRow(t, vals(t, out, "x_bitor_op_y"),
		[]any{int64(-108), int64(32), int64(58), int64(-1), nil}, "x | y")

	if _, err := d.SQL(context.Background(), `SELECT BIT_OR(x, y) FROM self`); err == nil {
		t.Fatalf("expected DuckDB to reject 2-arg BIT_OR(x, y)")
	}
	if _, err := d.SQL(context.Background(), `SELECT BITOR(y, x) FROM self`); err == nil {
		t.Fatalf("expected DuckDB to reject polars-only BITOR() alias")
	}
}

// test_bitwise_xor: polars uses the `XOR` keyword and BIT_XOR(a,b)/BITXOR(a,b).
//
// DISCREPANCY: DuckDB has no `XOR` operator keyword and no 2-arg BIT_XOR
// (aggregate-only). DuckDB spells row-wise bitwise xor as the xor(a,b) function,
// which MATCHes polars on values. The polars syntaxes are asserted to error.
func TestBitwiseXor(t *testing.T) {
	d := bitwiseFrame(t)
	want := []any{int64(-108), int64(32), int64(56), int64(-89), nil}

	// DuckDB's row-wise bitwise xor is the xor(a,b) function → MATCH on values.
	out := query(t, d, `SELECT xor(x, y) AS x_bitxor_fn FROM self ORDER BY idx`)
	eqRow(t, vals(t, out, "x_bitxor_fn"), want, "xor(x, y)")

	// DISCREPANCY: the polars `XOR` keyword is not a DuckDB operator.
	if _, err := d.SQL(context.Background(), `SELECT x XOR y FROM self`); err == nil {
		t.Fatalf("expected DuckDB to reject the XOR keyword operator")
	}
	// DISCREPANCY: 2-arg BIT_XOR(x,y)/BITXOR(x,y) rejected (BIT_XOR is an aggregate).
	if _, err := d.SQL(context.Background(), `SELECT BIT_XOR(x, y) FROM self`); err == nil {
		t.Fatalf("expected DuckDB to reject 2-arg BIT_XOR(x, y)")
	}
	if _, err := d.SQL(context.Background(), `SELECT BITXOR(y, x) FROM self`); err == nil {
		t.Fatalf("expected DuckDB to reject polars-only BITXOR() alias")
	}
}
