package datatypes

// Ported from py-polars/tests/unit/datatypes/test_decimal.py (py-1.28.1, representative subset)
//
// gopolars Decimal series accept decimal values as strings (or dtypes.DecimalValue).

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDecimalConstructionFromStrings(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "d", DType: polars.Decimal, Values: []any{"1.5", "2.25", "3.125"}})
	if err != nil {
		t.Fatalf("new decimal: %v", err)
	}
	if s.DataType() != polars.Decimal {
		t.Fatalf("dtype: got %v, want Decimal", s.DataType())
	}
	if s.Len() != 3 {
		t.Fatalf("len: got %d, want 3", s.Len())
	}
}

func TestDecimalNulls(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "d", DType: polars.Decimal, Values: []any{"1.0", nil}})
	if err != nil {
		t.Fatalf("new decimal: %v", err)
	}
	if s.NullCount() != 1 {
		t.Fatalf("null count: got %d, want 1", s.NullCount())
	}
}
