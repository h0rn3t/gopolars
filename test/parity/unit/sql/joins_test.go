//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_joins.py (py-1.28.1).
//
// gopolars runs SQL through an embedded DuckDB engine (build -tags "duckdb
// duckdb_arrow"), so behavior is measured against DuckDB's dialect, not polars'
// native sql engine. Cases that match polars are asserted directly; polars-only
// dialect/behavior is adapted to DuckDB or pinned with a // DISCREPANCY: / //
// GAP: note. All multi-row results are wrapped in ORDER BY for determinism, and
// overlapping output columns are explicitly aliased because DuckDB does NOT add
// polars' "<name>:<suffix>" rename for `SELECT *` joins (it keeps duplicate
// column names, and lookups by name resolve to the first occurrence).
package sql

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// abcFrames is the shared tbl_a/tbl_b/tbl_c fixture used by several join tests.
func abcFrames(t *testing.T) map[string]polars.DataFrame {
	t.Helper()
	return map[string]polars.DataFrame{
		"tbl_a": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(4), int64(0), int64(6)}},
			frame.SeriesInput{Name: "c", Values: []any{"w", "y", "z"}},
		),
		"tbl_b": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(3), int64(2), int64(1)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(6), int64(5), int64(4)}},
			frame.SeriesInput{Name: "c", Values: []any{"x", "y", "z"}},
		),
		"tbl_c": mustFrame(t,
			frame.SeriesInput{Name: "c", Values: []any{"w", "y", "z"}},
			frame.SeriesInput{Name: "d", Values: []any{10.5, -50.0, 25.5}},
		),
	}
}

// execErr registers the tables and returns the error from executing+collecting q.
func execErr(t *testing.T, q string, tables map[string]polars.DataFrame) error {
	t.Helper()
	ctx := polars.NewSQLContext()
	for n, d := range tables {
		if err := ctx.Register(n, d); err != nil {
			t.Fatalf("register %q: %v", n, err)
		}
	}
	lf, err := ctx.Execute(context.Background(), q)
	if err != nil {
		return err
	}
	_, err = lf.Collect(context.Background())
	return err
}

// test_join_cross: CROSS JOIN of two 3-row frames -> 9 rows.
func TestJoinsCross(t *testing.T) {
	tables := abcFrames(t)
	// Alias output cols (both sides share a/b/c) so we can assert each side.
	out := queryCtx(t, `
		SELECT tbl_a.a AS aa, tbl_a.b AS ab, tbl_a.c AS ac,
		       tbl_b.a AS ba, tbl_b.b AS bb, tbl_b.c AS bc
		FROM tbl_a CROSS JOIN tbl_b
		ORDER BY aa, ab, ac, ba`, tables)
	if out.Height() != 9 {
		t.Fatalf("cross join height = %d, want 9", out.Height())
	}
	// rows for tbl_a side appear in groups of 3 (sorted), each crossed with all of tbl_b.
	eqRow(t, vals(t, out, "aa"), []any{int64(1), int64(1), int64(1), int64(2), int64(2), int64(2), int64(3), int64(3), int64(3)}, "cross aa")
	// tbl_b 'a' values within each group are [1,2,3] after the ba sort.
	eqRow(t, vals(t, out, "ba"), []any{int64(1), int64(2), int64(3), int64(1), int64(2), int64(3), int64(1), int64(2), int64(3)}, "cross ba")
}

// test_join_cross_11927: CROSS JOIN + WHERE acting as an equi-filter.
func TestJoinsCross11927(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df1": mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3)}}),
		"df2": mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(3), int64(4), int64(5)}}),
		"df3": mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(4), int64(5), int64(6)}}),
	}
	out := queryCtx(t, "SELECT df1.id AS id FROM df1 CROSS JOIN df2 WHERE df1.id = df2.id ORDER BY id", tables)
	eqRow(t, vals(t, out, "id"), []any{int64(3)}, "cross-where overlap")

	empty := queryCtx(t, "SELECT df1.id AS id FROM df1 CROSS JOIN df3 WHERE df1.id = df3.id", tables)
	if empty.Height() != 0 {
		t.Fatalf("cross-where disjoint height = %d, want 0", empty.Height())
	}
}

