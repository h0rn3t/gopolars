//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_strings.py (py-1.28.1).
//
// gopolars runs SQL through an embedded DuckDB engine (-tags "duckdb
// duckdb_arrow"), so behavior is measured against DuckDB's dialect, not polars'
// native polars-sql engine. String functions are one of the most divergent
// catalogs between the two: several polars functions are absent or named
// differently in DuckDB, and a few share a name but differ in semantics. Cases
// that agree are asserted directly (MATCH); polars-only spellings are adapted to
// DuckDB's equivalent or pinned to DuckDB's actual behavior with a // DISCREPANCY
// note; functions with no DuckDB equivalent are // GAP (t.Skip).
package sql

import (
	"context"
	"reflect"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_string_case: INITCAP/UPPER/LOWER.
//
// DISCREPANCY: DuckDB has no INITCAP scalar function (Catalog Error). UPPER/LOWER
// are standard SQL and MATCH. We assert UPPER/LOWER directly and pin that INITCAP
// is rejected.
func TestStringsCase(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "words", Values: []any{"Test SOME words"}})

	out := query(t, d, `SELECT UPPER(words) AS upper, LOWER(words) AS lower FROM self`)
	if got := col(t, out, "upper").Value(0).(string); got != "TEST SOME WORDS" {
		t.Fatalf("upper = %q, want TEST SOME WORDS", got)
	}
	if got := col(t, out, "lower").Value(0).(string); got != "test some words" {
		t.Fatalf("lower = %q, want test some words", got)
	}

	// DISCREPANCY: INITCAP is polars-only; DuckDB rejects it (no such scalar fn).
	if _, err := d.SQL(context.Background(), `SELECT INITCAP(words) FROM self`); err == nil {
		t.Fatalf("expected DuckDB to reject INITCAP (no such function)")
	}
}

// test_string_concat: || operator, CONCAT, CONCAT_WS.
//
// DISCREPANCY (NULL handling): polars CONCAT/|| propagate NULL — any NULL operand
// yields NULL. DuckDB differs:
//   - The `||` operator DOES propagate NULL (matches polars).
//   - CONCAT() SKIPS NULL operands (treats them as empty), so a NULL arg does not
//     null the row. CONCAT_WS likewise skips NULLs.
//
// We assert the `||` rows (MATCH) and the CONCAT/CONCAT_WS rows pinned to DuckDB's
// skip-NULL behavior.
func TestStringsConcat(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "x", Values: []any{"a", nil, "c"}},
		frame.SeriesInput{Name: "y", Values: []any{"d", "e", "f"}},
		frame.SeriesInput{Name: "z", Values: []any{int64(1), int64(2), int64(3)}},
	)
	out := query(t, d, `
		SELECT
		  (x || x || y)             AS c0,
		  (x || y || z)             AS c1,
		  CONCAT((x || '-'), y)     AS c2,
		  CONCAT(x, x, y)           AS c3,
		  CONCAT(x, y, (z * 2))     AS c4,
		  CONCAT_WS(':', x, y, z)   AS c5,
		  CONCAT_WS('', y, z, '!')  AS c6
		FROM self
	`)

	// c0,c1: `||` propagates NULL → row 1 (x=NULL) is NULL. MATCH with polars.
	eqRow(t, vals(t, out, "c0"), []any{"aad", nil, "ccf"}, "c0")
	eqRow(t, vals(t, out, "c1"), []any{"ad1", nil, "cf3"}, "c1")

	// DISCREPANCY: CONCAT skips NULL. polars expects ["a-d", "e", "c-f"] (it
	// preserves the non-null parts too, but via NULL-as-empty); DuckDB's skip-NULL
	// yields the same visible strings here.
	eqRow(t, vals(t, out, "c2"), []any{"a-d", "e", "c-f"}, "c2")
	eqRow(t, vals(t, out, "c3"), []any{"aad", "e", "ccf"}, "c3")
	eqRow(t, vals(t, out, "c4"), []any{"ad2", "e4", "cf6"}, "c4")
	eqRow(t, vals(t, out, "c5"), []any{"a:d:1", "e:2", "c:f:3"}, "c5")
	eqRow(t, vals(t, out, "c6"), []any{"d1!", "e2!", "f3!"}, "c6")
}

