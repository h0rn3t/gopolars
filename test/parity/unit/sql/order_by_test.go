//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_order_by.py (py-1.28.1).
//
// DuckDB-backed; behavior measured against DuckDB's dialect. NULL-ordering and
// polars-only error messages are the main divergence zones here. Tests needing
// the foods1.ipc fixture are omitted (// GAP: external data fixture).
package sql

import (
	"reflect"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func vals(t *testing.T, d polars.DataFrame, name string) []any {
	t.Helper()
	s := col(t, d, name)
	out := make([]any, s.Len())
	for i := range out {
		out[i] = s.Value(i)
	}
	return out
}

func eqRow(t *testing.T, got, want []any, msg string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s:\n got=%v\nwant=%v", msg, got, want)
	}
}

func nullFrame(t *testing.T) polars.DataFrame {
	return mustFrame(t,
		frame.SeriesInput{Name: "x", Values: []any{nil, int64(1), nil, int64(3)}},
		frame.SeriesInput{Name: "y", Values: []any{int64(3), int64(2), nil, int64(1)}},
	)
}

// test_order_by_misc_16579: projection order is preserved; ORDER BY a string col.
func TestOrderByColumnOrderPreserved(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "x", Values: []any{"apple", "orange"}},
		frame.SeriesInput{Name: "y", Values: []any{"sheep", "alligator"}},
		frame.SeriesInput{Name: "z", Values: []any{"hello", "world"}},
	)
	out := query(t, d, "SELECT z, y, x FROM self ORDER BY y DESC")
	if got := out.Columns(); len(got) != 3 || got[0] != "z" || got[1] != "y" || got[2] != "x" {
		t.Fatalf("columns = %v, want [z y x]", out.Columns())
	}
	eqRow(t, vals(t, out, "z"), []any{"hello", "world"}, "z")
	eqRow(t, vals(t, out, "x"), []any{"apple", "orange"}, "x")
}

// test_order_by_ordinal: ORDER BY positional ordinals (DuckDB supports these).
// ASC null-last matches polars; DESC null placement is verified explicitly below.
func TestOrderByOrdinalAsc(t *testing.T) {
	out := query(t, nullFrame(t), "SELECT * FROM self ORDER BY 1, 2")
	eqRow(t, vals(t, out, "x"), []any{int64(1), int64(3), nil, nil}, "x")
	eqRow(t, vals(t, out, "y"), []any{int64(2), int64(1), int64(3), nil}, "y")
}

// Explicit NULLS LAST / NULLS FIRST are deterministic across engines → MATCH.
func TestOrderByExplicitNulls(t *testing.T) {
	out := query(t, nullFrame(t), "SELECT * FROM self ORDER BY x NULLS FIRST, y NULLS FIRST")
	eqRow(t, vals(t, out, "x"), []any{nil, nil, int64(1), int64(3)}, "x")
	eqRow(t, vals(t, out, "y"), []any{nil, int64(3), int64(2), int64(1)}, "y")

	out = query(t, nullFrame(t), "SELECT * FROM self ORDER BY 1 DESC NULLS LAST, 2 ASC")
	eqRow(t, vals(t, out, "x"), []any{int64(3), int64(1), nil, nil}, "x")
	eqRow(t, vals(t, out, "y"), []any{int64(1), int64(2), int64(3), nil}, "y")
}

// ORDER BY ALL — DuckDB supports it; equivalent to ordering by every column.
func TestOrderByAll(t *testing.T) {
	out := query(t, nullFrame(t), "SELECT * FROM self ORDER BY ALL")
	eqRow(t, vals(t, out, "x"), []any{int64(1), int64(3), nil, nil}, "x")
	eqRow(t, vals(t, out, "y"), []any{int64(2), int64(1), int64(3), nil}, "y")
}

// SELECT * EXCLUDE — DuckDB supports the EXCLUDE wildcard option.
func TestOrderByExcludeWildcard(t *testing.T) {
	out := query(t, nullFrame(t), "SELECT * EXCLUDE y FROM self ORDER BY y NULLS FIRST")
	if out.Width() != 1 {
		t.Fatalf("EXCLUDE: width = %d, want 1", out.Width())
	}
	// y = [3,2,null,1]; NULLS FIRST asc → null,1,2,3 rows → x = [None,3,1,None].
	eqRow(t, vals(t, out, "x"), []any{nil, int64(3), int64(1), nil}, "x")
}

// ORDER BY an expression.
func TestOrderByExpression(t *testing.T) {
	// x%y = [null, 1, null, 0]; NULLS FIRST asc → null,null,0,1.
	out := query(t, nullFrame(t), "SELECT (x % y) AS xmy FROM self ORDER BY x % y NULLS FIRST")
	eqRow(t, vals(t, out, "xmy"), []any{nil, nil, int64(0), int64(1)}, "xmy")
}
