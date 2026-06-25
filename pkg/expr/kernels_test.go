package expr

import (
	"math"
	"testing"
	"time"
)

// kernelRow backs Eval with a fixed set of named values.
type kernelRow map[string]any

func (r kernelRow) ValueByName(name string) (any, bool) {
	v, ok := r[name]
	return v, ok
}

func evalKernel(t *testing.T, e Expr, row kernelRow) any {
	t.Helper()
	v, err := Eval(e, row)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	return v
}

func TestKernelEndsWith(t *testing.T) {
	row := kernelRow{"s": "johnson", "null": nil}
	if v := evalKernel(t, Col("s").EndsWith(Lit("son")), row); v != true {
		t.Fatalf("ends_with: got %v, want true", v)
	}
	if v := evalKernel(t, Col("s").EndsWith(Lit("xyz")), row); v != false {
		t.Fatalf("ends_with: got %v, want false", v)
	}
	if v := evalKernel(t, Col("null").EndsWith(Lit("son")), row); v != nil {
		t.Fatalf("ends_with null: got %v, want nil", v)
	}
}

func TestKernelStrPos(t *testing.T) {
	row := kernelRow{"s": "héllo world", "null": nil}
	if v := evalKernel(t, Col("s").StrPos(Lit("world")), row); v != int64(7) {
		t.Fatalf("str_pos: got %v, want 7 (rune-based)", v)
	}
	if v := evalKernel(t, Col("s").StrPos(Lit("zzz")), row); v != nil {
		t.Fatalf("str_pos missing: got %v, want nil", v)
	}
	if v := evalKernel(t, Col("null").StrPos(Lit("a")), row); v != nil {
		t.Fatalf("str_pos null: got %v, want nil", v)
	}
}

func TestKernelLeftRight(t *testing.T) {
	row := kernelRow{"s": "abcdef", "null": nil}
	if v := evalKernel(t, Col("s").StrLeft(Lit(int64(3))), row); v != "abc" {
		t.Fatalf("left: got %v, want abc", v)
	}
	if v := evalKernel(t, Col("s").StrLeft(Lit(int64(-2))), row); v != "abcd" {
		t.Fatalf("left negative: got %v, want abcd", v)
	}
	if v := evalKernel(t, Col("s").StrRight(Lit(int64(2))), row); v != "ef" {
		t.Fatalf("right: got %v, want ef", v)
	}
	if v := evalKernel(t, Col("s").StrRight(Lit(int64(-2))), row); v != "cdef" {
		t.Fatalf("right negative: got %v, want cdef", v)
	}
	if v := evalKernel(t, Col("s").StrLeft(Lit(int64(99))), row); v != "abcdef" {
		t.Fatalf("left overflow: got %v, want abcdef", v)
	}
	if v := evalKernel(t, Col("null").StrLeft(Lit(int64(1))), row); v != nil {
		t.Fatalf("left null: got %v, want nil", v)
	}
}

func TestKernelRegexpLikeAndAtan2(t *testing.T) {
	row := kernelRow{"s": "abc123", "y": 1.0, "x": 1.0, "null": nil}
	if v := evalKernel(t, Col("s").RegexpLike(Lit(`^[a-z]+\d+$`)), row); v != true {
		t.Fatalf("regexp_like: got %v, want true", v)
	}
	if v := evalKernel(t, Col("null").RegexpLike(Lit("a")), row); v != nil {
		t.Fatalf("regexp_like null: got %v, want nil", v)
	}
	v := evalKernel(t, Col("y").Atan2(Col("x")), row)
	if f, ok := v.(float64); !ok || math.Abs(f-math.Pi/4) > 1e-12 {
		t.Fatalf("atan2: got %v, want pi/4", v)
	}
	if v := evalKernel(t, Col("null").Atan2(Col("x")), row); v != nil {
		t.Fatalf("atan2 null: got %v, want nil", v)
	}
}

