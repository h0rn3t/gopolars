package simd

import (
	"math"
	"testing"
)

// naive* are single-accumulator left-to-right reference reductions: the
// canonical scalar semantics the restructured generic kernels must agree with.
func naiveSum(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s
}

func naiveMin(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	m := v[0]
	for _, x := range v[1:] {
		if x < m {
			m = x
		}
	}
	return m
}

func naiveMax(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	m := v[0]
	for _, x := range v[1:] {
		if x > m {
			m = x
		}
	}
	return m
}

func exactOrBothNaN(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	return a == b
}

// TestReductionsEquivalence pins the restructured (multiple-accumulator) reduce
// kernels to the scalar reference. min/max are order-independent and must match
// exactly (NaN handling included); sum reorders additions and must match within
// a relative reduction-order tolerance. This is the cross-arch contract: every
// architecture's reduce kernels (the generic path here; turboslice on amd64)
// must satisfy it.
func TestReductionsEquivalence(t *testing.T) {
	// A large fractional input that actually exercises floating-point rounding.
	large := make([]float64, 12_345)
	for i := range large {
		large[i] = math.Sin(float64(i))*1e6 + float64(i%7)*0.1
	}

	cases := map[string][]float64{
		"empty":        {},
		"single":       {42.5},
		"unaligned_5":  {1.5, -2.25, 3.125, -4, 5.0625},
		"all_negative": {-1, -2.5, -3.75, -100, -0.5, -9, -8},
		"with_pos_inf": {1, 2, math.Inf(1), 3, 4},
		"with_neg_inf": {1, 2, math.Inf(-1), 3, 4},
		"nan_first":    {math.NaN(), 1, 2, 3, 4, 5},
		"nan_middle":   {1, 2, 3, math.NaN(), 4, 5, 6, 7, 8},
		"large_mixed":  large,
	}

	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			// Sum within reduction-order tolerance.
			gotSum := SumFloat64(in)
			wantSum := naiveSum(in)
			if math.IsNaN(wantSum) || math.IsInf(wantSum, 0) {
				if !exactOrBothNaN(gotSum, wantSum) && gotSum != wantSum {
					t.Fatalf("Sum: got %v, want %v (non-finite)", gotSum, wantSum)
				}
			} else if diff := math.Abs(gotSum - wantSum); diff > 1e-9*(math.Abs(wantSum)+1) {
				t.Fatalf("Sum: got %v, want %v (diff %g exceeds tolerance)", gotSum, wantSum, diff)
			}

			// Min/Max are order-independent: must match the reference exactly.
			if got, want := MinFloat64(in), naiveMin(in); !exactOrBothNaN(got, want) {
				t.Fatalf("Min: got %v, want %v", got, want)
			}
			if got, want := MaxFloat64(in), naiveMax(in); !exactOrBothNaN(got, want) {
				t.Fatalf("Max: got %v, want %v", got, want)
			}

			// MinMax must equal the separate Min and Max.
			gotMin, gotMax := MinMaxFloat64(in)
			if !exactOrBothNaN(gotMin, naiveMin(in)) {
				t.Fatalf("MinMax min: got %v, want %v", gotMin, naiveMin(in))
			}
			if !exactOrBothNaN(gotMax, naiveMax(in)) {
				t.Fatalf("MinMax max: got %v, want %v", gotMax, naiveMax(in))
			}
		})
	}
}

// TestCompareMaskBitExact pins the compare kernel: its boolean mask must equal a
// naive per-element comparison bit-for-bit (the cross-arch mask-equality
// contract). Comparisons carry no reduction-order ambiguity, so this is exact.
func TestCompareMaskBitExact(t *testing.T) {
	vals := make([]float64, 1000)
	for i := range vals {
		vals[i] = math.Sin(float64(i)) * 100
	}
	vals[10] = math.NaN()
	vals[20] = math.Inf(1)
	vals[30] = math.Inf(-1)

	for _, threshold := range []float64{-100, 0, 50, 100} {
		got := CompareGTFloat64(vals, threshold)
		for i, v := range vals {
			want := v > threshold
			if got[i] != want {
				t.Fatalf("threshold %v idx %d (v=%v): got %v, want %v", threshold, i, v, got[i], want)
			}
		}
	}
}