// test_join_inner_multi (ON + USING variants): chain of inner joins, single
// surviving row (1,4,"z",25.5).
func TestJoinsInnerMulti(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"tbl_a": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(4), nil, int64(6)}},
		),
		"tbl_b": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(3), int64(2), int64(1)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(6), int64(5), int64(4)}},
			frame.SeriesInput{Name: "c", Values: []any{"x", "y", "z"}},
		),
		"tbl_c": mustFrame(t,
			frame.SeriesInput{Name: "c", Values: []any{"w", "y", "z"}},
			frame.SeriesInput{Name: "d", Values: []any{10.5, -50.0, 25.5}},
		),
	}
	using := `
		SELECT a, b, c, d FROM tbl_a
		INNER JOIN tbl_b USING (a,b)
		INNER JOIN tbl_c USING (c)
		ORDER BY a DESC`
	on := `
		SELECT tbl_a.a AS a, tbl_a.b AS b, tbl_b.c AS c, tbl_c.d AS d FROM tbl_a
		INNER JOIN tbl_b ON tbl_a.a = tbl_b.a AND tbl_a.b = tbl_b.b
		INNER JOIN tbl_c ON tbl_b.c = tbl_c.c
		ORDER BY a DESC`
	for _, q := range []string{using, on} {
		out := queryCtx(t, q, tables)
		eqRow(t, vals(t, out, "a"), []any{int64(1)}, "inner-multi a")
		eqRow(t, vals(t, out, "b"), []any{int64(4)}, "inner-multi b")
		eqRow(t, vals(t, out, "c"), []any{"z"}, "inner-multi c")
		eqRow(t, vals(t, out, "d"), []any{25.5}, "inner-multi d")
	}
}

// test_join_inner_15663: INNER JOIN ... USING with table aliases and renamed
// projections.
func TestJoinsInner15663(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df_a": mustFrame(t,
			frame.SeriesInput{Name: "LOCID", Values: []any{int64(1), int64(2), int64(3)}},
			frame.SeriesInput{Name: "VALUE", Values: []any{0.1, 0.2, 0.3}},
		),
		"df_b": mustFrame(t,
			frame.SeriesInput{Name: "LOCID", Values: []any{int64(1), int64(2), int64(3)}},
			frame.SeriesInput{Name: "VALUE", Values: []any{25.6, 53.4, 12.7}},
		),
	}
	out := queryCtx(t, `
		SELECT a.LOCID AS LOCID, a.VALUE AS VALUE_A, b.VALUE AS VALUE_B
		FROM df_a AS a
		INNER JOIN df_b AS b USING (LOCID)
		ORDER BY LOCID`, tables)
	eqRow(t, vals(t, out, "LOCID"), []any{int64(1), int64(2), int64(3)}, "loc")
	eqRow(t, vals(t, out, "VALUE_A"), []any{0.1, 0.2, 0.3}, "value_a")
	eqRow(t, vals(t, out, "VALUE_B"), []any{25.6, 53.4, 12.7}, "value_b")
}

// test_join_left_multi: chain of LEFT joins, preserving all of tbl_a with nulls.
func TestJoinsLeftMulti(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"tbl_a": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(4), nil, int64(6)}},
		),
		"tbl_b": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(3), int64(2), int64(1)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(6), int64(5), int64(4)}},
			frame.SeriesInput{Name: "c", Values: []any{"x", "y", "z"}},
		),
		"tbl_c": mustFrame(t,
			frame.SeriesInput{Name: "c", Values: []any{"w", "y", "z"}},
			frame.SeriesInput{Name: "d", Values: []any{10.5, -50.0, 25.5}},
		),
	}
	out := queryCtx(t, `
		SELECT a, b, c, d FROM tbl_a
		LEFT JOIN tbl_b USING (a,b)
		LEFT JOIN tbl_c USING (c)
		ORDER BY a DESC`, tables)
	// py expected (ORDER BY a DESC): (3,6,"x",None),(2,None,None,None),(1,4,"z",25.5)
	eqRow(t, vals(t, out, "a"), []any{int64(3), int64(2), int64(1)}, "left-multi a")
	eqRow(t, vals(t, out, "b"), []any{int64(6), nil, int64(4)}, "left-multi b")
	eqRow(t, vals(t, out, "c"), []any{"x", nil, "z"}, "left-multi c")
	eqRow(t, vals(t, out, "d"), []any{nil, nil, 25.5}, "left-multi d")
}

