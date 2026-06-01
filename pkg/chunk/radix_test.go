package chunk

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

func TestOrderPreservingFloatTransform(t *testing.T) {
	vals := []float64{-1e308, -1, -0.5, math.Copysign(0, -1), 0.0, 0.5, 1, 1e308, math.Inf(1), math.Inf(-1)}
	sorted := append([]float64(nil), vals...)
	sort.Float64s(sorted)
	// The transform must be monotonic: a <= b  <=>  key(a) <= key(b).
	for i := 0; i < len(sorted); i++ {
		for j := i; j < len(sorted); j++ {
			ki := orderPreservingFloat(sorted[i])
			kj := orderPreservingFloat(sorted[j])
			if ki > kj {
				t.Errorf("transform not order-preserving: %v(%d) > %v(%d)", sorted[i], ki, sorted[j], kj)
			}
		}
	}
}

func TestArgsortFloat64MatchesStdSort(t *testing.T) {
	r := rand.New(rand.NewSource(99))
	n := 10000
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = r.NormFloat64() * 1e6
	}
	idx := ArgsortFloat64(vals)
	if len(idx) != n {
		t.Fatalf("len = %d", len(idx))
	}
	// Output values must be non-decreasing.
	for i := 1; i < n; i++ {
		if vals[idx[i-1]] > vals[idx[i]] {
			t.Fatalf("not sorted at %d: %v > %v", i, vals[idx[i-1]], vals[idx[i]])
		}
	}
	// Must be a permutation.
	seen := make([]bool, n)
	for _, k := range idx {
		if seen[k] {
			t.Fatalf("index %d duplicated", k)
		}
		seen[k] = true
	}
}

func TestArgsortInt64MatchesStdSort(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	n := 8000
	vals := make([]int64, n)
	for i := range vals {
		vals[i] = r.Int63n(1_000_000) - 500_000
	}
	idx := ArgsortInt64(vals)
	for i := 1; i < n; i++ {
		if vals[idx[i-1]] > vals[idx[i]] {
			t.Fatalf("not sorted at %d", i)
		}
	}
}

func TestArgsortStableForEqualKeys(t *testing.T) {
	// Equal keys must keep input order (LSD radix is stable). This lets the
	// frame sort produce a deterministic tie order.
	vals := []float64{2, 1, 2, 1, 2}
	idx := ArgsortFloat64(vals)
	// The two 1s are at original positions 1,3; they must appear in that order.
	var ones []int
	for _, k := range idx {
		if vals[k] == 1 {
			ones = append(ones, k)
		}
	}
	if len(ones) != 2 || ones[0] != 1 || ones[1] != 3 {
		t.Errorf("equal-key order not stable: %v", ones)
	}
}
