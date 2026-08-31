package micro

import (
	"math/rand"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/simd"
)

var benchSizes = []int{1_000, 10_000, 100_000, 1_000_000, 10_000_000}

func makeFloat64Slice(n int) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = rand.Float64()*100 - 50
	}
	return s
}

func BenchmarkSumFloat64(b *testing.B) {
	for _, n := range benchSizes {
		data := makeFloat64Slice(n)
		b.Run("size_"+humanize(n), func(b *testing.B) {
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				simd.SumFloat64(data)
			}
		})
	}
}

func BenchmarkMinFloat64(b *testing.B) {
	for _, n := range benchSizes {
		data := makeFloat64Slice(n)
		b.Run("size_"+humanize(n), func(b *testing.B) {
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				simd.MinFloat64(data)
			}
		})
	}
}

func BenchmarkMaxFloat64(b *testing.B) {
	for _, n := range benchSizes {
		data := makeFloat64Slice(n)
		b.Run("size_"+humanize(n), func(b *testing.B) {
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				simd.MaxFloat64(data)
			}
		})
	}
}

func BenchmarkMinMaxFloat64(b *testing.B) {
	for _, n := range benchSizes {
		data := makeFloat64Slice(n)
		b.Run("size_"+humanize(n), func(b *testing.B) {
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				simd.MinMaxFloat64(data)
			}
		})
	}
}

// BenchmarkSumWhereFloat64 covers the fused filter+sum kernel behind the
// cross-language benchmark (run-bench.sh). The literal is 0 over data uniform in
// [-50,50), so selectivity is ~50% — the case the flagship benchmark measures.
func BenchmarkSumWhereFloat64(b *testing.B) {
	for _, n := range benchSizes {
		data := makeFloat64Slice(n)
		b.Run("size_"+humanize(n), func(b *testing.B) {
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				simd.SumWhereFloat64(data, simd.CmpGT, 0, nil)
			}
		})
	}
}

func BenchmarkMinMaxWhereFloat64(b *testing.B) {
	for _, n := range benchSizes {
		data := makeFloat64Slice(n)
		b.Run("size_"+humanize(n), func(b *testing.B) {
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				simd.MinMaxWhereFloat64(data, simd.CmpGT, 0, nil)
			}
		})
	}
}

func humanize(n int) string {
	switch n {
	case 1_000:
		return "1K"
	case 10_000:
		return "10K"
	case 100_000:
		return "100K"
	case 1_000_000:
		return "1M"
	case 10_000_000:
		return "10M"
	}
	return "unknown"
}
