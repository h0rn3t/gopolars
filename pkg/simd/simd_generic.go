//go:build !amd64 || !simd

package simd

// SumFloat64 returns the sum of vals using a scalar loop.
func SumFloat64(vals []float64) float64 {
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum
}

// MinFloat64 returns the minimum value in vals using a scalar loop.
// Returns 0 for an empty slice.
func MinFloat64(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	min := vals[0]
	for _, v := range vals[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// MaxFloat64 returns the maximum value in vals using a scalar loop.
// Returns 0 for an empty slice.
func MaxFloat64(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	max := vals[0]
	for _, v := range vals[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// MinMaxFloat64 returns both the minimum and maximum values in vals
// using a single scalar pass. Returns (0, 0) for an empty slice.
func MinMaxFloat64(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return 0, 0
	}
	min, max := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

// CountFloat64 returns the number of elements in vals.
func CountFloat64(vals []float64) int64 {
	return int64(len(vals))
}

// AddSlicesFloat64 returns a new slice where each element is a[i] + b[i].
// The result length is min(len(a), len(b)).
func AddSlicesFloat64(a, b []float64) []float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		result[i] = a[i] + b[i]
	}
	return result
}

// MulSlicesFloat64 returns a new slice where each element is a[i] * b[i].
// The result length is min(len(a), len(b)).
func MulSlicesFloat64(a, b []float64) []float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		result[i] = a[i] * b[i]
	}
	return result
}

// DotProductFloat64 returns the dot product of a and b using a scalar loop.
// Only elements up to min(len(a), len(b)) are used.
func DotProductFloat64(a, b []float64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += a[i] * b[i]
	}
	return sum
}
