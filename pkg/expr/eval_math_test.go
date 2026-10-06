package expr

import (
	"errors"
	"math"
	"testing"
	"time"
)

func evalFloat(t *testing.T, e Expr, row RowValueGetter) float64 {
	t.Helper()
	v, err := Eval(e, row)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	f, ok := v.(float64)
	if !ok {
		t.Fatalf("result %v (%T) is not float64", v, v)
	}
	return f
}

// TestEvalMathUnary builds each numeric unary op via its builder and evaluates
// it, asserting against the math package. This covers both the expr.go builder
// and the matching Eval branch.
func TestEvalMathUnary(t *testing.T) {
	t.Parallel()

	const x = 0.5
	row := mapRow{"x": x}
	cases := []struct {
		name string
		e    Expr
		want float64
	}{
		{"abs", Col("x").Neg().Abs(), 0.5},
		{"exp", Col("x").Exp(), math.Exp(x)},
		{"log", Col("x").Log(), math.Log(x)},
		{"log10", Col("x").Log10(), math.Log10(x)},
		{"log1p", Col("x").Log1p(), math.Log1p(x)},
		{"sqrt", Col("x").Sqrt(), math.Sqrt(x)},
		{"cbrt", Col("x").Cbrt(), math.Cbrt(x)},
		{"ceil", Col("x").Ceil(), 1},
		{"floor", Col("x").Floor(), 0},
		{"sign", Col("x").Sign(), 1},
		{"sin", Col("x").Sin(), math.Sin(x)},
		{"cos", Col("x").Cos(), math.Cos(x)},
		{"tan", Col("x").Tan(), math.Tan(x)},
		{"sinh", Col("x").Sinh(), math.Sinh(x)},
		{"cosh", Col("x").Cosh(), math.Cosh(x)},
		{"tanh", Col("x").Tanh(), math.Tanh(x)},
		{"cot", Col("x").Cot(), 1 / math.Tan(x)},
		{"arcsin", Col("x").Arcsin(), math.Asin(x)},
		{"arccos", Col("x").Arccos(), math.Acos(x)},
		{"arctan", Col("x").Arctan(), math.Atan(x)},
		{"arcsinh", Col("x").Arcsinh(), math.Asinh(x)},
		{"arctanh", Col("x").Arctanh(), math.Atanh(x)},
		{"degrees", Col("x").Degrees(), x * 180 / math.Pi},
		{"radians", Col("x").Radians(), x * math.Pi / 180},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalFloat(t, tc.e, row)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("%s = %v, want %v", tc.name, got, tc.want)
			}
		})
	}

	// arccosh needs an argument >= 1.
	if got := evalFloat(t, Col("y").Arccosh(), mapRow{"y": 2.0}); math.Abs(got-math.Acosh(2)) > 1e-9 {
		t.Fatalf("arccosh = %v, want %v", got, math.Acosh(2))
	}
	// round on a float rounds half away from zero.
	if got := evalFloat(t, Col("z").Round(), mapRow{"z": 2.5}); got != 3 {
		t.Fatalf("round(2.5) = %v, want 3", got)
	}
	// round on int64 is identity (returns int64).
	if v, err := Eval(Col("z").Round(), mapRow{"z": int64(4)}); err != nil || v != int64(4) {
		t.Fatalf("round(int 4) = %v err=%v", v, err)
	}
	// neg on int64.
	if v, err := Eval(Col("z").Neg(), mapRow{"z": int64(5)}); err != nil || v != int64(-5) {
		t.Fatalf("neg(int 5) = %v err=%v", v, err)
	}
}

