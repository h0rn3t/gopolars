package chunk

import (
	"fmt"
	"testing"
)

// SliceParallel must be byte-identical to Slice for any index set; sharding only
// changes which core fills which output span. These cases cover every typed
// dtype, with and without nulls, at index counts straddling
// parallelGatherThreshold so both the inline and the sharded paths run, plus the
// empty and out-of-order/repeated-index cases.

func sampleColumn(dtype string, n int, nullEvery int) *Column {
	var nulls []bool
	if nullEvery > 0 {
		nulls = make([]bool, n)
		for i := range nulls {
			nulls[i] = i%nullEvery == 0
		}
	}
	switch dtype {
	case "int64":
		v := make([]int64, n)
		for i := range v {
			v[i] = int64(i) * 7
		}
		return NewInt64(v, nulls)
	case "float64":
		v := make([]float64, n)
		for i := range v {
			v[i] = float64(i)*0.25 - 1
		}
		return NewFloat64(v, nulls)
	case "string":
		v := make([]string, n)
		for i := range v {
			v[i] = fmt.Sprintf("s%d", i)
		}
		return NewString(v, nulls)
	case "bool":
		v := make([]bool, n)
		for i := range v {
			v[i] = i%3 == 0
		}
		return NewBool(v, nulls)
	default:
		panic("unknown dtype")
	}
}

func TestSliceParallelEqualsSlice(t *testing.T) {
	const n = 70_000 // > parallelGatherThreshold so the sharded path engages
	dtypes := []string{"int64", "float64", "string", "bool"}
	indexSets := map[string][]int{
		"empty":           {},
		"below-threshold": rangeIdx(0, 1000),       // serial fallback inside SliceParallel
		"above-threshold": rangeIdx(0, 50_000),     // sharded path
		"strided":         stridedIdx(n, 3),        // ~23k, exercises gaps
		"reversed":        reversedIdx(n, 40_000),  // out-of-order sources
		"repeated":        repeatedIdx(40, 40_000), // duplicate sources
	}
	for _, dt := range dtypes {
		for _, nullEvery := range []int{0, 5} {
			col := sampleColumn(dt, n, nullEvery)
			for name, idx := range indexSets {
				t.Run(fmt.Sprintf("%s/null=%d/%s", dt, nullEvery, name), func(t *testing.T) {
					want := col.Slice(idx)
					got := col.SliceParallel(idx, 4)
					assertSameColumn(t, want, got)
				})
			}
		}
	}
}

func assertSameColumn(t *testing.T, want, got *Column) {
	t.Helper()
	if got.Len() != want.Len() {
		t.Fatalf("length: got %d want %d", got.Len(), want.Len())
	}
	for i := 0; i < want.Len(); i++ {
		if got.IsNull(i) != want.IsNull(i) {
			t.Fatalf("null at %d: got %v want %v", i, got.IsNull(i), want.IsNull(i))
		}
		if want.IsNull(i) {
			continue
		}
		if got.ValueAt(i) != want.ValueAt(i) {
			t.Fatalf("value at %d: got %v want %v", i, got.ValueAt(i), want.ValueAt(i))
		}
	}
}

func rangeIdx(lo, hi int) []int {
	out := make([]int, 0, hi-lo)
	for i := lo; i < hi; i++ {
		out = append(out, i)
	}
	return out
}

func stridedIdx(n, stride int) []int {
	out := make([]int, 0, n/stride+1)
	for i := 0; i < n; i += stride {
		out = append(out, i)
	}
	return out
}

func reversedIdx(n, count int) []int {
	out := make([]int, 0, count)
	for i := range count {
		out = append(out, n-1-i)
	}
	return out
}

func repeatedIdx(src, count int) []int {
	out := make([]int, count)
	for i := range out {
		out[i] = src
	}
	return out
}
