package chunk

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

var (
	nanA = math.NaN()
	nanB = math.Float64frombits(0xfff8000000000001) // negative NaN, other payload
)

func TestFloat64Key(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b float64
		same bool
	}{
		{"signed zeros", 0, negZero(), true},
		{"nan payloads", nanA, nanB, true},
		{"equal values", 1.5, 1.5, true},
		{"opposite signs", 1, -1, false},
		{"zero and nan", 0, nanA, false},
		{"infinities", math.Inf(1), math.Inf(-1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ka, kb := float64Key(tt.a), float64Key(tt.b)
			if got := ka == kb; got != tt.same {
				t.Errorf("float64Key(%v) == float64Key(%v) = %t (%#x, %#x), want %t", tt.a, tt.b, got, ka, kb, tt.same)
			}
		})
	}
}

func TestCanonicalKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"negative zero", negZero(), "0"},
		{"positive zero", 0.0, "0"},
		{"nan", nanB, "NaN"},
		{"float", -2.5, "-2.5"},
		{"int", int64(-3), "-3"},
		{"string", "-0", "-0"},
		{"nil", nil, "<nil>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fmt.Sprintf("%v", CanonicalKey(tt.in)); got != tt.want {
				t.Errorf("CanonicalKey(%v) formats as %q, want %q", tt.in, got, tt.want)
			}
		})
	}
	if f, ok := CanonicalKey(negZero()).(float64); !ok || math.Signbit(f) {
		t.Errorf("CanonicalKey(-0.0) = %v, want float64 +0", CanonicalKey(negZero()))
	}
}

// signedZeroKeys repeats [0, -0, NaN, NaN', 1] to n rows: three distinct keys.
func signedZeroKeys(n int) []float64 {
	base := []float64{0, negZero(), nanA, nanB, 1}
	out := make([]float64, n)
	for i := range out {
		out[i] = base[i%len(base)]
	}
	return out
}

func TestGroupIDsSignedZeroAndNaN(t *testing.T) {
	t.Parallel()
	for _, n := range []int{5, 65536} {
		keys := NewFloat64(signedZeroKeys(n), nil)
		other := NewInt64(make([]int64, n), nil)
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			t.Parallel()
			if _, first := GroupIDs([]*Column{keys}, n); len(first) != 3 {
				t.Errorf("GroupIDs(float key, n=%d) groups = %d, want 3", n, len(first))
			}
			if _, first := GroupIDs([]*Column{keys, other}, n); len(first) != 3 {
				t.Errorf("GroupIDs(float and int keys, n=%d) groups = %d, want 3", n, len(first))
			}
			if _, groups := GroupIDsUnordered([]*Column{keys}, n); groups != 3 {
				t.Errorf("GroupIDsUnordered(float key, n=%d) groups = %d, want 3", n, groups)
			}
		})
	}
}

func TestJoinKeysSignedZero(t *testing.T) {
	t.Parallel()
	c := NewFloat64([]float64{0, negZero()}, nil)
	keyAt, ok := PackKeyFunc(c)
	if !ok {
		t.Fatal("PackKeyFunc(Float64) ok = false, want true")
	}
	if keyAt(0) != keyAt(1) {
		t.Errorf("PackKeyFunc: key(0.0) = %#x, key(-0.0) = %#x, want equal", keyAt(0), keyAt(1))
	}
	k0 := string(AppendRowKey(nil, []*Column{c}, 0))
	k1 := string(AppendRowKey(nil, []*Column{c}, 1))
	if k0 != k1 {
		t.Errorf("AppendRowKey: key(0.0) = %x, key(-0.0) = %x, want equal", k0, k1)
	}
}

// TestMarkSharedConcurrent runs MarkShared, IsShared and CloneIfShared on one
// column from several goroutines; run with -race.
func TestMarkSharedConcurrent(t *testing.T) {
	t.Parallel()
	c := NewInt64([]int64{1, 2, 3}, nil)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				c.MarkShared()
				_ = c.IsShared()
				_ = c.CloneIfShared()
			}
		})
	}
	wg.Wait()
	if !c.IsShared() {
		t.Error("IsShared() = false after concurrent MarkShared, want true")
	}
}
