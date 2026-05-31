package evalbatch

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
)

// rowAcc is a minimal RowValueGetter over typed chunks for row-wise expr.Eval,
// used as the conformance oracle.
type rowAcc struct {
	cols map[string]*chunk.Column
	row  int
}

func (r rowAcc) ValueByName(name string) (any, bool) {
	c, ok := r.cols[name]
	if !ok {
		return nil, false
	}
	return c.ValueAt(r.row), true
}

func fixtures() (map[string]*chunk.Column, int) {
	a := chunk.NewInt64([]int64{1, 2, 3, 4, 0}, []bool{false, false, false, false, true})
	b := chunk.NewFloat64([]float64{1.5, 2.5, 3.5, 0, 5.5}, []bool{false, false, false, true, false})
	c := chunk.NewInt64([]int64{10, 20, 30, 40, 50}, nil)
	d := chunk.NewFloat64([]float64{2, 7, 4, 5, 8}, nil)
	flag := chunk.NewBool([]bool{true, false, true, false, true}, nil)
	city := chunk.NewString([]string{"kyiv", "lviv", "kyiv", "odesa", "lviv"}, nil)
	return map[string]*chunk.Column{"a": a, "b": b, "c": c, "d": d, "flag": flag, "city": city}, 5
}

func TestBatchMatchesRowWise(t *testing.T) {
	t.Parallel()

	cols, height := fixtures()

	cases := []struct {
		name string
		e    expr.Expr
	}{
		{"int_gt_lit", expr.Col("a").Gt(expr.Lit(int64(2)))},
		{"float_lt_lit", expr.Col("b").Lt(expr.Lit(5.0))},
		{"int_lt_intcol", expr.Col("a").Lt(expr.Col("c"))},
		{"float_eq_floatcol", expr.Col("b").Eq(expr.Col("d"))},
		{"int_add_intcol", expr.Col("a").Add(expr.Col("c"))},
		{"float_mul_lit", expr.Col("b").Mul(expr.Lit(2.0))},
		{"int_div_lit", expr.Col("c").Div(expr.Lit(int64(2)))},
		{"int_mod_lit", expr.Col("c").Mod(expr.Lit(int64(7)))},
		{"int_floordiv_lit", expr.Col("c").FloorDiv(expr.Lit(int64(3)))},
		{"int_sub_lit", expr.Col("a").Sub(expr.Lit(int64(1)))},
		{"float_compare_col", expr.Col("b").Ge(expr.Col("d"))},
		{"and", expr.Col("a").Gt(expr.Lit(int64(1))).And(expr.Col("flag"))},
		{"or", expr.Col("a").Lt(expr.Lit(int64(2))).Or(expr.Col("flag"))},
		{"not", expr.Col("flag").Not()},
		{"string_eq", expr.Col("city").Eq(expr.Lit("kyiv"))},
		{"string_gt", expr.Col("city").Gt(expr.Lit("kyiv"))},
		{"cast_int_to_float", expr.Col("a").Cast(dtypes.Float64)},
		{"cast_float_to_int", expr.Col("b").Cast(dtypes.Int64)},
		{"nested", expr.Col("a").Add(expr.Col("c")).Gt(expr.Lit(int64(20)))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, ok := Compile(tc.e)
			if !ok {
				t.Fatalf("Compile(%s) returned not-supported, want supported", tc.name)
			}
			got, err := plan.Eval(cols, height)
			if err != nil {
				t.Fatalf("batch Eval: %v", err)
			}
			if got.Len() != height {
				t.Fatalf("batch len = %d, want %d", got.Len(), height)
			}
			for i := 0; i < height; i++ {
				want, werr := expr.Eval(tc.e, rowAcc{cols: cols, row: i})
				if werr != nil {
					t.Fatalf("row-wise eval row %d: %v", i, werr)
				}
				gotV := got.ValueAt(i)
				if want == nil {
					if gotV != nil {
						t.Fatalf("row %d: batch=%v want null", i, gotV)
					}
					continue
				}
				if !valuesEqual(gotV, want) {
					t.Fatalf("row %d: batch=%v (%T) row-wise=%v (%T)", i, gotV, gotV, want, want)
				}
			}
		})
	}
}

func valuesEqual(a, b any) bool {
	switch av := a.(type) {
	case float64:
		bv, ok := b.(float64)
		return ok && (av == bv || (av != av && bv != bv))
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case time.Time:
		bv, ok := b.(time.Time)
		return ok && av.Equal(bv)
	default:
		return a == b
	}
}

func TestCompileRejectsUnsupported(t *testing.T) {
	t.Parallel()

	// Aggregate is not a per-row batch op -> must be rejected for fallback.
	agg := expr.Col("a").Sum()
	if _, ok := Compile(agg); ok {
		t.Fatal("Compile should reject aggregate/unary-sum expression")
	}
	// Reverse row access is explicitly a row-wise-only op.
	if _, ok := Compile(expr.Col("a").Reverse()); ok {
		t.Fatal("Compile should reject reverse expression")
	}
}

func TestEvalBoolMaskAndValidity(t *testing.T) {
	t.Parallel()

	cols, height := fixtures()
	plan, ok := Compile(expr.Col("a").Gt(expr.Lit(int64(2))))
	if !ok {
		t.Fatal("compile failed")
	}
	mask, valid, err := plan.EvalBool(cols, height)
	if err != nil {
		t.Fatalf("EvalBool: %v", err)
	}
	// a = {1,2,3,4,null} > 2 -> compare with null operand yields false (not null)
	want := []bool{false, false, true, true, false}
	for i := 0; i < height; i++ {
		if mask[i] != want[i] || !valid[i] {
			t.Fatalf("row %d: mask=%v valid=%v want mask=%v valid=true", i, mask[i], valid[i], want[i])
		}
	}
}
