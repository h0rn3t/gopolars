package polars

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestSortAcrossDtypes drives lessAny and compareSortValues over int/float/
// string/bool columns, ascending and descending, with nulls and NaN, toggling
// NullsLast.
func TestSortAcrossDtypes(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "i", Values: []any{int64(3), int64(1), nil, int64(2)}},
		{Name: "f", Values: []any{2.0, math.NaN(), nil, 1.0}},
		{Name: "s", Values: []any{"b", "a", nil, "c"}},
		{Name: "bl", Values: []any{true, false, nil, true}},
	}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}

	for _, col := range []string{"i", "f", "s", "bl"} {
		for _, desc := range []bool{false, true} {
			for _, nl := range []bool{false, true} {
				out, err := df.Sort(SortInput{
					By:         []string{col},
					Descending: []bool{desc},
					NullsLast:  nl,
				})
				if err != nil {
					t.Fatalf("Sort by %s desc=%v nl=%v: %v", col, desc, nl, err)
				}
				if out.Height() != 4 {
					t.Errorf("Sort by %s: Height = %d, want 4", col, out.Height())
				}
			}
		}
	}
}

// TestCastValueMatrix drives castValueToType across every supported conversion
// plus an invalid cast.
func TestCastValueMatrix(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "i", Values: []any{int64(1), int64(2)}},
		{Name: "f", Values: []any{1.5, 2.5}},
		{Name: "si", Values: []any{"10", "20"}},   // numeric strings
		{Name: "sf", Values: []any{"1.5", "2.5"}}, // float strings
		{Name: "sb", Values: []any{"true", "false"}},
		{Name: "bl", Values: []any{true, false}},
	}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}

	good := []map[string]dtypes.DataType{
		{"i": dtypes.Float64},  // int → float
		{"i": dtypes.String},   // int → string
		{"f": dtypes.Int64},    // float → int
		{"f": dtypes.String},   // float → string
		{"si": dtypes.Int64},   // string → int
		{"sf": dtypes.Float64}, // string → float
		{"sb": dtypes.Boolean}, // string → bool
		{"bl": dtypes.String},  // bool → string
		{"bl": dtypes.Boolean}, // bool → bool identity
	}
	for _, m := range good {
		if _, err := df.Cast(m); err != nil {
			t.Errorf("Cast(%v): %v", m, err)
		}
	}

	// Invalid: non-numeric string → int.
	bad, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "s", Values: []any{"abc"}},
	}})
	if _, err := bad.Cast(map[string]dtypes.DataType{"s": dtypes.Int64}); err == nil {
		t.Error("Cast non-numeric string → int: expected error")
	}
}

// TestPivotAggMatrix drives aggregateValues across every aggregation kind by
// pivoting a frame whose (index,column) cells hold multiple values.
func TestPivotAggMatrix(t *testing.T) {
	makeDF := func() DataFrame {
		df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
			{Name: "g", Values: []any{"a", "a", "b", "b"}},
			{Name: "k", Values: []any{"x", "x", "x", "x"}},
			{Name: "v", Values: []any{int64(1), int64(3), int64(5), int64(7)}},
		}})
		if err != nil {
			t.Fatalf("NewDataFrame: %v", err)
		}
		return df
	}

	for _, agg := range []string{"sum", "mean", "count", "min", "max", "first", "unknown_agg"} {
		out, err := makeDF().Pivot(PivotInput{
			Index:   "g",
			Columns: "k",
			Values:  "v",
			Agg:     agg,
		})
		if err != nil {
			t.Fatalf("Pivot agg=%s: %v", agg, err)
		}
		if out.Height() != 2 {
			t.Errorf("Pivot agg=%s: Height = %d, want 2", agg, out.Height())
		}
	}

	// Float values exercise the non-int sum/mean branch.
	fdf, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"a", "a"}},
		{Name: "k", Values: []any{"x", "x"}},
		{Name: "v", Values: []any{1.5, 2.5}},
	}})
	if _, err := fdf.Pivot(PivotInput{Index: "g", Columns: "k", Values: "v", Agg: "sum"}); err != nil {
		t.Fatalf("Pivot float sum: %v", err)
	}
}

// TestGroupByAggDtypes drives the typed aggregation kernels (sumAndCount,
// extreme) across int and float grouped columns with a null.
func TestGroupByAggDtypes(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"a", "a", "b", "b"}},
		{Name: "i", Values: []any{int64(1), nil, int64(5), int64(7)}},
		{Name: "f", Values: []any{1.5, 2.5, nil, 4.5}},
	}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	out, err := df.GroupBy("g").Agg(
		Sum(Col("i")).Alias("sum_i"),
		Mean(Col("f")).Alias("mean_f"),
		Min(Col("i")).Alias("min_i"),
		Max(Col("f")).Alias("max_f"),
	)
	if err != nil {
		t.Fatalf("GroupBy Agg: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("GroupBy Height = %d, want 2", out.Height())
	}
}
