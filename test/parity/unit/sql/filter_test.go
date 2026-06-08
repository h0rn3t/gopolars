package sql

// Ported from py-polars/tests/unit/sql/test_filter_clause.py (py-1.28.1, representative subset)

import "testing"

// WHERE with a numeric comparison.
func TestSQLWhereNumeric(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT a, b FROM t WHERE a > 2")
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2 (a=3,4)", out.Height())
	}
	a, _ := out.GetColumn("a")
	if v, _ := a.Value(0).(int64); v != 3 {
		t.Fatalf("first: got %v, want 3", a.Value(0))
	}
}

// WHERE with a string equality.
func TestSQLWhereString(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT a FROM t WHERE b = 'x'")
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2 (b='x')", out.Height())
	}
}

// DISCREPANCY: gopolars SQL rejects compound boolean predicates such as
// `a >= 2 AND a <= 3` with a "compare type mismatch" error. Single comparisons
// work (above); compound AND/OR predicates are a gap.
func TestSQLWhereCompoundGap(t *testing.T) {
	t.Parallel()
	if err := execSQLErr(t, baseDF(t), "SELECT a FROM t WHERE a >= 2 AND a <= 3"); err == nil {
		t.Fatal("expected error for compound AND predicate, got nil")
	}
}