// test_string_concat_errors: polars rejects CONCAT()/CONCAT_WS()/CONCAT_WS(':')
// with a SQLSyntaxError ("expects at least N arguments").
//
// DISCREPANCY (error class): DuckDB also rejects these, but as Binder Errors
// ("No function matches ... concat()"), not polars' SQLSyntaxError. We assert each
// query errors.
func TestStringsConcatErrors(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{"a", "b", "c"}})
	for _, q := range []string{
		`SELECT CONCAT() FROM self`,
		`SELECT CONCAT_WS() FROM self`,
		`SELECT CONCAT_WS(':') FROM self`,
	} {
		if _, err := d.SQL(context.Background(), q); err == nil {
			t.Fatalf("expected error for %q", q)
		}
	}
}

// test_string_left_right_reverse: LEFT/RIGHT/REVERSE plus polars-specific argument
// validation. LEFT/RIGHT/REVERSE are standard in DuckDB and MATCH on values,
// including NULL passthrough.
//
// DISCREPANCY (error class): polars raises a tailored SQLSyntaxError for a string
// or float n_chars; DuckDB rejects both too (Conversion/Binder Error) — we assert
// the queries error, not the specific message.
func TestStringsLeftRightReverse(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "txt", Values: []any{"abcde", "abc", "a", nil}})
	out := query(t, d, `
		SELECT
		  LEFT(txt,2)  AS l,
		  RIGHT(txt,2) AS r,
		  REVERSE(txt) AS rev
		FROM self
	`)
	eqRow(t, vals(t, out, "l"), []any{"ab", "ab", "a", nil}, "left")
	eqRow(t, vals(t, out, "r"), []any{"de", "bc", "a", nil}, "right")
	eqRow(t, vals(t, out, "rev"), []any{"edcba", "cba", "a", nil}, "reverse")

	// DISCREPANCY: invalid n_chars rejected. The string arg fails at collect time
	// (string→int cast), the float arg at bind time — both surface as errors.
	if !queryErrors(t, d, `SELECT LEFT(txt,'xyz') FROM self`) {
		t.Fatalf("expected LEFT(txt,'xyz') to error")
	}
	if !queryErrors(t, d, `SELECT RIGHT(txt,6.66) FROM self`) {
		t.Fatalf("expected RIGHT(txt,6.66) to error")
	}
}

// test_string_left_negative_expr: LEFT with negative / zero / NULL / expression
// counts. DuckDB's LEFT matches polars here — negative n means "all but last |n|
// chars", clamped to empty; NULL→NULL; supports a column arg. MATCH.
func TestStringsLeftNegativeExpr(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "s", Values: []any{"alphabet", "alphabet"}},
		frame.SeriesInput{Name: "n", Values: []any{int64(-6), int64(6)}},
	)
	out := query(t, d, `
		SELECT
		  LEFT(s,-50)      AS l0,
		  LEFT(s,-3)       AS l1,
		  LEFT(s,SIGN(-1)) AS l2,
		  LEFT(s,0)        AS l3,
		  LEFT(s,NULL)     AS l4,
		  LEFT(s,1)        AS l5,
		  LEFT(s,SIGN(1))  AS l6,
		  LEFT(s,3)        AS l7,
		  LEFT(s,50)       AS l8,
		  LEFT(s,n)        AS l9
		FROM self
	`)
	eqRow(t, vals(t, out, "l0"), []any{"", ""}, "l0")
	eqRow(t, vals(t, out, "l1"), []any{"alpha", "alpha"}, "l1")
	eqRow(t, vals(t, out, "l2"), []any{"alphabe", "alphabe"}, "l2")
	eqRow(t, vals(t, out, "l3"), []any{"", ""}, "l3")
	eqRow(t, vals(t, out, "l4"), []any{nil, nil}, "l4")
	eqRow(t, vals(t, out, "l5"), []any{"a", "a"}, "l5")
	eqRow(t, vals(t, out, "l6"), []any{"a", "a"}, "l6")
	eqRow(t, vals(t, out, "l7"), []any{"alp", "alp"}, "l7")
	eqRow(t, vals(t, out, "l8"), []any{"alphabet", "alphabet"}, "l8")
	eqRow(t, vals(t, out, "l9"), []any{"al", "alphab"}, "l9")
}

