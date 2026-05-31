package frame

import (
	"fmt"
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
	"github.com/h0rn3t/gopolars/pkg/simd"
)

// filterSumFusedSink keeps benchmark results live against dead-code elimination.
var filterSumFusedSink float64

// reduceFloat64Materialized mirrors exec.aggregateFrame's Float64 semantics over
// an already-filtered series. It is the "materialized-then-reduce" oracle the
// fused path must match: seed min/max on the first non-null row, plain < / > so
// NaN is sticky-from-seed, and a null (no non-null rows) for empty input.
func reduceFloat64Materialized(s series.Series, op string) any {
	var sum, mn, mx float64
	count := 0
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		v := s.Value(i).(float64)
		if count == 0 {
			mn, mx = v, v
		} else {
			if v < mn {
				mn = v
			}
			if v > mx {
				mx = v
			}
		}
		sum += v
		count++
	}
	switch op {
	case "count":
		return int64(count)
	case "mean":
		if count == 0 {
			return nil
		}
		return sum / float64(count)
	case "min":
		if count == 0 {
			return nil
		}
		return mn
	case "max":
		if count == 0 {
			return nil
		}
		return mx
	default: // sum
		if count == 0 {
			return nil
		}
		return sum
	}
}

// aggEqual compares two aggregation cell values, treating two NaNs as equal and
// two nils as equal.
func aggEqual(x, y any) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	switch xv := x.(type) {
	case float64:
		yv, ok := y.(float64)
		if !ok {
			return false
		}
		return xv == yv || (math.IsNaN(xv) && math.IsNaN(yv))
	case int64:
		yv, ok := y.(int64)
		return ok && xv == yv
	default:
		return x == y
	}
}

// buildAggFixture builds a two float64-column frame of height n. Column "a" is
// the predicate operand (with nulls); "b" is a payload with both nulls and NaNs
// so the reduction must handle both.
func buildAggFixture(t *testing.T, n int) DataFrame {
	t.Helper()
	av := make([]any, n)
	bv := make([]any, n)
	for i := range n {
		if i%7 == 0 {
			av[i] = nil
		} else {
			av[i] = float64((i % 101) - 50)
		}
		switch {
		case i%5 == 0:
			bv[i] = nil
		case i%50 == 3:
			bv[i] = math.NaN()
		default:
			bv[i] = float64(i) * 0.5
		}
	}
	a, err := series.New("a", dtypes.Float64, av)
	if err != nil {
		t.Fatalf("series a: %v", err)
	}
	b, err := series.New("b", dtypes.Float64, bv)
	if err != nil {
		t.Fatalf("series b: %v", err)
	}
	df, err := New(NewInput{Series: []series.Series{a, b}})
	if err != nil {
		t.Fatalf("new df: %v", err)
	}
	return df
}

// TestFilterAggregateMatchesMaterialized checks the fused filter+reduce result
// equals the materialized Filter()-then-reduce result for every supported op,
// including null and NaN handling, at sizes below and above the parallel
// threshold (so both the sequential and worker-partitioned reductions are
// exercised).
func TestFilterAggregateMatchesMaterialized(t *testing.T) {
	pred := expr.Col("a").Gt(expr.Lit(0.0))
	ops := []string{"sum", "count", "mean", "min", "max"}
	sizes := []int{1000, parallelFilterThreshold + 4096}

	for _, n := range sizes {
		df := buildAggFixture(t, n)
		materialized, err := df.Filter(pred) // the real (parallel) materializing path
		if err != nil {
			t.Fatalf("n=%d filter: %v", n, err)
		}
		for _, op := range ops {
			t.Run(fmt.Sprintf("n=%d/%s", n, op), func(t *testing.T) {
				fused, ok, err := df.FilterAggregate(pred, op, nil)
				if err != nil {
					t.Fatalf("FilterAggregate: %v", err)
				}
				if !ok {
					t.Fatalf("FilterAggregate declined a supported op %q", op)
				}
				if fused.Height() != 1 {
					t.Fatalf("fused result height = %d, want 1", fused.Height())
				}
				for _, col := range []string{"a", "b"} {
					ms, _ := materialized.Series(col)
					want := reduceFloat64Materialized(ms, op)
					fs, _ := fused.Series(col)
					var got any
					if !fs.IsNull(0) {
						got = fs.Value(0)
					}
					if !aggEqual(got, want) {
						t.Fatalf("col %s op %s: fused=%v want(materialized)=%v", col, op, got, want)
					}
				}
			})
		}
	}
}

