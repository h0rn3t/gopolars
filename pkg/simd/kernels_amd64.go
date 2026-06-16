//go:build amd64

package simd

import "golang.org/x/sys/cpu"

// amd64 backend: the exported float64 kernels dispatch at runtime to AVX2
// assembly (reduce_amd64.s / arith_amd64.s) when the CPU supports it, and fall
// back to the scalar multiple-accumulator bodies in scalar.go otherwise. No
// build tag is required — selection is purely at runtime, so one binary is
// correct on AVX2 and pre-AVX2 amd64 alike.
//
// Sum and DotProduct have no AVX2 variant: the Go compiler already
// auto-vectorizes their scalar float64 loops, so a hand-written kernel does not
// beat them (matching the turboslice project's own measured choice).

var hasAVX2 = cpu.X86.HasAVX2

// avx2MinLen is the smallest input the AVX2 reduction kernels are used for;
// below it the scalar path wins (no vector setup / horizontal-reduce overhead).
const avx2MinLen = 16

// AVX2 assembly kernels (reduce_amd64.s). Defined for len >= 1; the Go wrappers
// gate on hasAVX2 and length before calling.

//go:noescape
func minFloat64AVX2(vals []float64) float64

//go:noescape
func maxFloat64AVX2(vals []float64) float64

//go:noescape
func minMaxFloat64AVX2(vals []float64) (float64, float64)

//go:noescape
func addSlicesFloat64AVX2(dst, a, b []float64)

//go:noescape
func mulSlicesFloat64AVX2(dst, a, b []float64)

// SumFloat64 returns the sum of vals. Returns 0 for an empty slice.
func SumFloat64(vals []float64) float64 { return sumFloat64Scalar(vals) }

// MinFloat64 returns the minimum value in vals. Returns 0 for an empty slice.
func MinFloat64(vals []float64) float64 {
	if hasAVX2 && len(vals) >= avx2MinLen {
		return minFloat64AVX2(vals)
	}
	return minFloat64Scalar(vals)
}

// MaxFloat64 returns the maximum value in vals. Returns 0 for an empty slice.
func MaxFloat64(vals []float64) float64 {
	if hasAVX2 && len(vals) >= avx2MinLen {
		return maxFloat64AVX2(vals)
	}
	return maxFloat64Scalar(vals)
}

// MinMaxFloat64 returns both the minimum and maximum values in vals.
// Returns (0, 0) for an empty slice.
func MinMaxFloat64(vals []float64) (float64, float64) {
	if hasAVX2 && len(vals) >= avx2MinLen {
		return minMaxFloat64AVX2(vals)
	}
	return minMaxFloat64Scalar(vals)
}

// AddSlicesFloat64 returns a new slice where each element is a[i] + b[i].
// The result length is min(len(a), len(b)).
func AddSlicesFloat64(a, b []float64) []float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if hasAVX2 && n >= avx2MinLen {
		dst := make([]float64, n)
		addSlicesFloat64AVX2(dst, a[:n], b[:n])
		return dst
	}
	return addSlicesFloat64Scalar(a, b)
}

// MulSlicesFloat64 returns a new slice where each element is a[i] * b[i].
// The result length is min(len(a), len(b)).
func MulSlicesFloat64(a, b []float64) []float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if hasAVX2 && n >= avx2MinLen {
		dst := make([]float64, n)
		mulSlicesFloat64AVX2(dst, a[:n], b[:n])
		return dst
	}
	return mulSlicesFloat64Scalar(a, b)
}

// DotProductFloat64 returns the dot product of a and b.
// Only elements up to min(len(a), len(b)) are used.
func DotProductFloat64(a, b []float64) float64 { return dotProductFloat64Scalar(a, b) }
