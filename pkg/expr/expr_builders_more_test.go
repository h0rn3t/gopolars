package expr

import (
	"math"
	"testing"
	"time"
)

// TestStringBuilderEval covers the string-op builders end to end: each builder
// produces an Expr whose evaluation matches the documented semantics.
func TestStringBuilderEval(t *testing.T) {
	t.Parallel()

	row := mapRow{"s": "Hello World"}
	cases := []struct {
		name string
		e    Expr
		want any
	}{
		{"str_len", Col("s").StrLen(), int64(11)},
		{"str_lower", Col("s").StrLower(), "hello world"},
		{"str_upper", Col("s").StrUpper(), "HELLO WORLD"},
		{"str_replace_all", Col("s").StrReplaceAll("o", "0"), "Hell0 W0rld"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Eval(tc.e, row)
			if err != nil {
				t.Fatalf("eval: %v", err)
			}
			if got != tc.want {
				t.Fatalf("%s = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestStringPredicateBuilders covers StartsWith / Contains, which build binary
// exprs evaluated against a literal operand.
func TestStringPredicateBuilders(t *testing.T) {
	t.Parallel()

	row := mapRow{"s": "gopolars"}

	starts, err := Eval(Col("s").StartsWith(Lit("go")), row)
	if err != nil || starts != true {
		t.Fatalf("starts_with = %v err=%v", starts, err)
	}
	contains, err := Eval(Col("s").Contains(Lit("pol")), row)
	if err != nil || contains != true {
		t.Fatalf("contains = %v err=%v", contains, err)
	}
	missing, err := Eval(Col("s").Contains(Lit("zzz")), row)
	if err != nil || missing != false {
		t.Fatalf("contains(zzz) = %v err=%v", missing, err)
	}
}

// TestDatetimeBuilderEval covers the datetime accessor builders.
func TestDatetimeBuilderEval(t *testing.T) {
	t.Parallel()

	// 2026-06-14 is a Sunday -> ISO weekday 7.
	ts := time.Date(2026, 6, 14, 9, 30, 0, 0, time.UTC)
	row := mapRow{"ts": ts}

	year, err := Eval(Col("ts").DtYear(), row)
	if err != nil || year != int64(2026) {
		t.Fatalf("dt_year = %v err=%v", year, err)
	}
	hour, err := Eval(Col("ts").DtHour(), row)
	if err != nil || hour != int64(9) {
		t.Fatalf("dt_hour = %v err=%v", hour, err)
	}
	weekday, err := Eval(Col("ts").DtWeekday(), row)
	if err != nil || weekday != int64(7) {
		t.Fatalf("dt_weekday = %v err=%v", weekday, err)
	}
}

// TestListAndStructBuilders covers ListLen, ListGet, and StructField.
func TestListAndStructBuilders(t *testing.T) {
	t.Parallel()

	row := mapRow{
		"items":  []any{int64(10), int64(20), int64(30)},
		"record": map[string]any{"city": "kyiv", "pop": int64(3)},
	}

	length, err := Eval(Col("items").ListLen(), row)
	if err != nil || length != int64(3) {
		t.Fatalf("list_len = %v err=%v", length, err)
	}
	got, err := Eval(Col("items").ListGet(Lit(int64(1))), row)
	if err != nil || got != int64(20) {
		t.Fatalf("list_get(1) = %v err=%v", got, err)
	}
	oob, err := Eval(Col("items").ListGet(Lit(int64(9))), row)
	if err != nil || oob != nil {
		t.Fatalf("list_get(9) = %v err=%v, want nil", oob, err)
	}
	field, err := Eval(Col("record").StructField("city"), row)
	if err != nil || field != "kyiv" {
		t.Fatalf("struct_field(city) = %v err=%v", field, err)
	}
}

// TestReducerAndTransformBuilders verifies the AST shape of reducer/transform
// builders. These nodes are interpreted by the frame/group layers (the row-wise
// Eval treats them as pass-throughs), so the contract is the Op()/Kind() they
// produce, which the downstream layers switch on.
func TestReducerAndTransformBuilders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		e      Expr
		wantOp string
	}{
		{"sum", Col("v").Sum(), "sum"},
		{"mean", Col("v").Mean(), "mean"},
		{"std", Col("v").Std(), "std"},
		{"var", Col("v").Var(), "var"},
		{"skew", Col("v").Skew(), "skew"},
		{"reverse", Col("v").Reverse(), "reverse"},
		{"cum_sum", Col("v").CumSum(), "cum_sum"},
		{"cum_count", Col("v").CumCount(), "cum_count"},
		{"rank", Col("v").Rank(), "rank"},
		{"top_k", Col("v").TopK(3), "top_k:3"},
		{"upper_bound", Col("v").UpperBound(), "upper_bound"},
		{"to_physical", Col("v").ToPhysical(), "to_physical"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.e.Kind() != KindUnary {
				t.Fatalf("%s kind = %q, want unary", tc.name, tc.e.Kind())
			}
			if tc.e.Op() != tc.wantOp {
				t.Fatalf("%s op = %q, want %q", tc.name, tc.e.Op(), tc.wantOp)
			}
			if tc.e.Target() == nil || tc.e.Target().ColName() != "v" {
				t.Fatalf("%s target not preserved: %+v", tc.name, tc.e.Target())
			}
		})
	}
}

// TestTernaryBuilders covers TopKBy and Over, which build ternary/over nodes.
func TestTernaryBuilders(t *testing.T) {
	t.Parallel()

	topKBy := Col("v").TopKBy(Col("w"), 2)
	if topKBy.Kind() != KindTern || topKBy.Op() != "top_k_by:2" {
		t.Fatalf("top_k_by kind/op = %q/%q", topKBy.Kind(), topKBy.Op())
	}
	if topKBy.Left() == nil || topKBy.Left().ColName() != "w" {
		t.Fatalf("top_k_by by-operand not preserved")
	}

	over := Col("v").Sum().Over("grp", "region")
	if over.Op() != "over:grp,region" {
		t.Fatalf("over op = %q, want over:grp,region", over.Op())
	}
}

// TestRollingBuilders verifies each rolling builder encodes its window size in
// the op string (the executor parses it back out).
func TestRollingBuilders(t *testing.T) {
	t.Parallel()

	cases := map[string]Expr{
		"rolling_min:4":  Col("v").RollingMin(4),
		"rolling_max:4":  Col("v").RollingMax(4),
		"rolling_mean:4": Col("v").RollingMean(4),
		"rolling_sum:4":  Col("v").RollingSum(4),
		"rolling_std:4":  Col("v").RollingStd(4),
		"rolling_var:4":  Col("v").RollingVar(4),
	}
	for wantOp, e := range cases {
		if e.Op() != wantOp {
			t.Errorf("rolling op = %q, want %q", e.Op(), wantOp)
		}
		if e.Kind() != KindUnary {
			t.Errorf("rolling %q kind = %q, want unary", wantOp, e.Kind())
		}
	}
}

// TestFillAndReplaceBuilders covers FillNull, FillNaN (binary fill exprs) and
// Replace (ternary), evaluated against concrete rows.
func TestFillAndReplaceBuilders(t *testing.T) {
	t.Parallel()

	// FillNull replaces a nil operand with the fallback.
	row := mapRow{"a": nil, "b": int64(5)}
	filled, err := Eval(Col("a").FillNull(Lit(int64(-1))), row)
	if err != nil || filled != int64(-1) {
		t.Fatalf("fill_null = %v err=%v", filled, err)
	}
	kept, err := Eval(Col("b").FillNull(Lit(int64(-1))), row)
	if err != nil || kept != int64(5) {
		t.Fatalf("fill_null kept = %v err=%v", kept, err)
	}

	// FillNaN replaces a NaN float with the fallback.
	nanRow := mapRow{"f": math.NaN(), "g": float64(2.5)}
	fn, err := Eval(Col("f").FillNaN(Lit(float64(0))), nanRow)
	if err != nil || fn != float64(0) {
		t.Fatalf("fill_nan = %v err=%v", fn, err)
	}
	keptF, err := Eval(Col("g").FillNaN(Lit(float64(0))), nanRow)
	if err != nil || keptF != float64(2.5) {
		t.Fatalf("fill_nan kept = %v err=%v", keptF, err)
	}

	// Replace swaps an exact match, otherwise passes through.
	repRow := mapRow{"x": int64(7)}
	rep, err := Eval(Col("x").Replace(Lit(int64(7)), Lit(int64(70))), repRow)
	if err != nil || rep != int64(70) {
		t.Fatalf("replace match = %v err=%v", rep, err)
	}
	noRep, err := Eval(Col("x").Replace(Lit(int64(8)), Lit(int64(80))), repRow)
	if err != nil || noRep != int64(7) {
		t.Fatalf("replace no-match = %v err=%v", noRep, err)
	}
}

// TestConditionalBuilders covers When/Where/Xor/TrueDiv.
func TestConditionalBuilders(t *testing.T) {
	t.Parallel()

	row := mapRow{"a": int64(10), "flag": true, "p": true, "q": false}

	// When(cond, then, otherwise)
	when := When(Col("a").Gt(Lit(int64(5))), Lit("big"), Lit("small"))
	got, err := Eval(when, row)
	if err != nil || got != "big" {
		t.Fatalf("when true = %v err=%v", got, err)
	}
	got, err = Eval(When(Col("a").Lt(Lit(int64(5))), Lit("big"), Lit("small")), row)
	if err != nil || got != "small" {
		t.Fatalf("when false = %v err=%v", got, err)
	}

	// Where keeps the value when the mask is true, else nil.
	kept, err := Eval(Col("a").Where(Col("flag")), row)
	if err != nil || kept != int64(10) {
		t.Fatalf("where true = %v err=%v", kept, err)
	}

	// Xor is boolean exclusive-or.
	xor, err := Eval(Col("p").Xor(Col("q")), row)
	if err != nil || xor != true {
		t.Fatalf("xor = %v err=%v", xor, err)
	}

	// TrueDiv divides (as div); use matching float operands.
	div, err := Eval(Col("a").TrueDiv(Lit(float64(10))), mapRow{"a": float64(10)})
	if err != nil || div != float64(1) {
		t.Fatalf("true_div = %v err=%v", div, err)
	}
}

// TestNameAndNames covers the Name()/Names() metadata helpers.
func TestNameAndNames(t *testing.T) {
	t.Parallel()

	if got := Col("city").Name(); got != "city" {
		t.Fatalf("col name = %q, want city", got)
	}
	if got := Col("city").Alias("location").Name(); got != "location" {
		t.Fatalf("aliased name = %q, want location", got)
	}
	if got := Sum(Col("v")).Name(); got != "sum_v" {
		t.Fatalf("agg name = %q, want sum_v", got)
	}
	if got := Lit(int64(1)).Name(); got != "expr" {
		t.Fatalf("literal name = %q, want expr", got)
	}

	cols := Cols("a", "b", "c")
	if names := cols.Names(); len(names) != 3 || names[0] != "a" || names[2] != "c" {
		t.Fatalf("Names() = %v, want [a b c]", names)
	}
	if Col("a").Names() != nil {
		t.Fatalf("plain col Names() should be nil")
	}
}

// TestMapAggregates folds an aggregation nested in a row-wise expr into a
// literal, broadcasting it.
func TestMapAggregates(t *testing.T) {
	t.Parallel()

	// pl.col("b") + pl.col("v").sum()  — the sum() folds to the precomputed 100.
	e := Col("b").Add(Sum(Col("v")))
	folded, err := MapAggregates(e, func(node Expr) (any, bool, error) {
		if node.Kind() == KindAgg && node.Op() == "sum" {
			return int64(100), true, nil
		}
		return nil, false, nil
	})
	if err != nil {
		t.Fatalf("map_aggregates: %v", err)
	}
	got, err := Eval(folded, mapRow{"b": int64(5)})
	if err != nil || got != int64(105) {
		t.Fatalf("folded eval = %v err=%v, want 105", got, err)
	}
}

// TestMapColumnNames rewrites column references and leaves the wildcard intact.
func TestMapColumnNames(t *testing.T) {
	t.Parallel()

	e := Col("a").Add(Col("b"))
	renamed, err := MapColumnNames(e, func(name string) (string, error) {
		return "t_" + name, nil
	})
	if err != nil {
		t.Fatalf("map_column_names: %v", err)
	}
	got, err := Eval(renamed, mapRow{"t_a": int64(3), "t_b": int64(4)})
	if err != nil || got != int64(7) {
		t.Fatalf("renamed eval = %v err=%v, want 7", got, err)
	}

	// "*" wildcard is left untouched.
	star, err := MapColumnNames(All(), func(string) (string, error) {
		return "should_not_be_used", nil
	})
	if err != nil || star.ColName() != "*" {
		t.Fatalf("wildcard rewritten: %q err=%v", star.ColName(), err)
	}
}
