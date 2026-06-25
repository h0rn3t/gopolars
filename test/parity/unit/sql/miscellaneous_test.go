//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_miscellaneous.py (py-1.28.1).
//
// NOTE: the DISTINCT and COUNT(DISTINCT ...) cases from this source file (test_distinct
// and the count-distinct arms of test_count) are already ported in distinct_test.go and
// are deliberately NOT re-ported here. Func names are prefixed TestMisc... to avoid any
// collision with sibling files.
//
// gopolars runs SQL through embedded DuckDB (build -tags "duckdb duckdb_arrow"), so
// behavior is measured against DuckDB's dialect. polars-only SQLContext APIs
// (register_globals/tables/unregister-as-context-manager), pandas/pyarrow frame
// interop, file I/O (read_csv), and list-producing UNNEST projections are GAPs.
package sql

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_any_all: polars supports `x >= ALL(df.y)` / `x >= ANY(df.y)` quantified
// comparisons against a column. This is polars-native SQL syntax; DuckDB does not
// accept `>= ALL(<column reference>)` (ALL/ANY require a subquery/array). gopolars
// (DuckDB) rejects it → GAP.
func TestMiscAnyAll(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "x", Values: []any{int64(-1), int64(0), int64(1), int64(2), int64(3), int64(4)}},
		frame.SeriesInput{Name: "y", Values: []any{int64(1), int64(0), int64(0), int64(1), int64(2), int64(3)}},
	)
	// DISCREPANCY: (1) polars' `x >= ALL(df.y)` column form is written as a scalar
	// subquery `ALL(SELECT y FROM self)` in the DuckDB dialect, and `==` → `=`;
	// (2) polars' `!= ANY` means "differs from every value" (NOT IN), so it is
	// written as DuckDB `!= ALL` (SQL's `!= ANY` is the existential ∃, which differs).
	// The boolean results MATCH polars (polars renders them as 0/1).
	out := query(t, d, `
		SELECT
		  x >= ALL(SELECT y FROM self) AS a_geq,
		  x  > ALL(SELECT y FROM self) AS a_g,
		  x  < ALL(SELECT y FROM self) AS a_l,
		  x <= ALL(SELECT y FROM self) AS a_leq,
		  x >= ANY(SELECT y FROM self) AS n_geq,
		  x  > ANY(SELECT y FROM self) AS n_g,
		  x  < ANY(SELECT y FROM self) AS n_l,
		  x <= ANY(SELECT y FROM self) AS n_leq,
		  x  = ANY(SELECT y FROM self) AS n_eq,
		  x != ALL(SELECT y FROM self) AS n_neq
		FROM self ORDER BY x`)
	bs := func(v ...bool) []any {
		o := make([]any, len(v))
		for i := range v {
			o[i] = v[i]
		}
		return o
	}
	eqRow(t, vals(t, out, "a_geq"), bs(false, false, false, false, true, true), "All Geq")
	eqRow(t, vals(t, out, "a_g"), bs(false, false, false, false, false, true), "All G")
	eqRow(t, vals(t, out, "a_l"), bs(true, false, false, false, false, false), "All L")
	eqRow(t, vals(t, out, "a_leq"), bs(true, true, false, false, false, false), "All Leq")
	eqRow(t, vals(t, out, "n_geq"), bs(false, true, true, true, true, true), "Any Geq")
	eqRow(t, vals(t, out, "n_g"), bs(false, false, true, true, true, true), "Any G")
	eqRow(t, vals(t, out, "n_l"), bs(true, true, true, true, false, false), "Any L")
	eqRow(t, vals(t, out, "n_leq"), bs(true, true, true, true, true, false), "Any Leq")
	eqRow(t, vals(t, out, "n_eq"), bs(false, true, true, true, true, false), "Any eq")
	eqRow(t, vals(t, out, "n_neq"), bs(true, false, false, false, false, true), "Any Neq")
}

