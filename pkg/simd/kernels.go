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

// MaskedReduceFloat64 reduces vals over the rows that survive a filter — those
// where keep[i] is true and the value is non-null (nulls == nil || !nulls[i]) —
// in a single pass, returning the sum, min, max, and the count of contributing
// rows. It is the kernel behind the fused filter+reduce path: it avoids
// building a surviving-index slice or a materialized filtered column.
//
// min and max are seeded on the first contributing row in ascending index
// order and updated with plain < / >, so NaN is sticky-from-seed and otherwise
// ignored — identical to the engine's scalar aggregation. When count == 0 the
// returned sum/min/max are 0 and callers should treat the reduction as empty
// (null). keep must be at least len(vals); nulls is either nil or len(vals).
func MaskedReduceFloat64(vals []float64, keep []bool, nulls []bool) (sum, min, max float64, count int) {
	if nulls == nil {
		for i, v := range vals {
			if !keep[i] {
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
	for i, v := range vals {
		if !keep[i] || nulls[i] {
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

// CompressIndices converts a boolean mask into the indices of its true entries,
// in order. Used to gather surviving rows after a Filter.
//
// It uses a count-then-allocate strategy: a first popcount pass sizes the
// output to the exact number of survivors, then a second pass fills it. This
// allocates 8*popcount bytes instead of the old 8*len(mask): a low-selectivity
// filter (most rows dropped) no longer over-allocates an N-sized buffer
// (e.g. a 1M-row mask with zero survivors allocated 8MB; now it allocates none),
// and a dense result is sized exactly with no growth reallocations. Both passes
// are linear, cache-friendly reads of a []bool, cheap relative to the old alloc.
func CompressIndices(mask []bool) []int {
	count := 0
	for _, m := range mask {
		if m {
			count++
		}
	}
	out := make([]int, count)
	j := 0
	for i, m := range mask {
		if m {
			out[j] = i
			j++
		}
	}
	return out
}
