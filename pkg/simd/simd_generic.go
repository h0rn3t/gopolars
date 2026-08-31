//go:build !amd64

package simd

// Non-amd64 build: the exported float64 kernels are the scalar
// multiple-accumulator implementations in scalar.go, unless GOEXPERIMENT=simd
// supplies the portable vector kernels in vec_simd.go — on this architecture
// there is no hand-written assembly, so the vector path is the primary one when
// it is available. amd64 has its own runtime-dispatched file (kernels_amd64.go)
// that prefers its AVX2 assembly.

// SumFloat64 returns the sum of vals. Returns 0 for an empty slice.
func SumFloat64(vals []float64) float64 { return sumFloat64Scalar(vals) }

// MinFloat64 returns the minimum value in vals. Returns 0 for an empty slice.
func MinFloat64(vals []float64) float64 {
	if m, ok := minFloat64Vec(vals); ok {
		return m
	}
	return minFloat64Scalar(vals)
}

// MaxFloat64 returns the maximum value in vals. Returns 0 for an empty slice.
func MaxFloat64(vals []float64) float64 {
	if m, ok := maxFloat64Vec(vals); ok {
		return m
	}
	return maxFloat64Scalar(vals)
}

// MinMaxFloat64 returns both the minimum and maximum values in vals.
// Returns (0, 0) for an empty slice.
func MinMaxFloat64(vals []float64) (float64, float64) {
	if mn, mx, ok := minMaxFloat64Vec(vals); ok {
		return mn, mx
	}
	return minMaxFloat64Scalar(vals)
}

// AddSlicesFloat64 returns a new slice where each element is a[i] + b[i].
// The result length is min(len(a), len(b)).
func AddSlicesFloat64(a, b []float64) []float64 { return addSlicesFloat64Scalar(a, b) }

// MulSlicesFloat64 returns a new slice where each element is a[i] * b[i].
// The result length is min(len(a), len(b)).
func MulSlicesFloat64(a, b []float64) []float64 { return mulSlicesFloat64Scalar(a, b) }

// DotProductFloat64 returns the dot product of a and b.
// Only elements up to min(len(a), len(b)) are used.
func DotProductFloat64(a, b []float64) float64 { return dotProductFloat64Scalar(a, b) }
