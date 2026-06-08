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

// Compound boolean predicates (`a >= 2 AND a <= 3`) are supported (added in the
// sql-funcs work): the BETWEEN-style range keeps only a in {2,3}.
func TestSQLWhereCompound(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT a FROM t WHERE a >= 2 AND a <= 3")
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2 (a=2,3)", out.Height())
	}
	a, _ := out.GetColumn("a")
	for i, w := range []int64{2, 3} {
		if v, _ := a.Value(i).(int64); v != w {
			t.Fatalf("a[%d]: got %v, want %d", i, a.Value(i), w)
		}
	}
}
