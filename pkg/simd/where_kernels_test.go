package simd

import (
	"math"
	"testing"
)

// refWhere is the obvious single-pass reference the kernels must match: seed
// min/max on the first passing (non-null) row, combine with plain < / >.
func refWhere(vals []float64, op Cmp, lit float64, nulls []bool) (sum, min, max float64, count int) {
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
		sum += v
		count++
	}
	return
}

func approxEq(a, b float64) bool { return math.Abs(a-b) <= 1e-6*(math.Abs(b)+1) }

func TestSumWhereFloat64AllComparisons(t *testing.T) {
	const n = 100_003 // not a multiple of 8: exercises the scalar tail
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = math.Sin(float64(i)) * 100
	}
	const lit = 12.5
	for _, op := range []Cmp{CmpGT, CmpGE, CmpLT, CmpLE, CmpEQ, CmpNE} {
		sum, count := SumWhereFloat64(vals, op, lit, nil)
		rs, _, _, rc := refWhere(vals, op, lit, nil)
		if count != rc {
			t.Fatalf("op %d: count %d, want %d", op, count, rc)
		}
		if !approxEq(sum, rs) {
			t.Fatalf("op %d: sum %v, want %v", op, sum, rs)
		}
	}
}

func TestMinMaxWhereFloat64(t *testing.T) {
	const n = 4097
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = math.Cos(float64(i)) * 50
	}
	for _, op := range []Cmp{CmpGT, CmpGE, CmpLT, CmpLE} {
		mn, mx, count := MinMaxWhereFloat64(vals, op, 3.0, nil)
		_, rmn, rmx, rc := refWhere(vals, op, 3.0, nil)
		if count != rc || mn != rmn || mx != rmx {
			t.Fatalf("op %d: (min,max,count)=(%v,%v,%d), want (%v,%v,%d)", op, mn, mx, count, rmn, rmx, rc)
		}
	}
}

// TestSumWhereMatchesBitmapPath pins the kernel to the existing
// CompareGTFloat64Bitmap + MaskedReduceFloat64 two-pass path (spec scenario).
func TestSumWhereMatchesBitmapPath(t *testing.T) {
	const n = 200_000
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = math.Sin(float64(i)*0.7) * 1000
	}
	const lit = 0.0
	sum, count := SumWhereFloat64(vals, CmpGT, lit, nil)
	bm := CompareGTFloat64Bitmap(vals, lit)
	bs, _, _, bc := MaskedReduceFloat64(vals, bm, nil)
	if count != bc {
		t.Fatalf("count %d, bitmap path %d", count, bc)
	}
	if !approxEq(sum, bs) {
		t.Fatalf("sum %v, bitmap path %v", sum, bs)
	}
}

// TestSumWhereNaNInfNoPoison proves the bit-mask select keeps non-passing
// NaN/-Inf from contaminating the sum (a multiply-by-mask form would yield NaN).
func TestSumWhereNaNInfNoPoison(t *testing.T) {
	vals := []float64{1.0, math.NaN(), 2.0, math.Inf(-1), 3.0, math.NaN(), 4.0}
	sum, count := SumWhereFloat64(vals, CmpGT, 0.0, nil) // NaN>0 and -Inf>0 are false
	if count != 4 {
		t.Fatalf("count %d, want 4 (1,2,3,4)", count)
	}
	if math.IsNaN(sum) || sum != 10.0 {
		t.Fatalf("sum %v, want 10 (non-passing NaN/-Inf excluded, no poison)", sum)
	}
}

// TestMinMaxWhereNaNStickyFromSeed: when the first passing row is NaN (only
// reachable via ne), min/max are NaN — matching MaskedReduceFloat64.
func TestMinMaxWhereNaNStickyFromSeed(t *testing.T) {
	vals := []float64{math.NaN(), 1.0, 2.0}
	mn, mx, count := MinMaxWhereFloat64(vals, CmpNE, math.Inf(1), nil) // all pass; first is NaN
	if count != 3 {
		t.Fatalf("count %d, want 3", count)
	}
	if !math.IsNaN(mn) || !math.IsNaN(mx) {
		t.Fatalf("min %v max %v, want NaN/NaN (sticky from seed)", mn, mx)
	}
}

func TestSumWhereNulls(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	nulls := make([]bool, len(vals))
	nulls[2], nulls[5], nulls[10] = true, true, true
	sum, count := SumWhereFloat64(vals, CmpGT, 0, nulls)
	rs, _, _, rc := refWhere(vals, CmpGT, 0, nulls)
	if count != rc || !approxEq(sum, rs) {
		t.Fatalf("(sum,count)=(%v,%d), want (%v,%d)", sum, count, rs, rc)
	}
}

func TestSumWhereEmptyCleanZero(t *testing.T) {
	cases := [][]float64{{1, 2, 3}, {math.NaN(), 1, 2}}
	for _, vals := range cases {
		sum, count := SumWhereFloat64(vals, CmpGT, 1e18, nil) // nothing passes
		if count != 0 {
			t.Fatalf("vals %v: count %d, want 0", vals, count)
		}
		if sum != 0.0 || math.IsNaN(sum) {
			t.Fatalf("vals %v: sum %v, want clean 0.0", vals, sum)
		}
	}
}

func TestSumWhereNoAlloc(t *testing.T) {
	vals := make([]float64, 4096)
	for i := range vals {
		vals[i] = float64(i) - 2048
	}
	if allocs := testing.AllocsPerRun(20, func() {
		_, _ = SumWhereFloat64(vals, CmpGT, 0, nil)
	}); allocs != 0 {
		t.Fatalf("SumWhereFloat64 allocated %v objs/op, want 0 (no bitmap)", allocs)
	}
}
