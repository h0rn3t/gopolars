package functions

// Ported from py-polars/tests/unit/functions/as_datatype/test_struct.py (py-1.28.1)
//
// gopolars has no top-level pl.struct(["a","b"]) expression that packs several
// columns into a Struct column inside select/with_columns. What IS available:
// constructing a Struct Series directly from map values, and unnesting a struct
// column back into its fields. We port that round-trip.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Struct field extraction preserves the field dtype (Int64 -> 3), matching Polars.
func TestStructSeriesFieldAccess(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:  "s",
		DType: polars.Struct,
		Values: []any{
			map[string]any{"a": int64(1), "b": int64(2)},
			map[string]any{"a": int64(3), "b": int64(4)},
		},
	})
	if err != nil {
		t.Fatalf("struct series: %v", err)
	}
	a, err := s.Struct().Field("a")
	if err != nil {
		t.Fatalf("struct.field(a): %v", err)
	}
	if a.DataType() != polars.Int64 {
		t.Fatalf("field dtype: got %v, want Int64", a.DataType())
	}
	if v, ok := a.Value(1).(int64); !ok || v != 3 {
		t.Fatalf("a[1]: got %T(%v), want int64 3", a.Value(1), a.Value(1))
	}
}

// Unnest expands a struct column into its component fields (analogue of the
// inverse of pl.struct).
func TestStructUnnest(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "s", Values: []any{
				map[string]any{"a": int64(1), "b": int64(2)},
				map[string]any{"a": int64(3), "b": int64(4)},
			}},
		},
	})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.Unnest("s")
	if err != nil {
		t.Fatalf("unnest: %v", err)
	}
	// Unnest expands the struct into its fields (emitted in sorted order).
	cols := out.Columns()
	if len(cols) != 2 || cols[0] != "a" || cols[1] != "b" {
		t.Fatalf("unnest columns: got %v, want [a b]", cols)
	}
	a, _ := out.GetColumn("a")
	if v, ok := a.Value(0).(int64); !ok || v != 1 {
		t.Fatalf("a[0]: got %T(%v), want int64 1", a.Value(0), a.Value(0))
	}
}