// test_string_right_negative_expr: RIGHT with negative / zero / NULL / expression
// counts. Mirrors LEFT — DuckDB MATCHes polars (negative n = "all but first |n|").
func TestStringsRightNegativeExpr(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "s", Values: []any{"alphabet", "alphabet"}},
		frame.SeriesInput{Name: "n", Values: []any{int64(-6), int64(6)}},
	)
	out := query(t, d, `
		SELECT
		  RIGHT(s,-50)      AS l0,
		  RIGHT(s,-3)       AS l1,
		  RIGHT(s,SIGN(-1)) AS l2,
		  RIGHT(s,0)        AS l3,
		  RIGHT(s,NULL)     AS l4,
		  RIGHT(s,1)        AS l5,
		  RIGHT(s,SIGN(1))  AS l6,
		  RIGHT(s,3)        AS l7,
		  RIGHT(s,50)       AS l8,
		  RIGHT(s,n)        AS l9
		FROM self
	`)
	eqRow(t, vals(t, out, "l0"), []any{"", ""}, "l0")
	eqRow(t, vals(t, out, "l1"), []any{"habet", "habet"}, "l1")
	eqRow(t, vals(t, out, "l2"), []any{"lphabet", "lphabet"}, "l2")
	eqRow(t, vals(t, out, "l3"), []any{"", ""}, "l3")
	eqRow(t, vals(t, out, "l4"), []any{nil, nil}, "l4")
	eqRow(t, vals(t, out, "l5"), []any{"t", "t"}, "l5")
	eqRow(t, vals(t, out, "l6"), []any{"t", "t"}, "l6")
	eqRow(t, vals(t, out, "l7"), []any{"bet", "bet"}, "l7")
	eqRow(t, vals(t, out, "l8"), []any{"alphabet", "alphabet"}, "l8")
	eqRow(t, vals(t, out, "l9"), []any{"et", "phabet"}, "l9")
}

// test_string_lengths: LENGTH/CHAR_LENGTH/CHARACTER_LENGTH (char count),
// OCTET_LENGTH (byte count), BIT_LENGTH (bit count).
//
// MATCH: LENGTH/CHAR_LENGTH/CHARACTER_LENGTH all return the char count and agree
// with polars (Café→4, 東京→2, ""→0, NULL→NULL).
//
// DISCREPANCY (byte count): polars uses OCTET_LENGTH for the byte count, but
// DuckDB's OCTET_LENGTH only accepts BLOB (Binder Error on VARCHAR). DuckDB's
// byte-length-of-string function is STRLEN — it returns the same values polars'
// OCTET_LENGTH does (Café→5, 東京→6). We assert STRLEN for the byte count.
//
// GAP (BIT_LENGTH): DuckDB has no BIT_LENGTH over strings or blobs.
func TestStringsLengths(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "words", Values: []any{"Café", nil, "東京", ""}})
	out := query(t, d, `
		SELECT
		  LENGTH(words)            AS n_chrs1,
		  CHAR_LENGTH(words)       AS n_chrs2,
		  CHARACTER_LENGTH(words)  AS n_chrs3,
		  STRLEN(words)            AS n_bytes
		FROM self
	`)
	wantChrs := []any{int64(4), nil, int64(2), int64(0)}
	eqRow(t, vals(t, out, "n_chrs1"), wantChrs, "n_chrs1")
	eqRow(t, vals(t, out, "n_chrs2"), wantChrs, "n_chrs2")
	eqRow(t, vals(t, out, "n_chrs3"), wantChrs, "n_chrs3")
	// DISCREPANCY: STRLEN (DuckDB) == OCTET_LENGTH (polars) for byte count.
	eqRow(t, vals(t, out, "n_bytes"), []any{int64(5), nil, int64(6), int64(0)}, "n_bytes")

	// DISCREPANCY: OCTET_LENGTH(VARCHAR) is a Binder Error in DuckDB.
	if _, err := d.SQL(context.Background(), `SELECT OCTET_LENGTH(words) FROM self`); err == nil {
		t.Fatalf("expected OCTET_LENGTH(VARCHAR) to error in DuckDB")
	}
}

