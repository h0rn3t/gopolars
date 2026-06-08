package datatypes

// Ported from py-polars/tests/unit/datatypes/test_utils.py (py-1.28.1)
//
// The Python test exercises dtype_to_init_repr (rendering a dtype as its
// constructor source, including nested List/Struct/Array). gopolars has no
// dtype-to-init-repr helper; the closest is Series.ToInitRepr, which renders a
// Series (not a dtype) as a constructor snippet. We assert that produces a
// non-empty representation and record the dtype-repr gap.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestSeriesToInitRepr(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2)}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	if repr := s.ToInitRepr(); repr == "" {
		t.Fatal("ToInitRepr returned empty string")
	}
}
