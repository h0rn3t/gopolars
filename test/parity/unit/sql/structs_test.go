//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_structs.py (py-1.28.1).
//
// gopolars represents Struct columns via its boxed Struct dtype, and the Arrow bridge
// round-trips Arrow Struct <-> gopolars Struct (including nested structs), so a Struct
// *input* column can now be materialized into the DuckDB engine and its fields accessed
// via dot notation. Cases that rely on polars-SQL-only surface (JSON path operators
// -> / ->> / #> / #>>, struct.* wildcard EXCLUDE/RENAME, polars-specific error text)
// stay GAP/DISCREPANCY. DuckDB dialect notes are tagged // DISCREPANCY.
package sql

import (
	"context"
	"reflect"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// bookFrame builds the test_struct_field_nested_dot_notation fixture: two books each
// with a nested author struct {id, name}.
func bookFrame(t *testing.T) polars.DataFrame {
	t.Helper()
	return mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{"012345", "987654"}},
		frame.SeriesInput{Name: "name", Values: []any{"A Book", "Another Book"}},
		frame.SeriesInput{Name: "author", Values: []any{
			map[string]any{"id": "888888", "name": "Iain M. Banks"},
			map[string]any{"id": "444444", "name": "Dan Abnett"},
		}},
	)
}

// jsonMsgFrame builds the df_struct fixture: a single json_msg struct column wrapping
// id/name/age plus a nested other:{n} struct.
func jsonMsgFrame(t *testing.T) polars.DataFrame {
	t.Helper()
	return mustFrame(t, frame.SeriesInput{Name: "json_msg", Values: []any{
		map[string]any{"id": int64(200), "name": "Bob", "age": int64(45), "other": map[string]any{"n": 1.5}},
		map[string]any{"id": int64(300), "name": "David", "age": int64(19), "other": map[string]any{"n": nil}},
		map[string]any{"id": int64(400), "name": "Zoe", "age": int64(45), "other": map[string]any{"n": -0.5}},
	}})
}

// test_struct_field_nested_dot_notation_22107: struct dot-notation field access.
func TestStructsFieldNestedDotNotation(t *testing.T) {
	d := bookFrame(t)

	out := query(t, d, `SELECT id, author.id AS author_id FROM self ORDER BY id`)
	eqRow(t, vals(t, out, "id"), []any{"012345", "987654"}, "id")
	eqRow(t, vals(t, out, "author_id"), []any{"888888", "444444"}, "author_id")

	// author.name and self.author.name both resolve the struct field (col name "name").
	for _, expr := range []string{"author.name", "self.author.name"} {
		o := query(t, d, "SELECT "+expr+" FROM self ORDER BY id")
		eqRow(t, vals(t, o, o.Columns()[0]), []any{"Iain M. Banks", "Dan Abnett"}, expr)
	}

	// plain / table-qualified top-level column, ordered by self.id DESC.
	for _, expr := range []string{"name", "self.name"} {
		o := query(t, d, "SELECT "+expr+" FROM self ORDER BY self.id DESC")
		eqRow(t, vals(t, o, o.Columns()[0]), []any{"Another Book", "A Book"}, expr)
	}

	// DISCREPANCY: invalid table/column references error (DuckDB's Binder Error text
	// differs from polars' SQLInterfaceError wording).
	if _, err := d.SQL(context.Background(), `SELECT foo.id FROM self`); err == nil {
		t.Fatalf("expected error for invalid table/struct 'foo'")
	}
	if _, err := d.SQL(context.Background(), `SELECT self.foo FROM self`); err == nil {
		t.Fatalf("expected error for invalid column 'foo'")
	}
}

// test_struct_field_selection: nested struct field access (json_msg.other.n) with a
// WHERE filter and ORDER BY over a struct field.
//
// DISCREPANCY: py mixes the `frame.` alias and the `self.` name in one SELECT after
// `FROM self AS frame`; DuckDB binds `self` to the alias `frame`, so the qualifier is
// used consistently here. The semantics (and result) are identical.
func TestStructsFieldSelection(t *testing.T) {
	d := jsonMsgFrame(t)
	out := query(t, d, `
		SELECT frame.json_msg.id AS ID, frame.json_msg.name AS NAME, frame.json_msg.age AS AGE
		FROM self AS frame
		WHERE json_msg.age > 20 AND json_msg.other.n IS NOT NULL
		ORDER BY json_msg.id DESC`)
	eqRow(t, vals(t, out, "ID"), []any{int64(400), int64(200)}, "ID")
	eqRow(t, vals(t, out, "NAME"), []any{"Zoe", "Bob"}, "NAME")
	eqRow(t, vals(t, out, "AGE"), []any{int64(45), int64(45)}, "AGE")
}

// test_struct_field_group_by: GROUP BY a struct field with COUNT + ARRAY_AGG (list
// result). Struct input + list output both round-trip now.
//
// DISCREPANCY: polars COUNT yields UInt32; DuckDB COUNT yields BIGINT (int64).
func TestStructsFieldGroupBy(t *testing.T) {
	d := jsonMsgFrame(t)
	out := query(t, d, `
		SELECT COUNT(json_msg.age) AS n, ARRAY_AGG(json_msg.name) AS names
		FROM self GROUP BY json_msg.age ORDER BY 1 DESC`)
	eqRow(t, vals(t, out, "n"), []any{int64(2), int64(1)}, "n")
	names := col(t, out, "names")
	if !reflect.DeepEqual(names.Value(0), []any{"Bob", "Zoe"}) || !reflect.DeepEqual(names.Value(1), []any{"David"}) {
		t.Fatalf("names = [%v, %v], want [[Bob Zoe] [David]]", names.Value(0), names.Value(1))
	}
}

