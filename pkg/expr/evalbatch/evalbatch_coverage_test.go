package evalbatch

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/simd"
)

// extendedFixtures augments the base fixtures with datetime, NaN-bearing and a
// second boolean column to drive the remaining batch branches.
func extendedFixtures() (map[string]*chunk.Column, int) {
	cols, h := fixtures()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ts := make([]time.Time, h)
	ts2 := make([]time.Time, h)
	for i := 0; i < h; i++ {
		ts[i] = t0.AddDate(0, 0, i)
		ts2[i] = t0.AddDate(0, 0, h-1-i) // reversed so compares vary
	}
	cols["ts"] = chunk.NewTime(ts, nil)
	cols["ts2"] = chunk.NewTime(ts2, nil)
	cols["flag2"] = chunk.NewBool([]bool{false, false, true, true, false}, nil)
	cols["nanf"] = chunk.NewFloat64([]float64{1.0, nan(), 3.0, nan(), 5.0}, nil)
	return cols, h
}

func nan() float64 { var z float64; return z / z }

// TestBatchMatchesRowWiseExtended extends the conformance oracle to every
// comparison operator, cast target, string/time comparison, coalesce, and
// literal broadcast shape.
func TestBatchMatchesRowWiseExtended(t *testing.T) {
	cols, height := extendedFixtures()

	cases := []struct {
		name string
		e    expr.Expr
	}{
		// numericCompare: all ops, int/float/mixed.
		{"int_ge_lit", expr.Col("a").Ge(expr.Lit(int64(2)))},
		{"int_le_lit", expr.Col("a").Le(expr.Lit(int64(2)))},
		{"int_gt_intcol", expr.Col("c").Gt(expr.Col("a"))},
		{"float_gt_lit", expr.Col("b").Gt(expr.Lit(2.0))},      // SIMD gt-float fast path
		{"float_gt_floatcol", expr.Col("d").Gt(expr.Col("b"))}, // general cmpFloat gt
		{"float_lt_lit", expr.Col("b").Lt(expr.Lit(3.0))},
		{"float_le_floatcol", expr.Col("b").Le(expr.Col("d"))},
		{"nan_lt_lit", expr.Col("nanf").Lt(expr.Lit(2.0))}, // general loop NaN skip

		// float arithmetic.
		{"float_add_col", expr.Col("d").Add(expr.Col("b"))},
		{"float_sub_col", expr.Col("d").Sub(expr.Col("b"))},
		{"float_div_lit", expr.Col("d").Div(expr.Lit(2.0))},

		// cmpString: all ordering ops.
		{"string_ge", expr.Col("city").Ge(expr.Lit("kyiv"))},
		{"string_lt", expr.Col("city").Lt(expr.Lit("lviv"))},
		{"string_le", expr.Col("city").Le(expr.Lit("lviv"))},

		// eq/ne across types.
		{"int_eq_lit", expr.Col("a").Eq(expr.Lit(int64(2)))},
		{"int_ne_lit", expr.Col("a").Ne(expr.Lit(int64(2)))},
		{"string_ne", expr.Col("city").Ne(expr.Lit("kyiv"))},
		{"float_ne_col", expr.Col("b").Ne(expr.Col("d"))},
		{"bool_eq_lit", expr.Col("flag").Eq(expr.Lit(true))},
		{"bool_ne_lit", expr.Col("flag").Ne(expr.Lit(false))},

		// and/or fast path over two no-null boolean columns.
		{"and_two_cols", expr.Col("flag").And(expr.Col("flag2"))},
		{"or_two_cols", expr.Col("flag").Or(expr.Col("flag2"))},
		{"and_lit", expr.Col("flag").And(expr.Lit(true))}, // per-row batchBin and
		{"not_flag2", expr.Col("flag2").Not()},

		// NaN on the right operand of eq/ne (literal on the left).
		{"eq_right_nan", expr.Lit(1.0).Eq(expr.Col("nanf"))},
		{"ne_right_nan", expr.Lit(1.0).Ne(expr.Col("nanf"))},

		// mod/floordiv/pow over null-free numeric columns.
		{"float_mod", expr.Col("d").Mod(expr.Lit(3.0))},
		{"int_pow_lit", expr.Col("c").Pow(expr.Lit(int64(2)))},
		{"float_pow", expr.Col("d").Pow(expr.Lit(2.0))},

		// casts: every target + identity casts.
		{"cast_nullable_to_int", expr.Col("a").Cast(dtypes.Int64)},     // Int64 null branch
		{"cast_nullable_to_string", expr.Col("a").Cast(dtypes.String)}, // String null branch
		{"cast_int_identity", expr.Col("c").Cast(dtypes.Int64)},
		{"cast_float_identity", expr.Col("d").Cast(dtypes.Float64)},
		{"cast_int_to_string", expr.Col("c").Cast(dtypes.String)},
		{"cast_float_to_string", expr.Col("d").Cast(dtypes.String)},
		{"cast_bool_to_string", expr.Col("flag").Cast(dtypes.String)},
		{"cast_string_identity", expr.Col("city").Cast(dtypes.String)},
		{"cast_bool_identity", expr.Col("flag").Cast(dtypes.Boolean)},
		{"cast_time_to_string", expr.Col("ts").Cast(dtypes.String)},
		{"cast_time_identity", expr.Col("ts").Cast(dtypes.Datetime)},

		// datetime comparisons (compareAny time branch).
		{"time_gt", expr.Col("ts").Gt(expr.Col("ts2"))},
		{"time_ge", expr.Col("ts").Ge(expr.Col("ts2"))},
		{"time_lt", expr.Col("ts").Lt(expr.Col("ts2"))},
		{"time_le", expr.Col("ts").Le(expr.Col("ts2"))},
		{"time_eq", expr.Col("ts").Eq(expr.Col("ts2"))},
		{"time_ne", expr.Col("ts").Ne(expr.Col("ts2"))},

		// coalesce kernels.
		{"fill_null", expr.Col("b").FillNull(expr.Lit(0.0))},
		{"fill_nan", expr.Col("nanf").FillNaN(expr.Lit(0.0))},

		// NaN equality semantics.
		{"nan_eq", expr.Col("nanf").Eq(expr.Lit(1.0))},
		{"nan_ne", expr.Col("nanf").Ne(expr.Lit(1.0))},

		// literal broadcast of every type.
		{"lit_int", expr.Lit(int64(5))},
		{"lit_float", expr.Lit(3.14)},
		{"lit_string", expr.Lit("hi")},
		{"lit_bool", expr.Lit(true)},
		{"lit_time", expr.Lit(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))},
		{"lit_nil", expr.Lit(nil)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, ok := Compile(tc.e)
			if !ok {
				t.Fatalf("Compile(%s) not supported", tc.name)
			}
			got, err := plan.Eval(cols, height)
			if err != nil {
				t.Fatalf("batch Eval: %v", err)
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

// TestBatchEvalErrors drives the error-return contracts of the batch evaluator:
// arithmetic division by zero, mod/floordiv by zero, an invalid cast, a missing
// column, and a non-bool predicate through EvalBool.
func TestBatchEvalErrors(t *testing.T) {
	cols, height := extendedFixtures()

	errCases := []struct {
		name string
		e    expr.Expr
	}{
		{"int_div_zero", expr.Col("c").Div(expr.Lit(int64(0)))},
		{"mod_zero", expr.Col("c").Mod(expr.Lit(0.0))},
		{"floordiv_zero", expr.Col("c").FloorDiv(expr.Lit(0.0))},
		{"cast_string_to_int", expr.Col("city").Cast(dtypes.Int64)},
		{"missing_column", expr.Col("does_not_exist")},
	}
	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			plan, ok := Compile(tc.e)
			if !ok {
				t.Fatalf("Compile(%s) not supported", tc.name)
			}
			if _, err := plan.Eval(cols, height); err == nil {
				t.Errorf("%s: expected Eval error", tc.name)
			}
		})
	}

	// A non-bool expression evaluated as a predicate must error.
	plan, ok := Compile(expr.Col("a"))
	if !ok {
		t.Fatal("Compile(col a) not supported")
	}
	if _, _, err := plan.EvalBool(cols, height); err == nil {
		t.Error("EvalBool on int column: expected non-bool error")
	}
}

