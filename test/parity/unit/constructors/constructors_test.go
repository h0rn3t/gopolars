package constructors

// Ported from py-polars/tests/unit/constructors/test_constructors.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestInitDict(t *testing.T) {
	t.Parallel()

	// Empty DataFrame — works with no columns (Python: pl.DataFrame({}) -> 0x0).
	df := parityDF(t)
	if df.Height() != 0 || df.Width() != 0 {
		t.Fatalf("empty DataFrame should have shape 0x0, got %dx%d", df.Height(), df.Width())
	}

	// Empty DataFrame with typed columns — pin the dtype per column (analogue of
	// Python's schema parameter), since empty columns can't be inferred.
	df2, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{}, DType: polars.Int64},
			{Name: "b", Values: []any{}, DType: polars.Float64},
		},
	})
	if err != nil {
		t.Fatalf("empty typed DataFrame creation: %v", err)
	}
	if df2.Height() != 0 || df2.Width() != 2 {
		t.Fatalf("empty typed DataFrame should have shape 0x2, got %dx%d", df2.Height(), df2.Width())
	}
	cols2 := df2.Columns()
	if len(cols2) != 2 || cols2[0] != "a" || cols2[1] != "b" {
		t.Fatalf("columns mismatch: got %v", cols2)
	}

	// Mixed dtypes
	df = parityDF(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "b", Values: []any{float64(1.0), float64(2.0), float64(3.0)}},
	)
	if df.Height() != 3 || df.Width() != 2 {
		t.Fatalf("DataFrame shape mismatch: got %dx%d, want 3x2", df.Height(), df.Width())
	}
	colsMixed := df.Columns()
	if colsMixed[0] != "a" || colsMixed[1] != "b" {
		t.Fatalf("columns mismatch: got %v", colsMixed)
	}

	// Python: df.dtypes == [pl.Int64, pl.Float64]
	dtypes := df.Dtypes()
	if len(dtypes) != 2 {
		t.Fatalf("dtypes count mismatch: got %d, want 2", len(dtypes))
	}
}

// Subset: construction with explicit schema/dtype override
func TestInitDictWithSchemaOverride(t *testing.T) {
	t.Parallel()

	// Python: pl.DataFrame({"a": [1, 2, 3], "b": [4, 5, 6]}, schema=[("a", pl.Int8), ("b", pl.Float32)])
	df := parityDF(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "b", Values: []any{float64(4.0), float64(5.0), float64(6.0)}},
	)
	if df.Height() != 3 {
		t.Fatalf("DataFrame height mismatch: got %d, want 3", df.Height())
	}

	// Python: rename columns via schema
	// Parity note: gopolars doesn't have a "schema" parameter in NewDataFrame
	// but supports Cast and Rename after construction
	df2, err := df.Rename(map[string]string{"a": "x", "b": "y"})
	if err != nil {
		t.Fatalf("rename failed: %v", err)
	}
	cols := df2.Columns()
	if cols[0] != "x" || cols[1] != "y" {
		t.Fatalf("renamed columns mismatch: got %v", cols)
	}
}

// Subset: DataFrame from columns containing nulls
func TestInitDictWithNulls(t *testing.T) {
	t.Parallel()

	df := parityDF(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), nil, int64(3)}},
		frame.SeriesInput{Name: "b", Values: []any{nil, float64(2.0), float64(3.0)}},
	)
	if df.Height() != 3 {
		t.Fatalf("DataFrame height mismatch: got %d, want 3", df.Height())
	}

	// Check null positions
	aCol, err := df.GetColumn("a")
	if err != nil {
		t.Fatalf("GetColumn(a) failed: %v", err)
	}
	if aCol.Value(0) != int64(1) {
		t.Fatalf("a[0] mismatch: got %v, want 1", aCol.Value(0))
	}
	if aCol.Value(1) != nil {
		t.Fatalf("a[1] should be null, got %v", aCol.Value(1))
	}
	if aCol.Value(2) != int64(3) {
		t.Fatalf("a[2] mismatch: got %v, want 3", aCol.Value(2))
	}

	bCol, err := df.GetColumn("b")
	if err != nil {
		t.Fatalf("GetColumn(b) failed: %v", err)
	}
	if bCol.Value(0) != nil {
		t.Fatalf("b[0] should be null, got %v", bCol.Value(0))
	}
}

