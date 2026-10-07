package expr

import (
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

func TestInputColumns(t *testing.T) {
	t.Parallel()

	add := func(acc, next any) (any, error) { return acc, nil }
	tests := []struct {
		name     string
		e        Expr
		wantCols []string
		wantOK   bool
	}{
		{"column", Col("a"), []string{"a"}, true},
		{"aliased column", Col("a").Alias("x"), []string{"a"}, true},
		{"literal", Lit(int64(1)), nil, true},
		{"count without target", Count(), nil, true},
		{"nested binary distinct", Col("b").Add(Col("a")).Gt(Col("b")), []string{"b", "a"}, true},
		{"unary", Col("s").StrLen(), []string{"s"}, true},
		{"cast", Col("a").Cast(dtypes.Float64), []string{"a"}, true},
		{"aggregation", Sum(Col("v")), []string{"v"}, true},
		{"when", When(Col("c").Gt(Lit(1)), Col("d").Abs(), Col("e")), []string{"c", "d", "e"}, true},
		{"clip", Col("x").Clip(Col("lo"), Col("hi")), []string{"x", "lo", "hi"}, true},
		{"over partitions", Col("v").CumSum().Over("p1", "p2"), []string{"v", "p1", "p2"}, true},
		{"over without partitions", Col("v").Sum().Over(), []string{"v"}, true},
		{"over partition repeats target", Col("p").Rank().Over("p"), []string{"p"}, true},
		{"fold operands", Fold(int64(0), add, Col("a"), Col("b").Mul(Col("c"))), []string{"a", "b", "c"}, true},
		{"struct_pack", StructCols("x", "y"), []string{"x", "y"}, true},
		{"struct_pack field", StructCols("x", "y").StructField("y"), []string{"x", "y"}, true},
		{"cols selector", Cols("b", "a"), []string{"b", "a"}, true},
		{"all", All(), nil, false},
		{"exclude", Exclude("a"), nil, false},
		{"regex column", Col("^a.*$"), nil, false},
		{"selector nested in binary", Col("a").Add(Col("^x$")), nil, false},
		{"selector in fold", Fold(int64(0), add, Col("a"), All()), nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cols, ok := InputColumns(tt.e)
			if !slices.Equal(cols, tt.wantCols) || ok != tt.wantOK {
				t.Errorf("InputColumns(%s) = %q, %t; want %q, %t", tt.name, cols, ok, tt.wantCols, tt.wantOK)
			}
		})
	}
}

func TestIsElementwise(t *testing.T) {
	t.Parallel()

	add := func(acc, next any) (any, error) { return acc, nil }
	a := Col("a")
	tests := []struct {
		name string
		e    Expr
		want bool
	}{
		{"column", a, true},
		{"literal", Lit(int64(1)), true},
		{"all selector", All(), true},
		{"cols selector", Cols("a", "b"), true},
		{"cast", a.Cast(dtypes.Float64), true},
		{"comparison", a.Gt(Lit(int64(1))).Or(a.Le(Col("b"))), true},
		{"arithmetic", a.Add(Col("b")).Mul(Lit(int64(2))).Sub(a.Mod(Lit(int64(3)))), true},
		{"boolean", a.Gt(Lit(int64(1))).And(Col("b").IsNull()).Not(), true},
		{"when", When(a.Gt(Lit(int64(1))), Col("b"), Lit(int64(0))), true},
		{"string", Col("s").StrUpper().StrReplaceAll("x", "y").StrSubstr(1, 2).StrLen(), true},
		{"string binary", Col("s").Contains(Lit("x")).And(Col("s").StrConcatWS("-", Col("t")).EndsWith(Lit("z"))), true},
		{"string ternary", Col("s").StrPadStart(Lit(int64(3)), Lit("0")), true},
		{"datetime", Col("t").DtYear().Add(Col("t").DtOrdinalDay()), true},
		{"math", a.Sqrt().Abs().RoundDP(2).Pow(Lit(2.0)), true},
		{"bitwise", a.BitwiseCountOnes().BitwiseAnd(Lit(int64(1))), true},
		{"clip", a.Clip(Lit(int64(0)), Lit(int64(9))), true},
		{"is_between", a.IsBetween(Lit(int64(0)), Lit(int64(9))), true},
		{"fill_null literal", a.FillNull(Lit(int64(0))), true},
		{"is_in literal", a.IsIn(Lit([]any{int64(1), int64(2)})), true},
		{"struct field", StructCols("a", "b").StructField("a"), true},
		{"fold", Fold(int64(0), add, a, Col("b").Neg()), true},
		{"sum aggregation", Sum(a), false},
		{"count aggregation", Count(), false},
		{"sum method", a.Sum(), false},
		{"mean method", a.Mean(), false},
		{"nested aggregation", a.Sub(a.Mean()), false},
		{"over", a.Over("p"), false},
		{"elementwise over", a.Add(Lit(int64(1))).Over("p"), false},
		{"cum_sum", a.CumSum(), false},
		{"cum_count", a.CumCount(), false},
		{"shift", a.Shift(1), false},
		{"rolling", a.RollingMean(2), false},
		{"rolling by", a.RollingSumBy(Col("t"), 2), false},
		{"sort", a.Sort(false), false},
		{"sort by", a.SortBy(Col("b"), true), false},
		{"reverse", a.Reverse(), false},
		{"rank", a.Rank(), false},
		{"diff", a.Diff(), false},
		{"first", a.First(), false},
		{"head", a.Head(2), false},
		{"unique", a.Unique(), false},
		{"filter", a.Filter(Col("b").Gt(Lit(int64(0)))), false},
		{"is_in column", a.IsIn(Col("b")), false},
		{"fill_null aggregation", a.FillNull(a.Mean()), false},
		{"fold stateful operand", Fold(int64(0), add, a, Col("b").CumSum()), false},
		{"unknown unary", Expr{kind: KindUnary, op: "made_up", target: &a}, false},
		{"unknown binary", Expr{kind: KindBin, op: "made_up", left: &a, right: &a}, false},
		{"unknown ternary", a.InterpolateBy(Col("t")), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsElementwise(tt.e); got != tt.want {
				t.Errorf("IsElementwise(%s) = %t, want %t", tt.name, got, tt.want)
			}
		})
	}
}
