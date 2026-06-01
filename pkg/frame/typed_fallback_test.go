package frame

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// TestFillNullStringFallback verifies that filling a string column with a
// non-float value uses the boxed fallback correctly (the typed float path
// returns ok=false for non-float64 columns).
func TestFillNullStringFallback(t *testing.T) {
	g := series.FromString("g", []string{"a", "", "c"}, []bool{false, true, false})
	df, _ := New(NewInput{Series: []series.Series{g}})
	out, err := df.FillNull("Z")
	if err != nil {
		t.Fatalf("fill_null string: %v", err)
	}
	col := out.cols["g"]
	want := []string{"a", "Z", "c"}
	for i, w := range want {
		if col.Value(i).(string) != w {
			t.Errorf("fill_null[%d] = %v, want %v", i, col.Value(i), w)
		}
	}
}

// TestTypedStorageDisabledStillWorks confirms the optimized ops run correctly
// with GOPOLARS_TYPED_STORAGE=0 (the batch-eval expression path is forced
// row-wise; the unconditional typed kernels still produce correct results).
func TestTypedStorageDisabledStillWorks(t *testing.T) {
	t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
	g := series.FromString("g", []string{"a", "b", "a"}, nil)
	v := series.FromFloat64("v", []float64{1, 2, 3}, nil)
	df, _ := New(NewInput{Series: []series.Series{g, v}})

	sel, err := df.Select(expr.Col("v").Gt(expr.Lit(float64(1))).Alias("p"))
	if err != nil {
		t.Fatalf("select (row-wise): %v", err)
	}
	if sel.height != 3 {
		t.Errorf("select rows = %d, want 3", sel.height)
	}

	gb, err := df.GroupBy("g").Agg(expr.Sum(expr.Col("v")))
	if err != nil {
		t.Fatalf("group_by: %v", err)
	}
	if gb.height != 2 {
		t.Errorf("group_by groups = %d, want 2", gb.height)
	}

	uq, err := df.Unique("g")
	if err != nil {
		t.Fatalf("unique: %v", err)
	}
	if uq.height != 2 {
		t.Errorf("unique rows = %d, want 2", uq.height)
	}
}
