package evalbatch

import (
	"math"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/simd"
)

func TestIsInBatchMatchesRowWise(t *testing.T) {
	t.Parallel()
	noon := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	noonPlusTwo := noon.In(time.FixedZone("UTC+2", 2*3600))
	nan, negZero := math.NaN(), math.Copysign(0, -1)
	category, err := chunk.FromAny(dtypes.Categorical, []any{"kyiv", nil, "lviv", "odesa", "kyiv", "lviv"})
	if err != nil {
		t.Fatalf("FromAny(categorical): %v", err)
	}
	nulls := []bool{false, true, false, false, false, false}
	cols := map[string]*chunk.Column{
		"i":   chunk.NewInt64([]int64{1, 0, 3, 7, 1, -2}, nulls),
		"f":   chunk.NewFloat64([]float64{2.5, 0, nan, negZero, 1, 2.5}, nulls),
		"s":   chunk.NewString([]string{"kyiv", "", "lviv", "odesa", "kyiv", "lviv"}, nulls),
		"cat": category,
		"b":   chunk.NewBool([]bool{true, false, false, true, true, false}, nulls),
		"t":   chunk.NewTime([]time.Time{noon, {}, noon.Add(time.Hour), noonPlusTwo, noon.Add(-time.Hour), noon}, nulls),
	}
	const height = 6
	tests := []struct {
		name string
		e    expr.Expr
	}{
		{"int64 typed list", expr.Col("i").IsIn(expr.Lit([]int64{1, 7}))},
		{"int64 with other types and nil", expr.Col("i").IsIn(expr.Lit([]any{int64(3), 1.0, "1", nil}))},
		{"float64 with NaN and zero", expr.Col("f").IsIn(expr.Lit([]float64{nan, 0, 2.5}))},
		{"string", expr.Col("s").IsIn(expr.Lit([]string{"lviv", "x"}))},
		{"string with nil", expr.Col("s").IsIn(expr.Lit([]any{"kyiv", nil}))},
		{"categorical", expr.Col("cat").IsIn(expr.Lit([]string{"kyiv", "odesa"}))},
		{"bool", expr.Col("b").IsIn(expr.Lit([]bool{false}))},
		{"datetime in other zone", expr.Col("t").IsIn(expr.Lit([]time.Time{noonPlusTwo, noon.Add(-time.Hour)}))},
		{"datetime with nil", expr.Col("t").IsIn(expr.Lit([]any{noon, nil}))},
		{"empty list", expr.Col("i").IsIn(expr.Lit([]any{}))},
	}
	for _, tc := range tests {
		plan, ok := Compile(tc.e)
		if !ok {
			t.Errorf("Compile(%s) = not supported, want supported", tc.name)
			continue
		}
		got, err := plan.Eval(cols, height)
		if err != nil {
			t.Fatalf("Eval(%s): %v", tc.name, err)
		}
		mask, err := plan.EvalBool(cols, height)
		if err != nil {
			t.Fatalf("EvalBool(%s): %v", tc.name, err)
		}
		for i := range height {
			want, err := expr.Eval(tc.e, rowAcc{cols: cols, row: i})
			if err != nil {
				t.Fatalf("row-wise Eval(%s) row %d: %v", tc.name, i, err)
			}
			if v := got.ValueAt(i); v != want {
				t.Errorf("Eval(%s) row %d = %v, want %v (row-wise)", tc.name, i, v, want)
			}
			if bit := simd.BitmapGet(mask, i); bit != (want == true) {
				t.Errorf("EvalBool(%s) row %d = %v, want %v (row-wise)", tc.name, i, bit, want)
			}
		}
	}
	if _, ok := Compile(expr.Col("i").IsIn(expr.Col("i"))); ok {
		t.Error("Compile(is_in against a column) = supported, want the row-wise fallback")
	}
}
