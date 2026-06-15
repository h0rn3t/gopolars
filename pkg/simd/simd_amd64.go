//go:build amd64 && simd

package simd

import "github.com/atul-007/turboslice"

// SumFloat64 returns the sum of vals. For float64 turboslice does not
// currently accelerate Sum, so this delegates to the scalar typed helper.
func SumFloat64(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	return turboslice.SumFloat64(vals)
}

// MinFloat64 returns the minimum value in vals. Returns 0 for an empty slice.
func MinFloat64(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	return turboslice.MinFloat64(vals)
}

// MaxFloat64 returns the maximum value in vals. Returns 0 for an empty slice.
func MaxFloat64(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	return turboslice.MaxFloat64(vals)
}

// MinMaxFloat64 returns both the minimum and maximum values in vals.
// Returns (0, 0) for an empty slice.
func MinMaxFloat64(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return 0, 0
	}
	return turboslice.MinMax(vals)
}

// AddSlicesFloat64 returns a new slice where each element is a[i] + b[i].
// The result length is min(len(a), len(b)).
func AddSlicesFloat64(a, b []float64) []float64 {
	return turboslice.AddSlices(a, b)
}

// MulSlicesFloat64 returns a new slice where each element is a[i] * b[i].
// The result length is min(len(a), len(b)).
func MulSlicesFloat64(a, b []float64) []float64 {
	return turboslice.MulSlices(a, b)
}

// DotProductFloat64 returns the dot product of a and b.
// Only elements up to min(len(a), len(b)) are used.
func DotProductFloat64(a, b []float64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return 0
	}
	return turboslice.DotProductFloat64(a[:n], b[:n])
}
