//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_array.py (py-1.28.1).
//
// gopolars represents List/Array columns via its boxed List dtype, and the Arrow
// bridge round-trips Arrow List <-> gopolars List (both result and input columns).
// Cases that exercise polars-SQL-only constructs DuckDB does not implement
// (multi-array UNNEST table function, ARRAY_GET, LIMIT-in-aggregate, 3-arg
// ARRAY_TO_STRING) stay GAP/DISCREPANCY. DuckDB-backed dialect notes are tagged
// // DISCREPANCY where DuckDB diverges from polars.
package sql

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// test_array_agg: ARRAY_AGG(col2 ORDER BY col0 {dir} [LIMIT n]) GROUP BY col1.
// The result `arrs` is now a real List(String) column (Arrow List → gopolars List).
//
// DISCREPANCY: DuckDB rejects LIMIT inside an aggregate's argument list
// ("ARRAY_AGG(... LIMIT n)" → Parser/Binder error), so py's LIMIT parametrizations
// are exercised as expected-errors; the ORDER BY arms MATCH polars.
func TestArrayAgg(t *testing.T) {
	const data = `
		WITH data (col0, col1, col2) AS (
		  VALUES (1,'a','x'),(2,'a','y'),(4,'b','z'),(8,'b','X'),(7,'b','Y')
		)
		SELECT col1, ARRAY_AGG(col2%s) AS arrs FROM data GROUP BY col1 ORDER BY col1`
	d := mustFrame(t, frame.SeriesInput{Name: "k", Values: []any{int64(1)}})

	// (sort_order, expected per group a,b) — the deterministic ORDER BY arms.
	ordered := []struct {
		inner string
		a, b  []any
	}{
		{" ORDER BY col0 ASC", []any{"x", "y"}, []any{"z", "Y", "X"}},
		{" ORDER BY col0 DESC", []any{"y", "x"}, []any{"X", "Y", "z"}},
	}
	for _, c := range ordered {
		out := query(t, d, fmt.Sprintf(data, c.inner))
		arrs := col(t, out, "arrs")
		if !reflect.DeepEqual(arrs.Value(0), c.a) || !reflect.DeepEqual(arrs.Value(1), c.b) {
			t.Fatalf("ARRAY_AGG%s = [%v, %v], want [%v, %v]", c.inner, arrs.Value(0), arrs.Value(1), c.a, c.b)
		}
	}

	// No ORDER BY: DuckDB aggregates in scan (insertion) order here → matches polars.
	out := query(t, d, fmt.Sprintf(data, ""))
	arrs := col(t, out, "arrs")
	if !reflect.DeepEqual(arrs.Value(0), []any{"x", "y"}) || !reflect.DeepEqual(arrs.Value(1), []any{"z", "X", "Y"}) {
		t.Fatalf("ARRAY_AGG (no order) = [%v, %v]", arrs.Value(0), arrs.Value(1))
	}

	// DISCREPANCY: LIMIT inside the aggregate argument is a polars-SQL extension
	// DuckDB does not parse.
	if _, err := d.SQL(context.Background(), fmt.Sprintf(data, " ORDER BY col0 ASC LIMIT 2")); err == nil {
		t.Fatalf("expected DuckDB to reject LIMIT inside ARRAY_AGG")
	}
}

// test_array_literals: list literals + ARRAY_AGG/ARRAY_CONTAINS/ARRAY_REVERSE over a
// single row. List result columns now read back as gopolars List columns.
//
// DISCREPANCY: polars-sql treats a SELECT mixing plain columns with ARRAY_AGG as one
// implicit group; DuckDB requires the plain columns to be grouped, so `GROUP BY ALL`
// is added. With a single source row the result MATCHes polars either way.
func TestArrayLiterals(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "k", Values: []any{int64(1)}})
	out := query(t, d, `
		SELECT a1, a2,
		  ARRAY_AGG(a1) AS a3,
		  ARRAY_AGG(a2) AS a4,
		  ARRAY_CONTAINS(a1,20) AS i20,
		  ARRAY_CONTAINS(a2,'zz') AS izz,
		  ARRAY_REVERSE(a1) AS ar1,
		  ARRAY_REVERSE(a2) AS ar2
		FROM (SELECT [10,20,30] AS a1, ['a','b','c'] AS a2 FROM self) tbl
		GROUP BY ALL`)

	check := func(name string, want any) {
		got := col(t, out, name).Value(0)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
	check("a1", []any{int64(10), int64(20), int64(30)})
	check("a2", []any{"a", "b", "c"})
	check("a3", []any{[]any{int64(10), int64(20), int64(30)}})
	check("a4", []any{[]any{"a", "b", "c"}})
	check("ar1", []any{int64(30), int64(20), int64(10)})
	check("ar2", []any{"c", "b", "a"})
	if col(t, out, "i20").Value(0) != true {
		t.Fatalf("ARRAY_CONTAINS(a1,20) = %v, want true", col(t, out, "i20").Value(0))
	}
	if col(t, out, "izz").Value(0) != false {
		t.Fatalf("ARRAY_CONTAINS(a2,'zz') = %v, want false", col(t, out, "izz").Value(0))
	}
}

