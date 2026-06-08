package operations

// Ported from py-polars/tests/unit/operations/unique/test_unique_counts.py (py-1.28.1).
//
// DISCREPANCY (cosmetic): Python `unique_counts` returns a UInt32 Series that
// keeps the original name; gopolars returns an Int64 Series named
// "<name>_unique_counts". The COUNTS and their order (first appearance) match.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestUniqueCounts(t *testing.T) {
	t.Parallel()
	s, _ := polars.NewSeries(polars.NewSeriesInput{Name: "id", DType: polars.String, Values: []any{"a", "b", "b", "c", "c", "c"}})
	out := s.UniqueCounts()
	if got := collectInts(out); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("unique_counts: got %v, want [1 2 3]", got)
	}
	// gopolars dtype divergence: Int64 (Python UInt32).
	if out.DataType() != polars.Int64 {
		t.Fatalf("unique_counts dtype: got %v, want Int64 (Python UInt32)", out.DataType())
	}
}

func TestUniqueCountsNulls(t *testing.T) {
	t.Parallel()
	// [None] -> [1]; [None, None, None] -> [3]
	s1, _ := polars.NewSeries(polars.NewSeriesInput{Name: "x", DType: polars.Int64, Values: []any{nil}})
	if got := collectInts(s1.UniqueCounts()); len(got) != 1 || got[0] != 1 {
		t.Fatalf("unique_counts([None]): got %v, want [1]", got)
	}
	s3, _ := polars.NewSeries(polars.NewSeriesInput{Name: "x", DType: polars.Int64, Values: []any{nil, nil, nil}})
	if got := collectInts(s3.UniqueCounts()); len(got) != 1 || got[0] != 3 {
		t.Fatalf("unique_counts([None x3]): got %v, want [3]", got)
	}
}
