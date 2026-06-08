package sql

// Ported from py-polars/tests/unit/sql/test_order_by.py (py-1.28.1, representative subset)

import "testing"

func TestSQLOrderByDesc(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT a FROM t ORDER BY a DESC")
	a, _ := out.GetColumn("a")
	for i, w := range []int64{4, 3, 2, 1} {
		if v, _ := a.Value(i).(int64); v != w {
			t.Fatalf("desc[%d]: got %v, want %d", i, a.Value(i), w)
		}
	}
}

func TestSQLOrderByAsc(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT a FROM t ORDER BY a ASC")
	a, _ := out.GetColumn("a")
	for i, w := range []int64{1, 2, 3, 4} {
		if v, _ := a.Value(i).(int64); v != w {
			t.Fatalf("asc[%d]: got %v, want %d", i, a.Value(i), w)
		}
	}
}

// ORDER BY combined with LIMIT.
func TestSQLOrderByLimit(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT a FROM t ORDER BY a DESC LIMIT 2")
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2", out.Height())
	}
	a, _ := out.GetColumn("a")
	if v, _ := a.Value(0).(int64); v != 4 {
		t.Fatalf("top: got %v, want 4", a.Value(0))
	}
}
