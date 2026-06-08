package operations

// Ported from py-polars/tests/unit/operations/test_comparison.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func cmpDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

func cmpSelect(t *testing.T, e polars.Expr) []bool {
	t.Helper()
	out, err := cmpDF(t).Select(e.Alias("r"))
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	s, _ := out.GetColumn("r")
	res := make([]bool, s.Len())
	for i := 0; i < s.Len(); i++ {
		res[i], _ = s.Value(i).(bool)
	}
	return res
}

func eqBools(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestComparisonOperators(t *testing.T) {
	t.Parallel()
	two := polars.Lit(int64(2))
	cases := []struct {
		name string
		expr polars.Expr
		want []bool
	}{
		{"gt", polars.Col("a").Gt(two), []bool{false, false, true}},
		{"ge", polars.Col("a").Ge(two), []bool{false, true, true}},
		{"lt", polars.Col("a").Lt(two), []bool{true, false, false}},
		{"le", polars.Col("a").Le(two), []bool{true, true, false}},
		{"eq", polars.Col("a").Eq(two), []bool{false, true, false}},
		{"ne", polars.Col("a").Ne(two), []bool{true, false, true}},
	}
	for _, c := range cases {
		if got := cmpSelect(t, c.expr); !eqBools(got, c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
