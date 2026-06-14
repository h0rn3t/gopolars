package sql

import (
	"context"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/exec"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

func mkFrame(t *testing.T, cols ...frame.SeriesInput) frame.DataFrame {
	t.Helper()
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: cols})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	return df
}

// runSQL parses, binds, plans, and executes a query against the named source
// table in the catalog, returning the result frame.
func runSQL(t *testing.T, query, sourceTable string, tables map[string]frame.DataFrame) frame.DataFrame {
	t.Helper()
	parsed, err := Parse(query)
	if err != nil {
		t.Fatalf("Parse(%q): %v", query, err)
	}
	catalog := NewCatalog(tables)
	bound, err := Bind(parsed, catalog)
	if err != nil {
		t.Fatalf("Bind(%q): %v", query, err)
	}
	nodes, err := Plan(bound, catalog)
	if err != nil {
		t.Fatalf("Plan(%q): %v", query, err)
	}
	out, err := exec.New().Execute(context.Background(), tables[sourceTable], nodes)
	if err != nil {
		t.Fatalf("Execute(%q): %v", query, err)
	}
	return out
}

// TestSQLJoins exercises the join binder/planner path (joinKeys, joinTypeOf,
// matchJoinType, joinTableRefs, resolveQueryColumns).
func TestSQLJoins(t *testing.T) {
	t.Parallel()

	tables := map[string]frame.DataFrame{
		"orders": mkFrame(t,
			frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			frame.SeriesInput{Name: "cid", Values: []any{int64(10), int64(20), int64(99)}},
		),
		"customers": mkFrame(t,
			frame.SeriesInput{Name: "cid", Values: []any{int64(10), int64(20), int64(30)}},
			frame.SeriesInput{Name: "name", Values: []any{"alice", "bob", "carol"}},
		),
	}

	inner := runSQL(t, "SELECT * FROM orders JOIN customers ON orders.cid = customers.cid", "orders", tables)
	if inner.Height() != 2 {
		t.Fatalf("inner join height = %d, want 2 (cids 10,20)", inner.Height())
	}

	left := runSQL(t, "SELECT * FROM orders LEFT JOIN customers ON orders.cid = customers.cid", "orders", tables)
	if left.Height() != 3 {
		t.Fatalf("left join height = %d, want 3 (all orders)", left.Height())
	}
}

// TestSQLDateFunctions exercises EXTRACT and the year/month builders
// (parseExtract, buildDatePart).
func TestSQLDateFunctions(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	tables := map[string]frame.DataFrame{
		"events": mkFrame(t,
			frame.SeriesInput{Name: "ts", Values: []any{ts}},
		),
	}

	extract := runSQL(t, "SELECT EXTRACT(YEAR FROM ts) AS y FROM events", "events", tables)
	yc, ok := extract.Series("y")
	if !ok || yc.Value(0) != int64(2026) {
		t.Fatalf("EXTRACT(YEAR) = %v ok=%v, want 2026", yc.Value(0), ok)
	}

	datePart := runSQL(t, "SELECT DATE_PART('month', ts) AS m FROM events", "events", tables)
	mc, ok := datePart.Series("m")
	if !ok || mc.Value(0) != int64(6) {
		t.Fatalf("DATE_PART(month) = %v ok=%v, want 6", mc.Value(0), ok)
	}
}

// TestSQLGreatestLeast exercises the GREATEST/LEAST fold builders
// (buildExtremeFn, compareScalars, toFloatScalar).
func TestSQLGreatestLeast(t *testing.T) {
	t.Parallel()

	tables := map[string]frame.DataFrame{
		"nums": mkFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(3), int64(7)}},
			frame.SeriesInput{Name: "b", Values: []any{int64(5), int64(2)}},
		),
	}

	great := runSQL(t, "SELECT GREATEST(a, b) AS g FROM nums", "nums", tables)
	gc, _ := great.Series("g")
	if v, _ := toFloatScalar(gc.Value(0)); v != 5 {
		t.Fatalf("GREATEST row0 = %v, want 5", gc.Value(0))
	}
	if v, _ := toFloatScalar(gc.Value(1)); v != 7 {
		t.Fatalf("GREATEST row1 = %v, want 7", gc.Value(1))
	}

	least := runSQL(t, "SELECT LEAST(a, b) AS l FROM nums", "nums", tables)
	lc, _ := least.Series("l")
	if v, _ := toFloatScalar(lc.Value(0)); v != 3 {
		t.Fatalf("LEAST row0 = %v, want 3", lc.Value(0))
	}
}

// TestSQLWhereAndAggregate covers a filtered group-by aggregation end to end.
func TestSQLWhereAndAggregate(t *testing.T) {
	t.Parallel()

	tables := map[string]frame.DataFrame{
		"sales": mkFrame(t,
			frame.SeriesInput{Name: "region", Values: []any{"e", "e", "w", "w"}},
			frame.SeriesInput{Name: "amt", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
		),
	}

	out := runSQL(t, "SELECT region, SUM(amt) AS total FROM sales WHERE amt > 15 GROUP BY region", "sales", tables)
	if out.Height() != 2 {
		t.Fatalf("group-by height = %d, want 2", out.Height())
	}
}
