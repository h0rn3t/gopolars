package simd

import (
	"math"
	"reflect"
	"testing"
)

func TestCompareGTFloat64(t *testing.T) {
	cases := []struct {
		name      string
		in        []float64
		threshold float64
		want      []bool
	}{
		{"empty", []float64{}, 1, []bool{}},
		{"mixed", []float64{1, 2, 3, 4}, 2, []bool{false, false, true, true}},
		{"all_false", []float64{1, 1, 1}, 5, []bool{false, false, false}},
		{"negatives", []float64{-3, -1, 0}, -2, []bool{false, true, true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CompareGTFloat64(c.in, c.threshold)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("CompareGTFloat64(%v, %v) = %v, want %v", c.in, c.threshold, got, c.want)
			}
		})
	}
}

func TestCompareEQInt64(t *testing.T) {
	cases := []struct {
		name   string
		in     []int64
		target int64
		want   []bool
	}{
		{"empty", []int64{}, 1, []bool{}},
		{"mixed", []int64{1, 2, 2, 3}, 2, []bool{false, true, true, false}},
		{"none", []int64{1, 3, 5}, 2, []bool{false, false, false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CompareEQInt64(c.in, c.target)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("CompareEQInt64(%v, %v) = %v, want %v", c.in, c.target, got, c.want)
			}
		})
	}
}

