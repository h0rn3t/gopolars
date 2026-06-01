package simd

import "math/bits"

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
	for i := range n {
		out[i] = a[i] && b[i]
	}
	return out
}

// CompareGTFloat64Bitmap returns a Bitmap whose bit i is set iff vals[i] >
// threshold. It is the bitmap counterpart of CompareGTFloat64: one bit per row
// instead of one byte. Only matching bits are set, so the trailing bits of a
// partial last word stay zero (BitmapNew zeroes the buffer).
func CompareGTFloat64Bitmap(vals []float64, threshold float64) Bitmap {
	b := BitmapNew(len(vals))
	for i, v := range vals {
		if v > threshold {
			b[i>>6] |= 1 << (uint(i) & 63)
		}
	}
	return b
}

// CompareEQInt64Bitmap returns a Bitmap whose bit i is set iff vals[i] ==
// target. Bitmap counterpart of CompareEQInt64.
func CompareEQInt64Bitmap(vals []int64, target int64) Bitmap {
	b := BitmapNew(len(vals))
	for i, v := range vals {
		if v == target {
			b[i>>6] |= 1 << (uint(i) & 63)
		}
	}
	return b
}

// BitmapAnd returns the element-wise logical AND of bitmaps a and b over nRows
// rows, in a freshly allocated Bitmap of (nRows+63)/64 words. It is the bitmap
// replacement for AndMask: combining two predicate bitmaps is a word-at-a-time
// AND rather than a per-element bool loop. Trailing bits of a partial last word
// remain zero as long as both inputs keep them zero.
func BitmapAnd(a, b Bitmap, nRows int) Bitmap {
	words := (nRows + 63) / 64
	out := make(Bitmap, words)
	for i := range words {
		out[i] = a[i] & b[i]
	}
	return out
}

// MaskedReduceFloat64 reduces vals over the rows that survive a filter — those
// whose keep bit is set and whose value is non-null (nulls == nil || !nulls[i])
// — in a single pass, returning the sum, min, max, and the count of contributing
// rows. It is the kernel behind the fused filter+reduce path: it avoids building
// a surviving-index slice or a materialized filtered column.
//
// keep is a packed Bitmap. The reduction walks one word at a time, skipping
// zero words entirely and visiting only set bits via math/bits.TrailingZeros64 +
// x &= x-1 (Lemire 2018). Bits are visited in ascending index order (lowest set
// bit of the lowest word first), so the floating-point reduction order is
// identical to the previous row-major []bool loop. The partial last word is
// masked to len(vals) so stray high bits never index out of range.
//
// min and max are seeded on the first contributing row in ascending index order
// and updated with plain < / >, so NaN is sticky-from-seed and otherwise ignored
// — identical to the engine's scalar aggregation. When count == 0 the returned
// sum/min/max are 0 and callers should treat the reduction as empty (null).
// keep must address at least len(vals) rows; nulls is either nil or len(vals).
func MaskedReduceFloat64(vals []float64, keep Bitmap, nulls []bool) (sum, min, max float64, count int) {
	n := len(vals)
	nWords := (n + 63) / 64
	lastMask := ^uint64(0)
	if rem := n & 63; rem != 0 {
		lastMask = uint64(1)<<uint(rem) - 1
	}
	if nulls == nil {
		for wi := range nWords {
			w := keep[wi]
			if wi == nWords-1 {
				w &= lastMask
			}
			base := wi << 6
			for w != 0 {
				v := vals[base+bits.TrailingZeros64(w)]
				w &= w - 1
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
		}
		return sum, min, max, count
	}
	for wi := range nWords {
		w := keep[wi]
		if wi == nWords-1 {
			w &= lastMask
		}
		base := wi << 6
		for w != 0 {
			i := base + bits.TrailingZeros64(w)
			w &= w - 1
			if nulls[i] {
				continue
			}
			v := vals[i]
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
	}
	return sum, min, max, count
}

// CompressIndices converts a predicate Bitmap into the ascending indices of its
// set bits over nRows rows. Used to gather surviving rows after a Filter.
//
// It pre-sizes the output to the exact survivor count via BitmapPopcount (free
// from the bitmap), then fills it with a word-at-a-time walk: for each non-zero
// word, math/bits.TrailingZeros64 yields the next set bit and x &= x-1 clears
// it, so zero words are skipped entirely and only set bits are visited (≈2.6–3.4
// cycles/bit, Lemire 2018). A low-selectivity filter no longer over-allocates an
// N-sized buffer (a 1M-row bitmap with zero survivors allocates none) and a
// dense result is sized exactly with no growth reallocations. The partial last
// word is masked to nRows so stray high bits never produce out-of-range indices.
func CompressIndices(mask Bitmap, nRows int) []int {
	count := BitmapPopcount(mask, nRows)
	out := make([]int, count)
	if count == 0 {
		return out
	}
	j := 0
	fullWords := nRows >> 6
	for wi := range fullWords {
		w := mask[wi]
		base := wi << 6
		for w != 0 {
			out[j] = base + bits.TrailingZeros64(w)
			j++
			w &= w - 1
		}
	}
	if rem := nRows & 63; rem != 0 {
		w := mask[fullWords] & (uint64(1)<<uint(rem) - 1)
		base := fullWords << 6
		for w != 0 {
			out[j] = base + bits.TrailingZeros64(w)
			j++
			w &= w - 1
		}
	}
	return out
}
