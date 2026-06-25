//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_regex.py (py-1.28.1).
//
// gopolars runs SQL through an embedded DuckDB engine (-tags "duckdb
// duckdb_arrow"). Regex support is one of the most divergent areas between
// polars-sql and DuckDB:
//
//   - polars exposes RLIKE / REGEXP / REGEXP_LIKE and the `~`, `~*`, `!~`, `!~*`
//     operators, all performing a PARTIAL (search) match.
//   - DuckDB does NOT parse RLIKE / REGEXP as operators (Parser Error), has no
//     REGEXP_LIKE / `~*` / `!~*`, and its `~` / `!~` operators perform a FULL
//     (anchored) match via regexp_full_match — not a partial search.
//   - DuckDB's partial-search primitive is regexp_matches(s, pattern[, flags]),
//     which is what reproduces polars' RLIKE/REGEXP/REGEXP_LIKE semantics.
//
// So the values polars expects are reproduced here through regexp_matches, with
// the polars-only spellings pinned as rejected (// DISCREPANCY), and the `~`
// full-vs-partial divergence documented and asserted on DuckDB's actual output.
package sql

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// regexFoods returns the distinct food categories used by the operator/regexp_like
// tests (the py tests load these from foods1.ipc; we inline the relevant set).
func regexFoods(t *testing.T) frame.SeriesInput {
	t.Helper()
	return frame.SeriesInput{Name: "category", Values: []any{"vegetables", "seafood", "meat", "fruit"}}
}

// test_regex_expr_match: polars RLIKE / REGEXP (and the negated forms) match a
// column against a per-row pattern column, using PARTIAL search. Expected matches
// are idx [0, 3]; negated [1, 2, 4].
//
// DISCREPANCY: DuckDB does not parse RLIKE / REGEXP as operators at all (Parser
// Error). Its equivalent partial-search primitive is regexp_matches(str, pat),
// which reproduces the exact polars values. We assert regexp_matches and pin that
// the RLIKE / REGEXP operator spellings are rejected.
func TestRegexExprMatch(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "idx", Values: []any{int64(0), int64(1), int64(2), int64(3), int64(4)}},
		frame.SeriesInput{Name: "str", Values: []any{"ABC", "abc", "000", "A0C", "a0c"}},
		frame.SeriesInput{Name: "pat", Values: []any{"^A", "^A", "^A", `[AB]\d.*$`, ".*xxx$"}},
	)

	out := query(t, d, `SELECT idx FROM self WHERE regexp_matches(str, pat) ORDER BY idx`)
	eqRow(t, vals(t, out, "idx"), []any{int64(0), int64(3)}, "regexp_matches(str,pat)")

	outNot := query(t, d, `SELECT idx FROM self WHERE NOT regexp_matches(str, pat) ORDER BY idx`)
	eqRow(t, vals(t, outNot, "idx"), []any{int64(1), int64(2), int64(4)}, "NOT regexp_matches(str,pat)")

	// DISCREPANCY: RLIKE / REGEXP are not DuckDB operators (Parser Error).
	for _, op := range []string{"RLIKE", "REGEXP", "NOT RLIKE", "NOT REGEXP"} {
		if _, err := d.SQL(context.Background(), `SELECT idx FROM self WHERE str `+op+` pat`); err == nil {
			t.Fatalf("expected DuckDB to reject the %q operator", op)
		}
	}
}

// test_regex_operators: polars `~`, `~*`, `!~`, `!~*` partial-match operators.
//
// DISCREPANCY (full vs partial): DuckDB's `~` / `!~` perform a FULL (anchored)
// match (regexp_full_match), and `~*` / `!~*` do not exist (Catalog Error). So the
// polars operator cases cannot be reproduced via DuckDB's operators. We instead:
//   - assert DuckDB's `~` is a full match (e.g. 'veg' does NOT match 'vegetables',
//     but '^veg.*' / '.*eat' do), and
//   - reproduce the polars partial-search expectations via regexp_matches (with an
//     'i' flag / inline (?i) for the case-insensitive arms), and
//   - pin that `~*` / `!~*` are rejected.
func TestRegexOperators(t *testing.T) {
	d := mustFrame(t, regexFoods(t))

	// DuckDB `~` is a FULL match — confirm the divergence from polars' partial `~`.
	full := query(t, d, `SELECT category FROM self WHERE category ~ 'veg' ORDER BY category`)
	eqRow(t, vals(t, full, "category"), []any{}, "DuckDB ~ 'veg' (full match → no row)")
	fullAnchored := query(t, d, `SELECT category FROM self WHERE category ~ '^veg.*' ORDER BY category`)
	eqRow(t, vals(t, fullAnchored, "category"), []any{"vegetables"}, "DuckDB ~ '^veg.*'")

	// DISCREPANCY: `~*` / `!~*` are not DuckDB operators.
	for _, op := range []string{"~*", "!~*"} {
		if _, err := d.SQL(context.Background(), `SELECT category FROM self WHERE category `+op+` '^VEG'`); err == nil {
			t.Fatalf("expected DuckDB to reject the %q operator", op)
		}
	}

	// Reproduce the polars partial-search expectations via regexp_matches.
	cases := []struct {
		label  string
		where  string
		expect []any
	}{
		// `~ '^veg'`  → "vegetables"
		{"veg", `regexp_matches(category,'^veg')`, []any{"vegetables"}},
		// `~ '^VEG'`  → none (case-sensitive)
		{"VEG", `regexp_matches(category,'^VEG')`, []any{}},
		// `~* '^VEG'` → "vegetables" (case-insensitive)
		{"VEG_i", `regexp_matches(category,'^VEG','i')`, []any{"vegetables"}},
		// `!~ '(t|s)$'` → "seafood" (only seafood does NOT end in t or s)
		{"not_ts", `NOT regexp_matches(category,'(t|s)$')`, []any{"seafood"}},
		// `!~* '(T|S)$'` → "seafood"
		{"not_TS_i", `NOT regexp_matches(category,'(T|S)$','i')`, []any{"seafood"}},
		// `!~* '^.E'` → "fruit"
		{"not_dotE_i", `NOT regexp_matches(category,'^.E','i')`, []any{"fruit"}},
		// `!~* '[aeiOU]'` → none (every category has a vowel)
		{"not_vowel_i", `NOT regexp_matches(category,'[aeiOU]','i')`, []any{}},
	}
	for _, c := range cases {
		out := query(t, d, `SELECT category FROM self WHERE `+c.where+` ORDER BY category`)
		eqRow(t, vals(t, out, "category"), c.expect, "regexp_matches "+c.label)
	}
}