func TestKernelConcatWS(t *testing.T) {
	row := kernelRow{"a": "x", "b": "y", "null": nil}
	if v := evalKernel(t, Col("a").StrConcatWS("-", Col("b")), row); v != "x-y" {
		t.Fatalf("concat_ws: got %v, want x-y", v)
	}
	if v := evalKernel(t, Col("a").StrConcatWS("-", Col("null")), row); v != "x" {
		t.Fatalf("concat_ws skip null: got %v, want x", v)
	}
	if v := evalKernel(t, Col("null").StrConcatWS("-", Col("b")), row); v != "y" {
		t.Fatalf("concat_ws skip left null: got %v, want y", v)
	}
	if v := evalKernel(t, Col("null").StrConcatWS("-", Col("null")), row); v != nil {
		t.Fatalf("concat_ws both null: got %v, want nil", v)
	}
}

func TestKernelTrimReverseTitleCharLen(t *testing.T) {
	row := kernelRow{"s": "  hi  ", "w": "héllo wörld", "null": nil}
	if v := evalKernel(t, Col("s").StrLTrim(), row); v != "hi  " {
		t.Fatalf("ltrim: got %q", v)
	}
	if v := evalKernel(t, Col("s").StrRTrim(), row); v != "  hi" {
		t.Fatalf("rtrim: got %q", v)
	}
	if v := evalKernel(t, Col("w").StrReverse(), row); v != "dlröw olléh" {
		t.Fatalf("reverse: got %q", v)
	}
	if v := evalKernel(t, Col("w").StrToTitle(), row); v != "Héllo Wörld" {
		t.Fatalf("initcap: got %q", v)
	}
	if v := evalKernel(t, Col("w").StrCharLen(), row); v != int64(11) {
		t.Fatalf("char_len: got %v, want 11", v)
	}
	for _, e := range []Expr{Col("null").StrLTrim(), Col("null").StrRTrim(), Col("null").StrReverse(), Col("null").StrToTitle(), Col("null").StrCharLen()} {
		if v := evalKernel(t, e, row); v != nil {
			t.Fatalf("null propagation: got %v, want nil", v)
		}
	}
}

func TestKernelPad(t *testing.T) {
	row := kernelRow{"s": "hi", "long": "abcdef", "null": nil}
	if v := evalKernel(t, Col("s").StrPadStart(Lit(int64(5)), Lit("xy")), row); v != "xyxhi" {
		t.Fatalf("lpad: got %q, want xyxhi", v)
	}
	if v := evalKernel(t, Col("s").StrPadEnd(Lit(int64(5)), Lit("xy")), row); v != "hixyx" {
		t.Fatalf("rpad: got %q, want hixyx", v)
	}
	if v := evalKernel(t, Col("long").StrPadStart(Lit(int64(3)), Lit("x")), row); v != "abc" {
		t.Fatalf("lpad truncation: got %q, want abc", v)
	}
	if v := evalKernel(t, Col("null").StrPadStart(Lit(int64(3)), Lit("x")), row); v != nil {
		t.Fatalf("lpad null: got %v, want nil", v)
	}
}

func TestKernelSplitPart(t *testing.T) {
	row := kernelRow{"s": "a@b@c", "null": nil}
	if v := evalKernel(t, Col("s").StrSplitPart(Lit("@"), Lit(int64(2))), row); v != "b" {
		t.Fatalf("split_part: got %q, want b", v)
	}
	if v := evalKernel(t, Col("s").StrSplitPart(Lit("@"), Lit(int64(9))), row); v != "" {
		t.Fatalf("split_part out of range: got %q, want empty", v)
	}
	if v := evalKernel(t, Col("null").StrSplitPart(Lit("@"), Lit(int64(1))), row); v != nil {
		t.Fatalf("split_part null: got %v, want nil", v)
	}
	if _, err := Eval(Col("s").StrSplitPart(Lit("@"), Lit(int64(0))), row); err == nil {
		t.Fatalf("split_part position 0: want error")
	}
}

func TestKernelOrdinalDay(t *testing.T) {
	row := kernelRow{"d": time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), "null": nil}
	if v := evalKernel(t, Col("d").DtOrdinalDay(), row); v != int64(32) {
		t.Fatalf("ordinal_day: got %v, want 32", v)
	}
	if v := evalKernel(t, Col("null").DtOrdinalDay(), row); v != nil {
		t.Fatalf("ordinal_day null: got %v, want nil", v)
	}
}
