package operations

// Ported from py-polars/tests/unit/operations/namespaces/string/test_string.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func strDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "s", Values: []any{"Foo", "BAR", "baz"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

func TestStrLen(t *testing.T) {
	t.Parallel()
	out, err := strDF(t).Select(polars.Col("s").StrLen().Alias("len"))
	if err != nil {
		t.Fatalf("str.len: %v", err)
	}
	s, _ := out.GetColumn("len")
	for i, w := range []int64{3, 3, 3} {
		if toFloatAny(s.Value(i)) != float64(w) {
			t.Fatalf("len[%d]: got %v, want %d", i, s.Value(i), w)
		}
	}
}

func TestStrLower(t *testing.T) {
	t.Parallel()
	out, err := strDF(t).Select(polars.Col("s").StrLower().Alias("lower"))
	if err != nil {
		t.Fatalf("str.lower: %v", err)
	}
	s, _ := out.GetColumn("lower")
	for i, w := range []string{"foo", "bar", "baz"} {
		if v, _ := s.Value(i).(string); v != w {
			t.Fatalf("lower[%d]: got %v, want %s", i, s.Value(i), w)
		}
	}
}

func TestStrUpper(t *testing.T) {
	t.Parallel()
	out, err := strDF(t).Select(polars.Col("s").StrUpper().Alias("upper"))
	if err != nil {
		t.Fatalf("str.upper: %v", err)
	}
	s, _ := out.GetColumn("upper")
	for i, w := range []string{"FOO", "BAR", "BAZ"} {
		if v, _ := s.Value(i).(string); v != w {
			t.Fatalf("upper[%d]: got %v, want %s", i, s.Value(i), w)
		}
	}
}
