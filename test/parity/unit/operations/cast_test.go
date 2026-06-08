package operations

// Ported from py-polars/tests/unit/operations/test_cast.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestCastIntToFloat(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.Cast(polars.Float64)
	if err != nil {
		t.Fatalf("cast int->float: %v", err)
	}
	if out.DataType() != polars.Float64 {
		t.Fatalf("dtype: got %v, want Float64", out.DataType())
	}
	if v, ok := out.Value(0).(float64); !ok || v != 1.0 {
		t.Fatalf("value[0]: got %v, want 1.0", out.Value(0))
	}
}

func TestCastFloatToInt(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{1.0, 2.0, 3.0}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.Cast(polars.Int64)
	if err != nil {
		t.Fatalf("cast float->int: %v", err)
	}
	if v, ok := out.Value(2).(int64); !ok || v != 3 {
		t.Fatalf("value[2]: got %v, want 3", out.Value(2))
	}
}

func TestCastIntToString(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(10), int64(20)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.Cast(polars.String)
	if err != nil {
		t.Fatalf("cast int->string: %v", err)
	}
	if v, ok := out.Value(0).(string); !ok || v != "10" {
		t.Fatalf("value[0]: got %v, want \"10\"", out.Value(0))
	}
}

func TestCastNullsPreserved(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.Cast(polars.Float64)
	if err != nil {
		t.Fatalf("cast: %v", err)
	}
	if out.Value(1) != nil {
		t.Fatalf("null not preserved: got %v", out.Value(1))
	}
}