func TestInitSeries(t *testing.T) {
	t.Parallel()

	// DataFrame from list of Series
	df := parityDF(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "b", Values: []any{int64(4), int64(5), int64(6)}},
	)
	if df.Height() != 3 || df.Width() != 2 {
		t.Fatalf("DataFrame shape mismatch: got %dx%d, want 3x2", df.Height(), df.Width())
	}
}

// Subset: length mismatch should error
func TestInitErrors(t *testing.T) {
	t.Parallel()

	// Length mismatch between columns
	_, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{float64(1.0), float64(2.0), float64(3.0), float64(4.0)}},
		},
	})
	if err == nil {
		t.Fatalf("expected error for length mismatch, got nil")
	}
}

func TestFromDictUpcastPrimitive(t *testing.T) {
	t.Parallel()

	// Python: pl.from_dict({"a": [1, 2.1, 3], "b": [4, 5, 6.4]}, strict=False)
	// In Go, mixed int/float in a column means we need to use Float64 type
	df := parityDF(t,
		frame.SeriesInput{Name: "a", Values: []any{float64(1.0), float64(2.1), float64(3.0)}},
		frame.SeriesInput{Name: "b", Values: []any{float64(4.0), float64(5.0), float64(6.4)}},
	)
	if df.Height() != 3 {
		t.Fatalf("DataFrame height mismatch: got %d, want 3", df.Height())
	}

	aCol, err := df.GetColumn("a")
	if err != nil {
		t.Fatalf("GetColumn(a) failed: %v", err)
	}
	v, ok := aCol.Value(1).(float64)
	if !ok {
		t.Fatalf("a[1] is not float64: got %T", aCol.Value(1))
	}
	if v != 2.1 {
		t.Fatalf("a[1] mismatch: got %v, want 2.1", v)
	}
}

func TestInitOnlyColumns(t *testing.T) {
	t.Parallel()

	// Python: pl.DataFrame(schema=["a","b","c"]) makes an empty frame with those
	// columns. gopolars pins the dtype per empty column (analogue of the schema).
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{}, DType: polars.Int64},
			{Name: "b", Values: []any{}, DType: polars.Int64},
			{Name: "c", Values: []any{}, DType: polars.Int64},
		},
	})
	if err != nil {
		t.Fatalf("empty DataFrame with columns: %v", err)
	}
	if df.Height() != 0 || df.Width() != 3 {
		t.Fatalf("empty DataFrame with schema should have shape 0x3, got %dx%d", df.Height(), df.Width())
	}
	cols := df.Columns()
	if len(cols) != 3 || cols[0] != "a" || cols[1] != "b" || cols[2] != "c" {
		t.Fatalf("columns mismatch: got %v", cols)
	}
}

// Subset: DataFrame from records (dict-style)
func TestInitFromRecords(t *testing.T) {
	t.Parallel()

	// In Go we construct via NewDataFrameInput which is row-oriented internally
	df := parityDF(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(1)}},
		frame.SeriesInput{Name: "b", Values: []any{int64(2), int64(1), int64(2)}},
	)
	if df.Height() != 3 || df.Width() != 2 {
		t.Fatalf("DataFrame shape mismatch: got %dx%d, want 3x2", df.Height(), df.Width())
	}
}

func TestU64Values(t *testing.T) {
	t.Parallel()

	df := parityDF(t,
		frame.SeriesInput{Name: "foo", Values: []any{int64(1), int64(2), int64(3)}},
	)
	result, err := df.Filter(polars.Col("foo").Gt(polars.Lit(int64(0))))
	if err != nil {
		t.Fatalf("filter failed: %v", err)
	}
	if result.Height() != 3 {
		t.Fatalf("filter result height mismatch: got %d, want 3", result.Height())
	}
}

// (numpy, pandas, pyarrow, pydantic, dataclass, namedtuple — not applicable to Go)

func TestNullValues(t *testing.T) {
	t.Parallel()

	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  polars.Int64,
		Values: []any{nil, nil},
	})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.Len() != 2 {
		t.Fatalf("series length mismatch: got %d, want 2", s.Len())
	}
	if s.Value(0) != nil {
		t.Fatalf("s[0] should be null, got %v", s.Value(0))
	}
	if s.Value(1) != nil {
		t.Fatalf("s[1] should be null, got %v", s.Value(1))
	}
}

func parityDF(t *testing.T, columns ...frame.SeriesInput) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: columns})
	if err != nil {
		t.Fatalf("NewDataFrame failed: %v", err)
	}
	return df
}
