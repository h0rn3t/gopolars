package operations

// Ported from py-polars/tests/unit/operations/test_group_by.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func groupDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"a", "a", "b", "b", "b"}},
		{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// group_by(...).agg(sum) reduces each group to one row.
func TestGroupBySum(t *testing.T) {
	t.Parallel()
	out, err := groupDF(t).GroupBy("g").Agg(polars.Sum(polars.Col("v")).Alias("sum_v"))
	if err != nil {
		t.Fatalf("group_by sum: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("groups: got %d, want 2", out.Height())
	}
	// collect g -> sum mapping
	got := map[string]int64{}
	gcol, _ := out.GetColumn("g")
	scol, _ := out.GetColumn("sum_v")
	for i := 0; i < out.Height(); i++ {
		k, _ := gcol.Value(i).(string)
		switch v := scol.Value(i).(type) {
		case int64:
			got[k] = v
		case float64:
			got[k] = int64(v)
		}
	}
	if got["a"] != 3 {
		t.Fatalf("sum a: got %d, want 3", got["a"])
	}
	if got["b"] != 12 {
		t.Fatalf("sum b: got %d, want 12", got["b"])
	}
}

// group_by(...).agg(count) gives group sizes.
func TestGroupByCount(t *testing.T) {
	t.Parallel()
	out, err := groupDF(t).GroupBy("g").Agg(polars.Count().Alias("n"))
	if err != nil {
		t.Fatalf("group_by count: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("groups: got %d, want 2", out.Height())
	}
}

// group_by(...).agg(mean) computes per-group means.
func TestGroupByMean(t *testing.T) {
	t.Parallel()
	out, err := groupDF(t).GroupBy("g").Agg(polars.Mean(polars.Col("v")).Alias("mean_v"))
	if err != nil {
		t.Fatalf("group_by mean: %v", err)
	}
	gcol, _ := out.GetColumn("g")
	mcol, _ := out.GetColumn("mean_v")
	for i := 0; i < out.Height(); i++ {
		k, _ := gcol.Value(i).(string)
		m, _ := mcol.Value(i).(float64)
		if k == "a" && m != 1.5 {
			t.Fatalf("mean a: got %v, want 1.5", m)
		}
		if k == "b" && m != 4.0 {
			t.Fatalf("mean b: got %v, want 4.0", m)
		}
	}
}