// test_boolean_where_clauses (TRUE arms): a WHERE clause that is constant-true returns
// all rows unchanged. Standard SQL → MATCH.
func TestMiscBooleanWhereTrue(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{int64(1), int64(2), int64(3), int64(4)}})
	for _, cond := range []string{"TRUE", "1=1", "2 = 2", "'xx' = 'xx'", "TRUE AND 1=1"} {
		out := query(t, d, "SELECT * FROM self WHERE "+cond)
		eqRow(t, vals(t, out, "x"), []any{int64(1), int64(2), int64(3), int64(4)}, "where "+cond)
	}
}

// test_boolean_where_clauses (FALSE arms): a constant-false WHERE clause returns the
// empty frame. Standard SQL → MATCH.
func TestMiscBooleanWhereFalse(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{int64(1), int64(2), int64(3), int64(4)}})
	for _, cond := range []string{"false", "1!=1", "2 != 2", "'xx' != 'xx'", "FALSE OR 1!=1"} {
		out := query(t, d, "SELECT * FROM self WHERE "+cond)
		if out.Height() != 0 {
			t.Fatalf("where %s: height = %d, want 0", cond, out.Height())
		}
	}
}

// test_count (non-distinct arms): COUNT(col) counts non-null, COUNT(*) all rows,
// COUNT(NULL) is 0. (The COUNT(DISTINCT ...) arms live in distinct_test.go.)
// Standard SQL → MATCH.
func TestMiscCountNonDistinct(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
		frame.SeriesInput{Name: "b", Values: []any{int64(1), int64(1), int64(22), int64(22), int64(333)}},
		frame.SeriesInput{Name: "c", Values: []any{int64(1), int64(1), nil, nil, int64(2)}},
	)
	out := query(t, d, `
		SELECT
		  COUNT(a) AS count_a,
		  COUNT(b) AS count_b,
		  COUNT(c) AS count_c,
		  COUNT(*) AS count_star,
		  COUNT(NULL) AS count_null
		FROM self
	`)
	want := map[string]int64{"count_a": 5, "count_b": 5, "count_c": 3, "count_star": 5, "count_null": 0}
	for name, w := range want {
		if g := col(t, out, name).Value(0).(int64); g != w {
			t.Fatalf("%s = %d, want %d", name, g, w)
		}
	}
}

// test_frame_sql_globals_error (positive multi-table arm): a JOIN across two registered
// tables resolves and returns scalar columns. (The polars-only "df1.sql cannot see
// globals" error arm is API-specific and not ported.) Standard SQL → MATCH.
func TestMiscMultiTableJoin(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df1": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(4), int64(5), int64(6)}},
		),
		"df2": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(2), int64(3), int64(4)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(7), int64(6), int64(5)}},
		),
	}
	out := queryCtx(t, `
		SELECT df1.a AS a, df2.b AS b
		FROM df2 JOIN df1 ON df1.a = df2.a
		ORDER BY b DESC
	`, tables)
	eqRow(t, vals(t, out, "a"), []any{int64(2), int64(3)}, "a")
	eqRow(t, vals(t, out, "b"), []any{int64(7), int64(6)}, "b")
}

// test_in_no_ops_11946: WHERE col IN (literal list) filters rows. Standard SQL → MATCH.
func TestMiscInNoOps(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "i1", Values: []any{int64(1), int64(2), int64(3)}})
	out := query(t, d, "SELECT * FROM self WHERE i1 IN (1, 3) ORDER BY i1")
	eqRow(t, vals(t, out, "i1"), []any{int64(1), int64(3)}, "i1 IN (1,3)")
}

