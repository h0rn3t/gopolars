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
