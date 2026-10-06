package simd

// Scalar float64 reduction and element-wise kernels. These have no build tag.
// Sum and the element-wise kernels are the implementation on every
// architecture; the Min/Max reductions are the always-correct reference that
// both the non-amd64 build (simd_generic.go) and the amd64 non-AVX2 fallback
// (kernels_amd64.go) call, and that the AVX2 equivalence test pins the
// assembly against.
//
// The reductions use multiple independent accumulators over an unrolled loop so
// a superscalar core keeps several FADD/FCMP in flight (instruction-level
// parallelism) without a single-accumulator dependency chain. Min/Max are
// order-independent: each accumulator is seeded from vals[0], so a NaN seed is
// sticky (NaN compares false, so a later NaN is ignored) — identical to a strict
// scalar loop. Sum reorders additions, so it differs from a strict left-to-right
// sum only by floating-point reduction-order rounding.

// SumFloat64 returns the sum of vals, using eight independent accumulators.
// Returns 0 for an empty slice.
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

// minFloat64Scalar returns the minimum of vals using four independent
// accumulators. Returns 0 for an empty slice.
func minFloat64Scalar(vals []float64) float64 {
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

// maxFloat64Scalar returns the maximum of vals using four independent
// accumulators. Returns 0 for an empty slice.
func maxFloat64Scalar(vals []float64) float64 {
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

// minMaxFloat64Scalar returns both the minimum and maximum of vals using four
// independent accumulators per reduction in a single unrolled pass. Returns
// (0, 0) for an empty slice.
func minMaxFloat64Scalar(vals []float64) (float64, float64) {
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

// AddSlicesFloat64 returns a new slice where each element is a[i] + b[i].
// The result length is min(len(a), len(b)).
func AddSlicesFloat64(a, b []float64) []float64 {
	n := min(len(a), len(b))
	result := make([]float64, n)
	for i := range n {
		result[i] = a[i] + b[i]
	}
	return result
}

// MulSlicesFloat64 returns a new slice where each element is a[i] * b[i].
// The result length is min(len(a), len(b)).
func MulSlicesFloat64(a, b []float64) []float64 {
	n := min(len(a), len(b))
	result := make([]float64, n)
	for i := range n {
		result[i] = a[i] * b[i]
	}
	return result
}

// DotProductFloat64 returns the dot product of a and b.
// Only elements up to min(len(a), len(b)) are used.
func DotProductFloat64(a, b []float64) float64 {
	n := min(len(a), len(b))
	sum := 0.0
	for i := range n {
		sum += a[i] * b[i]
	}
	return sum
}
