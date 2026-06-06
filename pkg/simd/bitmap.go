package simd

import (
	"math/bits"
)

// Bitmap is a packed predicate mask: bit i%64 of word i/64 (LSB = bit 0)
// encodes whether row i survives a filter predicate. It replaces the older
// []bool byte-mask — one bit per row instead of one byte — which cuts the mask
// bandwidth 8x and lets the reduce/compress kernels operate a word at a time
// with math/bits (OnesCount64, TrailingZeros64), each a single instruction on
// arm64 and amd64. Word length for N rows is (N+63)/64; the trailing bits of a
// partial last word (positions N%64..63) are kept zero by every producer so a
// plain per-word popcount over the whole slice equals the row popcount.
type Bitmap []uint64

// BitmapNew allocates a zeroed Bitmap large enough to address n rows, i.e.
// (n+63)/64 words. n == 0 yields a zero-length Bitmap.
func BitmapNew(n int) Bitmap {
	return make(Bitmap, (n+63)/64)
}

// BitmapSet sets the bit for row i. i must be in range for the Bitmap.
func BitmapSet(b Bitmap, i int) {
	b[i>>6] |= 1 << (uint(i) & 63)
}

// BitmapGet reports whether the bit for row i is set.
func BitmapGet(b Bitmap, i int) bool {
	return b[i>>6]&(1<<(uint(i)&63)) != 0
}

// BitmapPopcount returns the number of set bits in positions 0..nRows-1, using
// math/bits.OnesCount64 per word. The final partial word is masked to nRows so
// the count is correct even if a producer left stray high bits set.
func BitmapPopcount(b Bitmap, nRows int) int {
	if nRows <= 0 {
		return 0
	}
	fullWords := nRows >> 6
	count := 0
	for i := range fullWords {
		count += bits.OnesCount64(b[i])
	}
	if rem := nRows & 63; rem != 0 {
		mask := uint64(1)<<uint(rem) - 1
		count += bits.OnesCount64(b[fullWords] & mask)
	}
	return count
}
