package sql

import "testing"

// TestParseErrorCases drives every error branch of the parser.
func TestParseErrorCases(t *testing.T) {
	bad := []string{
		"",                               // empty query
		"   ",                            // blank query
		"badleft union select a from t",  // set-op left side invalid
		"select a from t union badright", // set-op right side invalid
		"with foo",                       // WITH without AS
		"with foo as bar",                // WITH subquery not parenthesised
		"with foo as (select a from t",   // unterminated WITH subquery
		"with foo as (badinner) select a from foo", // WITH inner not a select
		"with foo as (select a from t) badouter",   // WITH outer not a select
		"update t set a=1",                         // not a select
		"select a",                                 // missing FROM
		"select a from (select a from t",           // unterminated FROM subquery
		"select a from (badinner) x",               // FROM subquery inner invalid
		"select a from t limit abc",                // non-numeric LIMIT
		"select , from t",                          // empty select list
		"select sum() from t",                      // empty aggregate argument
		"select min() from t",
		"select max() from t",
		"select mean() from t",
		"select n_unique() from t",
		"select sum() over (partition by a) from t", // empty window aggregate arg
		"select mean() over (partition by a) from t",
		"select min() over (partition by a) from t",
		"select max() over (partition by a) from t",
		"select count() over (partition by a) from t",
		"select foo(x) over (partition by a) from t", // unsupported window fn
		"select sum(x) over foo from t",              // OVER not parenthesised
	}
	for _, q := range bad {
		if _, err := Parse(q); err == nil {
			t.Errorf("Parse(%q): expected error, got nil", q)
		}
	}
}

// TestParseExpressionForms covers the aggregate/scalar select-expression forms.
func TestParseExpressionForms(t *testing.T) {
	good := []string{
		"select * from t",
		"select sum(x) from t",
		"select min(x) from t",
		"select max(x) from t",
		"select mean(x) from t",
		"select n_unique(x) from t",
		"select count(*) from t",
		"select count() from t",
		"select a, from t", // trailing empty item is skipped
		"select a as alias from t",
	}
	for _, q := range good {
		if _, err := Parse(q); err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", q, err)
		}
	}
}

// TestParseConditionsAndLiterals covers every comparison operator and literal kind.
func TestParseConditionsAndLiterals(t *testing.T) {
	good := []string{
		"select a from t where a = 1",
		"select a from t where a != 1",
		"select a from t where a > 1",
		"select a from t where a >= 1",
		"select a from t where a < 1",
		"select a from t where a <= 1",
		"select a from t where a = 1.5",     // float literal
		"select a from t where a = true",    // bool literal
		"select a from t where a = false",   // bool literal
		"select a from t where a = 'foo'",   // single-quoted string literal
		"select a from t where a = \"bar\"", // double-quoted string literal
		"select a from t where a = foo",     // bare identifier literal fallback
	}
	for _, q := range good {
		if _, err := Parse(q); err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", q, err)
		}
	}
}

// TestParseClauses covers GROUP BY / ORDER BY / LIMIT and their edge tokens.
func TestParseClauses(t *testing.T) {
	good := []string{
		"select a from t group by a",
		"select a from t order by a",
		"select a from t order by a desc",
		"select a from t order by a,", // trailing empty order item
		"select a from t order by a, b desc",
		"select a from t limit 5",
		"select a from t where a > 1 group by a order by a desc limit 3",
	}
	for _, q := range good {
		if _, err := Parse(q); err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", q, err)
		}
	}
}

// TestParseSubqueriesAndSetOps covers FROM subqueries, CTEs and set operations.
func TestParseSubqueriesAndSetOps(t *testing.T) {
	good := []string{
		"select a from (select a from t) sub", // aliased subquery
		"select a from (select a from t)",     // subquery without alias → _subquery
		"with cte as (select a from t) select a from cte",
		"select a from t1 union select a from t2",
		"select a from t1 intersect select a from t2",
		"select a from t1 except select a from t2",
	}
	for _, q := range good {
		if _, err := Parse(q); err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", q, err)
		}
	}
}

// TestWindowExpressions covers every supported window function and OVER variant.
func TestWindowExpressions(t *testing.T) {
	good := []string{
		"select sum(x) over (partition by g order by a desc) from t",
		"select mean(x) over (partition by g) from t",
		"select rolling_mean(x) over (partition by g) from t",
		"select min(x) over (order by a) from t",
		"select max(x) over (partition by g order by a) from t",
		"select count(x) over (partition by g) from t",
		"select row_number() over (order by a) from t",
		"select sum(x) over (partition by g) as w from t", // explicit alias
		"select sum(x) over (order by a,) from t",         // trailing empty order item
	}
	for _, q := range good {
		if _, err := Parse(q); err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", q, err)
		}
	}
}

// TestBindRejectsUnboundedWindow exercises the binder guard for a window that
// has neither PARTITION BY nor ORDER BY.
func TestBindRejectsUnboundedWindow(t *testing.T) {
	parsed, err := Parse("select sum(x) over () from t")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Bind(parsed, nil); err == nil {
		t.Error("Bind: expected error for window without PARTITION BY/ORDER BY")
	}
}

// TestPlanPipeline drives Parse→Bind→Plan across the structural planner paths:
// window functions, GROUP BY + HAVING, set ops, CTEs and subqueries.
func TestPlanPipeline(t *testing.T) {
	queries := []string{
		"select sum(x) over (partition by g order by a desc) from t",
		"select sum(x) from t group by a having sum(x) > 1",
		"select g, sum(x) from t group by g", // non-agg select item in group path
		"select a from t1 union select a from t2",
		"with cte as (select a from t) select a from cte",
		"select a from (select a from t) sub",
		"select a from t where a > 1 order by a desc limit 5",
	}
	for _, q := range queries {
		parsed, err := Parse(q)
		if err != nil {
			t.Fatalf("Parse(%q): %v", q, err)
		}
		bound, err := Bind(parsed, nil)
		if err != nil {
			t.Fatalf("Bind(%q): %v", q, err)
		}
		nodes, err := Plan(bound, nil)
		if err != nil {
			t.Fatalf("Plan(%q): %v", q, err)
		}
		if len(nodes) == 0 {
			t.Errorf("Plan(%q): produced no nodes", q)
		}
	}

	// Pure select-star planning is valid and intentionally yields no nodes
	// (the star projection is a no-op); exercise that path without asserting
	// a node count.
	star, err := Parse("select * from t")
	if err != nil {
		t.Fatalf("Parse(select *): %v", err)
	}
	bound, err := Bind(star, nil)
	if err != nil {
		t.Fatalf("Bind(select *): %v", err)
	}
	nodes, err := Plan(bound, nil)
	if err != nil {
		t.Fatalf("Plan(select *): %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("Plan(select *): want 0 nodes, got %d", len(nodes))
	}
}
