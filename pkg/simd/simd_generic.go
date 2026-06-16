//go:build !amd64

package simd

// Non-amd64 build: the exported float64 kernels are the scalar
// multiple-accumulator implementations in scalar.go. amd64 has its own
// runtime-dispatched file (kernels_amd64.go) that selects AVX2 or the same
// scalar bodies.

// SumFloat64 returns the sum of vals. Returns 0 for an empty slice.
func SumFloat64(vals []float64) float64 { return sumFloat64Scalar(vals) }

// MinFloat64 returns the minimum value in vals. Returns 0 for an empty slice.
func MinFloat64(vals []float64) float64 { return minFloat64Scalar(vals) }

// MaxFloat64 returns the maximum value in vals. Returns 0 for an empty slice.
func MaxFloat64(vals []float64) float64 { return maxFloat64Scalar(vals) }

// MinMaxFloat64 returns both the minimum and maximum values in vals.
// Returns (0, 0) for an empty slice.
func MinMaxFloat64(vals []float64) (float64, float64) { return minMaxFloat64Scalar(vals) }

// AddSlicesFloat64 returns a new slice where each element is a[i] + b[i].
// The result length is min(len(a), len(b)).
func AddSlicesFloat64(a, b []float64) []float64 { return addSlicesFloat64Scalar(a, b) }

// MulSlicesFloat64 returns a new slice where each element is a[i] * b[i].
// The result length is min(len(a), len(b)).
func MulSlicesFloat64(a, b []float64) []float64 { return mulSlicesFloat64Scalar(a, b) }

// DotProductFloat64 returns the dot product of a and b.
// Only elements up to min(len(a), len(b)) are used.
func DotProductFloat64(a, b []float64) float64 { return dotProductFloat64Scalar(a, b) }
