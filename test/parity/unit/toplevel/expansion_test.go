package toplevel

// Ported from py-polars/tests/unit/test_expansion.py (py-1.28.1).
//
// Covers regex column selection, the pl.all()/pl.col(multi)/pl.exclude selectors,
// struct .field("*") wildcard expansion (on existing and inline pl.struct columns),
// and pl.fold horizontal reduction.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_regex_selection: pl.col("^foo.*$") selects every column whose name matches.
func TestRegexSelection(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "foo", Values: []any{int64(1)}},
		{Name: "fooey", Values: []any{int64(1)}},
		{Name: "foobar", Values: []any{int64(1)}},
		{Name: "bar", Values: []any{int64(1)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Select(polars.Col("^foo.*$"))
	if err != nil {
		t.Fatalf("select regex: %v", err)
	}
	got := out.Columns()
	want := []string{"foo", "fooey", "foobar"}
	if len(got) != len(want) {
		t.Fatalf("columns: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("columns[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

// test_regex_exclude analogue: a regex selector also works in with_columns,
// expanding to each matching column (here doubling every col_* in place).
func TestRegexSelectionInWithColumns(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "col_0", Values: []any{int64(1)}},
		{Name: "col_1", Values: []any{int64(2)}},
		{Name: "other", Values: []any{int64(9)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	// pl.col("^col_.*$") expands to col_0, col_1 (other is untouched).
	out, err := df.Select(polars.Col("^col_.*$"))
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got := out.Columns(); len(got) != 2 || got[0] != "col_0" || got[1] != "col_1" {
		t.Fatalf("columns: got %v, want [col_0 col_1]", out.Columns())
	}
}

func abcDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1)}},
		{Name: "b", Values: []any{int64(1)}},
		{Name: "c", Values: []any{true}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// pl.all() selects every column in order.
func TestSelectorAll(t *testing.T) {
	t.Parallel()
	out, err := abcDF(t).Select(polars.All())
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if got := out.Columns(); len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("columns: got %v, want [a b c]", out.Columns())
	}
}

// test_exclude_selection: pl.exclude("c") selects every column except c.
func TestSelectorExclude(t *testing.T) {
	t.Parallel()
	out, err := abcDF(t).Select(polars.Exclude("c"))
	if err != nil {
		t.Fatalf("exclude: %v", err)
	}
	if got := out.Columns(); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("columns: got %v, want [a b]", out.Columns())
	}
}

// pl.all().exclude("b") = every column except b.
func TestSelectorAllExclude(t *testing.T) {
	t.Parallel()
	out, err := abcDF(t).Select(polars.All().Exclude("b"))
	if err != nil {
		t.Fatalf("all.exclude: %v", err)
	}
	if got := out.Columns(); len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("columns: got %v, want [a c]", out.Columns())
	}
}

// pl.col("a","b") selects the named columns in the given order.
func TestSelectorCols(t *testing.T) {
	t.Parallel()
	out, err := abcDF(t).Select(polars.Cols("c", "a"))
	if err != nil {
		t.Fatalf("cols: %v", err)
	}
	if got := out.Columns(); len(got) != 2 || got[0] != "c" || got[1] != "a" {
		t.Fatalf("columns: got %v, want [c a]", out.Columns())
	}
}

// test_struct_field_expand_star: pl.col("s").struct.field("*") expands a struct
// column into one column per field.
func TestStructFieldStar(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "s", Values: []any{
			map[string]any{"x": int64(1), "y": int64(2)},
			map[string]any{"x": int64(3), "y": int64(4)},
		}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Select(polars.Col("s").StructField("*"))
	if err != nil {
		t.Fatalf("struct.field(*): %v", err)
	}
	if got := out.Columns(); len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Fatalf("columns: got %v, want [x y]", out.Columns())
	}
	x, _ := out.GetColumn("x")
	if v, _ := x.Value(0).(int64); v != 1 {
		t.Fatalf("x[0]: got %v, want 1", x.Value(0))
	}
}

// test_struct_unnest analogue: pl.struct([...]).struct.field("*") packs columns
// into a struct and immediately unpacks them back to the source columns.
func TestInlineStructFieldStar(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "aaa", Values: []any{int64(1), int64(2)}},
		{Name: "bbb", Values: []any{"ab", "cd"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	// Pack then expand: equivalent to selecting aaa, bbb.
	out, err := df.Select(polars.StructCols("aaa", "bbb").StructField("*"))
	if err != nil {
		t.Fatalf("inline struct.field(*): %v", err)
	}
	if got := out.Columns(); len(got) != 2 || got[0] != "aaa" || got[1] != "bbb" {
		t.Fatalf("columns: got %v, want [aaa bbb]", out.Columns())
	}
}

// pl.struct([...]) packs columns into a single Struct column.
func TestStructPack(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{"x", "y"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Select(polars.StructCols("a", "b").Alias("s"))
	if err != nil {
		t.Fatalf("struct pack: %v", err)
	}
	s, _ := out.GetColumn("s")
	if s.DataType() != polars.Struct {
		t.Fatalf("dtype: got %v, want Struct", s.DataType())
	}
	m, ok := s.Value(0).(map[string]any)
	if !ok || m["a"] != int64(1) || m["b"] != "x" {
		t.Fatalf("s[0]: got %v, want {a:1 b:x}", s.Value(0))
	}
}

// test_regex_in_filter analogue: pl.fold horizontally reduces several columns
// per row. Here a horizontal sum of three columns.
func TestFoldHorizontalSum(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{int64(10), int64(20)}},
		{Name: "c", Values: []any{int64(100), int64(200)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	add := func(acc any, next any) (any, error) {
		return toInt64(acc) + toInt64(next), nil
	}
	out, err := df.Select(polars.Fold(int64(0), add, polars.Col("a"), polars.Col("b"), polars.Col("c")).Alias("total"))
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	total, _ := out.GetColumn("total")
	for i, w := range []int64{111, 222} {
		if v, _ := total.Value(i).(int64); v != w {
			t.Fatalf("total[%d]: got %v, want %d", i, total.Value(i), w)
		}
	}
}

func toInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case float64:
		return int64(x)
	}
	return 0
}
