package micro

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// BenchmarkFilterSumFloat64_1M measures filter+sum on 1M float64 rows using the
// typed vectorized path (evalbatch + chunk gather + float64 backing aggregation).
// Run with:
//
//	go test ./bench/micro -bench=BenchmarkFilterSumFloat64_1M -benchmem
func BenchmarkFilterSumFloat64_1M(b *testing.B) {
	const n = 1_000_000
	vals := make([]any, n)
	for i := range vals {
		vals[i] = float64(i % 1000)
	}
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "v", Values: vals},
		},
	})
	if err != nil {
		b.Fatalf("build df: %v", err)
	}

	b.ResetTimer()
	b.SetBytes(int64(n * 8))
	for i := 0; i < b.N; i++ {
		filtered, err := df.Filter(polars.Col("v").Gt(polars.Lit(500.0)))
		if err != nil {
			b.Fatalf("filter: %v", err)
		}
		sv, ok := filtered.Series("v")
		if !ok {
			b.Fatalf("series not found")
		}
		_ = sv.Sum()
	}
}

// BenchmarkFilterSumInt64_1M is the same workload for int64 columns.
func BenchmarkFilterSumInt64_1M(b *testing.B) {
	const n = 1_000_000
	vals := make([]any, n)
	for i := range vals {
		vals[i] = int64(i % 1000)
	}
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "v", Values: vals},
		},
	})
	if err != nil {
		b.Fatalf("build df: %v", err)
	}

	b.ResetTimer()
	b.SetBytes(int64(n * 8))
	for i := 0; i < b.N; i++ {
		filtered, err := df.Filter(polars.Col("v").Gt(polars.Lit(int64(500))))
		if err != nil {
			b.Fatalf("filter: %v", err)
		}
		sv, ok := filtered.Series("v")
		if !ok {
			b.Fatalf("series not found")
		}
		_ = sv.Sum()
	}
}

// BenchmarkGroupBySumFloat64_1M measures group_by + sum on 1M float64 rows via
// the typed chunk fast path (task 3.3/5.1). Baseline: row-wise expr.Eval per bucket.
func BenchmarkGroupBySumFloat64_1M(b *testing.B) {
	const n = 1_000_000
	const nGroups = 100
	keys := make([]any, n)
	vals := make([]any, n)
	for i := 0; i < n; i++ {
		keys[i] = int64(i % nGroups)
		vals[i] = float64(i)
	}
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "g", Values: keys},
			{Name: "v", Values: vals},
		},
	})
	if err != nil {
		b.Fatalf("build df: %v", err)
	}

	b.ResetTimer()
	b.SetBytes(int64(n * 8))
	for i := 0; i < b.N; i++ {
		_, err := df.GroupBy("g").Agg(polars.Sum(polars.Col("v")))
		if err != nil {
			b.Fatalf("groupby: %v", err)
		}
	}
}