// test_limit_offset: LIMIT n OFFSET k slices the result. Standard SQL → MATCH.
func TestMiscLimitOffset(t *testing.T) {
	a := make([]any, 11)
	b := make([]any, 11)
	for i := 0; i < 11; i++ {
		a[i] = int64(i)
		b[i] = int64(10 - i)
	}
	tables := map[string]polars.DataFrame{
		"tbl": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: a},
			frame.SeriesInput{Name: "b", Values: b},
		),
	}
	out := queryCtx(t, "SELECT * FROM tbl ORDER BY a LIMIT 3 OFFSET 4", tables)
	eqRow(t, vals(t, out, "a"), []any{int64(4), int64(5), int64(6)}, "a")
	eqRow(t, vals(t, out, "b"), []any{int64(6), int64(5), int64(4)}, "b")

	// spot-check height across a few (offset, limit) pairs
	for _, p := range []struct{ off, lim, n int }{{0, 3, 3}, {2, 3, 3}, {8, 5, 3}, {11, 1, 0}} {
		o := queryCtx(t, "SELECT * FROM tbl ORDER BY a LIMIT "+itoa(p.lim)+" OFFSET "+itoa(p.off), tables)
		if o.Height() != p.n {
			t.Fatalf("LIMIT %d OFFSET %d: height = %d, want %d", p.lim, p.off, o.Height(), p.n)
		}
	}
}

// test_register_context: exercises polars SQLContext.register_globals()/tables() and
// context-manager scoping — a polars-specific Python API with no gopolars analogue.
func TestMiscRegisterContext(t *testing.T) {
	t.Skip("GAP: polars SQLContext register_globals/tables/context-manager API has no gopolars analogue")
}

// test_sql_on_compatible_frame_types: runs SQL over pandas/pyarrow frame types and
// PyCapsule objects — Python interop with no gopolars analogue.
func TestMiscSQLOnCompatibleFrameTypes(t *testing.T) {
	t.Skip("GAP: pandas/pyarrow/PyCapsule frame interop is Python-only")
}

// test_nested_cte_column_aliasing: nested CTEs with multi-level column & table
// aliasing, projecting scalar ints. Standard SQL → MATCH.
func TestMiscNestedCTEColumnAliasing(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "k", Values: []any{int64(1)}})
	out := query(t, d, `
		WITH
		  x AS (SELECT w.* FROM (VALUES(1,2), (3,4)) AS w(a, b)),
		  y (m, n) AS (
		    WITH z(c, d) AS (SELECT a, b FROM x)
		      SELECT d*2 AS d2, c*3 AS c3 FROM z
		  )
		SELECT n, m FROM y ORDER BY n
	`)
	eqRow(t, vals(t, out, "n"), []any{int64(3), int64(9)}, "n")
	eqRow(t, vals(t, out, "m"), []any{int64(4), int64(8)}, "m")
}

// test_invalid_derived_table_column_aliases:
//   - alias declaring too many columns for a VALUES-derived table must error;
//   - a bare alias over the same VALUES clause selects the rows fine.
//
// polars and DuckDB both reject the column-count mismatch (message differs → pin only
// the error condition). Standard SQL → MATCH for the positive arm.
func TestMiscInvalidDerivedTableColumnAliases(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "k", Values: []any{int64(1)}})
	if _, err := d.SQL(context.Background(),
		`SELECT * FROM (VALUES (1,2), (3,4)) AS tbl(a, b, c, d, e)`); err == nil {
		t.Fatalf("expected error for alias declaring 5 columns over a 2-column VALUES table")
	}
	out := query(t, d, `SELECT * FROM (VALUES (1,2), (3,4)) tbl ORDER BY 1`)
	if out.Width() != 2 || out.Height() != 2 {
		t.Fatalf("bare-alias VALUES: got %dx%d, want 2x2", out.Height(), out.Width())
	}
}

