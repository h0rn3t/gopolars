package frame

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

func mkFillFrame() DataFrame {
	n := series.FromFloat64("n", []float64{1, 0, math.NaN(), 0, 5}, []bool{false, true, false, true, false})
	df, _ := New(NewInput{Series: []series.Series{n}})
	return df
}

// TestFillNullExprTypedVsRowWise checks the evalbatch fill_null_expr path
// matches the row-wise fallback.
func TestFillNullExprTypedVsRowWise(t *testing.T) {
	typed, err := mkFillFrame().Select(expr.Col("n").FillNull(expr.Lit(float64(99))))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
	rowwise, err := mkFillFrame().Select(expr.Col("n").FillNull(expr.Lit(float64(99))))
	if err != nil {
		t.Fatal(err)
	}
	tc := typed.cols[typed.order[0]]
	rc := rowwise.cols[rowwise.order[0]]
	for i := 0; i < typed.height; i++ {
		tv := tc.Value(i)
		rv := rc.Value(i)
		if !valueEquals(tv, rv) {
			t.Errorf("fill_null_expr[%d]: typed %v rowwise %v", i, tv, rv)
		}
	}
}

func TestFillNanExprTypedVsRowWise(t *testing.T) {
	typed, err := mkFillFrame().Select(expr.Col("n").FillNaN(expr.Lit(float64(7))))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
	rowwise, err := mkFillFrame().Select(expr.Col("n").FillNaN(expr.Lit(float64(7))))
	if err != nil {
		t.Fatal(err)
	}
	tc := typed.cols[typed.order[0]]
	rc := rowwise.cols[rowwise.order[0]]
	for i := 0; i < typed.height; i++ {
		if !valueEquals(tc.Value(i), rc.Value(i)) {
			t.Errorf("fill_nan_expr[%d]: typed %v rowwise %v", i, tc.Value(i), rc.Value(i))
		}
	}
}
