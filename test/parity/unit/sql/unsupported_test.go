package sql

// Ported from py-polars/tests/unit/sql (py-1.28.1): test_distinct/test_joins/
// test_cast/test_conditional/test_strings/test_operators/test_miscellaneous.
//
// The gopolars SQL engine was extended (sql-funcs work) and now supports DISTINCT,
// CAST, CASE WHEN, scalar string functions (UPPER/...), arithmetic projections,
// compound boolean WHERE, and IN predicates. The tests below assert those now
// work; the remaining genuine gap (JOIN) is still recorded as a gap.

import "testing"

// test_distinct.py: SELECT DISTINCT removes duplicate rows.
func TestSQLDistinct(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT DISTINCT b FROM t")
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2 (distinct b = x,y)", out.Height())
	}
	b, _ := out.GetColumn("b")
	seen := map[string]bool{}
	for i := 0; i < b.Len(); i++ {
		if s, ok := b.Value(i).(string); ok {
			seen[s] = true
		}
	}
	if !seen["x"] || !seen["y"] || len(seen) != 2 {
		t.Fatalf("distinct values: got %v, want {x,y}", seen)
	}
}

// test_joins.py: JOIN syntax is still not supported (genuine gap).
func TestSQLJoinGap(t *testing.T) {
	t.Parallel()
	if err := execSQLErr(t, baseDF(t), "SELECT a FROM t INNER JOIN t AS u ON t.a = u.a"); err == nil {
		t.Fatal("expected JOIN to be unsupported, got nil error")
	}
}

// test_cast.py: CAST(a AS FLOAT) in the projection casts the column.
func TestSQLCast(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT CAST(a AS FLOAT) AS af FROM t")
	if out.Height() != 4 {
		t.Fatalf("height: got %d, want 4", out.Height())
	}
	af, _ := out.GetColumn("af")
	for i, w := range []float64{1, 2, 3, 4} {
		v, ok := af.Value(i).(float64)
		if !ok || v != w {
			t.Fatalf("af[%d]: got %T(%v), want float64(%v)", i, af.Value(i), af.Value(i), w)
		}
	}
}

// test_conditional.py: CASE WHEN ... THEN ... ELSE ... END.
func TestSQLCaseWhen(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT CASE WHEN a > 2 THEN 'hi' ELSE 'lo' END AS c FROM t")
	c, _ := out.GetColumn("c")
	for i, w := range []string{"lo", "lo", "hi", "hi"} {
		if v, _ := c.Value(i).(string); v != w {
			t.Fatalf("c[%d]: got %v, want %q", i, c.Value(i), w)
		}
	}
}

// test_strings.py: scalar string function UPPER.
func TestSQLStringUpper(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT UPPER(b) AS ub FROM t")
	ub, _ := out.GetColumn("ub")
	for i, w := range []string{"X", "Y", "X", "Y"} {
		if v, _ := ub.Value(i).(string); v != w {
			t.Fatalf("ub[%d]: got %v, want %q", i, ub.Value(i), w)
		}
	}
}

// test_operators.py: arithmetic expression in the projection list.
func TestSQLArithmeticProjection(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT a + 1 AS a1 FROM t")
	a1, _ := out.GetColumn("a1")
	for i, w := range []int64{2, 3, 4, 5} {
		if v, _ := a1.Value(i).(int64); v != w {
			t.Fatalf("a1[%d]: got %v, want %d", i, a1.Value(i), w)
		}
	}
}

// test_miscellaneous.py: IN predicate filters to the listed values.
func TestSQLInPredicate(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT a FROM t WHERE a IN (1, 3)")
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2 (a in {1,3})", out.Height())
	}
	a, _ := out.GetColumn("a")
	for i, w := range []int64{1, 3} {
		if v, _ := a.Value(i).(int64); v != w {
			t.Fatalf("a[%d]: got %v, want %d", i, a.Value(i), w)
		}
	}
}

// A bare aggregate without GROUP BY (SELECT COUNT(*)) reduces to a single row.
func TestSQLBareAggregate(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT COUNT(*) AS n FROM t")
	if out.Height() != 1 {
		t.Fatalf("height: got %d, want 1", out.Height())
	}
	n, err := out.GetColumn("n")
	if err != nil {
		t.Fatalf("get n: %v", err)
	}
	switch v := n.Value(0).(type) {
	case int64:
		if v != 4 {
			t.Fatalf("count: got %d, want 4", v)
		}
	case float64:
		if v != 4 {
			t.Fatalf("count: got %v, want 4", v)
		}
	default:
		t.Fatalf("count: unexpected type %T (%v)", n.Value(0), n.Value(0))
	}
}
