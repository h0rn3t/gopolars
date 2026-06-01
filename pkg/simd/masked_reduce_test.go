package simd

import (
	"math"
	"testing"
)

// refMaskedReduce is the previous branchy []bool implementation, kept as the
// correctness reference the Bitmap word-walk must match exactly. Reduction order
// is ascending index in both, so even the floating-point sum is bit-identical.
func refMaskedReduce(vals []float64, keep []bool, nulls []bool) (sum, min, max float64, count int) {
	for i, v := range vals {
		if !keep[i] {
			continue
		}
		if nulls != nil && nulls[i] {
			continue
		}
		if count == 0 {
			min, max = v, v
		} else {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		sum += v
		count++
	}
	return sum, min, max, count
}

func TestMaskedReduceFloat64MatchesReference(t *testing.T) {
	sizes := []int{0, 1, 63, 64, 65, 1000, 4097}
	selectivities := []struct {
		name string
		keep func(i int) bool
	}{
		{"empty", func(int) bool { return false }},
		{"half", func(i int) bool { return i%2 == 0 }},
		{"full", func(int) bool { return true }},
		{"sparse", func(i int) bool { return i%37 == 0 }},
	}
	nullModes := []struct {
		name  string
		build func(n int) []bool
	}{
		{"no_nulls", func(int) []bool { return nil }},
		{"some_nulls", func(n int) []bool {
			nl := make([]bool, n)
			for i := range nl {
				nl[i] = i%5 == 0
			}
			return nl
		}},
	}
	nanModes := []struct {
		name  string
		apply func(vals []float64)
	}{
		{"no_nan", func([]float64) {}},
		{"nan_first_kept", func(vals []float64) {
			if len(vals) > 0 {
				vals[0] = math.NaN() // index 0 is kept by "half"/"full"
			}
		}},
		{"nan_middle", func(vals []float64) {
			if len(vals) > 100 {
				vals[100] = math.NaN()
			}
		}},
	}

	for _, sz := range sizes {
		for _, sel := range selectivities {
			for _, nm := range nullModes {
				for _, nan := range nanModes {
					t.Run(sel.name+"/"+nm.name+"/"+nan.name+"/"+itoa(sz), func(t *testing.T) {
						vals := make([]float64, sz)
						for i := range vals {
							vals[i] = math.Sin(float64(i))*1e3 + float64(i)*0.25
						}
						nan.apply(vals)

						boolKeep := make([]bool, sz)
						bm := BitmapNew(sz)
						for i := range vals {
							if sel.keep(i) {
								boolKeep[i] = true
								BitmapSet(bm, i)
							}
						}
						nulls := nm.build(sz)

						wSum, wMin, wMax, wCount := refMaskedReduce(vals, boolKeep, nulls)
						gSum, gMin, gMax, gCount := MaskedReduceFloat64(vals, bm, nulls)

						if gCount != wCount {
							t.Fatalf("count: got %d, want %d", gCount, wCount)
						}
						if !exactOrBothNaN(gSum, wSum) {
							t.Fatalf("sum: got %v, want %v", gSum, wSum)
						}
						if !exactOrBothNaN(gMin, wMin) {
							t.Fatalf("min: got %v, want %v", gMin, wMin)
						}
						if !exactOrBothNaN(gMax, wMax) {
							t.Fatalf("max: got %v, want %v", gMax, wMax)
						}
					})
				}
			}
		}
	}
}

// itoa is a tiny helper to avoid pulling strconv into the subtest name.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// BenchmarkMaskedReduceFloat64 measures the fused masked reduction at 50%
// selectivity over 1M float64 rows — the worst branch-predictor regime for the
// old per-row "if !keep[i]" loop. The Bitmap word-walk visits only set bits and
// skips zero words, targeting ~2x over the branchy baseline.
func BenchmarkMaskedReduceFloat64(b *testing.B) {
	const n = 1_000_000
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = float64(i%1000)*0.5 - 250
	}
	bm := BitmapNew(n)
	for i := range n {
		if i%2 == 0 {
			BitmapSet(bm, i)
		}
	}
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	var s float64
	for b.Loop() {
		sum, _, _, _ := MaskedReduceFloat64(vals, bm, nil)
		s = sum
	}
	_ = s
}
