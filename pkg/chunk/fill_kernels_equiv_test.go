package chunk

import (
	"math"
	"testing"
)

// These tests pin the parallel/no-op fill & drop kernels to a straightforward
// sequential reference across empty, short, threshold-boundary, all-NaN,
// all-null, mixed, and no-op inputs. Sizes straddle parallelFillThreshold so
// both the inline and the sharded execution paths are exercised.

// makeF64 builds a float64 column. nullEvery>0 marks every nth row null;
// nanEvery>0 sets every nth row's payload to NaN (independent of validity).
func makeF64(n, nullEvery, nanEvery int) *Column {
	vals := make([]float64, n)
	var nulls []bool
	if nullEvery > 0 {
		nulls = make([]bool, n)
	}
	for i := 0; i < n; i++ {
		vals[i] = float64(i)*0.5 - 3
		if nanEvery > 0 && i%nanEvery == 0 {
			vals[i] = math.NaN()
		}
		if nullEvery > 0 && i%nullEvery == 0 {
			nulls[i] = true
		}
	}
	return NewFloat64(vals, nulls)
}

func nullsOf(c *Column) []bool {
	out := make([]bool, c.Len())
	for i := 0; i < c.Len(); i++ {
		out[i] = c.IsNull(i)
	}
	return out
}

func f64Equal(a, b float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	return a == b
}

func assertColEqual(t *testing.T, got *Column, wantVals []float64, wantNulls []bool) {
	t.Helper()
	gv, _ := got.Float64s()
	if len(gv) != len(wantVals) {
		t.Fatalf("length: got %d want %d", len(gv), len(wantVals))
	}
	gn := nullsOf(got)
	for i := range wantVals {
		if gn[i] != wantNulls[i] {
			t.Fatalf("row %d null: got %v want %v", i, gn[i], wantNulls[i])
		}
		// A null row's payload is unobservable; only compare non-null values.
		if !wantNulls[i] && !f64Equal(gv[i], wantVals[i]) {
			t.Fatalf("row %d value: got %v want %v", i, gv[i], wantVals[i])
		}
	}
}

// refFillNull / refFillNaN / refDropNaN are the obvious sequential semantics.
func refFillNull(c *Column, fill float64) ([]float64, []bool) {
	n := c.Len()
	vals, _ := c.Float64s()
	out := make([]float64, n)
	nulls := make([]bool, n)
	for i := 0; i < n; i++ {
		if c.IsNull(i) {
			out[i] = fill
		} else {
			out[i] = vals[i]
		}
	}
	return out, nulls // every row non-null after fill_null
}

func refFillNaN(c *Column, fill float64) ([]float64, []bool) {
	n := c.Len()
	vals, _ := c.Float64s()
	out := make([]float64, n)
	nulls := make([]bool, n)
	for i := 0; i < n; i++ {
		nulls[i] = c.IsNull(i)
		v := vals[i]
		if !nulls[i] && math.IsNaN(v) {
			v = fill
		}
		out[i] = v
	}
	return out, nulls
}

func refDropNaN(c *Column) ([]float64, []bool) {
	n := c.Len()
	vals, _ := c.Float64s()
	var outV []float64
	var outN []bool
	for i := 0; i < n; i++ {
		isNull := c.IsNull(i)
		if !isNull && math.IsNaN(vals[i]) {
			continue
		}
		outV = append(outV, vals[i])
		outN = append(outN, isNull)
	}
	return outV, outN
}

var equivSizes = []int{0, 1, 7, parallelFillThreshold - 1, parallelFillThreshold, parallelFillThreshold + 1, 50_000}

func TestFillNullFloat64Equivalence(t *testing.T) {
	for _, n := range equivSizes {
		for _, nullEvery := range []int{0, 1, 7} { // 0=none, 1=all, 7=mixed
			c := makeF64(n, nullEvery, 11)
			got, ok := c.FillNullFloat64(-1)
			if !ok {
				t.Fatalf("FillNullFloat64 not supported")
			}
			wv, wn := refFillNull(c, -1)
			assertColEqual(t, got, wv, wn)
		}
	}
}

func TestFillNaNFloat64Equivalence(t *testing.T) {
	for _, n := range equivSizes {
		for _, nullEvery := range []int{0, 7} {
			for _, nanEvery := range []int{0, 1, 5} {
				c := makeF64(n, nullEvery, nanEvery)
				got, ok := c.FillNaNFloat64(-2)
				if !ok {
					t.Fatalf("FillNaNFloat64 not supported")
				}
				wv, wn := refFillNaN(c, -2)
				assertColEqual(t, got, wv, wn)
			}
		}
	}
}

func TestDropNaNFloat64Equivalence(t *testing.T) {
	for _, n := range equivSizes {
		for _, nullEvery := range []int{0, 7} {
			for _, nanEvery := range []int{0, 1, 5} {
				c := makeF64(n, nullEvery, nanEvery)
				got, ok := c.DropNaNFloat64()
				if !ok {
					t.Fatalf("DropNaNFloat64 not supported")
				}
				wv, wn := refDropNaN(c)
				assertColEqual(t, got, wv, wn)
			}
		}
	}
}

// TestFillNoOpSharesBuffer verifies the no-op fast paths reuse the input value
// buffer rather than allocating and copying a fresh one.
func TestFillNoOpSharesBuffer(t *testing.T) {
	const n = 50_000
	// fill_null on a null-free column is a no-op.
	c := makeF64(n, 0, 0)
	in, _ := c.Float64s()
	got, _ := c.FillNullFloat64(0)
	gv, _ := got.Float64s()
	if &gv[0] != &in[0] {
		t.Fatalf("FillNullFloat64 no-op did not share the input buffer")
	}
	// fill_nan on a NaN-free column is a no-op.
	c2 := makeF64(n, 7, 0)
	in2, _ := c2.Float64s()
	got2, _ := c2.FillNaNFloat64(0)
	gv2, _ := got2.Float64s()
	if &gv2[0] != &in2[0] {
		t.Fatalf("FillNaNFloat64 no-op did not share the input buffer")
	}
	// drop_nans on a NaN-free column is a no-op.
	got3, _ := c2.DropNaNFloat64()
	gv3, _ := got3.Float64s()
	if &gv3[0] != &in2[0] {
		t.Fatalf("DropNaNFloat64 no-op did not share the input buffer")
	}
}