func TestAndMask(t *testing.T) {
	cases := []struct {
		name string
		a    []bool
		b    []bool
		want []bool
	}{
		{"empty", []bool{}, []bool{}, []bool{}},
		{"equal", []bool{true, true, false}, []bool{true, false, false}, []bool{true, false, false}},
		{"mismatch_len", []bool{true, true, true}, []bool{true, false}, []bool{true, false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AndMask(c.a, c.b)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("AndMask(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

// bitmapFromBools builds a Bitmap from a []bool, used by tests to express the
// legacy mask shape against the new Bitmap-input kernels.
func bitmapFromBools(mask []bool) Bitmap {
	b := BitmapNew(len(mask))
	for i, m := range mask {
		if m {
			BitmapSet(b, i)
		}
	}
	return b
}

// TestCompressIndicesAllocation pins the popcount-presize strategy: an all-zero
// bitmap must not retain a buffer proportional to N, and a dense bitmap must be
// sized to exactly its survivor count (no growth slack).
func TestCompressIndicesAllocation(t *testing.T) {
	const n = 10_000

	empty := BitmapNew(n)
	got := CompressIndices(empty, n)
	if len(got) != 0 {
		t.Fatalf("empty mask: len = %d, want 0", len(got))
	}
	if cap(got) != 0 {
		t.Fatalf("empty mask: cap = %d, want 0 (no oversized N-proportional buffer)", cap(got))
	}

	dense := BitmapNew(n)
	want := 0
	for i := range n {
		if i%3 == 0 {
			BitmapSet(dense, i)
			want++
		}
	}
	got = CompressIndices(dense, n)
	if len(got) != want {
		t.Fatalf("dense mask: len = %d, want %d", len(got), want)
	}
	if cap(got) != want {
		t.Fatalf("dense mask: cap = %d, want exactly %d (no growth reallocations)", cap(got), want)
	}
}

func BenchmarkCompressIndices(b *testing.B) {
	const n = 1_000_000
	profiles := []struct {
		name string
		fill func(i int) bool
	}{
		{"empty", func(int) bool { return false }},
		{"half", func(i int) bool { return i%2 == 0 }},
		{"full", func(int) bool { return true }},
	}
	for _, p := range profiles {
		mask := BitmapNew(n)
		for i := range n {
			if p.fill(i) {
				BitmapSet(mask, i)
			}
		}
		b.Run(p.name, func(b *testing.B) {
			b.ReportAllocs()
			var sink []int
			for b.Loop() {
				sink = CompressIndices(mask, n)
			}
			_ = sink
		})
	}
}

func TestCompressIndices(t *testing.T) {
	cases := []struct {
		name string
		mask []bool
		want []int
	}{
		{"empty", []bool{}, []int{}},
		{"simple", []bool{true, false, true}, []int{0, 2}},
		{"none", []bool{false, false}, []int{}},
		{"all", []bool{true, true}, []int{0, 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CompressIndices(bitmapFromBools(c.mask), len(c.mask))
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("CompressIndices(%v) = %v, want %v", c.mask, got, c.want)
			}
		})
	}
}

// TestCompressIndicesMatchesLegacy confirms the Bitmap word-walk produces the
// same ascending []int as a naive []bool scan, across selectivities and a
// row count that is not a multiple of 64 (partial last word).
func TestCompressIndicesMatchesLegacy(t *testing.T) {
	for _, n := range []int{0, 1, 63, 64, 65, 1000, 4097} {
		for _, step := range []int{1, 2, 3, 7} {
			mask := make([]bool, n)
			var want []int
			for i := range n {
				if i%step == 0 {
					mask[i] = true
					want = append(want, i)
				}
			}
			if want == nil {
				want = []int{}
			}
			got := CompressIndices(bitmapFromBools(mask), n)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("n=%d step=%d: got %v, want %v", n, step, got, want)
			}
		}
	}
}

// TestCompareGTFloat64BitmapMatchesLegacy pins the bitmap compare kernel to the
// legacy []bool kernel bit-for-bit, including NaN/Inf and a partial last word.
func TestCompareGTFloat64BitmapMatchesLegacy(t *testing.T) {
	const n = 1000
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = math.Sin(float64(i)) * 100
	}
	vals[10] = math.NaN()
	vals[20] = math.Inf(1)
	vals[30] = math.Inf(-1)

	for _, threshold := range []float64{-100, 0, 50, 100} {
		legacy := CompareGTFloat64(vals, threshold)
		bm := CompareGTFloat64Bitmap(vals, threshold)
		for i := range vals {
			if BitmapGet(bm, i) != legacy[i] {
				t.Fatalf("threshold %v idx %d: bitmap=%v legacy=%v", threshold, i, BitmapGet(bm, i), legacy[i])
			}
		}
		// Trailing bits of the partial last word must be zero.
		if rem := n & 63; rem != 0 {
			last := bm[len(bm)-1]
			if last>>uint(rem) != 0 {
				t.Fatalf("threshold %v: trailing bits of last word set: %#x", threshold, last)
			}
		}
	}
}

// TestCompareEQInt64BitmapMatchesLegacy pins the int64 == bitmap kernel to the
// legacy []bool kernel bit-for-bit.
func TestCompareEQInt64BitmapMatchesLegacy(t *testing.T) {
	const n = 257 // not a multiple of 64
	vals := make([]int64, n)
	for i := range vals {
		vals[i] = int64(i % 5)
	}
	for _, target := range []int64{0, 2, 4, 9} {
		legacy := CompareEQInt64(vals, target)
		bm := CompareEQInt64Bitmap(vals, target)
		for i := range vals {
			if BitmapGet(bm, i) != legacy[i] {
				t.Fatalf("target %d idx %d: bitmap=%v legacy=%v", target, i, BitmapGet(bm, i), legacy[i])
			}
		}
	}
}

// TestBitmapAnd checks element-wise AND correctness and that no trailing bits
// past nRows leak into the result.
func TestBitmapAnd(t *testing.T) {
	const n = 200
	a := BitmapNew(n)
	b := BitmapNew(n)
	for i := range n {
		if i%2 == 0 {
			BitmapSet(a, i)
		}
		if i%3 == 0 {
			BitmapSet(b, i)
		}
	}
	got := BitmapAnd(a, b, n)
	for i := range n {
		want := i%2 == 0 && i%3 == 0
		if BitmapGet(got, i) != want {
			t.Fatalf("idx %d: got %v, want %v", i, BitmapGet(got, i), want)
		}
	}
	if rem := n & 63; rem != 0 {
		last := got[len(got)-1]
		if last>>uint(rem) != 0 {
			t.Fatalf("trailing bits of last word set: %#x", last)
		}
	}
}
