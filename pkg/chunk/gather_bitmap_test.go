package chunk

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/simd"
)

func TestGatherBitmapEqualsCompressIndicesSlice(t *testing.T) {
	lengths := []int{0, 1, 63, 64, 65, 100, 70_000}
	dtypes := []string{"int64", "float64", "string", "bool"}
	for _, n := range lengths {
		for _, dt := range dtypes {
			for _, nullEvery := range []int{0, 5} {
				if n == 0 && nullEvery != 0 {
					continue
				}
				col := sampleColumn(dt, n, nullEvery)
				for _, sel := range []string{"empty", "half", "full"} {
					t.Run(fmt.Sprintf("n=%d/%s/null=%d/%s", n, dt, nullEvery, sel), func(t *testing.T) {
						mask := selectivityMask(n, sel)
						want := col.Slice(simd.CompressIndices(mask, n))
						got := col.GatherBitmap(mask, n)
						assertSameColumn(t, want, got)
						if n >= parallelGatherThreshold {
							gotPar, ok := FilterGatherColumns([]*Column{col}, n, 4, shardsOf(mask))
							if !ok {
								t.Fatal("FilterGatherColumns declined")
							}
							assertSameColumn(t, want, gotPar[0])
						}
					})
				}
			}
		}
	}
}

// shardsOf adapts a precomputed full mask to FilterGatherColumns' evalShard:
// each shard returns its word-aligned window of the mask.
func shardsOf(mask simd.Bitmap) func(start, end int) (simd.Bitmap, bool) {
	return func(start, end int) (simd.Bitmap, bool) {
		return mask[start>>6 : (end+63)>>6], true
	}
}

func TestFilterGatherColumnsEqualsPerColumn(t *testing.T) {
	for _, n := range []int{0, 1, 63, 64, 65, 100, 70_000} {
		cols := []*Column{
			sampleColumn("int64", n, 0),
			sampleColumn("float64", n, 5),
			sampleColumn("string", n, 0),
			sampleColumn("bool", n, 5),
		}
		for _, sel := range []string{"empty", "half", "full"} {
			for _, workers := range []int{1, 4} {
				t.Run(fmt.Sprintf("n=%d/%s/workers=%d", n, sel, workers), func(t *testing.T) {
					mask := selectivityMask(n, sel)
					got, ok := FilterGatherColumns(cols, n, workers, shardsOf(mask))
					if !ok {
						t.Fatal("FilterGatherColumns declined")
					}
					for ci, col := range cols {
						assertSameColumn(t, col.GatherBitmap(mask, n), got[ci])
					}
				})
			}
		}
	}
}

func TestFilterGatherColumnsDecline(t *testing.T) {
	const n = 70_000
	cols := []*Column{sampleColumn("int64", n, 0), sampleColumn("string", n, 0)}
	mask := selectivityMask(n, "half")
	shard := shardsOf(mask)
	var calls atomic.Int64
	got, ok := FilterGatherColumns(cols, n, 4, func(start, end int) (simd.Bitmap, bool) {
		calls.Add(1)
		if start == 0 {
			return nil, false // first shard declines (err / null predicate)
		}
		return shard(start, end)
	})
	if ok || got != nil {
		t.Fatalf("want decline, got ok=%v cols=%v", ok, got)
	}
	if calls.Load() == 0 {
		t.Fatal("evalShard never called")
	}
}

func selectivityMask(n int, sel string) simd.Bitmap {
	mask := simd.BitmapNew(n)
	switch sel {
	case "empty":
		return mask
	case "full":
		for i := range n {
			simd.BitmapSet(mask, i)
		}
	case "half":
		for i := 0; i < n; i += 2 {
			simd.BitmapSet(mask, i)
		}
	}
	return mask
}