// test_join_left_multi_nested: LEFT join over a subquery-derived relation.
func TestJoinsLeftMultiNested(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"tbl_a": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(4), nil, int64(6)}},
		),
		"tbl_b": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(3), int64(2), int64(1)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(6), int64(5), int64(4)}},
			frame.SeriesInput{Name: "c", Values: []any{"x", "y", "z"}},
		),
		"tbl_c": mustFrame(t,
			frame.SeriesInput{Name: "c", Values: []any{"w", "y", "z"}},
			frame.SeriesInput{Name: "d", Values: []any{10.5, -50.0, 25.5}},
		),
	}
	out := queryCtx(t, `
		SELECT tbl_x.a AS a, tbl_x.b AS b, tbl_x.c AS c, tbl_c.d AS d
		FROM (
			SELECT tbl_a.a AS a, tbl_a.b AS b, tbl_b.c AS c
			FROM tbl_a
			LEFT JOIN tbl_b ON tbl_a.a = tbl_b.a AND tbl_a.b = tbl_b.b
		) tbl_x
		LEFT JOIN tbl_c ON tbl_x.c = tbl_c.c
		ORDER BY tbl_x.a ASC`, tables)
	// py expected (ORDER BY a ASC): (1,4,"z",25.5),(2,None,None,None),(3,6,"x",None)
	eqRow(t, vals(t, out, "a"), []any{int64(1), int64(2), int64(3)}, "nested a")
	eqRow(t, vals(t, out, "b"), []any{int64(4), nil, int64(6)}, "nested b")
	eqRow(t, vals(t, out, "c"), []any{"z", nil, "x"}, "nested c")
	eqRow(t, vals(t, out, "d"), []any{25.5, nil, nil}, "nested d")
}

// test_join_misc_13618: self-join on a single frame registered under two names.
func TestJoinsSelfJoin13618(t *testing.T) {
	df := mustFrame(t,
		frame.SeriesInput{Name: "A", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
		frame.SeriesInput{Name: "B", Values: []any{int64(5), int64(4), int64(3), int64(2), int64(1)}},
		frame.SeriesInput{Name: "fruits", Values: []any{"banana", "banana", "apple", "apple", "banana"}},
		frame.SeriesInput{Name: "cars", Values: []any{"beetle", "audi", "beetle", "beetle", "beetle"}},
	)
	tables := map[string]polars.DataFrame{"t": df, "t1": df}
	out := queryCtx(t, `
		SELECT t.A AS A, t.fruits AS fruits, t1.B AS B, t1.cars AS cars
		FROM t JOIN t1 ON t.A = t1.B
		ORDER BY t.A DESC`, tables)
	eqRow(t, vals(t, out, "A"), []any{int64(5), int64(4), int64(3), int64(2), int64(1)}, "self A")
	eqRow(t, vals(t, out, "fruits"), []any{"banana", "apple", "apple", "banana", "banana"}, "self fruits")
	eqRow(t, vals(t, out, "B"), []any{int64(5), int64(4), int64(3), int64(2), int64(1)}, "self B")
	eqRow(t, vals(t, out, "cars"), []any{"beetle", "audi", "beetle", "beetle", "beetle"}, "self cars")
}

// test_join_misc_16255: two single-row frames joined on id with aliases.
func TestJoinsMisc16255(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df1": mustFrame(t,
			frame.SeriesInput{Name: "id", Values: []any{int64(1)}},
			frame.SeriesInput{Name: "data", Values: []any{"open"}},
		),
		"df2": mustFrame(t,
			frame.SeriesInput{Name: "id", Values: []any{int64(1)}},
			frame.SeriesInput{Name: "data", Values: []any{"closed"}},
		),
	}
	out := queryCtx(t, `
		SELECT a.id AS id, a.data AS d1, b.data AS d2
		FROM df1 AS a JOIN df2 AS b ON a.id = b.id
		ORDER BY id`, tables)
	eqRow(t, vals(t, out, "id"), []any{int64(1)}, "16255 id")
	eqRow(t, vals(t, out, "d1"), []any{"open"}, "16255 d1")
	eqRow(t, vals(t, out, "d2"), []any{"closed"}, "16255 d2")
}

