package micro

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// BenchmarkFilterFrame1M replicates the top30 `filter` workload: a 4-column 1M
// frame filtered by `v > 0` (≈half survive), materializing the survivors across
// all columns. Go-only (no Python), so it can be CPU-profiled.
func BenchmarkFilterFrame1M(b *testing.B) {
	const n = 1_000_000
	g := make([]any, n)
	v := make([]any, n)
	nn := make([]any, n)
	iv := make([]any, n)
	for i := 0; i < n; i++ {
		g[i] = "grp"
		v[i] = float64(i%200 - 100) // ≈half are > 0
		nn[i] = float64(i)
		iv[i] = int64(i)
	}
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: g},
		{Name: "v", Values: v},
		{Name: "n", Values: nn},
		{Name: "i", Values: iv},
	}})
	if err != nil {
		b.Fatalf("new dataframe: %v", err)
	}
	pred := polars.Col("v").Gt(polars.Lit(float64(0)))
	b.ReportAllocs()
	b.ResetTimer()
	for k := 0; k < b.N; k++ {
		if _, err := df.Filter(pred); err != nil {
			b.Fatalf("filter: %v", err)
		}
	}
}
