package operations

// Ported from py-polars/tests/unit/operations/test_select.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func selectDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "foo", Values: []any{int64(1), int64(2)}},
		{Name: "bar", Values: []any{int64(3), int64(4)}},
		{Name: "ham", Values: []any{"a", "b"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// test_select_args_kwargs: single column by name.
func TestSelectSingleColumn(t *testing.T) {
	t.Parallel()
	out, err := selectDF(t).Select(polars.Col("foo"))
	if err != nil {
		t.Fatalf("select foo: %v", err)
	}
	if cols := out.Columns(); len(cols) != 1 || cols[0] != "foo" {
		t.Fatalf("columns: got %v, want [foo]", out.Columns())
	}
}

// Multiple columns by name preserve order.
func TestSelectMultipleColumns(t *testing.T) {
	t.Parallel()
	out, err := selectDF(t).Select(polars.Col("bar"), polars.Col("foo"))
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	cols := out.Columns()
	if len(cols) != 2 || cols[0] != "bar" || cols[1] != "foo" {
		t.Fatalf("columns: got %v, want [bar foo]", cols)
	}
}

// Select with alias (oof="foo").
func TestSelectWithAlias(t *testing.T) {
	t.Parallel()
	out, err := selectDF(t).Select(polars.Col("foo").Alias("oof"))
	if err != nil {
		t.Fatalf("select alias: %v", err)
	}
	if cols := out.Columns(); len(cols) != 1 || cols[0] != "oof" {
		t.Fatalf("columns: got %v, want [oof]", out.Columns())
	}
}

// Aggregation expressions in select reduce to a single row (Polars semantics).
func TestSelectAggregation(t *testing.T) {
	t.Parallel()
	out, err := selectDF(t).Select(polars.Sum(polars.Col("foo")).Alias("s"))
	if err != nil {
		t.Fatalf("select(sum): %v", err)
	}
	if out.Height() != 1 {
		t.Fatalf("height: got %d, want 1", out.Height())
	}
	s, _ := out.GetColumn("s")
	if v := toFloatAny(s.Value(0)); v != 3 {
		t.Fatalf("sum(foo): got %v, want 3", s.Value(0))
	}
}
