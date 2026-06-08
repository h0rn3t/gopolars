package lazyframe

// Ported from py-polars/tests/unit/lazyframe/test_schema.py and test_rename.py
// (py-1.28.1, representative subsets).

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_schema: Columns()/Width() are available on the lazy frame without collect.
func TestLazySchemaColumns(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1)}},
		{Name: "b", Values: []any{int64(2)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	lf := df.Lazy()
	if w := lf.Width(); w != 2 {
		t.Fatalf("lazy width: got %d, want 2", w)
	}
	cols := lf.Columns()
	if len(cols) != 2 || cols[0] != "a" || cols[1] != "b" {
		t.Fatalf("lazy columns: got %v, want [a b]", cols)
	}
}

// test_rename: lazy rename applies after collect.
func TestLazyRename(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Lazy().Rename(map[string]string{"a": "x"}).Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if cols := out.Columns(); len(cols) != 1 || cols[0] != "x" {
		t.Fatalf("rename columns: got %v, want [x]", out.Columns())
	}
}
