//go:build goexperiment.simd

package simd

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

// Equivalence tests for the portable vector kernels (vec_simd.go), in the shape
// of avx2_equiv_test.go: each kernel is pinned against a reference derived from
// the behavior it replaces — a strict row-major loop — not against the vector
// code itself.
//
// The references below are deliberately naive. Sums are compared with a relative
// tolerance because both the vector and the scalar kernels reorder additions;
// min, max and count are compared exactly (NaN-aware), because no kernel is
// permitted to change those.

// approxEqSum compares two reduction sums. It extends the package's approxEq
// with the non-finite cases these corpora reach: a corpus containing both
// infinities sums to NaN on every path, and a NaN among the passing rows makes
// the sum NaN by IEEE. Both sides agreeing on NaN or on the same infinity is a
// pass; approxEq alone reports false there because NaN compares false with
// everything, including itself.
func approxEqSum(got, want float64) bool {
	if exactOrBothNaN(got, want) {
		return true
	}
	if math.IsInf(got, 0) || math.IsInf(want, 0) {
		return false // one side infinite, the other not, or opposite signs
	}
	return approxEq(got, want)
}

func allCmps() []Cmp { return []Cmp{CmpGT, CmpGE, CmpLT, CmpLE, CmpEQ, CmpNE} }

func cmpName(op Cmp) string {
	return [...]string{"gt", "ge", "lt", "le", "eq", "ne"}[op]
}

// vecCorpus returns inputs long enough to reach the vector paths, spanning the
// value classes the kernels must not mishandle.
func vecCorpus() map[string][]float64 {
	mk := func(n int, f func(i int) float64) []float64 {
		s := make([]float64, n)
		for i := range s {
			s[i] = f(i)
		}
		return s
	}
	nan := math.NaN()
	return map[string][]float64{
		"ramp_128":      mk(128, func(i int) float64 { return float64(i) - 64 }),
		"unaligned_101": mk(101, func(i int) float64 { return math.Sin(float64(i)) * 100 }),
		"big_1009":      mk(1009, func(i int) float64 { return math.Sin(float64(i))*1e6 + float64(i%7)*0.1 }),
		"all_equal":     mk(64, func(int) float64 { return 7.5 }),
		"all_negative":  mk(64, func(i int) float64 { return -float64(i) - 1 }),
		"signed_zero":   mk(64, func(i int) float64 { return math.Copysign(0, float64(i%2)*2-1) }),
		"with_infs":     mk(64, func(i int) float64 { return []float64{math.Inf(1), math.Inf(-1), 1, -1}[i%4] }),
		"nan_first":     mk(64, func(i int) float64 { return map[bool]float64{true: nan, false: float64(i)}[i == 0] }),
		"nan_middle":    mk(64, func(i int) float64 { return map[bool]float64{true: nan, false: float64(i)}[i == 33] }),
		"nan_last":      mk(64, func(i int) float64 { return map[bool]float64{true: nan, false: float64(i)}[i == 63] }),
		"nan_every_5th": mk(128, func(i int) float64 { return map[bool]float64{true: nan, false: float64(i) - 64}[i%5 == 0] }),
		"all_nan":       mk(64, func(int) float64 { return nan }),
	}
}

func TestSumWhereVecMatchesReference(t *testing.T) {
	t.Parallel()
	for name, in := range vecCorpus() {
		for _, op := range allCmps() {
			for _, lit := range []float64{0, 7.5, -1e9} {
				t.Run(fmt.Sprintf("%s/%s/lit=%g", name, cmpName(op), lit), func(t *testing.T) {
					t.Parallel()
					gotSum, gotCount, ok := sumWhereFloat64Vec(in, op, lit)
					if !ok {
						t.Skip("input below the vector threshold")
					}
					wantSum, _, _, wantCount := refWhere(in, op, lit, nil)
					if gotCount != wantCount {
						t.Fatalf("count: vec=%d ref=%d", gotCount, wantCount)
					}
					if !approxEqSum(gotSum, wantSum) {
						t.Fatalf("sum: vec=%v ref=%v", gotSum, wantSum)
					}
				})
			}
		}
	}
}

func TestMinMaxWhereVecMatchesReference(t *testing.T) {
	t.Parallel()
	for name, in := range vecCorpus() {
		for _, op := range allCmps() {
			for _, lit := range []float64{0, 7.5, -1e9} {
				t.Run(fmt.Sprintf("%s/%s/lit=%g", name, cmpName(op), lit), func(t *testing.T) {
					t.Parallel()
					gotMin, gotMax, gotCount, ok := minMaxWhereFloat64Vec(in, op, lit)
					if !ok {
						t.Skip("input below the vector threshold")
					}
					_, wantMin, wantMax, wantCount := refWhere(in, op, lit, nil)
					if gotCount != wantCount {
						t.Fatalf("count: vec=%d ref=%d", gotCount, wantCount)
					}
					if !exactOrBothNaN(gotMin, wantMin) {
						t.Fatalf("min: vec=%v ref=%v", gotMin, wantMin)
					}
					if !exactOrBothNaN(gotMax, wantMax) {
						t.Fatalf("max: vec=%v ref=%v", gotMax, wantMax)
					}
				})
			}
		}
	}
}

