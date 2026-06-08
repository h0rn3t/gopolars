package sql

// Ported from py-polars/tests/unit/sql/test_select.py (py-1.28.1, representative subset)

import "testing"

// SELECT * returns all columns.
func TestSQLSelectStar(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT * FROM t")
	if out.Height() != 4 || out.Width() != 3 {
		t.Fatalf("shape: got %dx%d, want 4x3", out.Height(), out.Width())
	}
}

// SELECT a, b projects a subset of columns in order.
func TestSQLSelectColumns(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT a, b FROM t")
	cols := out.Columns()
	if len(cols) != 2 || cols[0] != "a" || cols[1] != "b" {
		t.Fatalf("columns: got %v, want [a b]", cols)
	}
}

// LIMIT caps the number of rows.
func TestSQLSelectLimit(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT a FROM t LIMIT 2")
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2", out.Height())
	}
}