// test_values_clause_table_registration (query arm): the SELECT over a VALUES-derived
// table with a column alias returns the expected scalar row. The polars-specific
// ctx.tables() registration-state assertions are API-only and not ported. MATCH.
func TestMiscValuesClauseSelect(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "k", Values: []any{int64(1)}})
	out := query(t, d, `SELECT x, y FROM (VALUES (-1,1)) AS tbl(x, y)`)
	if g := col(t, out, "x").Value(0).(int64); g != -1 {
		t.Fatalf("x = %d, want -1", g)
	}
	if g := col(t, out, "y").Value(0).(int64); g != 1 {
		t.Fatalf("y = %d, want 1", g)
	}
}

// test_read_csv: reads from a CSV file via SQL `read_csv(...)` — file-I/O table
// function, out of scope for the in-memory parity helpers.
func TestMiscReadCSV(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test_sql_read.csv")
	if err := os.WriteFile(p, []byte("label,num\nlorem,-1\ndolor,0\nipsum,1\n"), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	// DuckDB reads the file natively; read_csv crosses the Arrow bridge fine.
	d := mustFrame(t, frame.SeriesInput{Name: "k", Values: []any{int64(1)}})
	out := query(t, d, `SELECT * FROM read_csv('`+p+`') ORDER BY num`)
	eqRow(t, vals(t, out, "label"), []any{"lorem", "dolor", "ipsum"}, "label")
	eqRow(t, vals(t, out, "num"), []any{int64(-1), int64(0), int64(1)}, "num")

	// DISCREPANCY: polars rejects multiple positional args with a specific
	// SQLSyntaxError; DuckDB's read_csv treats them as multiple paths and errors on
	// the missing files — only the error condition is asserted.
	if _, err := d.SQL(context.Background(), `SELECT * FROM read_csv('a','b','c')`); err == nil {
		lf, _ := d.SQL(context.Background(), `SELECT * FROM read_csv('a','b','c')`)
		if lf != nil {
			if _, cerr := lf.Collect(context.Background()); cerr == nil {
				t.Fatalf("expected read_csv('a','b','c') to error")
			}
		}
	}
}

// test_global_variable_inference_17398: a CTE selecting from a registered table flows
// the value through to the result. Standard SQL → MATCH.
func TestMiscGlobalVariableInference(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"users": mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{"1"}}),
	}
	out := queryCtx(t, `
		WITH user_by_email AS (SELECT id FROM users)
		SELECT * FROM user_by_email
	`, tables)
	if g := col(t, out, "id").Value(0).(string); g != "1" {
		t.Fatalf("id = %q, want \"1\"", g)
	}
}

// test_invalid_cols: every query references a non-existent column and must error.
// Both polars and DuckDB raise column-not-found (message differs) → pin the error
// condition only. MATCH on rejection.
func TestMiscInvalidCols(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "key", Values: []any{"xx", "xx", "yy"}},
		frame.SeriesInput{Name: "n", Values: []any{"100", "200", "300"}},
	)
	queries := []string{
		"SELECT invalid_column FROM self",
		"SELECT key, invalid_column FROM self",
		"SELECT invalid_column * 2 FROM self",
		"SELECT * FROM self ORDER BY invalid_column",
		"SELECT * FROM self WHERE invalid_column = 200",
		"SELECT * FROM self WHERE invalid_column = '200'",
		"SELECT key, SUM(n) AS sum_n FROM self GROUP BY invalid_column",
	}
	for _, q := range queries {
		if _, err := d.SQL(context.Background(), q); err == nil {
			t.Fatalf("expected column-not-found error for %q", q)
		}
	}
}

