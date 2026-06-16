//go:build amd64

package simd

import (
	"math"
	"testing"
)

// avx2Inputs returns the non-empty input classes the AVX2 reduction kernels must
// match the scalar reference on. The AVX2 functions are called directly here
// (bypassing the hasAVX2 dispatch gate) because under Rosetta 2 cpu.X86.HasAVX2
// reports false yet the AVX2 instructions still execute correctly — this is the
// only way to exercise the assembly on the dev box.
func avx2Inputs() map[string][]float64 {
	mk := func(n int, f func(i int) float64) []float64 {
		s := make([]float64, n)
		for i := range s {
			s[i] = f(i)
		}
		return s
	}
	cases := map[string][]float64{
		"single":         {42.5},
		"three":          {3, 1, 2},
		"sub_vector_3":   {1.5, -2.25, 3.125},
		"exactly_4":      {4, -1, 9, 2},
		"five_unaligned": {1.5, -2.25, 3.125, -4, 5.0625},
		"fifteen":        mk(15, func(i int) float64 { return math.Sin(float64(i)) * 100 }),
		"exactly_16":     mk(16, func(i int) float64 { return float64(i%7)*0.5 - 3 }),
		"seventeen":      mk(17, func(i int) float64 { return math.Cos(float64(i)) * 50 }),
		"big_1009":       mk(1009, func(i int) float64 { return math.Sin(float64(i))*1e6 + float64(i%7)*0.1 }),
		"all_negative":   {-1, -2.5, -3.75, -100, -0.5, -9, -8, -0.001, -55, -2, -3, -4, -5, -6, -7, -8, -9, -10},
		"with_pos_inf":   {1, 2, math.Inf(1), 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
		"with_neg_inf":   {1, 2, math.Inf(-1), 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
		"signed_zero":    {0.0, math.Copysign(0, -1), 1, -1, 0.0, math.Copysign(0, -1), 2, -2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		"nan_first":      {math.NaN(), 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
		"nan_middle":     {1, 2, 3, math.NaN(), 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
		"nan_last":       {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, math.NaN()},
		"all_nan":        {math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN()},
	}
	return cases
}

func TestAVX2ReductionsMatchScalar(t *testing.T) {
	for name, in := range avx2Inputs() {
		t.Run(name, func(t *testing.T) {
			if got, want := minFloat64AVX2(in), minFloat64Scalar(in); !exactOrBothNaN(got, want) {
				t.Fatalf("min: avx2=%v scalar=%v", got, want)
			}
			if got, want := maxFloat64AVX2(in), maxFloat64Scalar(in); !exactOrBothNaN(got, want) {
				t.Fatalf("max: avx2=%v scalar=%v", got, want)
			}
			gotMin, gotMax := minMaxFloat64AVX2(in)
			wantMin, wantMax := minMaxFloat64Scalar(in)
			if !exactOrBothNaN(gotMin, wantMin) {
				t.Fatalf("minmax min: avx2=%v scalar=%v", gotMin, wantMin)
			}
			if !exactOrBothNaN(gotMax, wantMax) {
				t.Fatalf("minmax max: avx2=%v scalar=%v", gotMax, wantMax)
			}
		})
	}
}

func TestAVX2ArithMatchScalar(t *testing.T) {
	mkPair := func(n int) (a, b []float64) {
		a = make([]float64, n)
		b = make([]float64, n)
		for i := range a {
			a[i] = math.Sin(float64(i))*100 + 0.5
			b[i] = math.Cos(float64(i))*7 - 1.25
		}
		if n > 5 {
			a[3] = math.NaN()
			b[5] = math.Inf(1)
			a[n-1] = math.Inf(-1)
		}
		return a, b
	}
	// Sizes spanning the 16-wide loop, the 4-wide loop, and the scalar tail.
	for _, n := range []int{16, 17, 19, 20, 31, 64, 1009} {
		a, b := mkPair(n)
		dst := make([]float64, n)
		addSlicesFloat64AVX2(dst, a, b)
		want := addSlicesFloat64Scalar(a, b)
		for i := range want {
			if !exactOrBothNaN(dst[i], want[i]) {
				t.Fatalf("add n=%d idx=%d: avx2=%v scalar=%v", n, i, dst[i], want[i])
			}
		}
		dst2 := make([]float64, n)
		mulSlicesFloat64AVX2(dst2, a, b)
		wantMul := mulSlicesFloat64Scalar(a, b)
		for i := range wantMul {
			if !exactOrBothNaN(dst2[i], wantMul[i]) {
				t.Fatalf("mul n=%d idx=%d: avx2=%v scalar=%v", n, i, dst2[i], wantMul[i])
			}
		}
	}
}