// test_join_anti_semi: only the *bare* SEMI/ANTI keywords are exercised here.
// DISCREPANCY: polars accepts `LEFT SEMI`/`LEFT ANTI`/`RIGHT SEMI`/`RIGHT ANTI`;
// DuckDB's parser rejects those qualified forms ("syntax error at or near
// SEMI/ANTI") — see TestJoinsQualifiedSemiAntiUnsupported. Bare SEMI/ANTI behave
// like polars' LEFT SEMI/LEFT ANTI and yield the same rows.
func TestJoinsSemiAnti(t *testing.T) {
	tables := abcFrames(t)

	// SEMI JOIN USING (a,c): only tbl_a row with (a,c) present in tbl_b is (2,_,"y").
	semiAC := queryCtx(t, "SELECT a, b, c FROM tbl_a SEMI JOIN tbl_b USING (a,c) ORDER BY a", tables)
	eqRow(t, vals(t, semiAC, "a"), []any{int64(2)}, "semi(a,c) a")
	eqRow(t, vals(t, semiAC, "b"), []any{int64(0)}, "semi(a,c) b")
	eqRow(t, vals(t, semiAC, "c"), []any{"y"}, "semi(a,c) c")

	// SEMI JOIN USING (a): every tbl_a 'a' (1,2,3) is present in tbl_b -> all rows.
	semiA := queryCtx(t, "SELECT a, b, c FROM tbl_a SEMI JOIN tbl_b USING (a) ORDER BY a", tables)
	eqRow(t, vals(t, semiA, "a"), []any{int64(1), int64(2), int64(3)}, "semi(a) a")
	eqRow(t, vals(t, semiA, "b"), []any{int64(4), int64(0), int64(6)}, "semi(a) b")
	eqRow(t, vals(t, semiA, "c"), []any{"w", "y", "z"}, "semi(a) c")

	// ANTI JOIN USING (a): no tbl_a 'a' is absent from tbl_b -> empty.
	antiA := queryCtx(t, "SELECT a, b, c FROM tbl_a ANTI JOIN tbl_b USING (a) ORDER BY a", tables)
	if antiA.Height() != 0 {
		t.Fatalf("anti(a) height = %d, want 0", antiA.Height())
	}

	// ANTI JOIN USING (b): tbl_a.b = [4,0,6]; tbl_b.b = [6,5,4]; only b=0 unmatched -> (2,0,"y").
	antiB := queryCtx(t, "SELECT a, b, c FROM tbl_a ANTI JOIN tbl_b USING (b) ORDER BY a", tables)
	eqRow(t, vals(t, antiB, "a"), []any{int64(2)}, "anti(b) a")
	eqRow(t, vals(t, antiB, "b"), []any{int64(0)}, "anti(b) b")
	eqRow(t, vals(t, antiB, "c"), []any{"y"}, "anti(b) c")
}

