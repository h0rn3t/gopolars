package polars

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

func mustDF(t *testing.T, cols []frame.SeriesInput) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: cols})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	return df
}

func runSQL(t *testing.T, ctx SQLContext, query string) DataFrame {
	t.Helper()
	lf, err := ctx.Execute(context.Background(), query)
	if err != nil {
		t.Fatalf("Execute(%q): %v", query, err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect(%q): %v", query, err)
	}
	return out
}

func colValues(t *testing.T, df DataFrame, name string) []any {
	t.Helper()
	s, ok := df.Series(name)
	if !ok {
		t.Fatalf("missing column %q (have %v)", name, df.Columns())
	}
	out := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		out[i] = s.Value(i)
	}
	return out
}

func joinCtx(t *testing.T) SQLContext {
	t.Helper()
	customers := mustDF(t, []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "name", Values: []any{"a", "b", "c"}},
	})
	orders := mustDF(t, []frame.SeriesInput{
		{Name: "cust_id", Values: []any{int64(1), int64(1), int64(2)}},
		{Name: "amount", Values: []any{int64(10), int64(20), int64(30)}},
	})
	ctx := NewSQLContext()
	if err := ctx.RegisterMany(map[string]DataFrame{"customers": customers, "orders": orders}); err != nil {
		t.Fatalf("RegisterMany: %v", err)
	}
	return ctx
}

func TestSQLInnerJoin(t *testing.T) {
	ctx := joinCtx(t)
	out := runSQL(t, ctx, "SELECT name, amount FROM customers JOIN orders ON customers.id = orders.cust_id ORDER BY amount")
	if out.Height() != 3 {
		t.Fatalf("inner join height = %d, want 3", out.Height())
	}
	got := colValues(t, out, "amount")
	want := []any{int64(10), int64(20), int64(30)}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("amount[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestSQLLeftJoinKeepsUnmatched(t *testing.T) {
	ctx := joinCtx(t)
	out := runSQL(t, ctx, "SELECT name FROM customers LEFT JOIN orders ON customers.id = orders.cust_id")
	// id 1 -> 2 rows, id 2 -> 1 row, id 3 -> 1 row (unmatched) = 4
	if out.Height() != 4 {
		t.Fatalf("left join height = %d, want 4", out.Height())
	}
}

func TestSQLCrossJoin(t *testing.T) {
	ctx := joinCtx(t)
	out := runSQL(t, ctx, "SELECT * FROM customers CROSS JOIN orders")
	if out.Height() != 9 {
		t.Fatalf("cross join height = %d, want 9", out.Height())
	}
	commaOut := runSQL(t, ctx, "SELECT * FROM customers, orders")
	if commaOut.Height() != 9 {
		t.Fatalf("comma cross join height = %d, want 9", commaOut.Height())
	}
}

func TestSQLBooleanWhere(t *testing.T) {
	ctx := NewSQLContext()
	df := mustDF(t, []frame.SeriesInput{
		{Name: "a", Values: []any{int64(0), int64(2), int64(2)}},
		{Name: "b", Values: []any{int64(9), int64(1), int64(9)}},
		{Name: "c", Values: []any{int64(0), int64(9), int64(9)}},
	})
	_ = ctx.Register("t", df)
	out := runSQL(t, ctx, "SELECT a FROM t WHERE a > 1 AND b < 5 OR c = 0")
	// row0: (0>1 false) AND ... OR c=0(true) => true
	// row1: (2>1 AND 1<5) => true
	// row2: (2>1 AND 9<5 false) OR c=0 false => false
	if out.Height() != 2 {
		t.Fatalf("boolean where height = %d, want 2", out.Height())
	}
}

func TestSQLCaseAndIn(t *testing.T) {
	ctx := NewSQLContext()
	df := mustDF(t, []frame.SeriesInput{
		{Name: "x", Values: []any{int64(-1), int64(0), int64(5)}},
		{Name: "city", Values: []any{"A", "B", "C"}},
	})
	_ = ctx.Register("t", df)
	out := runSQL(t, ctx, "SELECT CASE WHEN x > 0 THEN 'pos' WHEN x < 0 THEN 'neg' ELSE 'zero' END AS sign FROM t")
	got := colValues(t, out, "sign")
	want := []any{"neg", "zero", "pos"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sign[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	inOut := runSQL(t, ctx, "SELECT city FROM t WHERE city IN ('A', 'C')")
	if inOut.Height() != 2 {
		t.Fatalf("IN height = %d, want 2", inOut.Height())
	}
}

func TestSQLDistinctAndOffset(t *testing.T) {
	ctx := NewSQLContext()
	df := mustDF(t, []frame.SeriesInput{
		{Name: "city", Values: []any{"x", "x", "y", "z"}},
		{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
	})
	_ = ctx.Register("t", df)
	d := runSQL(t, ctx, "SELECT DISTINCT city FROM t")
	if d.Height() != 3 {
		t.Fatalf("distinct height = %d, want 3", d.Height())
	}
	o := runSQL(t, ctx, "SELECT id FROM t ORDER BY id LIMIT 2 OFFSET 1")
	got := colValues(t, o, "id")
	if len(got) != 2 || got[0] != int64(2) || got[1] != int64(3) {
		t.Fatalf("limit/offset got %v, want [2 3]", got)
	}
}

func TestSQLScalarFunctions(t *testing.T) {
	ctx := NewSQLContext()
	df := mustDF(t, []frame.SeriesInput{
		{Name: "s", Values: []any{"abc", "de"}},
		{Name: "v", Values: []any{3.14159, 2.71828}},
	})
	_ = ctx.Register("t", df)
	out := runSQL(t, ctx, "SELECT UPPER(s) AS u, ROUND(v, 2) AS r FROM t")
	u := colValues(t, out, "u")
	if u[0] != "ABC" || u[1] != "DE" {
		t.Fatalf("UPPER got %v", u)
	}
	r := colValues(t, out, "r")
	if r[0] != 3.14 || r[1] != 2.72 {
		t.Fatalf("ROUND got %v", r)
	}
}
