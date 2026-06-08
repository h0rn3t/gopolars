package operations

// Ported from py-polars/tests/unit/operations/unique/test_is_unique.py (py-1.28.1).
// Covers Series.is_unique / is_duplicated (incl. null cases) and the
// DataFrame-level row-uniqueness masks. gopolars matches Python here.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// seriesBools extracts a boolean Series into a []bool for comparison with eqBools.
func seriesBools(s polars.Series) []bool {
	out := make([]bool, s.Len())
	for i := 0; i < s.Len(); i++ {
		out[i], _ = s.Value(i).(bool)
	}
	return out
}

func TestIsUniqueSeries(t *testing.T) {
	t.Parallel()
	s, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(2), int64(3)}})
	if got := seriesBools(s.IsUnique()); !eqBools(got, []bool{true, false, false, true}) {
		t.Fatalf("is_unique: got %v", got)
	}
	if got := seriesBools(s.IsDuplicated()); !eqBools(got, []bool{false, true, true, false}) {
		t.Fatalf("is_duplicated: got %v", got)
	}
}

func TestIsUniqueStrings(t *testing.T) {
	t.Parallel()
	s, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.String, Values: []any{"a", "b", "c", "a"}})
	if got := seriesBools(s.IsUnique()); !eqBools(got, []bool{false, true, true, false}) {
		t.Fatalf("is_unique: got %v", got)
	}
	if got := seriesBools(s.IsDuplicated()); !eqBools(got, []bool{true, false, false, true}) {
		t.Fatalf("is_duplicated: got %v", got)
	}
}

func TestIsUniqueNulls(t *testing.T) {
	t.Parallel()
	// [None] -> unique True, duplicated False
	s1, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{nil}})
	if got := seriesBools(s1.IsUnique()); !eqBools(got, []bool{true}) {
		t.Fatalf("is_unique([None]): got %v", got)
	}
	// [None, None, None] -> all duplicated
	s3, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{nil, nil, nil}})
	if got := seriesBools(s3.IsUnique()); !eqBools(got, []bool{false, false, false}) {
		t.Fatalf("is_unique([None x3]): got %v", got)
	}
	if got := seriesBools(s3.IsDuplicated()); !eqBools(got, []bool{true, true, true}) {
		t.Fatalf("is_duplicated([None x3]): got %v", got)
	}
}

func TestIsUniqueDataFrame(t *testing.T) {
	t.Parallel()
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "foo", Values: []any{int64(1), int64(2), int64(2)}},
		{Name: "bar", Values: []any{int64(6), int64(7), int64(7)}},
	}})
	if got := seriesBools(df.IsUnique()); !eqBools(got, []bool{true, false, false}) {
		t.Fatalf("df.is_unique: got %v", got)
	}
	if got := seriesBools(df.IsDuplicated()); !eqBools(got, []bool{false, true, true}) {
		t.Fatalf("df.is_duplicated: got %v", got)
	}
}