// DISCREPANCY: DuckDB rejects the qualified `LEFT/RIGHT SEMI|ANTI JOIN` syntax
// that polars supports. Pin the parser rejection so the divergence is recorded.
func TestJoinsQualifiedSemiAntiUnsupported(t *testing.T) {
	tables := abcFrames(t)
	for _, q := range []string{
		"SELECT * FROM tbl_a LEFT SEMI JOIN tbl_b USING (a)",
		"SELECT * FROM tbl_a LEFT ANTI JOIN tbl_b USING (a)",
		"SELECT * FROM tbl_a RIGHT SEMI JOIN tbl_b USING (b)",
		"SELECT * FROM tbl_a RIGHT ANTI JOIN tbl_b USING (b)",
	} {
		if err := execErr(t, q, tables); err == nil {
			t.Fatalf("expected DuckDB parser error for %q", q)
		}
	}
}

// test_wildcard_resolution_and_join_order: `df1.*` / `df2.*` selection under
// INNER/LEFT/RIGHT/FULL joins, with the two frames swapped on either side.
func TestJoinsWildcardResolution(t *testing.T) {
	df1 := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "b", Values: []any{"x", "y", "z"}},
		frame.SeriesInput{Name: "c", Values: []any{int64(100), int64(200), int64(300)}},
	)
	df2 := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(3), int64(4)}},
		frame.SeriesInput{Name: "b", Values: []any{"qq", "pp", "oo"}},
		frame.SeriesInput{Name: "c", Values: []any{int64(400), int64(500), int64(600)}},
	)
	tables := map[string]polars.DataFrame{"df1": df1, "df2": df2}

	type tc struct {
		q       string
		a, b, c []any
	}
	// NULLS LAST keeps coalesced/unmatched join keys at the end (matches polars'
	// row content; row order made deterministic via ORDER BY).
	cases := []tc{
		// INNER
		{"SELECT df1.* FROM df1 INNER JOIN df2 USING (a) ORDER BY a",
			[]any{int64(1), int64(3)}, []any{"x", "z"}, []any{int64(100), int64(300)}},
		{"SELECT df2.* FROM df1 INNER JOIN df2 USING (a) ORDER BY a",
			[]any{int64(1), int64(3)}, []any{"qq", "pp"}, []any{int64(400), int64(500)}},
		// LEFT
		{"SELECT df1.* FROM df1 LEFT JOIN df2 USING (a) ORDER BY a",
			[]any{int64(1), int64(2), int64(3)}, []any{"x", "y", "z"}, []any{int64(100), int64(200), int64(300)}},
		// ORDER BY df2.a (not the coalesced USING key `a`, which is never null here)
		// so the unmatched row sorts last under NULLS LAST.
		{"SELECT df2.* FROM df1 LEFT JOIN df2 USING (a) ORDER BY df2.a NULLS LAST",
			[]any{int64(1), int64(3), nil}, []any{"qq", "pp", nil}, []any{int64(400), int64(500), nil}},
		// RIGHT
		{"SELECT df1.* FROM df1 RIGHT JOIN df2 USING (a) ORDER BY df2.a",
			[]any{int64(1), int64(3), nil}, []any{"x", "z", nil}, []any{int64(100), int64(300), nil}},
		{"SELECT df2.* FROM df1 RIGHT JOIN df2 USING (a) ORDER BY df2.a",
			[]any{int64(1), int64(3), int64(4)}, []any{"qq", "pp", "oo"}, []any{int64(400), int64(500), int64(600)}},
		// FULL
		{"SELECT df1.* FROM df1 FULL JOIN df2 USING (a) ORDER BY df1.a NULLS LAST",
			[]any{int64(1), int64(2), int64(3), nil}, []any{"x", "y", "z", nil}, []any{int64(100), int64(200), int64(300), nil}},
		{"SELECT df2.* FROM df1 FULL JOIN df2 USING (a) ORDER BY df2.a NULLS LAST",
			[]any{int64(1), int64(3), int64(4), nil}, []any{"qq", "pp", "oo", nil}, []any{int64(400), int64(500), int64(600), nil}},
	}
	for _, c := range cases {
		out := queryCtx(t, c.q, tables)
		eqRow(t, vals(t, out, "a"), c.a, c.q+" :a")
		eqRow(t, vals(t, out, "b"), c.b, c.q+" :b")
		eqRow(t, vals(t, out, "c"), c.c, c.q+" :c")
	}
}

