//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_wildcard_opts.py (py-1.28.1).
//
// gopolars runs SQL through an embedded DuckDB engine (build -tags "duckdb
// duckdb_arrow"). Wildcard options (* EXCLUDE / * REPLACE / * RENAME / ILIKE /
// COLUMNS) are a DuckDB feature, so these largely MATCH. Divergences from the
// polars-sql dialect (EXCEPT-as-exclude, ILIKE+RENAME combos, REPLACE+RENAME on
// the same column, empty-projection handling) are pinned with // DISCREPANCY:.
package sql

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func wildcardDF(t *testing.T) polars.DataFrame {
	t.Helper()
	return mustFrame(t,
		frame.SeriesInput{Name: "ID", Values: []any{int64(333), int64(666), int64(999)}},
		frame.SeriesInput{Name: "FirstName", Values: []any{"Bruce", "Diana", "Clark"}},
		frame.SeriesInput{Name: "LastName", Values: []any{"Wayne", "Prince", "Kent"}},
		frame.SeriesInput{Name: "Address", Values: []any{"Batcave", "Paradise Island", "Fortress of Solitude"}},
		frame.SeriesInput{Name: "City", Values: []any{"Gotham", "Themyscira", "Metropolis"}},
	)
}

func colsEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// test_select_exclude: `* EXCLUDE (...)` keeps the remaining columns. polars also
// accepts the `EXCEPT` keyword in this position; DuckDB does NOT (EXCEPT is the
// set operator) — see TestWildcardExceptKeywordUnsupported.
func TestWildcardSelectExclude(t *testing.T) {
	df := wildcardDF(t)
	cases := []struct {
		excluded string
		expected []string
	}{
		{"ID", []string{"FirstName", "LastName", "Address", "City"}},
		{"(ID)", []string{"FirstName", "LastName", "Address", "City"}},
		{"(Address, LastName, FirstName)", []string{"ID", "City"}},
	}
	for _, c := range cases {
		out := query(t, df, "SELECT * EXCLUDE "+c.excluded+" FROM self")
		if !colsEqual(out.Columns(), c.expected) {
			t.Fatalf("EXCLUDE %s: cols=%v want=%v", c.excluded, out.Columns(), c.expected)
		}
	}
}

// DISCREPANCY: polars yields an empty (0-column) frame when every column is
// excluded; DuckDB raises a Binder Error ("SELECT list is empty after resolving
// * expressions"). We assert the DuckDB error.
func TestWildcardExcludeAllErrors(t *testing.T) {
	df := wildcardDF(t)
	q := `SELECT * EXCLUDE ("ID","FirstName","LastName","Address","City") FROM self`
	if _, err := df.SQL(t.Context(), q); err == nil {
		t.Fatalf("excluding all columns: expected Binder Error, got nil")
	}
}

// DISCREPANCY: polars accepts EXCLUDE and EXCEPT as aliases in the wildcard
// position; DuckDB only accepts EXCLUDE there and treats `* EXCEPT ...` as a
// parser error.
func TestWildcardExceptKeywordUnsupported(t *testing.T) {
	df := wildcardDF(t)
	if _, err := df.SQL(t.Context(), "SELECT * EXCEPT (ID) FROM self"); err == nil {
		t.Fatalf("* EXCEPT (ID): expected parser error in DuckDB dialect, got nil")
	}
}

// test_select_exclude_order_by: EXCLUDE plus ORDER BY (ordinal and named).
func TestWildcardExcludeOrderBy(t *testing.T) {
	df := wildcardDF(t)
	wantFirst := []any{"Diana", "Clark", "Bruce"}
	wantAddr := []any{"Paradise Island", "Fortress of Solitude", "Batcave"}
	for _, orderBy := range []string{"ORDER BY 1 DESC", "ORDER BY 2 DESC", "ORDER BY Address DESC"} {
		out := query(t, df, "SELECT * EXCLUDE (ID,LastName,City) FROM self "+orderBy)
		if !colsEqual(out.Columns(), []string{"FirstName", "Address"}) {
			t.Fatalf("%s: cols=%v", orderBy, out.Columns())
		}
		// Both columns sort to the same row order here (FirstName DESC == Address DESC).
		eqRow(t, vals(t, out, "FirstName"), wantFirst, "FirstName "+orderBy)
		eqRow(t, vals(t, out, "Address"), wantAddr, "Address "+orderBy)
	}
}

