package functions

// Ported from py-polars/tests/unit/functions/as_datatype/test_concat_list.py (py-1.28.1)
//
// gopolars has no top-level pl.concat_list(...) expression that horizontally
// concatenates several columns/values into a single List column. The closest
// available primitive is Series.Implode (collapse a Series into a single one-row
// List), which we exercise to document what IS available.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Implode is the nearest available list-building primitive: it collapses all
// values of a Series into a single List-valued row.
func TestImplodeAsListBuilder(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	out := s.Implode()
	if out.Len() != 1 {
		t.Fatalf("implode len: got %d, want 1 (single List row)", out.Len())
	}
}
