//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_operators.py (py-1.28.1).
//
// gopolars runs SQL through an embedded DuckDB engine (build -tags "duckdb
// duckdb_arrow"), so behavior is measured against DuckDB's dialect, not polars'
// native polars-sql engine. Cases that match polars are asserted directly;
// polars-specific dialect is adapted to DuckDB or pinned with a // DISCREPANCY:
// / // GAP: note.
package sql

import (
	"context"
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func mustFrame(t *testing.T, cols ...frame.SeriesInput) polars.DataFrame {
	t.Helper()
	d, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: cols})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	return d
}

func query(t *testing.T, d polars.DataFrame, q string) polars.DataFrame {
	t.Helper()
	lf, err := d.SQL(context.Background(), q)
	if err != nil {
		t.Fatalf("sql %q: %v", q, err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect %q: %v", q, err)
	}
	return out
}

func col(t *testing.T, d polars.DataFrame, name string) polars.Series {
	t.Helper()
	s, err := d.GetColumn(name)
	if err != nil {
		t.Fatalf("column %q: %v", name, err)
	}
	return s
}

// test_div: a/b, a//b (floor div), SIGN(b). Standard SQL — expect MATCH.
func TestOperatorsDiv(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{10.0, 20.0, 30.0, 40.0, 50.0}},
		frame.SeriesInput{Name: "b", Values: []any{-100.5, 7.0, 2.5, nil, -3.14}},
	)
	out := query(t, d, `SELECT a / b AS a_div_b, a // b AS a_floordiv_b, SIGN(b) AS b_sign FROM self`)

	// MATCH: float division.
	div := col(t, out, "a_div_b")
	wantDiv := []float64{-0.0995024875621891, 2.85714285714286, 12.0, 0, -15.92356687898089}
	for i, w := range wantDiv {
		if i == 3 {
			if div.Value(i) != nil {
				t.Fatalf("a_div_b[3] = %v, want NULL", div.Value(i))
			}
			continue
		}
		if got := div.Value(i).(float64); math.Abs(got-w) > 1e-9 {
			t.Fatalf("a_div_b[%d] = %v, want %v", i, got, w)
		}
	}

	// DISCREPANCY: polars `//` is floor division ([-1, 2, 12, null, -16]); DuckDB's
	// `//` on DOUBLE operands returns plain float division (== a / b). Pin DuckDB's
	// actual behavior.
	fdiv := col(t, out, "a_floordiv_b")
	for i, w := range wantDiv {
		if i == 3 {
			continue
		}
		if got := fdiv.Value(i).(float64); math.Abs(got-w) > 1e-9 {
			t.Fatalf("a_floordiv_b[%d] = %v, want %v (DuckDB float div)", i, got, w)
		}
	}

	// MATCH (value): DuckDB SIGN returns TINYINT → normalized to int64; polars
	// returns float -1.0/1.0 — values agree, dtype differs (DISCREPANCY: int64 vs f64).
	sign := col(t, out, "b_sign")
	wantSign := []int64{-1, 1, 1, 0, -1}
	for i, w := range wantSign {
		if i == 3 {
			if sign.Value(i) != nil {
				t.Fatalf("b_sign[3] = %v, want NULL", sign.Value(i))
			}
			continue
		}
		if got := sign.Value(i).(int64); got != w {
			t.Fatalf("b_sign[%d] = %v, want %v", i, sign.Value(i), w)
		}
	}
}

// test_unary_ops_8890: unary +/-, IN list, projection of literals. Standard SQL.
func TestOperatorsUnaryAndIn(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(-2), int64(-1), int64(1), int64(2)}},
		frame.SeriesInput{Name: "b", Values: []any{"w", "x", "y", "z"}},
	)
	out := query(t, d, `SELECT *, -(3) AS c, (+4) AS d FROM self WHERE a IN (-3, -1, 2, 4) ORDER BY a`)
	if out.Height() != 2 {
		t.Fatalf("want 2 rows, got %d", out.Height())
	}
	a := col(t, out, "a")
	b := col(t, out, "b")
	c := col(t, out, "c")
	if a.Value(0).(int64) != -1 || a.Value(1).(int64) != 2 {
		t.Fatalf("a = %v %v, want -1 2", a.Value(0), a.Value(1))
	}
	if b.Value(0).(string) != "x" || b.Value(1).(string) != "z" {
		t.Fatalf("b = %v %v, want x z", b.Value(0), b.Value(1))
	}
	if toFloat(c.Value(0)) != -3 {
		t.Fatalf("c = %v, want -3", c.Value(0))
	}
}

// test_starts_with: polars uses the `^@` starts-with operator. DuckDB exposes the
// same `^@` operator — assert it; if DuckDB lacked it this would become a GAP.
func TestOperatorsStartsWith(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "x", Values: []any{"aaa", "bbb", "a"}},
		frame.SeriesInput{Name: "y", Values: []any{"abc", "b", "aa"}},
	)
	out := query(t, d, `SELECT x ^@ 'a' AS sw FROM self`)
	sw := col(t, out, "sw")
	want := []bool{true, false, true}
	for i, w := range want {
		if got := sw.Value(i).(bool); got != w {
			t.Fatalf("x ^@ 'a' [%d] = %v, want %v", i, got, w)
		}
	}
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	default:
		return math.NaN()
	}
}