// test_ilike: `* ILIKE 'pattern'` keeps columns whose *name* matches the pattern.
func TestWildcardILike(t *testing.T) {
	df := wildcardDF(t)
	cases := []struct {
		pattern  string
		expected []string
	}{
		{"a%e", nil}, // matches no column name -> empty set (DuckDB: Binder Error, see below)
		{"%nam_", []string{"FirstName", "LastName"}},
		{"%a%e%", []string{"FirstName", "LastName", "Address"}},
	}
	for _, c := range cases {
		q := "SELECT * ILIKE '" + c.pattern + "' FROM self"
		if c.expected == nil {
			// DISCREPANCY: polars returns an empty frame when the pattern matches no
			// column name; DuckDB raises an empty-set error instead.
			if err := runErr(t, df, q); err == nil {
				t.Fatalf("ILIKE '%s': expected empty-set error, got nil", c.pattern)
			}
			continue
		}
		out := query(t, df, q)
		if !colsEqual(out.Columns(), c.expected) {
			t.Fatalf("ILIKE '%s': cols=%v want=%v", c.pattern, out.Columns(), c.expected)
		}
	}
}

// runErr runs a query through SQL+Collect and returns the first error (or nil).
func runErr(t *testing.T, d polars.DataFrame, q string) error {
	t.Helper()
	lf, err := d.SQL(t.Context(), q)
	if err != nil {
		return err
	}
	_, err = lf.Collect(t.Context())
	return err
}

// DISCREPANCY: polars allows `* ILIKE '...' RENAME (...)`; DuckDB rejects the
// combination with a parser error.
func TestWildcardILikeRenameUnsupported(t *testing.T) {
	df := wildcardDF(t)
	q := `SELECT * ILIKE '%I%' RENAME (FirstName AS Name) FROM self`
	if _, err := df.SQL(t.Context(), q); err == nil {
		t.Fatalf("ILIKE + RENAME: expected parser error in DuckDB, got nil")
	}
}

// test_select_rename: `* RENAME (...)` renames columns in place, preserving order.
func TestWildcardSelectRename(t *testing.T) {
	df := wildcardDF(t)
	cases := []struct {
		renames  string
		expected []string
	}{
		{"(Address AS Location)", []string{"ID", "FirstName", "LastName", "Location", "City"}},
		{`("Address" AS Location, "ID" AS PersonID)`, []string{"PersonID", "FirstName", "LastName", "Location", "City"}},
	}
	for _, c := range cases {
		out := query(t, df, "SELECT * RENAME "+c.renames+" FROM self")
		if !colsEqual(out.Columns(), c.expected) {
			t.Fatalf("RENAME %s: cols=%v want=%v", c.renames, out.Columns(), c.expected)
		}
	}
}

// test_select_rename_exclude_sort: EXCLUDE + RENAME (single, unparenthesized) +
// ORDER BY.
func TestWildcardExcludeRenameSort(t *testing.T) {
	df := wildcardDF(t)
	for _, orderBy := range []string{"1 DESC", "Name DESC", "FirstName DESC"} {
		// ORDER BY references must use the renamed/positional form that DuckDB
		// resolves; "FirstName DESC" still resolves to the source column.
		out := query(t, df, "SELECT * EXCLUDE (ID, City, LastName) RENAME FirstName AS Name FROM self ORDER BY "+orderBy)
		if !colsEqual(out.Columns(), []string{"Name", "Address"}) {
			t.Fatalf("ORDER BY %s: cols=%v", orderBy, out.Columns())
		}
		eqRow(t, vals(t, out, "Name"), []any{"Diana", "Clark", "Bruce"}, "Name ORDER BY "+orderBy)
		eqRow(t, vals(t, out, "Address"), []any{"Paradise Island", "Fortress of Solitude", "Batcave"}, "Address ORDER BY "+orderBy)
	}
}

