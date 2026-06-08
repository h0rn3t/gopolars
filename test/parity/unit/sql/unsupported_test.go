package sql

// Ported from py-polars/tests/unit/sql (py-1.28.1): records SQL feature gaps from
// test_distinct/test_joins/test_cast/test_conditional/test_strings/test_operators/
// test_miscellaneous and the advanced-feature families.
//
// The gopolars SQL engine supports a basic subset: SELECT (star + projection),
// single-comparison WHERE, GROUP BY with aggregates (SUM/COUNT/MAX/MIN/AVG),
// ORDER BY, and LIMIT. The families below are NOT supported and are recorded here
// so their tasks (10.x) have an explicit, executable record.

import "testing"

// test_distinct.py: DISTINCT is not parsed ("column DISTINCT b not found").
func TestSQLDistinctGap(t *testing.T) {
	t.Parallel()
	if err := execSQLErr(t, baseDF(t), "SELECT DISTINCT b FROM t"); err == nil {
		t.Fatal("expected DISTINCT to be unsupported, got nil error")
	}
}

// test_joins.py: JOIN syntax is not parsed.
func TestSQLJoinGap(t *testing.T) {
	t.Parallel()
	if err := execSQLErr(t, baseDF(t), "SELECT a FROM t INNER JOIN t AS u ON t.a = u.a"); err == nil {
		t.Fatal("expected JOIN to be unsupported, got nil error")
	}
}

// test_cast.py: CAST(...) in the projection list is unsupported.
func TestSQLCastGap(t *testing.T) {
	t.Parallel()
	if err := execSQLErr(t, baseDF(t), "SELECT CAST(a AS FLOAT) AS af FROM t"); err == nil {
		t.Fatal("expected CAST to be unsupported, got nil error")
	}
}

// test_conditional.py: CASE WHEN is unsupported.
func TestSQLCaseWhenGap(t *testing.T) {
	t.Parallel()
	if err := execSQLErr(t, baseDF(t), "SELECT CASE WHEN a > 2 THEN 'hi' ELSE 'lo' END AS c FROM t"); err == nil {
		t.Fatal("expected CASE WHEN to be unsupported, got nil error")
	}
}

// test_strings.py: scalar string functions (UPPER, LOWER, ...) are unsupported.
func TestSQLStringFunctionsGap(t *testing.T) {
	t.Parallel()
	if err := execSQLErr(t, baseDF(t), "SELECT UPPER(b) AS ub FROM t"); err == nil {
		t.Fatal("expected UPPER() to be unsupported, got nil error")
	}
}

// test_operators.py: arithmetic expressions in the projection list are unsupported.
func TestSQLArithmeticProjectionGap(t *testing.T) {
	t.Parallel()
	if err := execSQLErr(t, baseDF(t), "SELECT a + 1 AS a1 FROM t"); err == nil {
		t.Fatal("expected arithmetic projection to be unsupported, got nil error")
	}
}

// test_miscellaneous.py: IN predicates are unsupported.
func TestSQLInPredicateGap(t *testing.T) {
	t.Parallel()
	if err := execSQLErr(t, baseDF(t), "SELECT a FROM t WHERE a IN (1, 3)"); err == nil {
		t.Fatal("expected IN predicate to be unsupported, got nil error")
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
