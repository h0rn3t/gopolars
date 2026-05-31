package unit

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestPublicSeriesAPI(t *testing.T) {
	left, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), nil, int64(3)},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	right, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "b",
		DType:  polars.Int64,
		Values: []any{int64(10), int64(20), int64(30)},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	filled, err := left.FillNull(int64(0))
	if err != nil {
		t.Fatalf("fill null failed: %v", err)
	}
	sum, err := filled.Add(right)
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if sum.Value(0) != int64(11) || sum.Value(1) != int64(20) || sum.Value(2) != int64(33) {
		t.Fatalf("unexpected add result")
	}
	mask := left.IsNull()
	if mask.Value(1) != true || mask.Value(0) != false {
		t.Fatalf("unexpected is_null mask")
	}
	dropped := left.DropNulls()
	if dropped.Len() != 2 {
		t.Fatalf("unexpected drop_nulls size")
	}
	casted, err := right.Cast(polars.Float64)
	if err != nil {
		t.Fatalf("cast failed: %v", err)
	}
	if _, ok := casted.Value(0).(float64); !ok {
		t.Fatalf("expected float64 after cast")
	}
}
