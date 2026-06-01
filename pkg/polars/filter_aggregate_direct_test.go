package polars

import (
	"context"
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// approxEqual reports whether a and b agree within a relative reduction-order
// tolerance, treating +0.0 and -0.0 as equal.
func approxEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9*(math.Abs(b)+1)
}

// buildDirectFrame builds a single float64 column "a" of n rows where roughly
// fraction of the rows are positive (kept by col("a") > 0). Distinct kept/dropped
// values keep the sum meaningful.
func buildDirectFrame(t testing.TB, n int, fraction float64) DataFrame {
	t.Helper()
	vals := make([]any, n)
	for i := range vals {
		frac := float64(i%1000) / 1000.0
		if frac < fraction {
			vals[i] = 2.0
		} else {
			vals[i] = -1.0
		}
	}
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{{Name: "a", Values: vals}}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	return df
}

// lazySum runs the canonical lazy fused filter+sum over column "a" and returns
// the single resulting value.
func lazySum(t testing.TB, df DataFrame) float64 {
	t.Helper()
	out, err := df.Lazy().Filter(Col("a").Gt(Lit(0.0))).Sum().Collect(context.Background())
	if err != nil {
		t.Fatalf("lazy collect: %v", err)
	}
	s, ok := out.Series("a")
	if !ok {
		t.Fatalf("result column a not found")
	}
	v := s.Value(0)
	if v == nil {
		return 0
	}
	f, _ := v.(float64)
	return f
}

// TestFilterAggregateDirectSumMatchesLazy pins the eager fused sum to the lazy
// fused sum across 0%, 50%, and 100% selectivity (spec scenario).
func TestFilterAggregateDirectSumMatchesLazy(t *testing.T) {
	const n = 50_000
	for _, frac := range []float64{0.0, 0.5, 1.0} {
		df := buildDirectFrame(t, n, frac)
		got, err := df.FilterAggregateDirect(Col("a").Gt(Lit(0.0)), "sum", []string{"a"})
		if err != nil {
			t.Fatalf("frac %.1f: FilterAggregateDirect: %v", frac, err)
		}
		want := lazySum(t, df)
		if !approxEqual(got["a"], want) {
			t.Fatalf("frac %.1f: direct sum = %v, lazy sum = %v", frac, got["a"], want)
		}
	}
}

// TestFilterAggregateDirectEmpty confirms the zero-survivor selectivity gate
// returns a map with the requested key set to 0.
func TestFilterAggregateDirectEmpty(t *testing.T) {
	df := buildDirectFrame(t, 10_000, 0.0)
	got, err := df.FilterAggregateDirect(Col("a").Gt(Lit(0.0)), "sum", []string{"a"})
	if err != nil {
		t.Fatalf("FilterAggregateDirect: %v", err)
	}
	if v, ok := got["a"]; !ok || v != 0 {
		t.Fatalf("empty filter: got[a] = %v (present=%v), want 0", v, ok)
	}
}

// TestFilterAggregateDirectOps checks sum/min/max/mean/count against a manual
// reduction over the kept rows at ~50% selectivity, with a row count that is not
// a multiple of 64 (exercises the partial last word).
func TestFilterAggregateDirectOps(t *testing.T) {
	const n = 4097
	vals := make([]any, n)
	var kept []float64
	for i := range vals {
		v := math.Sin(float64(i)) * 10
		vals[i] = v
		if v > 0 {
			kept = append(kept, v)
		}
	}
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{{Name: "a", Values: vals}}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	pred := Col("a").Gt(Lit(0.0))

	wantSum, wantMin, wantMax := 0.0, kept[0], kept[0]
	for _, v := range kept {
		wantSum += v
		if v < wantMin {
			wantMin = v
		}
		if v > wantMax {
			wantMax = v
		}
	}
	cases := map[string]float64{
		"sum":   wantSum,
		"min":   wantMin,
		"max":   wantMax,
		"mean":  wantSum / float64(len(kept)),
		"count": float64(len(kept)),
	}
	for op, want := range cases {
		got, err := df.FilterAggregateDirect(pred, op, []string{"a"})
		if err != nil {
			t.Fatalf("op %s: %v", op, err)
		}
		if op == "sum" || op == "mean" {
			if !approxEqual(got["a"], want) {
				t.Fatalf("op %s: got %v, want %v", op, got["a"], want)
			}
		} else if got["a"] != want {
			t.Fatalf("op %s: got %v, want %v", op, got["a"], want)
		}
	}
}

// TestFilterAggregateDirectNullExclusion confirms null rows that pass the
// predicate are excluded from the sum (matching lazy/materialized semantics).
func TestFilterAggregateDirectNullExclusion(t *testing.T) {
	vals := []any{1.0, nil, 3.0, 4.0, nil}
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{{Name: "a", Values: vals}}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	got, err := df.FilterAggregateDirect(Col("a").Gt(Lit(0.0)), "sum", []string{"a"})
	if err != nil {
		t.Fatalf("FilterAggregateDirect: %v", err)
	}
	if got["a"] != 8.0 { // 1 + 3 + 4, nulls excluded
		t.Fatalf("sum with nulls = %v, want 8", got["a"])
	}
	if want := lazySum(t, df); !approxEqual(got["a"], want) {
		t.Fatalf("direct sum %v != lazy sum %v", got["a"], want)
	}
}

// TestFilterAggregateDirectNaNExcludedByPredicate confirms a NaN value does not
// pass a "> threshold" predicate (NaN comparisons are false), so it is excluded
// from min — matching the kernel's null/NaN semantics.
func TestFilterAggregateDirectNaNExcludedByPredicate(t *testing.T) {
	vals := []any{math.NaN(), 1.0, 2.0, 3.0}
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{{Name: "a", Values: vals}}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	got, err := df.FilterAggregateDirect(Col("a").Gt(Lit(math.Inf(-1))), "min", []string{"a"})
	if err != nil {
		t.Fatalf("FilterAggregateDirect: %v", err)
	}
	// NaN > -inf is false, so NaN is not kept; min is over {1,2,3} = 1.
	if got["a"] != 1.0 {
		t.Fatalf("min = %v, want 1 (NaN excluded by predicate)", got["a"])
	}
}

// TestFilterAggregateDirectAllocations pins the no-materialization contract: at
// 1M rows / 50% selectivity the eager fused path allocates only the predicate
// bitmap and the result map — no []int survivor slice, no sliced column, no
// DataFrame — so allocs/op are a small constant independent of row count.
func TestFilterAggregateDirectAllocations(t *testing.T) {
	if testing.Short() {
		t.Skip("allocation test builds a 1M-row frame")
	}
	const n = 1_000_000
	df := buildDirectFrame(t, n, 0.5)
	pred := Col("a").Gt(Lit(0.0))
	if _, err := df.FilterAggregateDirect(pred, "sum", []string{"a"}); err != nil {
		t.Fatalf("warmup: %v", err)
	}
	allocs := testing.AllocsPerRun(20, func() {
		if _, err := df.FilterAggregateDirect(pred, "sum", []string{"a"}); err != nil {
			t.Fatalf("FilterAggregateDirect: %v", err)
		}
	})
	if allocs > 8 {
		t.Fatalf("FilterAggregateDirect allocated %v objs/op at 1M/50%%, want <= 8 (no []int/DataFrame materialization)", allocs)
	}
}