// test_array_indexing: arr[i] over a list literal. The list is only an intermediate;
// the RESULT column is a scalar int (NULL when the index is out of range, including
// index 0). Scalar results → portable for the subscript arm.
//
// DISCREPANCY: py-polars also exposes ARRAY_GET(arr, i); DuckDB has no array_get
// scalar function ("Catalog Error: ... array_get does not exist"), so that arm is a
// GAP and only the `arr[i]` subscript is asserted.
//
// py covers indices -4..4 over [99,66,33] with expected
// [None,99,66,33,None,99,66,33,None] (index 0 → NULL, negatives count from the end).
// DuckDB's 1-based list subscript agrees on this list.
func TestArrayIndexing(t *testing.T) {
	type tc struct {
		idx  int
		want any // int64 or nil
	}
	cases := []tc{
		{-4, nil}, {-3, int64(99)}, {-2, int64(66)}, {-1, int64(33)},
		{0, nil}, {1, int64(99)}, {2, int64(66)}, {3, int64(33)}, {4, nil},
	}
	d := mustFrame(t, frame.SeriesInput{Name: "k", Values: []any{int64(1)}})
	for _, c := range cases {
		q := "SELECT arr[idxv] AS idx1 FROM (SELECT [99,66,33] AS arr, CAST(" +
			itoa(c.idx) + " AS BIGINT) AS idxv) tbl"
		out := query(t, d, q)
		got1 := col(t, out, "idx1").Value(0)
		if !eqScalar(got1, c.want) {
			t.Fatalf("arr[%d] idx1 = %v, want %v", c.idx, got1, c.want)
		}
	}
}

// test_array_indexing_by_expr: arr[idx] over a per-row LIST input column and a per-row
// int index. List input columns now materialize into the engine as real Arrow lists.
//
// DISCREPANCY: py also covers ARRAY_GET(arr, idx); DuckDB has no array_get scalar
// (Catalog Error), so only the `arr[idx]` subscript is asserted. py expected
// [2,5,None,None,8,5,1] (1-based; idx 0/NULL → NULL); DuckDB's list subscript MATCHes.
func TestArrayIndexingByExpr(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "idx", Values: []any{int64(-2), int64(-1), int64(0), nil, int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "arr", Values: []any{
			[]any{int64(0), int64(1), int64(2), int64(3)},
			[]any{int64(4), int64(5)},
			[]any{int64(6)},
			[]any{int64(7), int64(8), int64(9)},
			[]any{int64(8), int64(7)},
			[]any{int64(6), int64(5), int64(4)},
			[]any{int64(3), int64(2), int64(1)},
		}},
	)
	out := query(t, d, `SELECT arr[idx] AS idx1 FROM self`)
	eqRow(t, vals(t, out, "idx1"), []any{int64(2), int64(5), nil, nil, int64(8), int64(5), int64(1)}, "arr[idx]")

	// DISCREPANCY: ARRAY_GET is a polars-SQL scalar absent from DuckDB.
	if _, err := d.SQL(context.Background(), `SELECT ARRAY_GET(arr, idx) FROM self`); err == nil {
		t.Fatalf("expected ARRAY_GET to be unsupported in DuckDB")
	}
}

