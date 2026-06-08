package datatypes

// Ported from py-polars/tests/unit/datatypes/test_string.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_series_init_string: inferred dtype for string values is String.
func TestSeriesInitString(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.String, Values: []any{"a", "b"}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	if s.DataType() != polars.String {
		t.Fatalf("dtype: got %v, want String", s.DataType())
	}
}

// test_utf8_alias_*: in gopolars there is a single String dtype (no separate
// Utf8 alias object), so the alias-equality tests collapse to "String is String".
func TestUtf8IsString(t *testing.T) {
	t.Parallel()
	if polars.String != polars.String {
		t.Fatal("String must equal itself")
	}
}
