package evalbatch

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/simd"
)

func TestAsColCmpLitPositive(t *testing.T) {
	cases := []struct {
		e    expr.Expr
		want simd.Cmp
	}{
		{expr.Col("a").Gt(expr.Lit(5.0)), simd.CmpGT},
		{expr.Col("a").Ge(expr.Lit(5.0)), simd.CmpGE},
		{expr.Col("a").Lt(expr.Lit(5.0)), simd.CmpLT},
		{expr.Col("a").Le(expr.Lit(5.0)), simd.CmpLE},
	}
	for _, tc := range cases {
		p, ok := Compile(tc.e)
		if !ok {
			t.Fatalf("Compile failed for %v", tc.e)
		}
		col, cmp, litF, ok := p.AsColCmpLit()
		if !ok || col != "a" || cmp != tc.want || litF != 5.0 {
			t.Fatalf("AsColCmpLit = (%q,%d,%v,%v), want (a,%d,5,true)", col, cmp, litF, ok, tc.want)
		}
	}
}

func TestAsColCmpLitIntLiteral(t *testing.T) {
	p, ok := Compile(expr.Col("a").Gt(expr.Lit(int64(3))))
	if !ok {
		t.Fatal("Compile failed")
	}
	col, cmp, litF, ok := p.AsColCmpLit()
	if !ok || col != "a" || cmp != simd.CmpGT || litF != 3.0 {
		t.Fatalf("AsColCmpLit = (%q,%d,%v,%v), want (a,GT,3,true)", col, cmp, litF, ok)
	}
}

func TestAsColCmpLitNegative(t *testing.T) {
	cases := map[string]expr.Expr{
		"col vs col":  expr.Col("a").Gt(expr.Col("b")),
		"lit on left": expr.Lit(5.0).Gt(expr.Col("a")),
		"bare col":    expr.Col("a"),
	}
	for name, e := range cases {
		p, ok := Compile(e)
		if !ok {
			continue // unsupported by Compile is also "not fast-path"
		}
		if _, _, _, ok := p.AsColCmpLit(); ok {
			t.Fatalf("%s: AsColCmpLit reported ok, want false", name)
		}
	}
}