// test_regex_operators_error: polars rejects a non-string pattern for `~` / `!~*`
// with a tailored SQLSyntaxError.
//
// DISCREPANCY (error class): DuckDB also rejects a numeric pattern (Binder Error,
// "No function matches ... regexp_full_match(VARCHAR, INTEGER)"), and rejects the
// `!~*` operator outright (Catalog Error). We assert both error.
func TestRegexOperatorsError(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "sval", Values: []any{"ABC", "abc", "000", "A0C", "a0c"}})

	// DISCREPANCY: numeric pattern → Binder Error.
	if _, err := d.SQL(context.Background(), `SELECT * FROM self WHERE sval ~ 12345`); err == nil {
		t.Fatalf("expected `sval ~ 12345` to error")
	}
	// DISCREPANCY: `!~*` is not a DuckDB operator at all.
	if _, err := d.SQL(context.Background(), `SELECT * FROM self WHERE sval !~* abcde`); err == nil {
		t.Fatalf("expected `sval !~* abcde` to error")
	}
}

// test_regexp_like: polars REGEXP_LIKE(s, pattern[, flags]) partial match, with an
// optional 'i' flag.
//
// DISCREPANCY: DuckDB has no REGEXP_LIKE (Catalog Error). Its equivalent is
// regexp_matches(s, pattern[, flags]), which reproduces the polars values exactly
// (including the 'i' flag and inline (?i)). We assert regexp_matches and pin that
// REGEXP_LIKE itself is rejected.
func TestRegexpLike(t *testing.T) {
	d := mustFrame(t, regexFoods(t))

	// DISCREPANCY: REGEXP_LIKE is not a DuckDB function.
	if _, err := d.SQL(context.Background(), `SELECT category FROM self WHERE REGEXP_LIKE(category,'^veg')`); err == nil {
		t.Fatalf("expected DuckDB to reject REGEXP_LIKE")
	}

	cases := []struct {
		label  string
		where  string
		expect []any
	}{
		{"veg", `regexp_matches(category,'^veg')`, []any{"vegetables"}},
		{"VEG", `regexp_matches(category,'^VEG')`, []any{}},
		{"inline_VEG", `regexp_matches(category,'(?i)^VEG')`, []any{"vegetables"}},
		{"not_ts", `NOT regexp_matches(category,'(t|s)$')`, []any{"seafood"}},
		{"not_TS_i", `NOT regexp_matches(category,'T|S$','i')`, []any{"seafood"}},
		{"not_dotE_i", `NOT regexp_matches(category,'^.E','i')`, []any{"fruit"}},
		{"not_vowel_i", `NOT regexp_matches(category,'[aeiOU]','i')`, []any{}},
	}
	for _, c := range cases {
		out := query(t, d, `SELECT category FROM self WHERE `+c.where+` ORDER BY category`)
		eqRow(t, vals(t, out, "category"), c.expect, "regexp_matches "+c.label)
	}
}

// test_regexp_like_errors: polars raises tailored SQLSyntaxErrors for an empty
// flags string, non-string arguments, and a 1-arg call.
//
// DISCREPANCY: DuckDB (via regexp_matches):
//   - empty flags ”        → NO error; treated as no flags (returns no match here),
//   - numeric args (s,N,N)  → Binder Error,
//   - 1-arg regexp_matches  → Binder Error.
//
// We pin the non-erroring empty-flags case and assert the two Binder errors.
func TestRegexpLikeErrors(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "scol", Values: []any{"xyz"}})

	// DISCREPANCY: empty flags are accepted by DuckDB (no error); with [x-z]+ on
	// 'xyz' the pattern matches, so the row is returned.
	out := query(t, d, `SELECT scol FROM self WHERE regexp_matches(scol,'[x-z]+','') ORDER BY scol`)
	eqRow(t, vals(t, out, "scol"), []any{"xyz"}, "empty flags accepted")

	// DISCREPANCY: numeric arguments → Binder Error.
	if _, err := d.SQL(context.Background(), `SELECT scol FROM self WHERE regexp_matches(scol,999,999)`); err == nil {
		t.Fatalf("expected regexp_matches(scol,999,999) to error")
	}
	// DISCREPANCY: 1-arg call → Binder Error.
	if _, err := d.SQL(context.Background(), `SELECT scol FROM self WHERE regexp_matches(scol)`); err == nil {
		t.Fatalf("expected 1-arg regexp_matches to error")
	}
}