// TestFilterAggregateFallback verifies the fused path declines (ok=false) for
// an unsupported reduction, an unsupported column dtype, and an unsupported
// predicate — so the executor can fall back to materialize-then-aggregate.
func TestFilterAggregateFallback(t *testing.T) {
	pred := expr.Col("a").Gt(expr.Lit(0.0))

	t.Run("unsupported_op", func(t *testing.T) {
		df := buildAggFixture(t, 100)
		if _, ok, _ := df.FilterAggregate(pred, "median", nil); ok {
			t.Fatal("median should not be handled by the fused path")
		}
	})

	t.Run("non_float64_column", func(t *testing.T) {
		ai, _ := series.New("a", dtypes.Int64, []any{int64(1), int64(-2), int64(3)})
		df, err := New(NewInput{Series: []series.Series{ai}})
		if err != nil {
			t.Fatalf("new df: %v", err)
		}
		if _, ok, _ := df.FilterAggregate(expr.Col("a").Gt(expr.Lit(int64(0))), "sum", nil); ok {
			t.Fatal("int64 column should fall back, not fuse")
		}
	})

	t.Run("unsupported_predicate", func(t *testing.T) {
		df := buildAggFixture(t, 100)
		// Reverse is a row-wise-only expression that evalbatch.Compile rejects.
		if _, ok, _ := df.FilterAggregate(expr.Col("a").Reverse(), "sum", nil); ok {
			t.Fatal("unsupported predicate should fall back, not fuse")
		}
	})
}

// buildFilterSumBenchFrame builds a single Float64 column "a" of height n with
// values spanning [-50, 50) so that `a > 0` keeps ~50% and `a > 50` keeps none.
func buildFilterSumBenchFrame(b *testing.B, n int) DataFrame {
	b.Helper()
	vals := make([]any, n)
	for i := range n {
		vals[i] = float64((i % 100)) - 50
	}
	a, err := series.New("a", dtypes.Float64, vals)
	if err != nil {
		b.Fatalf("series: %v", err)
	}
	df, err := New(NewInput{Series: []series.Series{a}})
	if err != nil {
		b.Fatalf("new df: %v", err)
	}
	return df
}

// BenchmarkFilterSumFused compares the fused single-pass filter+sum
// (FilterAggregate) against the materializing path (Filter then Sum over the
// gathered column) at 1M/10M rows for both the empty (~0%) and half (~50%)
// selectivity profiles.
func BenchmarkFilterSumFused(b *testing.B) {
	profiles := []struct {
		name      string
		threshold float64
	}{
		{"empty", 50.0},
		{"half", 0.0},
	}
	sizes := []struct {
		name string
		n    int
	}{
		{"1M", 1_000_000},
		{"10M", 10_000_000},
	}
	for _, prof := range profiles {
		for _, sz := range sizes {
			df := buildFilterSumBenchFrame(b, sz.n)
			pred := expr.Col("a").Gt(expr.Lit(prof.threshold))

			b.Run(fmt.Sprintf("materialize/%s/%s", prof.name, sz.name), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(sz.n * 8))
				for b.Loop() {
					filtered, err := df.Filter(pred)
					if err != nil {
						b.Fatalf("filter: %v", err)
					}
					s, _ := filtered.Series("a")
					f64, _ := s.Column().Float64s()
					filterSumFusedSink = simd.SumFloat64(f64)
				}
			})

			b.Run(fmt.Sprintf("fused/%s/%s", prof.name, sz.name), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(sz.n * 8))
				for b.Loop() {
					res, ok, err := df.FilterAggregate(pred, "sum", nil)
					if err != nil || !ok {
						b.Fatalf("fused filter+sum: ok=%v err=%v", ok, err)
					}
					s, _ := res.Series("a")
					if !s.IsNull(0) {
						filterSumFusedSink = s.Value(0).(float64)
					}
				}
			})
		}
	}
}