// test_array_to_string: ARRAY_TO_STRING(list, sep) over LIST input columns. List input
// columns now materialize as real Arrow lists, so the 2-arg form MATCHes polars (both
// skip NULL elements).
//
// DISCREPANCY: py also covers the 3-arg form ARRAY_TO_STRING(list, sep, null_repr);
// DuckDB's array_to_string is a 2-arg macro and rejects the 3-arg null-replacement form.
func TestArrayToString(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "s_values", Values: []any{
			[]any{"aa", "bb"}, []any{nil, "cc"}, []any{"dd", nil},
		}},
		frame.SeriesInput{Name: "n_values", Values: []any{
			[]any{int64(999), int64(777)}, []any{nil, int64(555)}, []any{int64(333), nil},
		}},
	)
	out := query(t, d, `
		SELECT
		  ARRAY_TO_STRING(s_values, '')  AS vs1,
		  ARRAY_TO_STRING(s_values, ':') AS vs2,
		  ARRAY_TO_STRING(n_values, '')  AS vn1,
		  ARRAY_TO_STRING(n_values, ':') AS vn2
		FROM self`)
	eqRow(t, vals(t, out, "vs1"), []any{"aabb", "cc", "dd"}, "vs1")
	eqRow(t, vals(t, out, "vs2"), []any{"aa:bb", "cc", "dd"}, "vs2")
	eqRow(t, vals(t, out, "vn1"), []any{"999777", "555", "333"}, "vn1")
	eqRow(t, vals(t, out, "vn2"), []any{"999:777", "555", "333"}, "vn2")

	// DISCREPANCY: 3-arg ARRAY_TO_STRING (null replacement) is polars-SQL only.
	if _, err := d.SQL(context.Background(), `SELECT ARRAY_TO_STRING(s_values, ':', 'NA') FROM self`); err == nil {
		t.Fatalf("expected 3-arg ARRAY_TO_STRING to be unsupported in DuckDB")
	}
}

// test_unnest_table_function: zip parallel arrays into columns x,y,z.
//
// DISCREPANCY: polars' multi-array UNNEST *table function* `FROM UNNEST(a,b,c) AS
// tbl(x,y,z)` is not a DuckDB construct (its UNNEST table function takes one arg);
// DuckDB instead zips multiple UNNEST() calls in the SELECT list by position, which
// produces the same result. MATCH on values.
func TestArrayUnnestTableFunction(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "k", Values: []any{int64(1)}})
	// DISCREPANCY: decimal literals (23.0, …) are FLOAT in polars but DECIMAL in DuckDB
	// (a gopolars Decimal — see TestNumericDecimalType); the z list is cast to DOUBLE[]
	// to assert polars' float semantics.
	out := query(t, d, `
		SELECT
		  UNNEST([1,2,3,4])                       AS x,
		  UNNEST(['ww','xx','yy','zz'])            AS y,
		  UNNEST([23.0,24.5,28.0,27.5]::DOUBLE[])  AS z`)
	eqRow(t, vals(t, out, "x"), []any{int64(1), int64(2), int64(3), int64(4)}, "x")
	eqRow(t, vals(t, out, "y"), []any{"ww", "xx", "yy", "zz"}, "y")
	eqRow(t, vals(t, out, "z"), []any{23.0, 24.5, 28.0, 27.5}, "z")
}

// test_unnest_table_function_errors: polars raises specific diagnostics for malformed
// UNNEST table-function aliases.
//
// DISCREPANCY: DuckDB has no such alias-validation surface — it ACCEPTS the forms polars
// rejects: `UNNEST([..]) AS "frame data"` (alias, no column names), `AS tbl(a,b)` (extra
// column name), and a bare `UNNEST([..])` (auto-named column). The two forms DuckDB does
// reject (for its own reasons) are asserted: a multi-array UNNEST table function (its
// unnest takes a single arg) and `WITH OFFSET` (unsupported syntax).
func TestArrayUnnestTableFunctionErrors(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "k", Values: []any{int64(1)}})
	for _, q := range []string{
		`SELECT * FROM UNNEST([1,2,3], [3,4,5]) AS tbl (a)`,
		`SELECT * FROM UNNEST([1, 2, 3]) tbl (colx) WITH OFFSET`,
	} {
		if runErr(t, d, q) == nil {
			t.Fatalf("expected DuckDB to reject %q", q)
		}
	}
}

// eqScalar compares two scalar values (handles nil and int64) for these tests.
func eqScalar(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a == b
}
