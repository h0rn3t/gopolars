package lazyframe

// Ported from py-polars/tests/unit/lazyframe/test_lazyframe.py (py-1.28.1, representative subset)

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func lazyDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		{Name: "b", Values: []any{"w", "x", "y", "z"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// Lazy -> Collect round-trips the data unchanged.
func TestLazyCollectRoundTrip(t *testing.T) {
	t.Parallel()
	out, err := lazyDF(t).Lazy().Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if out.Height() != 4 || out.Width() != 2 {
		t.Fatalf("shape: got %dx%d, want 4x2", out.Height(), out.Width())
	}
}

// Lazy select narrows columns.
func TestLazySelect(t *testing.T) {
	t.Parallel()
	out, err := lazyDF(t).Lazy().Select(polars.Col("a")).Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if cols := out.Columns(); len(cols) != 1 || cols[0] != "a" {
		t.Fatalf("columns: got %v, want [a]", out.Columns())
	}
}

// Lazy filter + with_columns compose.
func TestLazyFilterWithColumns(t *testing.T) {
	t.Parallel()
	out, err := lazyDF(t).Lazy().
		Filter(polars.Col("a").Gt(polars.Lit(int64(2)))).
		WithColumns(polars.Col("a").Mul(polars.Lit(int64(10))).Alias("a10")).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2", out.Height())
	}
	a10, _ := out.GetColumn("a10")
	if v, _ := a10.Value(0).(int64); v != 30 {
		t.Fatalf("a10[0]: got %v, want 30", a10.Value(0))
	}
}

// Lazy group_by aggregation.
func TestLazyGroupBy(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"a", "a", "b"}},
		{Name: "v", Values: []any{int64(1), int64(2), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Lazy().GroupBy("g").Agg(polars.Sum(polars.Col("v")).Alias("s")).Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("groups: got %d, want 2", out.Height())
	}
}
