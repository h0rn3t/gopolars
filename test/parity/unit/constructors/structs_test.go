package constructors

// Ported from py-polars/tests/unit/constructors/test_structs.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Go doesn't have native Python dict-based structs, but gopolars supports struct Series

func TestStructConstructionFromMap(t *testing.T) {
	t.Parallel()

	// Python: pl.Series([{"a": 1, "b": 2}, {"a": 3, "b": 4}])
	// In Go, struct series are created via map[string]any values
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:  "s",
		DType: polars.Struct,
		Values: []any{
			map[string]any{"a": int64(1), "b": int64(2)},
			map[string]any{"a": int64(3), "b": int64(4)},
		},
	})
	if err != nil {
		// PARITY_FAIL: struct construction from maps may not work yet
		t.Fatalf("struct series creation failed: %v", err)
	}
	if s.Len() != 2 {
		t.Fatalf("struct series len: got %d, want 2", s.Len())
	}
}

func TestStructConstructionEmpty(t *testing.T) {
	t.Parallel()

	// Python: pl.Series([], dtype=pl.Struct([pl.Field("a", pl.Int64)]))
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "s",
		DType:  polars.Struct,
		Values: []any{},
	})
	if err != nil {
		t.Fatalf("empty struct series creation failed: %v", err)
	}
	if s.Len() != 0 {
		t.Fatalf("empty struct series len: got %d, want 0", s.Len())
	}
}