// test_natural_joins_01 (subset): chained NATURAL LEFT/INNER joins, common
// columns coalesced by name. Drops the use-of-COLUMNS regex (polars-specific) by
// projecting the surviving columns explicitly.
func TestJoinsNatural(t *testing.T) {
	df1 := mustFrame(t,
		frame.SeriesInput{Name: "CharacterID", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		frame.SeriesInput{Name: "FirstName", Values: []any{"Jernau Morat", "Cheradenine", "Byr", "Diziet"}},
	)
	df2 := mustFrame(t,
		frame.SeriesInput{Name: "CharacterID", Values: []any{int64(1), int64(2), int64(3), int64(5)}},
		frame.SeriesInput{Name: "Role", Values: []any{"Protagonist", "Protagonist", "Protagonist", "Antagonist"}},
	)
	df3 := mustFrame(t,
		frame.SeriesInput{Name: "CharacterID", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		frame.SeriesInput{Name: "Species", Values: []any{"Pan-human", "Human", "Human", "Oct"}},
	)
	tables := map[string]polars.DataFrame{"df1": df1, "df2": df2, "df3": df3}
	out := queryCtx(t, `
		SELECT CharacterID AS id, FirstName, Role, Species
		FROM df1
		NATURAL LEFT JOIN df2
		NATURAL INNER JOIN df3
		ORDER BY id`, tables)
	eqRow(t, vals(t, out, "id"), []any{int64(1), int64(2), int64(3), int64(4)}, "natural id")
	eqRow(t, vals(t, out, "FirstName"), []any{"Jernau Morat", "Cheradenine", "Byr", "Diziet"}, "natural FirstName")
	// CharacterID 4 has no row in df2 -> Role NULL (NATURAL LEFT JOIN).
	eqRow(t, vals(t, out, "Role"), []any{"Protagonist", "Protagonist", "Protagonist", nil}, "natural Role")
	eqRow(t, vals(t, out, "Species"), []any{"Pan-human", "Human", "Human", "Oct"}, "natural Species")
}

// test_nested_join (subset): join against a nested join relation.
// DISCREPANCY: polars supports referencing the inner table names through a bare
// parenthesized "(df2 JOIN df3 ON ...) AS r99" relation; DuckDB does not expose
// those inner names outside the parentheses ("Referenced table df2 not found").
// The portable form wraps the nested join in a derived SELECT that re-exposes the
// needed columns under the r99 alias — same result rows.
func TestJoinsNested(t *testing.T) {
	df1 := mustFrame(t,
		frame.SeriesInput{Name: "CharacterID", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		frame.SeriesInput{Name: "FirstName", Values: []any{"Jernau Morat", "Cheradenine", "Byr", "Diziet"}},
	)
	df2 := mustFrame(t,
		frame.SeriesInput{Name: "CharacterID", Values: []any{int64(1), int64(2), int64(3), int64(5)}},
		frame.SeriesInput{Name: "Role", Values: []any{"Protagonist", "Protagonist", "Protagonist", "Antagonist"}},
	)
	df3 := mustFrame(t,
		frame.SeriesInput{Name: "CharacterID", Values: []any{int64(1), int64(2), int64(5), int64(6)}},
		frame.SeriesInput{Name: "Species", Values: []any{"Pan-human", "Human", "Human", "Oct"}},
	)
	tables := map[string]polars.DataFrame{"df1": df1, "df2": df2, "df3": df3}
	out := queryCtx(t, `
		SELECT df1.CharacterID AS id, df1.FirstName AS FirstName, r99.Role AS Role, r99.Species AS Species
		FROM df1
		INNER JOIN (
			SELECT df2.CharacterID AS CharacterID, df2.Role AS Role, df3.Species AS Species
			FROM df2 JOIN df3 ON df2.CharacterID = df3.CharacterID
		) AS r99 ON df1.CharacterID = r99.CharacterID
		ORDER BY id`, tables)
	// Only IDs 1 and 2 are present across df1/df2/df3.
	eqRow(t, vals(t, out, "id"), []any{int64(1), int64(2)}, "nested id")
	eqRow(t, vals(t, out, "FirstName"), []any{"Jernau Morat", "Cheradenine"}, "nested FirstName")
	eqRow(t, vals(t, out, "Role"), []any{"Protagonist", "Protagonist"}, "nested Role")
	eqRow(t, vals(t, out, "Species"), []any{"Pan-human", "Human"}, "nested Species")
}

// test_non_equi_joins — DISCREPANCY: polars raises SQLInterfaceError ("only
// equi-join constraints ... are currently supported"); DuckDB fully supports
// non-equi (inequality) join conditions. Pin DuckDB's successful execution.
func TestJoinsNonEquiSupported(t *testing.T) {
	tbl := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "b", Values: []any{int64(4), int64(3), int64(2)}},
	)
	tables := map[string]polars.DataFrame{"tbl": tbl}
	// l.a < r.b : count matches per left row. With l.a in {1,2,3} and r.b in {4,3,2}:
	//   a=1 < {4,3,2} -> 3 rows; a=2 < {4,3} -> 2 rows; a=3 < {4} -> 1 row = 6 rows.
	out := queryCtx(t, `
		SELECT l.a AS la FROM tbl AS l
		LEFT JOIN tbl AS r ON l.a < r.b
		ORDER BY la`, tables)
	eqRow(t, vals(t, out, "la"), []any{int64(1), int64(1), int64(1), int64(2), int64(2), int64(3)}, "non-equi la")
}

// test_implicit_joins — DISCREPANCY: polars raises SQLInterfaceError (comma joins
// "not currently supported ... use explicit JOIN syntax"); DuckDB supports the
// implicit comma-join + WHERE form. Pin DuckDB's result.
func TestJoinsImplicitSupported(t *testing.T) {
	tbl := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "b", Values: []any{int64(4), int64(3), int64(2)}},
		frame.SeriesInput{Name: "c", Values: []any{"x", "y", "z"}},
	)
	tables := map[string]polars.DataFrame{"tbl": tbl}
	out := queryCtx(t, `
		SELECT t1.a AS a FROM tbl AS t1, tbl AS t2
		WHERE t1.a = t2.b
		ORDER BY a`, tables)
	// t1.a matches t2.b in {4,3,2}: a=2 (b=2), a=3 (b=3).
	eqRow(t, vals(t, out, "a"), []any{int64(2), int64(3)}, "implicit a")
}

// test_nulls_equal_19624: SQL equi-joins follow SQL NULL semantics (NULL != NULL),
// so rows with NULL keys never match. Verified via a LEFT join that leaves the
// NULL-keyed left rows unmatched.
func TestJoinsNullKeysDoNotMatch(t *testing.T) {
	tables := map[string]polars.DataFrame{
		"df1": mustFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), nil, nil}}),
		"df2": mustFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(1), int64(2), int64(2), nil}},
			frame.SeriesInput{Name: "b", Values: []any{int64(0), int64(1), int64(2), int64(3), int64(4)}},
		),
	}
	out := queryCtx(t, `
		SELECT df1.a AS a, df2.b AS b
		FROM df1 LEFT JOIN df2 ON df1.a = df2.a
		ORDER BY a NULLS LAST, b NULLS LAST`, tables)
	// df1 a=1 -> b {0,1}; a=2 -> b {2,3}; the two NULL-keyed rows stay unmatched (b NULL).
	eqRow(t, vals(t, out, "a"), []any{int64(1), int64(1), int64(2), int64(2), nil, nil}, "nullkey a")
	eqRow(t, vals(t, out, "b"), []any{int64(0), int64(1), int64(2), int64(3), nil, nil}, "nullkey b")
}