// TestEvalMathBinary covers the numeric binary ops.
func TestEvalMathBinary(t *testing.T) {
	t.Parallel()

	row := mapRow{"a": 8.0, "b": 3.0}

	if got := evalFloat(t, Col("a").Pow(Col("b")), row); got != 512 {
		t.Fatalf("pow = %v, want 512", got)
	}
	if got := evalFloat(t, Col("a").Mod(Col("b")), row); math.Abs(got-2) > 1e-9 {
		t.Fatalf("mod = %v, want 2", got)
	}
	if got := evalFloat(t, Col("a").FloorDiv(Col("b")), row); got != 2 {
		t.Fatalf("floordiv = %v, want 2", got)
	}
	if got := evalFloat(t, Col("a").Dot(Col("b")), row); got != 24 {
		t.Fatalf("dot = %v, want 24", got)
	}
	if got := evalFloat(t, Col("y").Atan2(Col("x")), mapRow{"y": 1.0, "x": 1.0}); math.Abs(got-math.Atan2(1, 1)) > 1e-9 {
		t.Fatalf("atan2 = %v", got)
	}

	// is_close: within 1e-9 tolerance.
	close, err := Eval(Col("a").IsClose(Lit(8.0)), row)
	if err != nil || close != true {
		t.Fatalf("is_close = %v err=%v", close, err)
	}
}