// test_select_output_heights_20058_21084: row-preserving projections keep the input
// height (3); unit-height aggregates collapse to a single row. Iterates over optional
// WHERE/ORDER BY clause variants. Standard SQL → MATCH.
func TestMiscSelectOutputHeights(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}})
	filters := []string{"", "WHERE 1 = 1", "WHERE a = 1 OR a != 1"}
	orders := []string{"", "ORDER BY 1", "ORDER BY a"}

	for _, f := range filters {
		for _, o := range orders {
			// row-preserving: literal projection broadcasts to input height
			out := query(t, d, "SELECT 1 as a FROM self "+f+" "+o)
			if out.Height() != 3 {
				t.Fatalf("SELECT 1 (f=%q o=%q): height = %d, want 3", f, o, out.Height())
			}
			for i := 0; i < out.Height(); i++ {
				if toFloat(col(t, out, "a").Value(i)) != 1 {
					t.Fatalf("SELECT 1 a[%d] != 1", i)
				}
			}

			// two-literal projection, height preserved
			out2 := query(t, d, "SELECT 1 + 1 as a, 1 as b FROM self "+f+" "+o)
			if out2.Height() != 3 {
				t.Fatalf("SELECT 1+1 (f=%q o=%q): height = %d, want 3", f, o, out2.Height())
			}
			for i := 0; i < out2.Height(); i++ {
				if toFloat(col(t, out2, "a").Value(i)) != 2 || toFloat(col(t, out2, "b").Value(i)) != 1 {
					t.Fatalf("SELECT 1+1 row %d mismatch", i)
				}
			}

			// aggregate: collapses to unit height
			outc := query(t, d, "SELECT COUNT(*) as a FROM self "+f+" "+o)
			if outc.Height() != 1 || toFloat(col(t, outc, "a").Value(0)) != 3 {
				t.Fatalf("COUNT(*) (f=%q o=%q): got %dx? a=%v, want 1 row a=3", f, o, outc.Height(), col(t, outc, "a").Value(0))
			}

			outs := query(t, d, "SELECT SUM(a) as a, 1 as b FROM self "+f+" "+o)
			if outs.Height() != 1 || toFloat(col(t, outs, "a").Value(0)) != 6 || toFloat(col(t, outs, "b").Value(0)) != 1 {
				t.Fatalf("SUM(a) (f=%q o=%q): unexpected result", f, o)
			}
		}
	}
}

// test_select_explode_height_filter_order_by: UNNEST(list_col) in the projection.
//
// DISCREPANCY: polars' SQL UNNEST(col) == Expr.explode() — it explodes ONE column and
// position-extends the others with NULL, so `ORDER BY sort_key` interleaves (polars
// returns [2,1,3,4,5,6]). DuckDB's UNNEST-in-SELECT is a standard lateral unnest that
// REPEATS the other columns per element, so ORDER BY groups by the repeated key
// ([4,5,6,1,2,3]). Pinned to DuckDB's actual output. The pre-explode FILTER arm agrees
// with polars (filter applies before the explode → [4,5,6]).
func TestMiscSelectExplodeHeight(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "list_long", Values: []any{
			[]any{int64(1), int64(2), int64(3)}, []any{int64(4), int64(5), int64(6)},
		}},
		frame.SeriesInput{Name: "sort_key", Values: []any{int64(2), int64(1)}},
		frame.SeriesInput{Name: "filter_mask", Values: []any{false, true}},
		frame.SeriesInput{Name: "filter_mask_all_true", Values: []any{true, true}},
	)
	repeated := []any{int64(4), int64(5), int64(6), int64(1), int64(2), int64(3)}
	for _, q := range []string{
		`SELECT UNNEST(list_long) AS list FROM self ORDER BY sort_key`,
		`SELECT UNNEST(list_long) AS list FROM self ORDER BY sort_key NULLS FIRST`,
		`SELECT UNNEST(list_long) AS list FROM self WHERE filter_mask_all_true ORDER BY sort_key`,
	} {
		out := query(t, d, q)
		eqRow(t, vals(t, out, "list"), repeated, q)
	}
	// MATCH: a filter that keeps one row before the explode → [4,5,6] (as in polars).
	out := query(t, d, `SELECT UNNEST(list_long) AS list FROM self WHERE filter_mask ORDER BY sort_key`)
	eqRow(t, vals(t, out, "list"), []any{int64(4), int64(5), int64(6)}, "filtered")
}
