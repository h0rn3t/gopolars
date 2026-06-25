//go:build duckdb && duckdb_arrow

package polars

import (
	"context"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

func sqlFrame(t *testing.T) DataFrame {
	t.Helper()
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		{Name: "g", Values: []any{"a", "a", "b", "b"}},
		{Name: "v", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	return d
}

func collect(t *testing.T, lf LazyFrame, err error) DataFrame {
	t.Helper()
	if err != nil {
		t.Fatalf("SQL: %v", err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	return out
}

func TestDuckDBSelectFilter(t *testing.T) {
	d := sqlFrame(t)
	lf, err := d.SQL(context.Background(), "SELECT id FROM self WHERE v > 20 ORDER BY id")
	out := collect(t, lf, err)
	if out.Width() != 1 || out.Height() != 2 {
		t.Fatalf("want 1x2, got %dx%d", out.Width(), out.Height())
	}
	col, _ := out.GetColumn("id")
	if col.Value(0).(int64) != 3 || col.Value(1).(int64) != 4 {
		t.Fatalf("unexpected ids: %v %v", col.Value(0), col.Value(1))
	}
}

func TestDuckDBGroupBy(t *testing.T) {
	d := sqlFrame(t)
	lf, err := d.SQL(context.Background(), "SELECT g, sum(v) AS s FROM self GROUP BY g ORDER BY g")
	out := collect(t, lf, err)
	if out.Height() != 2 {
		t.Fatalf("want 2 groups, got %d", out.Height())
	}
	s, _ := out.GetColumn("s")
	// integer SUM comes back as HUGEINT → normalized to int64
	if got := s.Value(0).(int64); got != 30 { // a: 10+20
		t.Fatalf("group a sum = %v, want 30", got)
	}
	if got := s.Value(1).(int64); got != 70 { // b: 30+40
		t.Fatalf("group b sum = %v, want 70", got)
	}
}

func TestDuckDBContextJoin(t *testing.T) {
	ctx := NewSQLContext()
	orders, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "uid", Values: []any{int64(1), int64(1), int64(2)}},
		{Name: "amount", Values: []any{int64(100), int64(200), int64(300)}},
	}})
	users, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2)}},
		{Name: "name", Values: []any{"alice", "bob"}},
	}})
	if err := ctx.Register("orders", orders); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := ctx.RegisterMany(map[string]DataFrame{"users": users}); err != nil {
		t.Fatalf("register_many: %v", err)
	}
	lf, err := ctx.Execute(context.Background(),
		"SELECT u.name, sum(o.amount) AS total FROM orders o JOIN users u ON o.uid = u.id GROUP BY u.name ORDER BY u.name")
	out := collect(t, lf, err)
	if out.Height() != 2 {
		t.Fatalf("want 2 rows, got %d", out.Height())
	}
	name, _ := out.GetColumn("name")
	total, _ := out.GetColumn("total")
	if name.Value(0).(string) != "alice" || total.Value(0).(int64) != 300 {
		t.Fatalf("alice total = %v, want 300", total.Value(0))
	}
	if name.Value(1).(string) != "bob" || total.Value(1).(int64) != 300 {
		t.Fatalf("bob total = %v, want 300", total.Value(1))
	}
}

func TestDuckDBTablesAndUnregister(t *testing.T) {
	ctx := NewSQLContext()
	d := sqlFrame(t)
	_ = ctx.Register("t1", d)
	_ = ctx.Register("t2", d)
	if got := ctx.Tables(); len(got) != 2 || got[0] != "t1" || got[1] != "t2" {
		t.Fatalf("tables = %v, want [t1 t2]", got)
	}
	ctx.Unregister("t1")
	if got := ctx.Tables(); len(got) != 1 || got[0] != "t2" {
		t.Fatalf("after unregister: %v, want [t2]", got)
	}
}

func TestDuckDBInvalidQuery(t *testing.T) {
	d := sqlFrame(t)
	if _, err := d.SQL(context.Background(), "SELECT FROM WHERE bad"); err == nil {
		t.Fatalf("expected error for invalid query")
	}
	if _, err := d.SQL(context.Background(), "SELECT * FROM missing_table"); err == nil {
		t.Fatalf("expected error for missing table")
	}
}

func TestDuckDBNullAndDatetimeRoundTrip(t *testing.T) {
	t0 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "v", Values: []any{int64(10), nil, int64(30)}},
		{Name: "ts", Values: []any{t0, t0.Add(time.Hour), t0.Add(2 * time.Hour)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	lf, err := d.SQL(context.Background(), "SELECT id, v, ts FROM self ORDER BY id")
	out := collect(t, lf, err)
	if out.Height() != 3 {
		t.Fatalf("want 3 rows, got %d", out.Height())
	}
	v, _ := out.GetColumn("v")
	if v.Value(1) != nil {
		t.Fatalf("null not preserved: row1 v = %v", v.Value(1))
	}
	if v.Value(0).(int64) != 10 || v.Value(2).(int64) != 30 {
		t.Fatalf("v values mangled: %v %v", v.Value(0), v.Value(2))
	}
	ts, _ := out.GetColumn("ts")
	got, ok := ts.Value(0).(time.Time)
	if !ok || !got.Equal(t0) {
		t.Fatalf("ts not preserved: %v (%T)", ts.Value(0), ts.Value(0))
	}
}

func TestDuckDBIOScalar(t *testing.T) {
	io := NewIO()
	lf, err := io.SQL(context.Background(), "SELECT 1 AS x, 'hi' AS y")
	out := collect(t, lf, err)
	if out.Height() != 1 || out.Width() != 2 {
		t.Fatalf("want 1x2, got %dx%d", out.Height(), out.Width())
	}
}
