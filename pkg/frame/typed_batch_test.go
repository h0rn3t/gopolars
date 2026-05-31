package frame

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

func buildBatchTestDF(t *testing.T) DataFrame {
	t.Helper()
	a, err := series.New("a", dtypes.Int64, []any{int64(1), int64(2), int64(3), int64(4), int64(5)})
	if err != nil {
		t.Fatalf("series a: %v", err)
	}
	b, err := series.New("b", dtypes.Float64, []any{1.5, 2.5, 3.5, 4.5, 5.5})
	if err != nil {
		t.Fatalf("series b: %v", err)
	}
	city, err := series.New("city", dtypes.String, []any{"kyiv", "lviv", "kyiv", "odesa", "lviv"})
	if err != nil {
		t.Fatalf("series city: %v", err)
	}
	df, err := New(NewInput{Series: []series.Series{a, b, city}})
	if err != nil {
		t.Fatalf("new df: %v", err)
	}
	return df
}

func assertFramesEqual(t *testing.T, got, want DataFrame) {
	t.Helper()
	eq, err := got.Equals(want)
	if err != nil {
		t.Fatalf("Equals: %v", err)
	}
	if !eq {
		t.Fatalf("frames differ:\n typed-on=%v\n typed-off=%v", got.ToDicts(), want.ToDicts())
	}
}

func TestFilterBatchParityWithRowWise(t *testing.T) {
	pred := expr.Col("a").Gt(expr.Lit(int64(2)))

	on, err := buildBatchTestDF(t).Filter(pred) // typed storage default-on
	if err != nil {
		t.Fatalf("filter (typed on): %v", err)
	}

	t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
	off, err := buildBatchTestDF(t).Filter(pred)
	if err != nil {
		t.Fatalf("filter (typed off): %v", err)
	}

	assertFramesEqual(t, on, off)
	if on.Height() != 3 {
		t.Fatalf("expected 3 surviving rows, got %d", on.Height())
	}
}

func TestWithColumnsBatchParityWithRowWise(t *testing.T) {
	exprs := []expr.Expr{
		expr.Col("a").Gt(expr.Lit(int64(3))).Alias("big"),
		expr.Col("city").Eq(expr.Lit("kyiv")).Alias("is_kyiv"),
		expr.Col("a").Gt(expr.Lit(int64(1))).And(expr.Col("city").Eq(expr.Lit("kyiv"))).Alias("both"),
	}

	on, err := buildBatchTestDF(t).WithColumns(exprs...) // typed on
	if err != nil {
		t.Fatalf("with_columns (typed on): %v", err)
	}

	t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
	off, err := buildBatchTestDF(t).WithColumns(exprs...)
	if err != nil {
		t.Fatalf("with_columns (typed off): %v", err)
	}

	assertFramesEqual(t, on, off)

	// sanity: the derived boolean columns exist with the right dtype
	s, ok := on.Series("big")
	if !ok || s.DataType() != dtypes.Boolean {
		t.Fatalf("expected boolean 'big' column, got ok=%v dtype=%v", ok, s.DataType())
	}
}