// test_string_like: LIKE / ILIKE / NOT LIKE / NOT ILIKE plus the `~~`, `~~*`,
// `!~~`, `!~~*` operator spellings. All standard in DuckDB and MATCH polars.
func TestStringsLike(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "idx", Values: []any{int64(0), int64(1), int64(2), int64(3), int64(4)}},
		frame.SeriesInput{Name: "txt", Values: []any{"ABC", "abc", "000", "A[0]*C", "a0c?"}},
	)
	cases := []struct {
		pattern  string
		like     string
		expected []any
	}{
		{"a%", "LIKE", []any{int64(1), int64(4)}},
		{"a%", "ILIKE", []any{int64(0), int64(1), int64(3), int64(4)}},
		{"ab%", "LIKE", []any{int64(1)}},
		{"AB%", "ILIKE", []any{int64(0), int64(1)}},
		{"ab_", "LIKE", []any{int64(1)}},
		{"A__", "ILIKE", []any{int64(0), int64(1)}},
		{"_0%_", "LIKE", []any{int64(2), int64(4)}},
		{"%0", "LIKE", []any{int64(2)}},
		{"0%", "LIKE", []any{int64(2)}},
		{"__0%", "~~", []any{int64(2), int64(3)}},
		{"%*%", "~~*", []any{int64(3)}},
		{"____", "~~", []any{int64(4)}},
		{"a%C", "~~", []any{}},
		{"a%C", "~~*", []any{int64(0), int64(1), int64(3)}},
		{"%C?", "~~*", []any{int64(4)}},
		{"a0c?", "~~", []any{int64(4)}},
		{"000", "~~", []any{int64(2)}},
		{"00", "~~", []any{}},
	}
	allIdx := []any{int64(0), int64(1), int64(2), int64(3), int64(4)}
	for _, c := range cases {
		// positive form
		out := query(t, d, `SELECT idx FROM self WHERE txt `+c.like+` '`+c.pattern+`' ORDER BY idx`)
		eqRow(t, vals(t, out, "idx"), c.expected, "txt "+c.like+" '"+c.pattern+"'")

		// negated form: NOT for LIKE/ILIKE, ! prefix for the ~~ family
		var notQ string
		if c.like == "LIKE" || c.like == "ILIKE" {
			notQ = `SELECT idx FROM self WHERE txt NOT ` + c.like + ` '` + c.pattern + `' ORDER BY idx`
		} else {
			notQ = `SELECT idx FROM self WHERE txt !` + c.like + ` '` + c.pattern + `' ORDER BY idx`
		}
		out2 := query(t, d, notQ)
		eqRow(t, vals(t, out2, "idx"), complementAny(allIdx, c.expected),
			"negated "+c.like+" '"+c.pattern+"'")
	}
}

// test_string_like_multiline: LIKE / ILIKE anchored against multi-line strings.
// Standard SQL semantics in DuckDB → MATCH.
func TestStringsLikeMultiline(t *testing.T) {
	s1, s2, s3 := "Hello World", "Hello\nWorld", "hello\nWORLD"
	d := mustFrame(t,
		frame.SeriesInput{Name: "idx", Values: []any{int64(0), int64(1), int64(2)}},
		frame.SeriesInput{Name: "txt", Values: []any{s1, s2, s3}},
	)
	out1 := query(t, d, `SELECT txt FROM self WHERE txt LIKE 'Hello%' ORDER BY idx`)
	eqRow(t, vals(t, out1, "txt"), []any{s1, s2}, "LIKE Hello%")

	out2 := query(t, d, `SELECT txt FROM self WHERE txt ILIKE 'HELLO%' ORDER BY idx`)
	eqRow(t, vals(t, out2, "txt"), []any{s1, s2, s3}, "ILIKE HELLO%")

	out3 := query(t, d, `SELECT txt FROM self WHERE txt LIKE '%WORLD' ORDER BY idx`)
	eqRow(t, vals(t, out3, "txt"), []any{s3}, "LIKE %WORLD")

	out4 := query(t, d, "SELECT txt FROM self WHERE txt ILIKE '%\nWORLD' ORDER BY idx")
	eqRow(t, vals(t, out4, "txt"), []any{s2, s3}, "ILIKE %\\nWORLD")
}

