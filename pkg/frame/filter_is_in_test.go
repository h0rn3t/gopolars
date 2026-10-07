package frame

import (
	"fmt"
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

// TestIsInBatchMatchesRowWiseFrame compares Filter and WithColumns with an
// is_in predicate on the typed batch path against the row-wise path
// (GOPOLARS_TYPED_STORAGE=0), above the parallel filter threshold.
func TestIsInBatchMatchesRowWiseFrame(t *testing.T) {
	df := aggBenchFrame(t, 100_000)
	names := make([]any, 0, 51)
	for i := range 50 {
		names = append(names, fmt.Sprintf("zone-%d", i*5))
	}
	names = append(names, nil)
	preds := map[string]expr.Expr{
		"int64":         expr.Col("id").IsIn(expr.Lit([]int64{0, 7, 499, 1000})),
		"string":        expr.Col("name").IsIn(expr.Lit(names)),
		"nullable int":  expr.Col("iw").IsIn(expr.Lit([]any{int64(1), nil})),
		"is_in and cmp": expr.Col("id").IsIn(expr.Lit([]int64{1, 2, 3})).And(expr.Col("v").Gt(expr.Lit(1000.0))),
	}
	for name, pred := range preds {
		t.Setenv("GOPOLARS_TYPED_STORAGE", "1")
		batch, err := df.Filter(pred)
		if err != nil {
			t.Fatalf("Filter(%s) batch: %v", name, err)
		}
		batchCol, err := df.WithColumns(pred.Alias("hit"))
		if err != nil {
			t.Fatalf("WithColumns(%s) batch: %v", name, err)
		}
		t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
		rowWise, err := df.Filter(pred)
		if err != nil {
			t.Fatalf("Filter(%s) row-wise: %v", name, err)
		}
		rowCol, err := df.WithColumns(pred.Alias("hit"))
		if err != nil {
			t.Fatalf("WithColumns(%s) row-wise: %v", name, err)
		}
		if got, want := columnInt64s(t, batch, "seq"), columnInt64s(t, rowWise, "seq"); !slices.Equal(got, want) {
			t.Errorf("Filter(%s) kept %d rows, want %d (row-wise)", name, len(got), len(want))
		}
		got, _ := batchCol.Series("hit")
		want, _ := rowCol.Series("hit")
		for i := range df.Height() {
			if got.Value(i) != want.Value(i) {
				t.Errorf("WithColumns(%s) row %d = %v, want %v (row-wise)", name, i, got.Value(i), want.Value(i))
				break
			}
		}
	}
}
