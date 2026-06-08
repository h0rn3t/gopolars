package constructors

// Ported from py-polars/tests/unit/constructors/test_any_value_fallbacks.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Python has complex fallback logic for mixed types; Go requires explicit DType

func TestAnyValueFallbackBoolToInt(t *testing.T) {
	t.Parallel()

	// Python: pl.Series([True, 1]) with strict=False → Int64([1, 1])
	// Go: explicit DType required
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), int64(1)},
	})
	if err != nil {
		t.Fatalf("series creation failed: %v", err)
	}
	if s.Len() != 2 {
		t.Fatalf("len: got %d, want 2", s.Len())
	}
}

func TestAnyValueFallbackIntToFloat(t *testing.T) {
	t.Parallel()

	// Python: pl.Series([1, 1.0]) → Float64([1.0, 1.0])
	// Go: explicit Float64 type
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Float64,
		Values: []any{float64(1.0), float64(1.0)},
	})
	if err != nil {
		t.Fatalf("series creation failed: %v", err)
	}
	if s.Len() != 2 {
		t.Fatalf("len: got %d, want 2", s.Len())
	}
}

func TestAnyValueFallbackStringMixedInt(t *testing.T) {
	t.Parallel()

	// Python: pl.Series([1, "foo"], strict=False) → String(["1", "foo"])
	// Go: require explicit String DType
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.String,
		Values: []any{"1", "foo"},
	})
	if err != nil {
		t.Fatalf("series creation failed: %v", err)
	}
	if s.Len() != 2 {
		t.Fatalf("len: got %d, want 2", s.Len())
	}
}

func TestAnyValueFallbackNullInMixedType(t *testing.T) {
	t.Parallel()

	// Python: null values are common in mixed-type fallbacks
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), nil, int64(3)},
	})
	if err != nil {
		t.Fatalf("series creation failed: %v", err)
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
}

func TestAnyValueFallbackEmptyWithDtype(t *testing.T) {
	t.Parallel()

	// Python: pl.Series([], dtype=pl.Float64)
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Float64,
		Values: []any{},
	})
	if err != nil {
		t.Fatalf("series creation failed: %v", err)
	}
	if s.Len() != 0 {
		t.Fatalf("len: got %d, want 0", s.Len())
	}
}

func TestAnyValueFallbackNullOnly(t *testing.T) {
	t.Parallel()

	// Python: pl.Series([None, None], dtype=pl.Int64)
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{nil, nil},
	})
	if err != nil {
		t.Fatalf("series creation failed: %v", err)
	}
	if s.Len() != 2 {
		t.Fatalf("len: got %d, want 2", s.Len())
	}
	for i := 0; i < 2; i++ {
		if s.Value(i) != nil {
			t.Fatalf("value[%d]: should be nil, got %v", i, s.Value(i))
		}
	}
}