// test_select_replace: `* REPLACE (expr AS col)` swaps a column's values while
// keeping its position. // floor-div `//` on integers behaves as polars expects.
func TestWildcardSelectReplace(t *testing.T) {
	df := wildcardDF(t)

	// (ID // 3 AS ID)
	out := query(t, df, "SELECT * REPLACE (ID // 3 AS ID) FROM self ORDER BY ID DESC")
	if !colsEqual(out.Columns(), []string{"ID", "FirstName", "LastName", "Address", "City"}) {
		t.Fatalf("REPLACE ID//3: cols=%v", out.Columns())
	}
	eqRow(t, vals(t, out, "ID"), []any{int64(333), int64(222), int64(111)}, "ID//3")

	// ((City || ':' || City) AS City, ID // -3 AS ID)
	out2 := query(t, df, "SELECT * REPLACE ((City || ':' || City) AS City, ID // -3 AS ID) FROM self ORDER BY ID DESC")
	eqRow(t, vals(t, out2, "ID"), []any{int64(-111), int64(-222), int64(-333)}, "ID//-3")
	eqRow(t, vals(t, out2, "City"),
		[]any{"Gotham:Gotham", "Themyscira:Themyscira", "Metropolis:Metropolis"}, "City concat")
}

// DISCREPANCY: polars allows REPLACE and RENAME to reference the same column
// (e.g. `REPLACE (ID // 3 AS ID) RENAME (ID AS Identifier)`); DuckDB rejects it
// with "Column ... cannot occur in both REPLACE and RENAME list".
func TestWildcardReplaceRenameSameColumnUnsupported(t *testing.T) {
	df := wildcardDF(t)
	q := "SELECT * REPLACE (ID // 3 AS ID) RENAME (ID AS Identifier) FROM self"
	if _, err := df.SQL(t.Context(), q); err == nil {
		t.Fatalf("REPLACE+RENAME same column: expected parser error, got nil")
	}
}

// test_select_wildcard_errors: invalid wildcard option combinations.
//   - EXCLUDE + ILIKE together: polars raises SQLInterfaceError("ILIKE"); DuckDB
//     resolves to an empty column set and raises a Binder Error. Both error.
func TestWildcardExcludeIlikeTogetherErrors(t *testing.T) {
	df := wildcardDF(t)
	if _, err := df.SQL(t.Context(), "SELECT * EXCLUDE Address ILIKE '%o%' FROM self"); err == nil {
		t.Fatalf("EXCLUDE + ILIKE: expected error, got nil")
	}
}

// COLUMNS(...) selector — a DuckDB wildcard feature polars does not expose via
// SQL. Both the regex form and the lambda form select by column name. This is an
// EXTRA capability beyond the py test (// DISCREPANCY: DuckDB-only feature).
func TestWildcardColumnsSelector(t *testing.T) {
	df := wildcardDF(t)
	out := query(t, df, `SELECT COLUMNS('Name$') FROM self`)
	if !colsEqual(out.Columns(), []string{"FirstName", "LastName"}) {
		t.Fatalf("COLUMNS('Name$'): cols=%v", out.Columns())
	}
	out2 := query(t, df, `SELECT COLUMNS(c -> c LIKE '%Name') FROM self`)
	if !colsEqual(out2.Columns(), []string{"FirstName", "LastName"}) {
		t.Fatalf("COLUMNS(lambda): cols=%v", out2.Columns())
	}
}
