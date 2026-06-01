package chunk

import "math"

// LSD radix argsort for numeric columns. These return an index permutation that
// orders the input ascending, in O(n) time (8 passes over 8-bit digits) rather
// than the O(n log n) of a comparison sort. They use the Lemire/Herf
// order-preserving key transform so unsigned-integer digit order equals the
// signed/float numeric order.
//
// The decision (design D7) was to implement this inline rather than vendor a
// library: the kernel is small, has no external dependency or supply-chain
// risk, and a sequential LSD radix pass is already memory-bandwidth bound, so a
// parallel merge would add complexity for limited extra gain at these sizes.

// orderPreservingFloat maps a float64 to a uint64 whose unsigned order matches
// ascending float order: flip all bits for negatives (sign set), flip just the
// sign bit for non-negatives. Callers must exclude NaN beforehand.
func orderPreservingFloat(v float64) uint64 {
	bits := math.Float64bits(v)
	mask := uint64(int64(bits)>>63) | (uint64(1) << 63)
	return bits ^ mask
}

// orderPreservingInt maps an int64 to a uint64 whose unsigned order matches
// ascending signed order (flip the sign bit).
func orderPreservingInt(v int64) uint64 {
	return uint64(v) ^ (uint64(1) << 63)
}

// ArgsortFloat64 returns indices that sort vals ascending. Behavior is undefined
// for NaN inputs (callers gate the radix path on a NaN-free, null-free column).
func ArgsortFloat64(vals []float64) []int {
	keys := make([]uint64, len(vals))
	for i, v := range vals {
		keys[i] = orderPreservingFloat(v)
	}
	return radixArgsort(keys)
}

// ArgsortInt64 returns indices that sort vals ascending.
func ArgsortInt64(vals []int64) []int {
	keys := make([]uint64, len(vals))
	for i, v := range vals {
		keys[i] = orderPreservingInt(v)
	}
	return radixArgsort(keys)
}

// radixArgsort sorts indices [0,n) by their uint64 key via stable LSD radix
// (eight 8-bit passes).
func radixArgsort(keys []uint64) []int {
	n := len(keys)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	tmp := make([]int, n)
	var count [256]int
	for shift := uint(0); shift < 64; shift += 8 {
		for i := range count {
			count[i] = 0
		}
		for i := 0; i < n; i++ {
			count[(keys[idx[i]]>>shift)&0xff]++
		}
		sum := 0
		for i := 0; i < 256; i++ {
			c := count[i]
			count[i] = sum
			sum += c
		}
		for i := 0; i < n; i++ {
			d := (keys[idx[i]] >> shift) & 0xff
			tmp[count[d]] = idx[i]
			count[d]++
		}
		idx, tmp = tmp, idx
	}
	return idx
}