// TestCompileRejectsMore drives the remaining reject branches of supported():
// an unsupported cast target and an unsupported binary op.
func TestCompileRejectsMore(t *testing.T) {
	if _, ok := Compile(expr.Col("a").Cast(dtypes.Categorical)); ok {
		t.Error("Compile should reject cast to an unsupported dtype")
	}
	if _, ok := Compile(expr.Col("a").NeMissing(expr.Lit(int64(1)))); ok {
		t.Error("Compile should reject an unsupported binary op")
	}
}

// TestEvalBitmapShapes drives every direct-bitmap predicate shape and the
// fallback path of EvalBool, checking each bit against the row-wise evaluator.
func TestEvalBitmapShapes(t *testing.T) {
	cols, height := extendedFixtures()
	cols["flagN"] = chunk.NewBool([]bool{true, false, true, false, true}, []bool{false, true, false, false, false})

	cases := []struct {
		name string
		e    expr.Expr
	}{
		{"bare_bool", expr.Col("flag")},                                                 // KindCol bool, no nulls
		{"nullable_bool_fallback", expr.Col("flagN")},                                   // hasNulls → fallback
		{"float_gt_lit_fast", expr.Col("b").Gt(expr.Lit(2.0))},                          // cmpBitmap gt-float + clearNullBits
		{"int_eq_lit_fast", expr.Col("a").Eq(expr.Lit(int64(2)))},                       // cmpBitmap eq-int + clearNullBits
		{"int_gt_lit_general", expr.Col("c").Gt(expr.Lit(int64(25)))},                   // cmpBitmap general int
		{"float_ge_lit_general", expr.Col("b").Ge(expr.Lit(2.5))},                       // cmpBitmap general float
		{"eq_float_fallback", expr.Col("b").Eq(expr.Lit(2.5))},                          // eq-float → fallback pack
		{"and_predicates", expr.Col("c").Gt(expr.Lit(int64(15))).And(expr.Col("flag"))}, // evalBitmap AND
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, ok := Compile(tc.e)
			if !ok {
				t.Fatalf("Compile(%s) not supported", tc.name)
			}
			mask, _, err := plan.EvalBool(cols, height)
			if err != nil {
				t.Fatalf("EvalBool: %v", err)
			}
			for i := 0; i < height; i++ {
				want, werr := expr.Eval(tc.e, rowAcc{cols: cols, row: i})
				if werr != nil {
					// A nullable bool used as a predicate errors row-wise; the
					// bitmap folds it to 0. Skip the per-row equality there.
					continue
				}
				wb, _ := want.(bool)
				if simd.BitmapGet(mask, i) != (want != nil && wb) {
					t.Fatalf("%s row %d: bit=%v want %v", tc.name, i, simd.BitmapGet(mask, i), wb)
				}
			}
		})
	}
}