// test_string_normalize: NORMALIZE(txt, NFKC|NFKD).
//
// GAP: DuckDB has no NORMALIZE / unicode-normalization scalar function.
// test_string_normalize: NORMALIZE(txt, NFKC|NFKD) compatibility-normalizes Unicode.
//
// DISCREPANCY: DuckDB ships only nfc_normalize (NFC), so gopolars registers a
// `normalize(text, form)` scalar UDF (backed by golang.org/x/text/unicode/norm) and
// the polars bareword form NORMALIZE(txt, NFKC) is written as normalize(txt,'NFKC').
// All five compatibility variants normalize to "Test". MATCH.
func TestStringsNormalize(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "txt", Values: []any{
		"Ｔｅｓｔ", "𝕋𝕖𝕤𝕥", "𝕿𝖊𝖘𝖙", "𝗧𝗲𝘀𝘁", "Ⓣⓔⓢⓣ",
	}})
	for _, form := range []string{"NFKC", "NFKD"} {
		out := query(t, d, `SELECT normalize(txt,'`+form+`') AS norm_txt FROM self`)
		eqRow(t, vals(t, out, "norm_txt"),
			[]any{"Test", "Test", "Test", "Test", "Test"}, "normalize "+form)
	}
}

// test_string_position: POSITION(x IN s) and STRPOS(s, x). Both standard in DuckDB
// and MATCH polars on values (1-indexed, 0 when not found). DuckDB returns int64;
// polars returns UInt32 — values agree (DISCREPANCY: dtype only).
func TestStringsPosition(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "city", Values: []any{
		"Dubai", "Abu Dhabi", "Sharjah", "Al Ain", "Ajman", "Ras Al Khaimah",
	}})
	out := query(t, d, `
		SELECT
		  POSITION('a' IN city) AS a_lc1,
		  POSITION('A' IN city) AS a_uc1,
		  STRPOS(city,'a')      AS a_lc2,
		  STRPOS(city,'A')      AS a_uc2
		FROM self
	`)
	wantLc := []any{int64(4), int64(7), int64(3), int64(0), int64(4), int64(2)}
	wantUc := []any{int64(0), int64(1), int64(0), int64(1), int64(1), int64(5)}
	eqRow(t, vals(t, out, "a_lc1"), wantLc, "a_lc1")
	eqRow(t, vals(t, out, "a_uc1"), wantUc, "a_uc1")
	eqRow(t, vals(t, out, "a_lc2"), wantLc, "a_lc2")
	eqRow(t, vals(t, out, "a_uc2"), wantUc, "a_uc2")

	d2 := mustFrame(t, frame.SeriesInput{Name: "txt", Values: []any{"AbCdEXz", "XyzFDkE"}})
	out2 := query(t, d2, `
		SELECT
		  POSITION('E' IN txt) AS match_E,
		  STRPOS(txt,'X')      AS match_X
		FROM self
	`)
	eqRow(t, vals(t, out2, "match_E"), []any{int64(5), int64(7)}, "match_E")
	eqRow(t, vals(t, out2, "match_X"), []any{int64(6), int64(1)}, "match_X")
}

// test_string_replace: nested REPLACE(s, from, to). Standard SQL → MATCH on values
// (including ""→"" and NULL→NULL passthrough).
//
// DISCREPANCY (error class): polars rejects a 2-arg REPLACE with a tailored
// SQLSyntaxError ("REPLACE expects 3 arguments"); DuckDB rejects it as a Binder
// Error. We assert it errors.
func TestStringsReplace(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "words", Values: []any{
		"Yemeni coffee is the best coffee", "", nil,
	}})
	out := query(t, d, `
		SELECT REPLACE(
		  REPLACE(words, 'coffee', 'tea'),
		  'Yemeni',
		  'English breakfast'
		) AS words
		FROM self
	`)
	eqRow(t, vals(t, out, "words"),
		[]any{"English breakfast tea is the best tea", "", nil}, "replace")

	// DISCREPANCY: 2-arg REPLACE rejected (Binder Error in DuckDB).
	if _, err := d.SQL(context.Background(), `SELECT REPLACE(words,'coffee') FROM self`); err == nil {
		t.Fatalf("expected 2-arg REPLACE to error")
	}
}

