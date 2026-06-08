package constructors

// Ported from py-polars/tests/unit/constructors/test_dataframe.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDataFrameConstructionFromDict(t *testing.T) {
	t.Parallel()

	df := parityDF(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "b", Values: []any{float64(1.0), float64(2.0), float64(3.0)}},
	)
	if df.Height() != 3 || df.Width() != 2 {
		t.Fatalf("shape: got %dx%d, want 3x2", df.Height(), df.Width())
	}
	cols := df.Columns()
	if cols[0] != "a" || cols[1] != "b" {
		t.Fatalf("columns: got %v", cols)
	}
}

func TestDataFrameConstructionWithNulls(t *testing.T) {
	t.Parallel()

	df := parityDF(t,
		frame.SeriesInput{Name: "x", Values: []any{int64(1), nil, int64(3)}},
		frame.SeriesInput{Name: "y", Values: []any{nil, float64(2.0), float64(3.0)}},
	)
	if df.Height() != 3 {
		t.Fatalf("height: got %d, want 3", df.Height())
	}

	// Verify null positions via null count
	nc := df.NullCount()
	// x column should have 1 null, y column should have 1 null
	xCol, err := df.GetColumn("x")
	if err != nil {
		t.Fatalf("GetColumn(x) failed: %v", err)
	}
	yCol, err := df.GetColumn("y")
	if err != nil {
		t.Fatalf("GetColumn(y) failed: %v", err)
	}
	_ = xCol
	_ = yCol
	if xCol.Value(1) != nil {
		t.Fatalf("x[1] should be null")
	}
	if yCol.Value(0) != nil {
		t.Fatalf("y[0] should be null")
	}
	_ = nc
}

func TestDataFrameConstructionEmpty(t *testing.T) {
	t.Parallel()

	df := parityDF(t)
	if df.Height() != 0 || df.Width() != 0 {
		t.Fatalf("empty df: got %dx%d, want 0x0", df.Height(), df.Width())
	}
}

func TestDataFrameConstructionFromSeries(t *testing.T) {
	t.Parallel()

	s1, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{int64(1), int64(2), int64(3)},
	})
	if err != nil {
		t.Fatalf("series creation failed: %v", err)
	}
	s2, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "b",
		DType:  polars.Int64,
		Values: []any{int64(4), int64(5), int64(6)},
	})
	if err != nil {
		t.Fatalf("series creation failed: %v", err)
	}

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: s1.Name(), Values: s1.ToList()},
			{Name: s2.Name(), Values: s2.ToList()},
		},
	})
	if err != nil {
		t.Fatalf("dataframe from series failed: %v", err)
	}
	if df.Height() != 3 || df.Width() != 2 {
		t.Fatalf("shape: got %dx%d, want 3x2", df.Height(), df.Width())
	}
}

func TestDataFrameDtypes(t *testing.T) {
	t.Parallel()

	df := parityDF(t,
		frame.SeriesInput{Name: "int_col", Values: []any{int64(1), int64(2)}},
		frame.SeriesInput{Name: "float_col", Values: []any{float64(1.0), float64(2.0)}},
		frame.SeriesInput{Name: "str_col", Values: []any{"a", "b"}},
		frame.SeriesInput{Name: "bool_col", Values: []any{true, false}},
	)
	if df.Height() != 2 || df.Width() != 4 {
		t.Fatalf("shape: got %dx%d, want 2x4", df.Height(), df.Width())
	}

	dtypes := df.Dtypes()
	if len(dtypes) != 4 {
		t.Fatalf("dtypes count: got %d, want 4", len(dtypes))
	}
}
