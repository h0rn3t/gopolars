package simd

import (
	"math/bits"
	"sync"
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

// bitmapMaxPoolWords caps the size of buffers retained by the pool at 1<<20
// rows (16384 words = 128 KiB) — the fused-filter working set the proposal
// targets for 0 allocs/op on the hot path. Releasing a larger buffer drops it
// (GC-eligible) rather than pinning hundreds of KiB of RSS per buffer for the
// process lifetime (design D4). (The change tasks' "2048 words" figure was an
// arithmetic slip — 1M rows is 15625 words, not 2048 — so the binding
// BitmapAcquire(1_000_000) 0-alloc scenario fixes the real cap here.)
const bitmapMaxPoolWords = (1 << 20) / 64 // 16384 words, up to 1,048,576 rows

// bitmapPool vends reusable backing arrays for the fused filter hot path. It
// stores *Bitmap wrappers, never a bare slice: a *Bitmap fits directly in the
// sync.Pool interface word, so Get/Put move only a pointer and never box a slice
// header. Both BitmapAcquire and BitmapRelease do exactly one Get + one Put of a
// wrapper, so the wrapper count is conserved and steady-state reuse allocates
// nothing — the detach-on-acquire / attach-on-release pattern. (A naive
// Put(&local) instead escapes a fresh slice header to the heap every call, which
// is the 1-alloc/op trap this avoids.)
var bitmapPool = sync.Pool{
	New: func() any { return new(Bitmap) },
}

// BitmapAcquire returns a zeroed Bitmap of (n+63)/64 words. For n up to ~1M rows
// the backing array is drawn from a sync.Pool and reused across calls, so a
// warm pool produces 0 allocs/op; larger requests allocate fresh. The result
// MUST be returned with BitmapRelease once the caller is done with it.
//
// It detaches the buffer from its wrapper (setting the wrapper's slice to nil)
// before returning the wrapper to the pool, so the buffer it hands out is never
// simultaneously reachable through a pooled wrapper — no two callers can be
// handed the same backing array.
func BitmapAcquire(n int) Bitmap {
	words := (n + 63) / 64
	if words > bitmapMaxPoolWords {
		return make(Bitmap, words)
	}
	bp := bitmapPool.Get().(*Bitmap)
	b := *bp
	*bp = nil
	bitmapPool.Put(bp)
	if cap(b) < words {
		b = make(Bitmap, words, bitmapMaxPoolWords)
	} else {
		b = b[:words]
		clear(b)
	}
	return b
}

// BitmapRelease returns a Bitmap previously obtained from BitmapAcquire to the
// pool by attaching it to a recycled wrapper. Buffers whose capacity exceeds the
// pool cap (or are empty) are dropped so oversized allocations are not retained.
// The caller must not use b after releasing it.
func BitmapRelease(b Bitmap) {
	if cap(b) == 0 || cap(b) > bitmapMaxPoolWords {
		return
	}
	bp := bitmapPool.Get().(*Bitmap)
	*bp = b[:cap(b)]
	bitmapPool.Put(bp)
}
