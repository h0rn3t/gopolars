package operations

// Ported from py-polars/tests/unit/operations/unique/test_n_unique.py (py-1.28.1).
//
// DISCREPANCY: gopolars Series.NUnique() SKIPS nulls, whereas Python
// `Series.n_unique()` counts null as a single distinct value. Each affected
// assertion below documents both the gopolars result and the Python result.
// DataFrame.NUnique(cols...) (unique-row / subset counting) matches Python.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestNUniqueSeriesSkipsNulls(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.Int64,
		Values: []any{int64(11), int64(11), int64(11), int64(22), int64(22), int64(33), nil, nil, nil}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	// Python: 4 (three values + null). gopolars skips nulls -> 3.
	if got := s.NUnique(); got != 3 {
		t.Fatalf("n_unique: got %d, want 3 (gopolars skips null; Python=4)", got)
	}
}

func TestNUniqueAllNull(t *testing.T) {
	t.Parallel()
	// Python: [None] -> 1, [None, None] -> 1. gopolars: 0 (skips nulls).
	s1, _ := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.Int64, Values: []any{nil}})
	if got := s1.NUnique(); got != 0 {
		t.Fatalf("n_unique([None]): got %d, want 0 (Python=1)", got)
	}
	s2, _ := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.Int64, Values: []any{nil, nil}})
	if got := s2.NUnique(); got != 0 {
		t.Fatalf("n_unique([None,None]): got %d, want 0 (Python=1)", got)
	}
}

func TestNUniqueStringNoNulls(t *testing.T) {
	t.Parallel()
	s, _ := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.String, Values: []any{"a", "b", "b", "c"}})
	if got := s.NUnique(); got != 3 {
		t.Fatalf("n_unique: got %d, want 3", got)
	}
}

func TestNUniqueDataFrameSubsets(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(1), int64(2), int64(3), int64(4), int64(5)}},
		{Name: "b", Values: []any{0.5, 0.5, 1.0, 2.0, 3.0, 3.0}},
		{Name: "c", Values: []any{true, true, true, false, true, true}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	// No subset -> count of unique rows.
	if got, err := df.NUnique(); err != nil || got != 5 {
		t.Fatalf("df.n_unique(): got %d err %v, want 5", got, err)
	}
	// Subset of column names -> unique combinations.
	if got, err := df.NUnique("b", "c"); err != nil || got != 4 {
		t.Fatalf("df.n_unique([b,c]): got %d err %v, want 4", got, err)
	}
	if got, err := df.NUnique("c"); err != nil || got != 2 {
		t.Fatalf("df.n_unique([c]): got %d err %v, want 2", got, err)
	}
}