// TestEvalBitwise covers the bitwise unary and binary ops with hand-verified
// values.
func TestEvalBitwise(t *testing.T) {
	t.Parallel()

	row := mapRow{"a": int64(6), "b": int64(3), "seven": int64(7), "one": int64(1), "eight": int64(8), "neg1": int64(-1)}

	cases := []struct {
		name string
		e    Expr
		want int64
	}{
		{"and", Col("a").BitwiseAnd(Col("b")), 2},
		{"or", Col("a").BitwiseOr(Col("b")), 7},
		{"xor", Col("a").BitwiseXor(Col("b")), 5},
		{"count_ones", Col("seven").BitwiseCountOnes(), 3},
		{"count_zeros", Col("one").BitwiseCountZeros(), 63},
		{"leading_zeros", Col("one").BitwiseLeadingZeros(), 63},
		{"trailing_zeros", Col("eight").BitwiseTrailingZeros(), 3},
		{"trailing_ones", Col("seven").BitwiseTrailingOnes(), 3},
		{"leading_ones", Col("neg1").BitwiseLeadingOnes(), 64},
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

// TestEvalListAndMembership covers list/membership binary ops.
func TestEvalListAndMembership(t *testing.T) {
	t.Parallel()

	row := mapRow{"items": []any{int64(1), int64(2), int64(3)}, "v": int64(2)}

	got, err := Eval(Col("items").Get(Lit(int64(0))), row)
	if err != nil || got != int64(1) {
		t.Fatalf("get(0) = %v err=%v", got, err)
	}
	got, err = Eval(Col("items").Gather(Lit(int64(2))), row)
	if err != nil || got != int64(3) {
		t.Fatalf("gather(2) = %v err=%v", got, err)
	}
	got, err = Eval(Col("v").IsIn(Lit([]any{int64(1), int64(2)})), row)
	if err != nil || got != true {
		t.Fatalf("is_in = %v err=%v", got, err)
	}
	got, err = Eval(Col("items").IndexOf(Lit(int64(3))), row)
	if err != nil || got != int64(2) {
		t.Fatalf("index_of = %v err=%v", got, err)
	}
	got, err = Eval(Col("items").Append(Lit(int64(4))), row)
	if err != nil {
		t.Fatalf("append err=%v", err)
	}
	if list, ok := got.([]any); !ok || len(list) != 4 || list[3] != int64(4) {
		t.Fatalf("append = %v", got)
	}
	got, err = Eval(Col("v").EqMissing(Lit(int64(2))), row)
	if err != nil || got != true {
		t.Fatalf("eq_missing = %v err=%v", got, err)
	}
	got, err = Eval(Col("v").NeMissing(Lit(int64(9))), row)
	if err != nil || got != true {
		t.Fatalf("ne_missing = %v err=%v", got, err)
	}
}

// TestEvalFloatPredicates covers the finite/nan/infinite predicates.
func TestEvalFloatPredicates(t *testing.T) {
	t.Parallel()

	row := mapRow{"f": 1.5, "nan": math.NaN(), "inf": math.Inf(1)}

	checks := []struct {
		name string
		e    Expr
		want any
	}{
		{"is_finite", Col("f").IsFinite(), true},
		{"is_finite_inf", Col("inf").IsFinite(), false},
		{"is_infinite", Col("inf").IsInfinite(), true},
		{"is_nan", Col("nan").IsNan(), true},
		{"is_not_nan", Col("f").IsNotNan(), true},
	}
	for _, tc := range checks {
		got, err := Eval(tc.e, row)
		if err != nil {
			t.Errorf("%s: err %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestEvalLogicalReducers covers All/Any/Not_ over scalar and list operands and
// HasNulls over a list, plus FromJSON.
func TestEvalLogicalReducers(t *testing.T) {
	t.Parallel()

	row := mapRow{
		"allTrue":  []any{true, true},
		"someTrue": []any{false, true},
		"flag":     true,
		"withNull": []any{int64(1), nil},
		"json":     `{"k":1}`,
	}

	if got, err := Eval(Col("allTrue").All(), row); err != nil || got != true {
		t.Fatalf("all = %v err=%v", got, err)
	}
	if got, err := Eval(Col("someTrue").All(), row); err != nil || got != false {
		t.Fatalf("all(some) = %v err=%v", got, err)
	}
	if got, err := Eval(Col("someTrue").Any(), row); err != nil || got != true {
		t.Fatalf("any = %v err=%v", got, err)
	}
	if got, err := Eval(Col("flag").Not_(), row); err != nil || got != false {
		t.Fatalf("not_ = %v err=%v", got, err)
	}
	if got, err := Eval(Col("withNull").HasNulls(), row); err != nil || got != true {
		t.Fatalf("has_nulls = %v err=%v", got, err)
	}
	got, err := Eval(Col("json").FromJSON(), row)
	if err != nil {
		t.Fatalf("from_json err=%v", err)
	}
	if m, ok := got.(map[string]any); !ok || m["k"] != float64(1) {
		t.Fatalf("from_json = %v", got)
	}
}

// TestEvalNumericTernary covers Clip and IsBetween.
func TestEvalNumericTernary(t *testing.T) {
	t.Parallel()

	row := mapRow{"v": 5.0, "lo": 2.0, "hi": 10.0}

	clip, err := Eval(Col("v").Clip(Lit(2.0), Lit(4.0)), row)
	if err != nil || clip != 4.0 {
		t.Fatalf("clip(5,[2,4]) = %v err=%v, want 4", clip, err)
	}
	clipLow, err := Eval(Col("lo").Clip(Lit(3.0), Lit(10.0)), row)
	if err != nil || clipLow != 3.0 {
		t.Fatalf("clip(2,[3,10]) = %v err=%v, want 3", clipLow, err)
	}
	between, err := Eval(Col("v").IsBetween(Col("lo"), Col("hi")), row)
	if err != nil || between != true {
		t.Fatalf("is_between = %v err=%v", between, err)
	}
}

// TestEvalStringKernels covers the string builders end to end.
func TestEvalStringKernels(t *testing.T) {
	t.Parallel()

	row := mapRow{"s": "hello world", "a": "foo", "b": "bar", "pad": "ab", "csv": "x,y,z"}

	cases := []struct {
		name string
		e    Expr
		want any
	}{
		{"str_like", Col("s").StrLike("hello%"), true},
		{"str_like_no", Col("s").StrLike("bye%"), false},
		{"str_concat", Col("a").StrConcat(Col("b")), "foobar"},
		{"str_substr", Col("s").StrSubstr(1, 5), "hello"},
		{"str_ltrim", Lit("  x").StrLTrim(), "x"},
		{"str_rtrim", Lit("x  ").StrRTrim(), "x"},
		{"str_reverse", Lit("abc").StrReverse(), "cba"},
		{"str_to_title", Lit("hello world").StrToTitle(), "Hello World"},
		{"str_char_len", Lit("héllo").StrCharLen(), int64(5)},
		{"ends_with", Col("s").EndsWith(Lit("world")), true},
		{"str_pos", Col("s").StrPos(Lit("world")), int64(7)},
		{"str_left", Col("s").StrLeft(Lit(int64(5))), "hello"},
		{"str_right", Col("s").StrRight(Lit(int64(5))), "world"},
		{"regexp_like", Col("s").RegexpLike(Lit("^h.*d$")), true},
		{"concat_ws", Col("a").StrConcatWS("-", Col("b")), "foo-bar"},
		{"pad_start", Col("pad").StrPadStart(Lit(int64(4)), Lit("*")), "**ab"},
		{"pad_end", Col("pad").StrPadEnd(Lit(int64(4)), Lit("*")), "ab**"},
		{"split_part", Col("csv").StrSplitPart(Lit(","), Lit(int64(2))), "y"},
		{"round_dp", Lit(3.14159).RoundDP(2), 3.14},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Eval(tc.e, row)
			if err != nil {
				t.Fatalf("eval: %v", err)
			}
			if got != tc.want {
				t.Fatalf("%s = %v (%T), want %v (%T)", tc.name, got, got, tc.want, tc.want)
			}
		})
	}
}

// TestEvalDatetimeKernels covers the datetime accessors.
func TestEvalDatetimeKernels(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 3, 1, 14, 25, 36, 0, time.UTC)
	row := mapRow{"ts": ts}

	minute, err := Eval(Col("ts").DtMinute(), row)
	if err != nil || minute != int64(25) {
		t.Fatalf("dt_minute = %v err=%v", minute, err)
	}
	second, err := Eval(Col("ts").DtSecond(), row)
	if err != nil || second != int64(36) {
		t.Fatalf("dt_second = %v err=%v", second, err)
	}
	// 2026-03-01 is the 60th day of the year.
	ordinal, err := Eval(Col("ts").DtOrdinalDay(), row)
	if err != nil || ordinal != int64(60) {
		t.Fatalf("dt_ordinal_day = %v err=%v", ordinal, err)
	}
}

// evalUnaryOp evaluates the unary op on the single value v.
func evalUnaryOp(op string, v any) (any, error) {
	target := Col("v")
	return Eval(Expr{kind: KindUnary, op: op, target: &target}, mapRow{"v": v})
}

// TestEvalNumericUnaryFamilies pins every float-math, datetime and bitwise unary
// op: its value for each accepted operand type, and the exact, unwrapped error
// text for a mismatched operand and for null.
func TestEvalNumericUnaryFamilies(t *testing.T) {
	t.Parallel()

	floatOps := map[string]func(float64) float64{
		"arccos": math.Acos, "arccosh": math.Acosh, "arcsin": math.Asin,
		"arcsinh": math.Asinh, "arctan": math.Atan, "arctanh": math.Atanh,
		"cbrt": math.Cbrt, "ceil": math.Ceil, "cos": math.Cos, "sin": math.Sin,
		"cosh": math.Cosh, "sinh": math.Sinh, "tan": math.Tan, "tanh": math.Tanh,
		"floor": math.Floor, "log10": math.Log10, "log1p": math.Log1p,
		"log": math.Log, "sqrt": math.Sqrt, "exp": math.Exp,
		"cot":     func(f float64) float64 { return 1 / math.Tan(f) },
		"degrees": func(f float64) float64 { return f * 180 / math.Pi },
		"radians": func(f float64) float64 { return f * math.Pi / 180 },
		"sign": func(f float64) float64 {
			switch {
			case f < 0:
				return -1
			case f > 0:
				return 1
			}
			return 0
		},
	}
	inputs := []struct {
		v any
		f float64
	}{
		{0.25, 0.25}, {-2.0, -2}, {0.0, 0}, {math.NaN(), math.NaN()}, {math.Inf(1), math.Inf(1)},
		{int64(3), 3}, {int64(-4), -4},
	}
	for op, fn := range floatOps {
		for _, in := range inputs {
			got, err := evalUnaryOp(op, in.v)
			want := fn(in.f)
			g, ok := got.(float64)
			if err != nil || !ok || (g != want && !(math.IsNaN(g) && math.IsNaN(want))) {
				t.Errorf("%s(%v) = %v (%T), %v; want %v", op, in.v, got, got, err, want)
			}
		}
		for _, bad := range []any{"x", nil, true} {
			got, err := evalUnaryOp(op, bad)
			if got != nil || err == nil || err.Error() != op+" expects numeric" || errors.Unwrap(err) != nil {
				t.Errorf("%s(%v) = %v, %v; want %q", op, bad, got, err, op+" expects numeric")
			}
		}
	}

	sunday := time.Date(2024, 3, 17, 13, 45, 30, 0, time.UTC)
	monday := time.Date(2025, 12, 1, 0, 1, 2, 0, time.UTC)
	dtOps := []struct {
		op           string
		sunday, mond int64
	}{
		{"dt_year", 2024, 2025}, {"dt_month", 3, 12}, {"dt_day", 17, 1}, {"dt_hour", 13, 0},
		{"dt_weekday", 7, 1}, {"dt_minute", 45, 1}, {"dt_second", 30, 2},
	}
	for _, tc := range dtOps {
		for in, want := range map[time.Time]int64{sunday: tc.sunday, monday: tc.mond} {
			if got, err := evalUnaryOp(tc.op, in); err != nil || got != want {
				t.Errorf("%s(%v) = %v, %v; want %d", tc.op, in, got, err, want)
			}
		}
		for _, bad := range []any{"x", nil, int64(1)} {
			got, err := evalUnaryOp(tc.op, bad)
			if got != nil || err == nil || err.Error() != tc.op+" expects datetime" || errors.Unwrap(err) != nil {
				t.Errorf("%s(%v) = %v, %v; want %q", tc.op, bad, got, err, tc.op+" expects datetime")
			}
		}
	}

	// Operands: int64(11) = 0b1011, int64(-1) (all ones), 6.9 (truncated to 6 = 0b110).
	bitOps := []struct {
		op   string
		want [3]int64
	}{
		{"bitwise_count_ones", [3]int64{3, 64, 2}},
		{"bitwise_count_zeros", [3]int64{61, 0, 62}},
		{"bitwise_leading_ones", [3]int64{0, 64, 0}},
		{"bitwise_leading_zeros", [3]int64{60, 0, 61}},
		{"bitwise_trailing_ones", [3]int64{2, 64, 0}},
		{"bitwise_trailing_zeros", [3]int64{0, 0, 1}},
	}
	for _, tc := range bitOps {
		for i, in := range []any{int64(11), int64(-1), 6.9} {
			if got, err := evalUnaryOp(tc.op, in); err != nil || got != tc.want[i] {
				t.Errorf("%s(%v) = %v, %v; want %d", tc.op, in, got, err, tc.want[i])
			}
		}
		for _, bad := range []any{"x", nil, true} {
			got, err := evalUnaryOp(tc.op, bad)
			if got != nil || err == nil || err.Error() != tc.op+" expects int" || errors.Unwrap(err) != nil {
				t.Errorf("%s(%v) = %v, %v; want %q", tc.op, bad, got, err, tc.op+" expects int")
			}
		}
	}

	// Pass-through unary ops return the operand unchanged, null included.
	for _, op := range []string{"cum_sum", "cum_count", "rank", "dt", "str", "list", "struct", "agg_groups", "var"} {
		for _, in := range []any{int64(7), nil} {
			if got, err := evalUnaryOp(op, in); err != nil || got != in {
				t.Errorf("%s(%v) = %v, %v; want the operand", op, in, got, err)
			}
		}
	}
}