// test_struct_field_group_by_errors: an ungrouped struct field in an aggregate query
// must error (DuckDB Binder Error; polars raises SQLSyntaxError with different text).
func TestStructsFieldGroupByErrors(t *testing.T) {
	d := jsonMsgFrame(t)
	// DISCREPANCY: error text differs; we assert that DuckDB rejects the query.
	if _, err := d.SQL(context.Background(),
		`SELECT json_msg.name, COUNT(json_msg.age) FROM self GROUP BY json_msg.age`); err == nil {
		t.Fatalf("expected error for ungrouped struct field json_msg.name")
	}
}

// test_struct_field_operator_access: -> / ->> / #> / #>> path operators.
//
// DISCREPANCY: DuckDB has no #> / #>> operators (parser error), and its -> / ->>
// operators return JSON *text* (quoted strings) rather than polars' native typed
// values. The portable -> arms are pinned to DuckDB's actual JSON output; the #>
// arms are confirmed unsupported.
func TestStructsFieldOperatorAccess(t *testing.T) {
	nested := mustFrame(t, frame.SeriesInput{Name: "nested", Values: []any{
		map[string]any{"0": []any{"baz"}, "b": []any{"foo", "bar"}, "c": []any{int64(3), int64(2), int64(1)}},
	}})
	cases := []struct {
		expr string
		want any
	}{
		{`nested -> '0' -> 0`, `"baz"`}, // polars "baz"; DuckDB returns JSON-quoted text
		{`nested -> 'c' -> -1`, `1`},    // polars int 1; DuckDB returns JSON text
		{`nested -> 'c' ->> 2`, `1`},    // polars "1"; DuckDB ->> returns text → MATCH
	}
	for _, c := range cases {
		out := query(t, nested, `SELECT `+c.expr+` AS r FROM self`)
		if got := col(t, out, "r").Value(0); got != c.want {
			t.Fatalf("%s = %#v, want %#v", c.expr, got, c.want)
		}
	}
	// DISCREPANCY: #> / #>> are not DuckDB operators (parser error).
	if runErr(t, nested, `SELECT nested #> '{c,1}' AS r FROM self`) == nil {
		t.Fatalf("expected #> to be unsupported in DuckDB")
	}
}

// test_struct_field_selection_wildcards: json_msg.* with EXCLUDE/RENAME modifiers.
// DuckDB supports the top-level struct.* expansion with EXCLUDE and RENAME, so those
// arms now port (asserted per-column, since gopolars' struct export sorts fields so
// the expanded column ORDER differs from polars — a DISCREPANCY).
//
// DISCREPANCY: the nested `json_msg.other.*` arms are NOT supported (DuckDB: "syntax
// error at or near *"), so only the top-level-star arms are covered.
func TestStructsFieldSelectionWildcards(t *testing.T) {
	d := jsonMsgFrame(t)
	structN := func(o polars.DataFrame, name string) []any {
		s := col(t, o, name)
		out := make([]any, s.Len())
		for i := range out {
			if m, ok := s.Value(i).(map[string]any); ok {
				out[i] = m["n"]
			}
		}
		return out
	}

	// Arm 0: json_msg.* EXCLUDE (age)  → id, name, other
	o0 := query(t, d, `SELECT json_msg.* EXCLUDE (age) FROM self ORDER BY json_msg.id`)
	eqRow(t, vals(t, o0, "id"), []any{int64(200), int64(300), int64(400)}, "c0 id")
	eqRow(t, vals(t, o0, "name"), []any{"Bob", "David", "Zoe"}, "c0 name")
	eqRow(t, structN(o0, "other"), []any{1.5, nil, -0.5}, "c0 other.n")

	// Arm 1: json_msg.* EXCLUDE (name) RENAME (other AS misc)  → id, age, misc
	o1 := query(t, d, `SELECT json_msg.* EXCLUDE (name) RENAME (other AS misc) FROM self ORDER BY json_msg.id`)
	eqRow(t, vals(t, o1, "id"), []any{int64(200), int64(300), int64(400)}, "c1 id")
	eqRow(t, vals(t, o1, "age"), []any{int64(45), int64(19), int64(45)}, "c1 age")
	eqRow(t, structN(o1, "misc"), []any{1.5, nil, -0.5}, "c1 misc.n")

	// Arm 2: json_msg.* EXCLUDE (age,other) RENAME (name AS ident)  → id, ident.
	// DISCREPANCY: the table-qualified `self.json_msg.*` star is a DuckDB parser error
	// ("syntax error at or near *"), so the unqualified star is used.
	o2 := query(t, d, `SELECT json_msg.* EXCLUDE (age, other) RENAME (name AS ident) FROM self ORDER BY json_msg.id`)
	eqRow(t, vals(t, o2, "id"), []any{int64(200), int64(300), int64(400)}, "c2 id")
	eqRow(t, vals(t, o2, "ident"), []any{"Bob", "David", "Zoe"}, "c2 ident")
}

// test_struct_field_selection_errors: invalid struct field/path diagnostics are
// polars-specific (StructFieldNotFoundError / SQLSyntaxError) and not reproduced
// verbatim by DuckDB's Binder Error.
func TestStructsFieldSelectionErrors(t *testing.T) {
	d := jsonMsgFrame(t)
	// DISCREPANCY: DuckDB rejects an unknown struct field, with different error text.
	if _, err := d.SQL(context.Background(), `SELECT json_msg.unknown_field FROM self`); err == nil {
		t.Fatalf("expected error for unknown struct field")
	}
}
