package expr

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

type mapRow map[string]any

func (m mapRow) ValueByName(name string) (any, bool) {
	v, ok := m[name]
	return v, ok
}

func TestEvalCastAndComparisons(t *testing.T) {
	row := mapRow{"id": int64(2), "name": "beta", "score": float64(2.5)}

	got, err := Eval(Col("id").Cast(dtypes.Float64), row)
	if err != nil || got != float64(2) {
		t.Fatalf("cast int64->float64: %v err=%v", got, err)
	}

	ne, err := Eval(Col("id").Ne(Lit(int64(1))), row)
	if err != nil || ne != true {
		t.Fatalf("ne: %v err=%v", ne, err)
	}

	lt, err := Eval(Col("name").Lt(Lit("gamma")), row)
	if err != nil || lt != true {
		t.Fatalf("string lt: %v err=%v", lt, err)
	}

	sum, err := Eval(Col("id").Add(Lit(int64(3))), row)
	if err != nil || sum != int64(5) {
		t.Fatalf("add: %v err=%v", sum, err)
	}

	mul, err := Eval(Col("score").Mul(Lit(float64(2))), row)
	if err != nil || mul != float64(5) {
		t.Fatalf("mul: %v err=%v", mul, err)
	}

	ge, err := Eval(Col("score").Ge(Lit(float64(2))), row)
	if err != nil || ge != true {
		t.Fatalf("float ge: %v err=%v", ge, err)
	}

	asStr, err := Eval(Col("id").Cast(dtypes.String), row)
	if err != nil || asStr != "2" {
		t.Fatalf("cast to string: %v err=%v", asStr, err)
	}

	toInt, err := Eval(Col("score").Cast(dtypes.Int64), row)
	if err != nil || toInt != int64(2) {
		t.Fatalf("cast float to int64: %v err=%v", toInt, err)
	}
}

func TestEvalBooleanAndDatetimeCast(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC)
	row := mapRow{"flag": true, "ts": ts}

	asBool, err := Eval(Col("flag").Cast(dtypes.Boolean), row)
	if err != nil || asBool != true {
		t.Fatalf("cast bool: %v err=%v", asBool, err)
	}

	_, err = Eval(Lit("x").Cast(dtypes.Datetime), row)
	if err == nil {
		t.Fatal("очікували помилку cast до datetime")
	}

	le, err := Eval(Col("ts").Le(Lit(ts.Add(time.Hour))), row)
	if err != nil || le != true {
		t.Fatalf("datetime le: %v err=%v", le, err)
	}
}

func TestEvalUnaryAndLogicalOps(t *testing.T) {
	row := mapRow{"flag": true, "name": "  Kyiv  ", "items": []any{int64(1), int64(2)}}

	notVal, err := Eval(Col("flag").Not(), row)
	if err != nil || notVal != false {
		t.Fatalf("not: %v err=%v", notVal, err)
	}

	trimmed, err := Eval(Col("name").StrTrim(), row)
	if err != nil || trimmed != "Kyiv" {
		t.Fatalf("str_trim: %v err=%v", trimmed, err)
	}

	replaced, err := Eval(Col("name").StrReplace("Kyiv", "Київ"), row)
	if err != nil {
		t.Fatalf("str_replace: %v", err)
	}
	if replaced == nil {
		t.Fatal("str_replace повернув nil")
	}

	contains, err := Eval(Col("items").ListContains(Lit(int64(2))), row)
	if err != nil || contains != true {
		t.Fatalf("list_contains: %v err=%v", contains, err)
	}
}

func TestEvalDatetimeParts(t *testing.T) {
	ts := time.Date(2026, 5, 28, 15, 30, 0, 0, time.UTC)
	row := mapRow{"ts": ts}

	month, err := Eval(Col("ts").DtMonth(), row)
	if err != nil || month != int64(5) {
		t.Fatalf("dt_month: %v err=%v", month, err)
	}
	day, err := Eval(Col("ts").DtDay(), row)
	if err != nil || day != int64(28) {
		t.Fatalf("dt_day: %v err=%v", day, err)
	}
}

func TestExprBuildersCoverage(t *testing.T) {
	_ = Min(Col("v"))
	_ = Max(Col("v"))
	_ = Count()
	_ = NUnique(Col("v"))
	_ = Col("v").Sub(Lit(int64(1)))
	_ = Col("v").Div(Lit(int64(2)))
	_ = Col("a").And(Col("b"))
	_ = Col("a").Or(Col("b"))
	_ = Col("a").IsNull()
	_ = Col("a").IsNotNull()
}