// test_string_split: STRING_TO_ARRAY(s, ',') → List(String). The list result now
// reads back as a gopolars List column. MATCH (DuckDB splits "" → [""], NULL → NULL).
func TestStringsSplit(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "s", Values: []any{
		"xx,yy,zz", "abc,,xyz", "", nil,
	}})
	out := query(t, d, `SELECT STRING_TO_ARRAY(s,',') AS s_array FROM self`)
	sa := col(t, out, "s_array")
	want := []any{
		[]any{"xx", "yy", "zz"},
		[]any{"abc", "", "xyz"},
		[]any{""},
		nil,
	}
	for i, w := range want {
		if !reflect.DeepEqual(sa.Value(i), w) {
			t.Fatalf("s_array[%d] = %#v, want %#v", i, sa.Value(i), w)
		}
	}
}

// test_string_split_part: SPLIT_PART(s, ',', n), including a negative index.
// Standard in DuckDB and MATCH polars (1-indexed; negative counts from the end;
// missing part → ""; NULL→NULL).
func TestStringsSplitPart(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "s", Values: []any{
		"xx,yy,zz", "abc,,xyz,???,hmm", "", nil,
	}})
	out := query(t, d, `
		SELECT
		  SPLIT_PART(s,',',1)  AS "s+1",
		  SPLIT_PART(s,',',3)  AS "s+3",
		  SPLIT_PART(s,',',-2) AS "s-2"
		FROM self
	`)
	eqRow(t, vals(t, out, "s+1"), []any{"xx", "abc", "", nil}, "s+1")
	eqRow(t, vals(t, out, "s+3"), []any{"zz", "xyz", "", nil}, "s+3")
	eqRow(t, vals(t, out, "s-2"), []any{"yy", "???", "", nil}, "s-2")
}

// test_string_substr: SUBSTR(s, start[, len]) with 1-indexed start, including
// negative starts/lengths.
//
// Positive starts MATCH polars. NULL→NULL MATCHes.
//
// DISCREPANCY (negative start semantics): polars clamps a negative start to the
// string head and counts `len` chars from position 1 (so SUBSTR(s,-3,5) keeps just
// the first char); DuckDB instead treats a negative start as an offset from the
// string head where the substring spans positions [start, start+len), counting
// only the portion with positive index. This yields different strings — pinned to
// DuckDB's actual output below. DuckDB also accepts a negative length (returns "")
// where polars raises.
func TestStringsSubstr(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "scol", Values: []any{"abcdefg", "abcde", "abc", nil}},
		frame.SeriesInput{Name: "n", Values: []any{int64(-2), int64(3), int64(2), nil}},
	)
	out := query(t, d, `
		SELECT
		  SUBSTR(scol,1)      AS s1,
		  SUBSTR(scol,2)      AS s2,
		  SUBSTR(scol,3)      AS s3,
		  SUBSTR(scol,1,5)    AS s1_5,
		  SUBSTR(scol,2,2)    AS s2_2,
		  SUBSTR(scol,3,1)    AS s3_1,
		  SUBSTR(scol,-3)     AS "s-3",
		  SUBSTR(scol,-3,3)   AS "s-3_3",
		  SUBSTR(scol,-3,4)   AS "s-3_4",
		  SUBSTR(scol,-3,5)   AS "s-3_5",
		  SUBSTR(scol,-10,13) AS "s-10_13",
		  SUBSTR(scol,n,2)    AS "s-n2",
		  SUBSTR(scol,2,n+3)  AS "s-2n3"
		FROM self
	`)
	// positive starts — MATCH polars
	eqRow(t, vals(t, out, "s1"), []any{"abcdefg", "abcde", "abc", nil}, "s1")
	eqRow(t, vals(t, out, "s2"), []any{"bcdefg", "bcde", "bc", nil}, "s2")
	eqRow(t, vals(t, out, "s3"), []any{"cdefg", "cde", "c", nil}, "s3")
	eqRow(t, vals(t, out, "s1_5"), []any{"abcde", "abcde", "abc", nil}, "s1_5")
	eqRow(t, vals(t, out, "s2_2"), []any{"bc", "bc", "bc", nil}, "s2_2")
	eqRow(t, vals(t, out, "s3_1"), []any{"c", "c", "c", nil}, "s3_1")

	// DISCREPANCY: DuckDB interprets a negative start as an offset from the end of
	// the string ("last |start| chars"), then extends `len` from there. polars
	// instead clamps the start to position 1 and keeps `len + start - 1` leading
	// chars. The two produce different strings — pinned to DuckDB's actual output.
	//   polars s-3   = ["abcdefg","abcde","abc"] ; DuckDB = last 3 chars
	//   polars s-3_3 = ["",...]                  ; DuckDB = last 3 chars
	//   polars s-10_13 = ["ab",...]              ; DuckDB = whole string
	eqRow(t, vals(t, out, "s-3"), []any{"efg", "cde", "abc", nil}, "s-3")
	eqRow(t, vals(t, out, "s-3_3"), []any{"efg", "cde", "abc", nil}, "s-3_3")
	eqRow(t, vals(t, out, "s-3_4"), []any{"efg", "cde", "abc", nil}, "s-3_4")
	eqRow(t, vals(t, out, "s-3_5"), []any{"efg", "cde", "abc", nil}, "s-3_5")
	eqRow(t, vals(t, out, "s-10_13"), []any{"abcdefg", "abcde", "abc", nil}, "s-10_13")
	eqRow(t, vals(t, out, "s-n2"), []any{"fg", "cd", "bc", nil}, "s-n2")
	eqRow(t, vals(t, out, "s-2n3"), []any{"b", "bcde", "bc", nil}, "s-2n3")

	// DISCREPANCY: DuckDB accepts a negative length where polars raises. With
	// SUBSTR('abc',2,-99) DuckDB clamps the (start+len) window to the string head,
	// yielding the single leading char "a" rather than erroring.
	out2 := query(t, d, `SELECT SUBSTR('abc',2,-99) AS v FROM self LIMIT 1`)
	if got := col(t, out2, "v").Value(0).(string); got != "a" {
		t.Fatalf("SUBSTR('abc',2,-99) = %q, want \"a\" (DuckDB accepts negative length)", got)
	}
}

