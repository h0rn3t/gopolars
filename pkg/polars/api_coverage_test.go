package polars

import (
	"context"
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// statFrame builds a numeric frame for the aggregate reducers.
//
//	x: 1 2 3 4
//	y: 10 20 30 40
func statFrame(t *testing.T) DataFrame {
	t.Helper()
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "x", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		{Name: "y", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
	}})
	if err != nil {
		t.Fatalf("statFrame: %v", err)
	}
	return d
}

// TestDataFrameStatReducers covers the map-returning statistical reducers with
// hand-checked values.
func TestDataFrameStatReducers(t *testing.T) {
	t.Parallel()

	d := statFrame(t)

	if got := d.Sum()["x"]; got != 10 {
		t.Errorf("Sum[x] = %v, want 10", got)
	}
	if got := d.Mean()["y"]; got != 25 {
		t.Errorf("Mean[y] = %v, want 25", got)
	}
	if got := d.Median()["x"]; got != 2.5 {
		t.Errorf("Median[x] = %v, want 2.5", got)
	}
	if got := d.Product()["x"]; got != 24 {
		t.Errorf("Product[x] = %v, want 24", got)
	}
	if got := d.Quantile(0.5)["x"]; got != 3 {
		// q=0.5 over [1,2,3,4]: idx = round(0.5*3) = 2 -> vals[2] = 3
		t.Errorf("Quantile(.5)[x] = %v, want 3", got)
	}
	// var of [1,2,3,4] sample = 1.6667; std = sqrt(1.6667).
	if got := d.Var()["x"]; math.Abs(got-(5.0/3.0)) > 1e-9 {
		t.Errorf("Var[x] = %v, want %v", got, 5.0/3.0)
	}
	if got := d.Std()["x"]; math.Abs(got-math.Sqrt(5.0/3.0)) > 1e-9 {
		t.Errorf("Std[x] = %v, want %v", got, math.Sqrt(5.0/3.0))
	}
	if got := d.Max()["y"]; got != int64(40) {
		t.Errorf("Max[y] = %v, want 40", got)
	}
	if got := d.Min()["x"]; got != int64(1) {
		t.Errorf("Min[x] = %v, want 1", got)
	}
}

// TestDataFrameCorr covers the pairwise correlation helper (perfectly correlated
// columns -> 1.0).
func TestDataFrameCorr(t *testing.T) {
	t.Parallel()

	d := statFrame(t) // y = 10*x, perfectly correlated
	corr, err := d.Corr("x", "y")
	if err != nil {
		t.Fatalf("corr: %v", err)
	}
	if math.Abs(corr-1.0) > 1e-9 {
		t.Fatalf("corr(x,y) = %v, want 1.0", corr)
	}
}

// TestDataFrameHorizontalAgg covers the row-wise horizontal aggregations.
func TestDataFrameHorizontalAgg(t *testing.T) {
	t.Parallel()

	d := statFrame(t)

	check := func(name string, build func(string) (DataFrame, error), wantFirst float64) {
		out, err := build("h")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		s, ok := out.Series("h")
		if !ok {
			t.Fatalf("%s: result column missing", name)
		}
		got, _ := toFloatLocal(s.Value(0))
		if got != wantFirst {
			t.Fatalf("%s row0 = %v, want %v", name, got, wantFirst)
		}
	}

	// row 0 is x=1, y=10.
	check("max", d.MaxHorizontal, 10)
	check("min", d.MinHorizontal, 1)
	check("mean", d.MeanHorizontal, 5.5)
	check("sum", d.SumHorizontal, 11)
}

// TestDataFrameTopBottomK covers TopK/BottomK ordering.
func TestDataFrameTopBottomK(t *testing.T) {
	t.Parallel()

	d := statFrame(t)

	top, err := d.TopK(2, "x")
	if err != nil {
		t.Fatalf("top_k: %v", err)
	}
	if top.Height() != 2 {
		t.Fatalf("top_k height = %d, want 2", top.Height())
	}
	sx, _ := top.Series("x")
	if sx.Value(0) != int64(4) {
		t.Fatalf("top_k first x = %v, want 4 (largest)", sx.Value(0))
	}

	bottom, err := d.BottomK(2, "x")
	if err != nil {
		t.Fatalf("bottom_k: %v", err)
	}
	bx, _ := bottom.Series("x")
	if bx.Value(0) != int64(1) {
		t.Fatalf("bottom_k first x = %v, want 1 (smallest)", bx.Value(0))
	}
}