// TestEvalParamUnaryDispatch pins how "name:arg" unary ops route: known names
// evaluate, malformed arguments report their op, and unknown names fall through
// to the unsupported-op error.
func TestEvalParamUnaryDispatch(t *testing.T) {
	t.Parallel()

	row := mapRow{"x": int64(7), "s": "a-b-b"}
	cases := []struct {
		op      string
		col     string
		want    any
		wantErr string
	}{
		{op: "head:3", col: "x", want: int64(7)},
		{op: "rolling_rank:2", col: "x", want: int64(7)},
		{op: "round_dp:2", col: "x", want: int64(7)},
		{op: "str_replace:b:c", col: "s", want: "a-c-b"},
		{op: "str_replace_all:b:c", col: "s", want: "a-c-c"},
		{op: "str_substr:3:1", col: "s", want: "b"},
		{op: "str_replace:b", col: "s", wantErr: "invalid str_replace configuration"},
		{op: "str_substr:x:1", col: "s", wantErr: "invalid str_substr configuration"},
		{op: "round_dp:x", col: "x", wantErr: "invalid round_dp configuration"},
		{op: "struct_field:k", col: "x", wantErr: "struct_field expects struct"},
		{op: "round_sig_figs:2", col: "s", wantErr: "round_sig_figs expects numeric"},
		{op: "nope:1", col: "x", wantErr: "unsupported unary op nope:1"},
		{op: "head", col: "x", wantErr: "unsupported unary op head"},
	}
	for _, tc := range cases {
		target := Col(tc.col)
		e := Expr{kind: KindUnary, op: tc.op, target: &target}
		got, err := Eval(e, row)
		if tc.wantErr != "" {
			if err == nil || err.Error() != tc.wantErr {
				t.Errorf("Eval(%s on %s) error = %v, want %q", tc.op, tc.col, err, tc.wantErr)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("Eval(%s on %s) = %v, %v; want %v, nil", tc.op, tc.col, got, err, tc.want)
		}
	}
}

// TestEvalTernOperands pins the ternary dispatch: operand ops evaluate target,
// then left, then right and stop at the first error; pass-through ops return
// the target without evaluating their other operands; any other op is a when.
func TestEvalTernOperands(t *testing.T) {
	t.Parallel()

	row := mapRow{"s": "ab", "x": int64(5)}
	tern := func(op string, target, left, right Expr) Expr {
		return Expr{kind: KindTern, op: op, target: &target, left: &left, right: &right}
	}
	operandOps := []string{"str_pad_start", "str_pad_end", "str_split_part", "clip", "replace", "replace_strict", "is_between"}
	for _, op := range operandOps {
		for _, tc := range []struct{ target, left, right, wantErr string }{
			{"t_missing", "l_missing", "r_missing", "column t_missing not found"},
			{"x", "l_missing", "r_missing", "column l_missing not found"},
			{"x", "x", "r_missing", "column r_missing not found"},
		} {
			got, err := Eval(tern(op, Col(tc.target), Col(tc.left), Col(tc.right)), row)
			if got != nil || err == nil || err.Error() != tc.wantErr {
				t.Errorf("%s(%s, %s, %s) = %v, %v; want error %q", op, tc.target, tc.left, tc.right, got, err, tc.wantErr)
			}
		}
	}
	for op, wantErr := range map[string]string{"clip": "clip expects numeric", "is_between": "is_between expects numeric"} {
		if _, err := Eval(tern(op, Col("s"), Col("x"), Col("x")), row); err == nil || err.Error() != wantErr {
			t.Errorf("%s on a string: error = %v, want %q", op, err, wantErr)
		}
	}

	passthrough := []string{
		"bottom_k_by:2", "top_k_by:2", "sort_by:desc",
		"ewm_mean_by", "extend_constant", "max_by", "min_by", "interpolate_by",
		"rolling_max_by:2", "rolling_mean_by:2", "rolling_min_by:2", "rolling_sum_by:2", "rolling_std_by:2",
		"rolling_var_by:2", "rolling_median_by:2", "rolling_quantile_by:2", "rolling_rank_by:2",
	}
	for _, op := range passthrough {
		if got, err := Eval(tern(op, Col("x"), Col("missing"), Col("missing")), row); err != nil || got != int64(5) {
			t.Errorf("%s = %v, %v; want the target 5", op, got, err)
		}
	}
	// A prefix op without its argument, or an exact op with one, is a when.
	for _, op := range []string{"sort_by", "rolling_max_by", "min_by:x", "interpolate_by:x"} {
		if _, err := Eval(tern(op, Col("x"), Col("missing"), Col("x")), row); err == nil || err.Error() != "column missing not found" {
			t.Errorf("%s: error = %v, want the when condition's error", op, err)
		}
	}
	if _, err := Eval(tern("when", Col("x"), Col("x"), Col("x")), row); err == nil || err.Error() != "when expects bool condition" {
		t.Errorf("when on int condition: error = %v", err)
	}
	if _, err := Eval(tern("when", Col("x"), Lit(false), Col("x")), row); err == nil || err.Error() != "when expects otherwise branch" {
		t.Errorf("when without otherwise: error = %v", err)
	}
}
