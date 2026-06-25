//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_group_by.py (py-1.28.1).
//
// DuckDB-backed. Omitted: foods1.ipc-fixture cases (// GAP: external data),
// DATE-typed grouping (// GAP: gopolars has no Date dtype — only Datetime),
// struct-output and polars-only error-message tests.
package sql

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// test_group_by (frame part): COUNT(DISTINCT) + HAVING + GROUP BY.
func TestGroupByCountDistinctHaving(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "grp", Values: []any{"a", "b", "c", "c", "b"}},
		frame.SeriesInput{Name: "att", Values: []any{"x", "y", "x", "y", "y"}},
	)
	out := query(t, d, `SELECT grp, COUNT(DISTINCT att) AS n FROM self GROUP BY grp HAVING COUNT(DISTINCT att) > 1 ORDER BY grp`)
	eqRow(t, vals(t, out, "grp"), []any{"c"}, "grp")
	eqRow(t, vals(t, out, "n"), []any{int64(2)}, "n")
}

// test_group_by_all (basic): GROUP BY ALL / ORDER BY.
func TestGroupByAll(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{"xx", "yy", "xx", "yy", "xx", "zz"}},
		frame.SeriesInput{Name: "b", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}},
		frame.SeriesInput{Name: "c", Values: []any{int64(99), int64(99), int64(66), int64(66), int64(66), int64(66)}},
	)
	out := query(t, d, `SELECT a, SUM(b) AS sb, SUM(c) AS sc, COUNT(*) AS n FROM self GROUP BY ALL ORDER BY a`)
	eqRow(t, vals(t, out, "a"), []any{"xx", "yy", "zz"}, "a")
	eqRow(t, vals(t, out, "sb"), []any{int64(9), int64(6), int64(6)}, "sb")
	eqRow(t, vals(t, out, "sc"), []any{int64(231), int64(165), int64(66)}, "sc")
	eqRow(t, vals(t, out, "n"), []any{int64(3), int64(2), int64(1)}, "n")
}

// test_group_by_all (nested agg + aliased key).
func TestGroupByAllNestedAgg(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{"xx", "yy", "xx", "yy", "xx", "zz"}},
		frame.SeriesInput{Name: "b", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}},
		frame.SeriesInput{Name: "c", Values: []any{int64(99), int64(99), int64(66), int64(66), int64(66), int64(66)}},
	)
	out := query(t, d, `SELECT SUM(b) AS sum_b, SUM(c) AS sum_c, (SUM(b) + SUM(c)) / 2.0 AS sum_bc_over_2, a AS grp FROM self GROUP BY ALL ORDER BY grp`)
	eqRow(t, vals(t, out, "sum_b"), []any{int64(9), int64(6), int64(6)}, "sum_b")
	eqRow(t, vals(t, out, "grp"), []any{"xx", "yy", "zz"}, "grp")
	wantBC := []float64{120.0, 85.5, 36.0}
	bc := col(t, out, "sum_bc_over_2")
	for i, w := range wantBC {
		if got := bc.Value(i).(float64); math.Abs(got-w) > 1e-9 {
			t.Fatalf("sum_bc_over_2[%d] = %v, want %v", i, got, w)
		}
	}
}

// test_group_by_ordinal_position: GROUP BY 1 (ordinal), COUNT vs COUNT(*) on nulls.
func TestGroupByOrdinal(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{"xx", "yy", "xx", "yy", "xx", "zz"}},
		frame.SeriesInput{Name: "b", Values: []any{int64(1), nil, int64(3), int64(4), int64(5), int64(6)}},
		frame.SeriesInput{Name: "c", Values: []any{int64(99), int64(99), int64(66), int64(66), int64(66), int64(66)}},
	)
	out := query(t, d, `SELECT c, SUM(b) AS total_b, COUNT(b) AS count_b, COUNT(*) AS count_star FROM self GROUP BY 1 ORDER BY c`)
	eqRow(t, vals(t, out, "c"), []any{int64(66), int64(99)}, "c")
	eqRow(t, vals(t, out, "total_b"), []any{int64(18), int64(1)}, "total_b")
	eqRow(t, vals(t, out, "count_b"), []any{int64(4), int64(1)}, "count_b")
	eqRow(t, vals(t, out, "count_star"), []any{int64(4), int64(2)}, "count_star")
}
