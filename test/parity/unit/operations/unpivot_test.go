package operations

// Ported from py-polars/tests/unit/operations/test_unpivot.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// unpivot melts value columns into variable/value pairs.
func TestUnpivot(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2)}},
		{Name: "x", Values: []any{int64(10), int64(20)}},
		{Name: "y", Values: []any{int64(30), int64(40)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Unpivot(polars.MeltInput{
		IDVars:      []string{"id"},
		ValueVars:   []string{"x", "y"},
		VariableCol: "variable",
		ValueCol:    "value",
	})
	if err != nil {
		t.Fatalf("unpivot: %v", err)
	}
	// 2 ids x 2 value columns = 4 rows
	if out.Height() != 4 {
		t.Fatalf("unpivot height: got %d, want 4", out.Height())
	}
	cols := out.Columns()
	hasVar, hasVal := false, false
	for _, c := range cols {
		if c == "variable" {
			hasVar = true
		}
		if c == "value" {
			hasVal = true
		}
	}
	if !hasVar || !hasVal {
		t.Fatalf("unpivot columns: got %v, want variable+value", cols)
	}
}
