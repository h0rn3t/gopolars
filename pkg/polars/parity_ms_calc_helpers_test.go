package polars

// Usage-driven parity tests for the gopolars public API.
//
// Fixtures and expected values in the parity_ms_calc_*_test.go files are derived from how the
// Python service ../ms-calculations uses the `polars` library (energy balance / profiling
// pipelines). Each test reproduces a real call site as a self-contained Go fixture and asserts
// gopolars produces the polars result. Where gopolars lacks a polars equivalent or diverges in
// semantics, the test reimplements the behavior on gopolars primitives — it never asserts a false pass.
//
// The shared helpers below match the existing pkg/polars test conventions (standard testing,
// helper constructors, flat assertions, no testify).

import (
	"math"
	"math/big"
	"strconv"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// mscEpsKWh mirrors VOLUME_PER_DAY_INVARIANT_EPS_KWH from
// ../ms-calculations/app/services/balance/volume_invariants.py:6.
const mscEpsKWh = 0.01

// mscFrame builds a DataFrame from named columns, following the existing newDFTestFrame style.
func mscFrame(t *testing.T, cols ...frame.SeriesInput) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: cols})
	if err != nil {
		t.Fatalf("mscFrame: %v", err)
	}
	return df
}

// mscCol is a small constructor for a single named column.
func mscCol(name string, values ...any) frame.SeriesInput {
	return frame.SeriesInput{Name: name, Values: values}
}

// mscApprox reports whether a and b are within eps of each other.
func mscApprox(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

// asFloat coerces the numeric/any values gopolars returns (int64, float64, etc.) to float64.
func asFloat(t *testing.T, v any) float64 {
	t.Helper()
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	case int:
		return float64(x)
	default:
		t.Fatalf("asFloat: unexpected type %T (%v)", v, v)
		return 0
	}
}

// roundHalfEvenN rounds x to n decimal places using round-half-to-even (banker's rounding) on the
// decimal value, mirroring round_bankers in ../ms-calculations/app/docs/banking_rounding2.py.
//
// It rounds the literal decimal the float represents (recovered via the shortest round-trippable
// decimal string) rather than the raw binary approximation, so exact midpoints such as 0.0115 are
// treated as true ties — matching the documented sample vectors.
func roundHalfEvenN(x float64, n int) float64 {
	r, ok := new(big.Rat).SetString(strconv.FormatFloat(x, 'g', -1, 64))
	if !ok {
		return x
	}
	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt(pow))
	rounded := ratRoundHalfEven(scaled)
	out, _ := new(big.Rat).SetFrac(rounded, pow).Float64()
	return out
}

// ratRoundHalfEven rounds a rational to the nearest integer, breaking ties to the even integer.
func ratRoundHalfEven(r *big.Rat) *big.Int {
	num := new(big.Int).Abs(r.Num())
	den := r.Denom()
	neg := r.Num().Sign() < 0
	q := new(big.Int)
	m := new(big.Int)
	q.QuoRem(num, den, m) // q,m >= 0
	twoM := new(big.Int).Lsh(m, 1)
	switch twoM.Cmp(den) {
	case 1: // remainder > half -> round away from zero
		q.Add(q, big.NewInt(1))
	case 0: // exact half -> ties to even
		if q.Bit(0) == 1 {
			q.Add(q, big.NewInt(1))
		}
	}
	if neg {
		q.Neg(q)
	}
	return q
}

// bankersResidueCarry applies round-half-to-even to 3 decimals with the sequential residue carry
// from round_for_nparray in ../ms-calculations/app/docs/banking_rounding2.py:43: each value is
// adjusted by the accumulated rounding residue before being rounded, conserving the running total.
func bankersResidueCarry(vals []float64) []float64 {
	out := make([]float64, len(vals))
	residue := 0.0
	for i, v := range vals {
		prime := v + residue
		r := roundHalfEvenN(prime, 3)
		residue = v - r + residue
		out[i] = r
	}
	return out
}

// sumFloats is a tiny reduction helper used by the rounding/precision tests.
func sumFloats(vals []float64) float64 {
	total := 0.0
	for _, v := range vals {
		total += v
	}
	return total
}