// TestDataFrameTranspose covers Transpose.
func TestDataFrameTranspose(t *testing.T) {
	t.Parallel()

	d := statFrame(t) // 4 rows x 2 cols
	tr, err := d.Transpose()
	if err != nil {
		t.Fatalf("transpose: %v", err)
	}
	// transposed: 2 rows (x,y), 4 value columns.
	if tr.Height() != 2 {
		t.Fatalf("transpose height = %d, want 2", tr.Height())
	}
}

// TestExprWrappers covers the package-level selector/fold wrappers that delegate
// to pkg/expr.
func TestExprWrappers(t *testing.T) {
	t.Parallel()

	if All().Op() != "" && All().ColName() != "*" {
		t.Fatalf("All() wrapper wrong: %+v", All())
	}
	if got := Cols("a", "b").Names(); len(got) != 2 {
		t.Fatalf("Cols() names = %v", got)
	}
	if Exclude("a").Op() == "" {
		t.Fatalf("Exclude() should carry an exclude op")
	}
	if StructCols("a", "b").Op() != "struct_pack" {
		t.Fatalf("StructCols() op = %q", StructCols("a", "b").Op())
	}
	fold := Fold(int64(0), func(acc any, next any) (any, error) { return acc, nil }, Col("a"))
	if fold.Op() != "fold" {
		t.Fatalf("Fold() op = %q, want fold", fold.Op())
	}
}

// TestLazyFrameSliceMethods covers the lazy Head/Tail/First/Last/Slice/Reverse
// methods through to Collect.
func TestLazyFrameSliceMethods(t *testing.T) {
	t.Parallel()

	d := statFrame(t)
	ctx := context.Background()

	head, err := d.Lazy().Head(2).Collect(ctx)
	if err != nil || head.Height() != 2 {
		t.Fatalf("head: height=%d err=%v", head.Height(), err)
	}
	tail, err := d.Lazy().Tail(1).Collect(ctx)
	if err != nil || tail.Height() != 1 {
		t.Fatalf("tail: height=%d err=%v", tail.Height(), err)
	}
	first, err := d.Lazy().First().Collect(ctx)
	if err != nil || first.Height() != 1 {
		t.Fatalf("first: height=%d err=%v", first.Height(), err)
	}
	last, err := d.Lazy().Last().Collect(ctx)
	if err != nil || last.Height() != 1 {
		t.Fatalf("last: height=%d err=%v", last.Height(), err)
	}
	sx, _ := first.Series("x")
	if sx.Value(0) != int64(1) {
		t.Fatalf("first x = %v, want 1", sx.Value(0))
	}

	rev, err := d.Lazy().Reverse().Collect(ctx)
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	rx, _ := rev.Series("x")
	if rx.Value(0) != int64(4) {
		t.Fatalf("reverse first x = %v, want 4", rx.Value(0))
	}

	slice, err := d.Lazy().Slice(1, 2).Collect(ctx)
	if err != nil || slice.Height() != 2 {
		t.Fatalf("slice: height=%d err=%v", slice.Height(), err)
	}
}

// TestLazyFrameTopBottomK covers the lazy TopK/BottomK/ApproxNUnique methods.
func TestLazyFrameTopBottomK(t *testing.T) {
	t.Parallel()

	d := statFrame(t)
	ctx := context.Background()

	top, err := d.Lazy().TopK(2, "x").Collect(ctx)
	if err != nil || top.Height() != 2 {
		t.Fatalf("lazy top_k: height=%d err=%v", top.Height(), err)
	}
	bottom, err := d.Lazy().BottomK(2, "x").Collect(ctx)
	if err != nil || bottom.Height() != 2 {
		t.Fatalf("lazy bottom_k: height=%d err=%v", bottom.Height(), err)
	}
	// ApproxNUnique delegates to Unique (row dedup); x is all-distinct -> 4 rows.
	nun, err := d.Lazy().ApproxNUnique("x").Collect(ctx)
	if err != nil {
		t.Fatalf("lazy approx_n_unique: %v", err)
	}
	if nun.Height() != 4 {
		t.Fatalf("approx_n_unique height = %d, want 4", nun.Height())
	}
	// With no columns it is an identity transform.
	ident, err := d.Lazy().ApproxNUnique().Collect(ctx)
	if err != nil || ident.Height() != 4 {
		t.Fatalf("approx_n_unique() identity: height=%d err=%v", ident.Height(), err)
	}
}
