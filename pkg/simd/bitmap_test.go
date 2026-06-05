package simd

import "testing"

func TestBitmapNew(t *testing.T) {
	cases := []struct {
		n     int
		words int
	}{
		{0, 0},
		{1, 1},
		{64, 1},
		{65, 2},
		{100, 2},
		{128, 2},
		{129, 3},
	}
	for _, c := range cases {
		b := BitmapNew(c.n)
		if len(b) != c.words {
			t.Fatalf("BitmapNew(%d): len = %d, want %d", c.n, len(b), c.words)
		}
		for i, w := range b {
			if w != 0 {
				t.Fatalf("BitmapNew(%d): word %d = %#x, want 0", c.n, i, w)
			}
		}
	}
}

func TestBitmapSetGet(t *testing.T) {
	b := BitmapNew(130)
	set := []int{0, 2, 63, 64, 65, 129}
	for _, i := range set {
		BitmapSet(b, i)
	}
	want := map[int]bool{}
	for _, i := range set {
		want[i] = true
	}
	for i := range 130 {
		if got := BitmapGet(b, i); got != want[i] {
			t.Fatalf("BitmapGet(%d) = %v, want %v", i, got, want[i])
		}
	}
	// Word 0 should encode exactly bits 0, 2, 63.
	if b[0] != (1<<0)|(1<<2)|(1<<63) {
		t.Fatalf("word 0 = %#x, want %#x", b[0], uint64((1<<0)|(1<<2)|(1<<63)))
	}
}

func TestBitmapPopcount(t *testing.T) {
	cases := []struct {
		name string
		n    int
		set  []int
		want int
	}{
		{"empty", 100, nil, 0},
		{"sparse", 130, []int{0, 64, 129}, 3},
		{"partial_last_word", 70, []int{0, 1, 2, 3, 4, 5, 6, 65, 66, 67}, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := BitmapNew(c.n)
			for _, i := range c.set {
				BitmapSet(b, i)
			}
			if got := BitmapPopcount(b, c.n); got != c.want {
				t.Fatalf("BitmapPopcount = %d, want %d", got, c.want)
			}
		})
	}
}

// TestBitmapPopcountAllOnes confirms an all-set bitmap (including a partial last
// word) counts exactly nRows, i.e. stray high bits of the last word are masked.
func TestBitmapPopcountAllOnes(t *testing.T) {
	const n = 1000
	b := BitmapNew(n)
	for i := range b {
		b[i] = ^uint64(0) // set every bit, including past nRows in the last word
	}
	if got := BitmapPopcount(b, n); got != n {
		t.Fatalf("BitmapPopcount(all-ones, %d) = %d, want %d", n, got, n)
	}
}