// test_string_trim: TRIM(LEADING 'vmf' FROM s). DuckDB supports the
// LEADING/TRAILING/BOTH ... FROM syntax and MATCHes polars.
//
// DISCREPANCY: polars rejects the 2-arg snowflake-style TRIM(s, chars) as
// "unsupported TRIM syntax"; DuckDB SUPPORTS it (strips the given char set from
// both ends). We assert DuckDB's actual 2-arg result.
func TestStringsTrim(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "category", Values: []any{
		"vegetables", "fruit", "meat", "seafood",
	}})
	out := query(t, d, `
		SELECT DISTINCT TRIM(LEADING 'vmf' FROM category) AS new_category
		FROM self ORDER BY new_category DESC
	`)
	eqRow(t, vals(t, out, "new_category"),
		[]any{"seafood", "ruit", "egetables", "eat"}, "trim leading")

	// DISCREPANCY: 2-arg TRIM is supported in DuckDB (polars rejects it).
	out2 := query(t, d, `SELECT TRIM('*^xxxx^*', '^*') AS v FROM self LIMIT 1`)
	if got := col(t, out2, "v").Value(0).(string); got != "xxxx" {
		t.Fatalf("TRIM('*^xxxx^*','^*') = %q, want \"xxxx\" (DuckDB strips '^*' from both ends)", got)
	}
}

// --- local helpers (unique to this file) -----------------------------------

// queryErrors reports whether running q fails either at plan (SQL) or collect time.
func queryErrors(t *testing.T, d polars.DataFrame, q string) bool {
	t.Helper()
	lf, err := d.SQL(context.Background(), q)
	if err != nil {
		return true
	}
	if _, err := lf.Collect(context.Background()); err != nil {
		return true
	}
	return false
}

// complementAny returns the elements of all not present in sub (preserving order).
func complementAny(all, sub []any) []any {
	out := []any{}
	for _, v := range all {
		in := false
		for _, s := range sub {
			if s == v {
				in = true
				break
			}
		}
		if !in {
			out = append(out, v)
		}
	}
	return out
}
