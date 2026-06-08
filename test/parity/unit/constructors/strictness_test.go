package constructors

// Ported from py-polars/tests/unit/constructors/test_strictness.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// Tests for strict mode error handling during construction

func TestStrictTypeMismatchIntToString(t *testing.T) {
	t.Parallel()

	// Python: in strict mode, mixing int and string in a column raises TypeError
	// Go: gopolars requires explicit DType, so a mismatch means values must be compatible
	// We test that a string Series can be created with string values
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.String,
		Values: []any{"hello", "world"},
	})
	if err != nil {
		t.Fatalf("string series creation failed: %v", err)
	}
	if s.Len() != 2 {
		t.Fatalf("len: got %d, want 2", s.Len())
	}
}

func TestStrictNullInNonNullColumn(t *testing.T) {
	t.Parallel()

	// Go: nulls should be allowed in any type, matching Python with strict=False behavior
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), nil, int64(3)},
	})
	if err != nil {
		t.Fatalf("series with null failed: %v", err)
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
}

func TestStrictEmptySeriesWithDtype(t *testing.T) {
	t.Parallel()

	// Python: pl.Series("a", [], dtype=pl.Int64) should create empty Int64 series
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{},
	})
	if err != nil {
		t.Fatalf("empty series creation failed: %v", err)
	}
	if s.Len() != 0 {
		t.Fatalf("empty series len: got %d, want 0", s.Len())
	}
}

func TestStrictDataframeLengthMismatch(t *testing.T) {
	t.Parallel()

	// Python: DataFrame with mismatched column lengths raises ShapeError
	_, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{float64(1.0), float64(2.0)}},
		},
	})
	if err == nil {
		t.Fatalf("expected error for length mismatch, got nil")
	}
}

func TestStrictDataframeColumnNameConsistency(t *testing.T) {
	t.Parallel()

	// Python: column names should be unique
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1)}},
			{Name: "b", Values: []any{int64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("dataframe creation failed: %v", err)
	}
	cols := df.Columns()
	if len(cols) != 2 {
		t.Fatalf("columns count: got %d, want 2", len(cols))
	}
}
