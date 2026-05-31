//go:build !amd64 || !simd

package simd

// The reductions below use multiple independent accumulators over an unrolled
// loop so non-amd64 builds (notably arm64, where the amd64+simd turboslice path
// is excluded) do not run a single-accumulator dependency-chain loop. Breaking
// the chain lets the CPU keep several FADD/FCMP in flight (instruction-level
// parallelism) and gives the compiler an auto-vectorizable shape, hoisting
// bounds via reslicing. The unrolled remainder is handled by a scalar tail.
//
// Sum reorders additions across the accumulators, so its result differs from a
// strict left-to-right sum by floating-point rounding (within reduction-order
// tolerance). Min/Max are order-independent: each accumulator is seeded from
// vals[0] so NaN handling (NaN compares false, so a NaN seed is sticky and a
// later NaN is ignored) is identical to the original scalar loop.

// SumFloat64 returns the sum of vals using eight independent accumulators over
// an unrolled loop, with a scalar tail for the remainder.
func SumFloat64(vals []float64) float64 {
	var s0, s1, s2, s3, s4, s5, s6, s7 float64
	rest := vals
	for len(rest) >= 8 {
		s0 += rest[0]
		s1 += rest[1]
		s2 += rest[2]
		s3 += rest[3]
		s4 += rest[4]
		s5 += rest[5]
		s6 += rest[6]
		s7 += rest[7]
		rest = rest[8:]
	}
	sum := ((s0 + s1) + (s2 + s3)) + ((s4 + s5) + (s6 + s7))
	for _, v := range rest {
		sum += v
	}
	return sum
}

// MinFloat64 returns the minimum value in vals using four independent
// accumulators. Returns 0 for an empty slice.
func MinFloat64(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	m0, m1, m2, m3 := vals[0], vals[0], vals[0], vals[0]
	rest := vals
	for len(rest) >= 4 {
		if rest[0] < m0 {
			m0 = rest[0]
		}
		if rest[1] < m1 {
			m1 = rest[1]
		}
		if rest[2] < m2 {
			m2 = rest[2]
		}
		if rest[3] < m3 {
			m3 = rest[3]
		}
		rest = rest[4:]
	}
	min := m0
	if m1 < min {
		min = m1
	}
	if m2 < min {
		min = m2
	}
	if m3 < min {
		min = m3
	}
	for _, v := range rest {
		if v < min {
			min = v
		}
	}
	return min
}

// MaxFloat64 returns the maximum value in vals using four independent
// accumulators. Returns 0 for an empty slice.
func MaxFloat64(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	m0, m1, m2, m3 := vals[0], vals[0], vals[0], vals[0]
	rest := vals
	for len(rest) >= 4 {
		if rest[0] > m0 {
			m0 = rest[0]
		}
		if rest[1] > m1 {
			m1 = rest[1]
		}
		if rest[2] > m2 {
			m2 = rest[2]
		}
		if rest[3] > m3 {
			m3 = rest[3]
		}
		rest = rest[4:]
	}
	max := m0
	if m1 > max {
		max = m1
	}
	if m2 > max {
		max = m2
	}
	if m3 > max {
		max = m3
	}
	for _, v := range rest {
		if v > max {
			max = v
		}
	}
	return max
}

// MinMaxFloat64 returns both the minimum and maximum values in vals using four
// independent accumulators per reduction in a single unrolled pass. Returns
// (0, 0) for an empty slice.
func MinMaxFloat64(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return 0, 0
	}
	mn0, mn1, mn2, mn3 := vals[0], vals[0], vals[0], vals[0]
	mx0, mx1, mx2, mx3 := vals[0], vals[0], vals[0], vals[0]
	rest := vals
	for len(rest) >= 4 {
		if rest[0] < mn0 {
			mn0 = rest[0]
		} else if rest[0] > mx0 {
			mx0 = rest[0]
		}
		if rest[1] < mn1 {
			mn1 = rest[1]
		} else if rest[1] > mx1 {
			mx1 = rest[1]
		}
		if rest[2] < mn2 {
			mn2 = rest[2]
		} else if rest[2] > mx2 {
			mx2 = rest[2]
		}
		if rest[3] < mn3 {
			mn3 = rest[3]
		} else if rest[3] > mx3 {
			mx3 = rest[3]
		}
		rest = rest[4:]
	}
	min, max := mn0, mx0
	if mn1 < min {
		min = mn1
	}
	if mn2 < min {
		min = mn2
	}
	if mn3 < min {
		min = mn3
	}
	if mx1 > max {
		max = mx1
	}
	if mx2 > max {
		max = mx2
	}
	if mx3 > max {
		max = mx3
	}
	for _, v := range rest {
		if v < min {
			min = v
		} else if v > max {
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
