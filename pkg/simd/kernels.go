package simd

import (
	"math"
	"math/bits"
)

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
//
// Each 64-bit word is accumulated in a register and stored once. The inner
// compare is written as "set a 0/1 bit, then shift-or unconditionally" rather
// than "if cond { word |= mask }" so the compiler emits a branchless SETcc
// instead of a data-dependent branch: on ~50%-selective input the branchy form
// mispredicts every other row and also does a dependent memory RMW per element,
// which measured ~10x slower than this form (~2.2 GB/s vs ~23 GB/s, the latter
// at memory bandwidth — so an AVX2 kernel would add nothing here).
func CompareGTFloat64Bitmap(vals []float64, threshold float64) Bitmap {
	b := BitmapNew(len(vals))
	n := len(vals)
	for w := range b {
		base := w << 6
		hi := min(base+64, n)
		var word uint64
		for i := base; i < hi; i++ {
			var bit uint64
			if vals[i] > threshold {
				bit = 1
			}
			word |= bit << uint(i-base)
		}
		b[w] = word
	}
	return b
}

// CompareEQInt64Bitmap returns a Bitmap whose bit i is set iff vals[i] ==
// target. Bitmap counterpart of CompareEQInt64. Built word-at-a-time with a
// branchless inner compare for the same reason as CompareGTFloat64Bitmap.
func CompareEQInt64Bitmap(vals []int64, target int64) Bitmap {
	b := BitmapNew(len(vals))
	n := len(vals)
	for w := range b {
		base := w << 6
		hi := min(base+64, n)
		var word uint64
		for i := base; i < hi; i++ {
			var bit uint64
			if vals[i] == target {
				bit = 1
			}
			word |= bit << uint(i-base)
		}
		b[w] = word
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

// Cmp identifies a `column <cmp> literal` comparison for the single-pass
// filter-reduce kernels (SumWhereFloat64 / MinMaxWhereFloat64).
type Cmp uint8

const (
	CmpGT Cmp = iota // v > lit
	CmpGE            // v >= lit
	CmpLT            // v < lit
	CmpLE            // v <= lit
	CmpEQ            // v == lit
	CmpNE            // v != lit
)

// whereKeep reports whether v passes `v <cmp> lit` using Go's native float
// comparison. For gt/ge/lt/le/eq a NaN operand never passes (NaN compares
// false); ne returns true for a NaN v (NaN != lit), matching IEEE.
func whereKeep(v, lit float64, op Cmp) bool {
	switch op {
	case CmpGT:
		return v > lit
	case CmpGE:
		return v >= lit
	case CmpLT:
		return v < lit
	case CmpLE:
		return v <= lit
	case CmpEQ:
		return v == lit
	default: // CmpNE
		return v != lit
	}
}

func boolToU64(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

// whereMask returns all-ones (passing) or zero (not passing) so a caller can
// select a value branchlessly via bit-AND.
func whereMask(v, lit float64, op Cmp) uint64 {
	return -boolToU64(whereKeep(v, lit, op))
}

// SumWhereFloat64 reduces vals over the rows passing `vals[i] <cmp> lit` in a
// single pass, returning the sum of the passing values and their count. It
// builds no predicate bitmap and makes no second pass.
//
// The sum is accumulated branchlessly via a bit-mask select —
// Float64frombits(Float64bits(v) & mask) yields exactly 0.0 for a non-passing
// row, never NaN — so a non-passing NaN/±Inf value cannot poison the sum (a
// `v * b2f(pass)` form would compute 0*NaN = NaN). Eight independent
// accumulators over an unrolled loop (mirroring SumFloat64) keep several FADDs
// in flight; a scalar tail handles the remainder. nulls, when non-nil, excludes
// nulls[i] rows (also branchlessly).
//
// Under GOEXPERIMENT=simd a null-free input long enough to fill the vector
// accumulators takes the portable hardware path in vec_simd.go instead
// (compare -> mask -> masked add). Both forms reorder the additions, so they
// agree with each other and with a row-major scalar sum only up to
// reduction-order rounding.
func SumWhereFloat64(vals []float64, op Cmp, lit float64, nulls []bool) (sum float64, count int) {
	if nulls == nil {
		if s, c, ok := sumWhereFloat64Vec(vals, op, lit); ok {
			return s, c
		}
	}
	if nulls != nil {
		for i, v := range vals {
			m := whereMask(v, lit, op) &^ -boolToU64(nulls[i])
			sum += math.Float64frombits(math.Float64bits(v) & m)
			count += int(m & 1)
		}
		return sum, count
	}
	var s0, s1, s2, s3, s4, s5, s6, s7 float64
	var c0, c1, c2, c3, c4, c5, c6, c7 uint64
	rest := vals
	for len(rest) >= 8 {
		m0 := whereMask(rest[0], lit, op)
		m1 := whereMask(rest[1], lit, op)
		m2 := whereMask(rest[2], lit, op)
		m3 := whereMask(rest[3], lit, op)
		m4 := whereMask(rest[4], lit, op)
		m5 := whereMask(rest[5], lit, op)
		m6 := whereMask(rest[6], lit, op)
		m7 := whereMask(rest[7], lit, op)
		s0 += math.Float64frombits(math.Float64bits(rest[0]) & m0)
		s1 += math.Float64frombits(math.Float64bits(rest[1]) & m1)
		s2 += math.Float64frombits(math.Float64bits(rest[2]) & m2)
		s3 += math.Float64frombits(math.Float64bits(rest[3]) & m3)
		s4 += math.Float64frombits(math.Float64bits(rest[4]) & m4)
		s5 += math.Float64frombits(math.Float64bits(rest[5]) & m5)
		s6 += math.Float64frombits(math.Float64bits(rest[6]) & m6)
		s7 += math.Float64frombits(math.Float64bits(rest[7]) & m7)
		c0 += m0 & 1
		c1 += m1 & 1
		c2 += m2 & 1
		c3 += m3 & 1
		c4 += m4 & 1
		c5 += m5 & 1
		c6 += m6 & 1
		c7 += m7 & 1
		rest = rest[8:]
	}
	sum = ((s0 + s1) + (s2 + s3)) + ((s4 + s5) + (s6 + s7))
	count = int(((c0 + c1) + (c2 + c3)) + ((c4 + c5) + (c6 + c7)))
	for _, v := range rest {
		m := whereMask(v, lit, op)
		sum += math.Float64frombits(math.Float64bits(v) & m)
		count += int(m & 1)
	}
	return sum, count
}

// MinMaxWhereFloat64 reduces vals over the rows passing `vals[i] <cmp> lit` in a
// single pass, returning the min, max, and count of passing (non-null) values.
// It builds no predicate bitmap. min and max are seeded on the first passing row
// in ascending index order and combined with plain < / >, so NaN handling is
// identical to MaskedReduceFloat64 (sticky-from-seed). When count == 0 the
// returned min/max are 0 and callers treat the reduction as empty.
//
// Under GOEXPERIMENT=simd a null-free input long enough to fill the vector
// accumulators takes the portable hardware path in vec_simd.go instead. min and
// max are exact in both forms — only the sum kernels reorder.
func MinMaxWhereFloat64(vals []float64, op Cmp, lit float64, nulls []bool) (min, max float64, count int) {
	if nulls == nil {
		if mn, mx, c, ok := minMaxWhereFloat64Vec(vals, op, lit); ok {
			return mn, mx, c
		}
	}
	for i, v := range vals {
		if !whereKeep(v, lit, op) {
			continue
		}
		if nulls != nil && nulls[i] {
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
		count++
	}
	return min, max, count
}
