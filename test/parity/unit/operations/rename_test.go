package operations

// Ported from py-polars/tests/unit/operations/test_rename.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestRename(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{int64(3), int64(4)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Rename(map[string]string{"a": "x"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	cols := out.Columns()
	hasX, hasA := false, false
	for _, c := range cols {
		if c == "x" {
			hasX = true
		}
		if c == "a" {
			hasA = true
		}
	}
	if !hasX || hasA {
		t.Fatalf("rename columns: got %v, want a->x", cols)
	}
	// data preserved under new name
	x, _ := out.GetColumn("x")
	if v, _ := x.Value(0).(int64); v != 1 {
		t.Fatalf("x[0]: got %v, want 1", x.Value(0))
	}
}

func TestRenameMultiple(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1)}},
		{Name: "b", Values: []any{int64(2)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Rename(map[string]string{"a": "x", "b": "y"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	for _, c := range out.Columns() {
		if c != "x" && c != "y" {
			t.Fatalf("unexpected column %q in %v", c, out.Columns())
		}
	}
}