func TestMinMaxVecMatchesScalar(t *testing.T) {
	t.Parallel()
	for name, in := range vecCorpus() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			gotMin, minOK := minFloat64Vec(in)
			gotMax, maxOK := maxFloat64Vec(in)
			gotBothMin, gotBothMax, bothOK := minMaxFloat64Vec(in)
			if !minOK || !maxOK || !bothOK {
				t.Skip("input below the vector threshold")
			}
			if want := minFloat64Scalar(in); !exactOrBothNaN(gotMin, want) {
				t.Fatalf("min: vec=%v scalar=%v", gotMin, want)
			}
			if want := maxFloat64Scalar(in); !exactOrBothNaN(gotMax, want) {
				t.Fatalf("max: vec=%v scalar=%v", gotMax, want)
			}
			wantMin, wantMax := minMaxFloat64Scalar(in)
			if !exactOrBothNaN(gotBothMin, wantMin) {
				t.Fatalf("minmax min: vec=%v scalar=%v", gotBothMin, wantMin)
			}
			if !exactOrBothNaN(gotBothMax, wantMax) {
				t.Fatalf("minmax max: vec=%v scalar=%v", gotBothMax, wantMax)
			}
		})
	}
}

// TestVecKernelsRandomizedNaN sweeps lengths across the vector thresholds with a
// dense NaN population, so a NaN lands in every lane position. This is the test
// that fails if min/max ever go back to Float64s.Min/Max: NEON FMIN propagates
// NaN, and which lane the NaN occupies then decides the result.
func TestVecKernelsRandomizedNaN(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(0x5EED, 0xF00D))
	nan := math.NaN()
	for range 500 {
		n := 1 + r.IntN(160)
		in := make([]float64, n)
		for i := range in {
			if r.IntN(6) == 0 {
				in[i] = nan
			} else {
				in[i] = r.Float64()*20 - 10
			}
		}
		if got, ok := minFloat64Vec(in); ok {
			if want := minFloat64Scalar(in); !exactOrBothNaN(got, want) {
				t.Fatalf("min n=%d: vec=%v scalar=%v in=%v", n, got, want, in)
			}
		}
		if got, ok := maxFloat64Vec(in); ok {
			if want := maxFloat64Scalar(in); !exactOrBothNaN(got, want) {
				t.Fatalf("max n=%d: vec=%v scalar=%v in=%v", n, got, want, in)
			}
		}
		if gotMin, gotMax, ok := minMaxFloat64Vec(in); ok {
			wantMin, wantMax := minMaxFloat64Scalar(in)
			if !exactOrBothNaN(gotMin, wantMin) || !exactOrBothNaN(gotMax, wantMax) {
				t.Fatalf("minmax n=%d: vec=(%v,%v) scalar=(%v,%v) in=%v", n, gotMin, gotMax, wantMin, wantMax, in)
			}
		}
		for _, op := range allCmps() {
			if gotSum, gotCount, ok := sumWhereFloat64Vec(in, op, 0); ok {
				wantSum, _, _, wantCount := refWhere(in, op, 0, nil)
				if gotCount != wantCount || !approxEqSum(gotSum, wantSum) {
					t.Fatalf("sumwhere %s n=%d: vec=(%v,%d) ref=(%v,%d) in=%v",
						cmpName(op), n, gotSum, gotCount, wantSum, wantCount, in)
				}
			}
			if gotMin, gotMax, gotCount, ok := minMaxWhereFloat64Vec(in, op, 0); ok {
				_, wantMin, wantMax, wantCount := refWhere(in, op, 0, nil)
				if gotCount != wantCount || !exactOrBothNaN(gotMin, wantMin) || !exactOrBothNaN(gotMax, wantMax) {
					t.Fatalf("minmaxwhere %s n=%d: vec=(%v,%v,%d) ref=(%v,%v,%d) in=%v",
						cmpName(op), n, gotMin, gotMax, gotCount, wantMin, wantMax, wantCount, in)
				}
			}
		}
	}
}

// TestVecKernelsAllocFree pins the stack-buffer design: the horizontal tail of
// each reduction stores into a [maxLanes]float64 array, which must not escape.
// A heap allocation here would show up as an allocation in the fused
// filter-aggregate path, which pkg/frame asserts is allocation-free.
func TestVecKernelsAllocFree(t *testing.T) {
	in := make([]float64, 4096)
	for i := range in {
		in[i] = float64(i%97) - 48
	}
	cases := map[string]func(){
		"minMaxWhere": func() { minMaxWhereFloat64Vec(in, CmpGT, 0) },
		"min":         func() { minFloat64Vec(in) },
		"max":         func() { maxFloat64Vec(in) },
		"minMax":      func() { minMaxFloat64Vec(in) },
	}
	for name, fn := range cases {
		if got := testing.AllocsPerRun(100, fn); got != 0 {
			t.Errorf("%s: got %v allocs/op, want 0", name, got)
		}
	}
}
