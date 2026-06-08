package operations

// Ported from py-polars/tests/unit/operations/namespaces/string/test_string.py
// (py-1.28.1, representative subset for the str methods gopolars exposes)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func strMethodsDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "s", Values: []any{"  hello  ", "world", "foofoo"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// str.strip_chars() with no argument trims whitespace from both ends.
func TestStrTrim(t *testing.T) {
	t.Parallel()
	out, err := strMethodsDF(t).Select(polars.Col("s").StrTrim().Alias("t"))
	if err != nil {
		t.Fatalf("str.trim: %v", err)
	}
	col, _ := out.GetColumn("t")
	for i, w := range []string{"hello", "world", "foofoo"} {
		if v, _ := col.Value(i).(string); v != w {
			t.Fatalf("trim[%d]: got %q, want %q", i, col.Value(i), w)
		}
	}
}

// str.starts_with(prefix) returns a Boolean mask.
func TestStrStartsWith(t *testing.T) {
	t.Parallel()
	out, err := strMethodsDF(t).Select(polars.Col("s").StrTrim().StartsWith(polars.Lit("foo")).Alias("b"))
	if err != nil {
		t.Fatalf("starts_with: %v", err)
	}
	col, _ := out.GetColumn("b")
	for i, w := range []bool{false, false, true} {
		if v, _ := col.Value(i).(bool); v != w {
			t.Fatalf("starts_with[%d]: got %v, want %v", i, col.Value(i), w)
		}
	}
}

// str.contains(substr) returns a Boolean mask. The substrings used here have no
// regex metacharacters, so gopolars' literal match equals Python's default
// (regex) match.
func TestStrContains(t *testing.T) {
	t.Parallel()
	out, err := strMethodsDF(t).Select(polars.Col("s").StrTrim().Contains(polars.Lit("or")).Alias("b"))
	if err != nil {
		t.Fatalf("contains: %v", err)
	}
	col, _ := out.GetColumn("b")
	for i, w := range []bool{false, true, false} {
		if v, _ := col.Value(i).(bool); v != w {
			t.Fatalf("contains[%d]: got %v, want %v", i, col.Value(i), w)
		}
	}
}

// str.replace(pattern, value) replaces only the FIRST match (pattern is a regex):
// "foofoo" -> "barfoo".
func TestStrReplaceFirst(t *testing.T) {
	t.Parallel()
	out, err := strMethodsDF(t).Select(polars.Col("s").StrReplace("foo", "bar").Alias("r"))
	if err != nil {
		t.Fatalf("str.replace: %v", err)
	}
	col, _ := out.GetColumn("r")
	if v, _ := col.Value(2).(string); v != "barfoo" {
		t.Fatalf("replace[2]: got %q, want %q", col.Value(2), "barfoo")
	}
}

// str.replace_all(pattern, value) replaces every match: "foofoo" -> "barbar".
func TestStrReplaceAll(t *testing.T) {
	t.Parallel()
	out, err := strMethodsDF(t).Select(polars.Col("s").StrReplaceAll("foo", "bar").Alias("r"))
	if err != nil {
		t.Fatalf("str.replace_all: %v", err)
	}
	col, _ := out.GetColumn("r")
	if v, _ := col.Value(2).(string); v != "barbar" {
		t.Fatalf("replace_all[2]: got %q, want %q", col.Value(2), "barbar")
	}
}
