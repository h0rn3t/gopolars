package simd

// This file holds column kernels used by the vectorized expression engine.
// They are defined without a build tag so the same implementation compiles in
// both the SIMD (amd64 && simd) and generic builds; correctness never depends on
// SIMD being enabled. On amd64 with GOEXPERIMENT=simd these are the designated
// entry points where SIMD acceleration can be slotted in without changing call
// sites in pkg/expr/evalbatch.

// CompareGTFloat64 returns a boolean mask where mask[i] == (vals[i] > threshold).
func CompareGTFloat64(vals []float64, threshold float64) []bool {
	out := make([]bool, len(vals))
	for i, v := range vals {
		out[i] = v > threshold
	}
	return out
}

// CompareEQInt64 returns a boolean mask where mask[i] == (vals[i] == target).
func CompareEQInt64(vals []int64, target int64) []bool {
	out := make([]bool, len(vals))
	for i, v := range vals {
		out[i] = v == target
	}
	return out
}

// AndMask returns the element-wise logical AND of a and b. The result length is
// min(len(a), len(b)).
func AndMask(a, b []bool) []bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	out := make([]bool, n)
	for i := 0; i < n; i++ {
		out[i] = a[i] && b[i]
	}
	return out
}

// CompressIndices converts a boolean mask into the indices of its true entries,
// in order. Used to gather surviving rows after a Filter.
func CompressIndices(mask []bool) []int {
	out := make([]int, 0, len(mask))
	for i, m := range mask {
		if m {
			out = append(out, i)
		}
	}
	return out
}
