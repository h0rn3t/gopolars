package chunk

import (
	"math/bits"
	"sync"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/simd"
)

// GatherBitmap gathers rows whose bits are set in mask into a new Column,
// preserving ascending row order. Equivalent to Slice(CompressIndices(mask))
// without allocating the intermediate []int keep list.
func (c *Column) GatherBitmap(mask simd.Bitmap, nRows int) *Column {
	n := simd.BitmapPopcount(mask, nRows)
	out := allocGatherOut(c, n)
	needNulls := out.nulls != nil
	switch c.dtype {
	case dtypes.Int64:
		gatherBitmapSlice(out.i64, c.i64, mask, nRows, needNulls, out.nulls, c.nulls)
	case dtypes.Float64:
		gatherBitmapSlice(out.f64, c.f64, mask, nRows, needNulls, out.nulls, c.nulls)
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		gatherBitmapSlice(out.str, c.str, mask, nRows, needNulls, out.nulls, c.nulls)
	case dtypes.Boolean:
		gatherBitmapSlice(out.bln, c.bln, mask, nRows, needNulls, out.nulls, c.nulls)
	case dtypes.Datetime:
		gatherBitmapSlice(out.tim, c.tim, mask, nRows, needNulls, out.nulls, c.nulls)
	default:
		gatherBitmapSlice(out.boxed, c.boxed, mask, nRows, needNulls, out.nulls, c.nulls)
	}
	return out
}

// FilterGatherColumns runs the fused single-wave batch-filter materialization:
// one set of workers evaluates the predicate mask for word-aligned row shards
// (via evalShard, called concurrently), a barrier on the caller's goroutine
// computes shared prefix-popcount offsets and allocates the output columns,
// then the same workers gather every column of their shard into disjoint
// output spans. No per-column spawn waves and no second wave for the gather.
//
// evalShard returns the local mask covering [start,end) (start is always
// word-aligned) or ok=false to decline the batch path (evaluation error or
// null predicate result). Any declined shard aborts the gather phase and the
// call returns ok=false.
func FilterGatherColumns(cols []*Column, nRows, workers int, evalShard func(start, end int) (simd.Bitmap, bool)) ([]*Column, bool) {
	ranges := alignedRowRanges(nRows, workers)
	global := simd.BitmapNew(nRows)
	okShard := make([]bool, len(ranges))
	offsets := make([]int, len(ranges)+1)
	out := make([]*Column, len(cols))
	// doGather is written before close(proceed) and read after <-proceed, so
	// the channel close orders it without atomics.
	doGather := false
	proceed := make(chan struct{})
	var evalDone, wave sync.WaitGroup
	evalDone.Add(len(ranges))
	wave.Add(len(ranges))
	for i, rg := range ranges {
		go func(i, start, end int) {
			defer wave.Done()
			if mask, ok := evalShard(start, end); ok {
				okShard[i] = true
				copy(global[start>>6:], mask)
			}
			evalDone.Done()
			<-proceed
			if !doGather || offsets[i] == offsets[i+1] {
				return
			}
			for ci, c := range cols {
				gatherBitmapRange(out[ci], c, global, start, end, offsets[i], out[ci].nulls != nil)
			}
		}(i, rg[0], rg[1])
	}
	evalDone.Wait()
	allOK := true
	for _, ok := range okShard {
		allOK = allOK && ok
	}
	if allOK {
		for i, rg := range ranges {
			offsets[i+1] = offsets[i] + bitmapPopcountRange(global, rg[0], rg[1])
		}
		for i, c := range cols {
			out[i] = allocGatherOut(c, offsets[len(ranges)])
		}
		doGather = true
	}
	close(proceed)
	wave.Wait()
	if !allOK {
		return nil, false
	}
	return out, true
}

func allocGatherOut(c *Column, n int) *Column {
	out := &Column{dtype: c.dtype, n: n, nullCount: unknownNullCount}
	needNulls := c.NullCount() != 0
	if needNulls {
		out.nulls = make([]bool, n)
	} else {
		out.nullCount = 0
	}
	switch c.dtype {
	case dtypes.Int64:
		out.i64 = make([]int64, n)
	case dtypes.Float64:
		out.f64 = make([]float64, n)
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		out.str = make([]string, n)
	case dtypes.Boolean:
		out.bln = make([]bool, n)
	case dtypes.Datetime:
		out.tim = make([]time.Time, n)
	default:
		out.boxed = make([]any, n)
	}
	return out
}

func gatherBitmapSlice[T any](dst, src []T, mask simd.Bitmap, nRows int, needNulls bool, outNulls, srcNulls []bool) {
	j := 0
	fullWords := nRows >> 6
	for wi := range fullWords {
		w := mask[wi]
		base := wi << 6
		for w != 0 {
			s := base + bits.TrailingZeros64(w)
			if needNulls && srcNulls != nil && srcNulls[s] {
				outNulls[j] = true
			} else {
				dst[j] = src[s]
			}
			j++
			w &= w - 1
		}
	}
	if rem := nRows & 63; rem != 0 {
		w := mask[fullWords] & (uint64(1)<<uint(rem) - 1)
		base := fullWords << 6
		for w != 0 {
			s := base + bits.TrailingZeros64(w)
			if needNulls && srcNulls != nil && srcNulls[s] {
				outNulls[j] = true
			} else {
				dst[j] = src[s]
			}
			j++
			w &= w - 1
		}
	}
}

func gatherBitmapRange(out, c *Column, mask simd.Bitmap, start, end, outOff int, needNulls bool) {
	switch c.dtype {
	case dtypes.Int64:
		gatherBitmapSliceRange(out.i64[outOff:], c.i64, mask, start, end, needNulls, nullsFrom(out.nulls, outOff), c.nulls)
	case dtypes.Float64:
		gatherBitmapSliceRange(out.f64[outOff:], c.f64, mask, start, end, needNulls, nullsFrom(out.nulls, outOff), c.nulls)
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		gatherBitmapSliceRange(out.str[outOff:], c.str, mask, start, end, needNulls, nullsFrom(out.nulls, outOff), c.nulls)
	case dtypes.Boolean:
		gatherBitmapSliceRange(out.bln[outOff:], c.bln, mask, start, end, needNulls, nullsFrom(out.nulls, outOff), c.nulls)
	case dtypes.Datetime:
		gatherBitmapSliceRange(out.tim[outOff:], c.tim, mask, start, end, needNulls, nullsFrom(out.nulls, outOff), c.nulls)
	default:
		gatherBitmapSliceRange(out.boxed[outOff:], c.boxed, mask, start, end, needNulls, nullsFrom(out.nulls, outOff), c.nulls)
	}
}

func nullsFrom(nulls []bool, off int) []bool {
	if nulls == nil {
		return nil
	}
	return nulls[off:]
}

// gatherBitmapSliceRange copies set bits in source rows [start,end) into dst
// starting at dst[0]. start must be word-aligned (as produced by alignedRowRanges).
func gatherBitmapSliceRange[T any](dst, src []T, mask simd.Bitmap, start, end int, needNulls bool, outNulls, srcNulls []bool) {
	j := 0
	wi := start >> 6
	wordEnd := end >> 6
	for ; wi < wordEnd; wi++ {
		w := mask[wi]
		base := wi << 6
		for w != 0 {
			s := base + bits.TrailingZeros64(w)
			if needNulls && srcNulls != nil && srcNulls[s] {
				outNulls[j] = true
			} else {
				dst[j] = src[s]
			}
			j++
			w &= w - 1
		}
	}
	if remStart := wi << 6; remStart < end {
		rem := end - remStart
		w := mask[wi] & (uint64(1)<<uint(rem) - 1)
		for w != 0 {
			s := remStart + bits.TrailingZeros64(w)
			if needNulls && srcNulls != nil && srcNulls[s] {
				outNulls[j] = true
			} else {
				dst[j] = src[s]
			}
			j++
			w &= w - 1
		}
	}
}

// bitmapPopcountRange counts set bits in [start,end). start must be word-aligned.
func bitmapPopcountRange(mask simd.Bitmap, start, end int) int {
	if end <= start {
		return 0
	}
	return simd.BitmapPopcount(mask[start>>6:], end-start)
}

// alignedRowRanges splits [0,n) into up to workers contiguous ranges whose
// starts (except possibly when n < 64) fall on 64-bit word boundaries.
func alignedRowRanges(n, workers int) [][2]int {
	workers = max(1, min(workers, n))
	chunkSize := (n + workers - 1) / workers
	chunkSize = (chunkSize + 63) &^ 63
	if chunkSize == 0 {
		chunkSize = 64
	}
	ranges := make([][2]int, 0, workers)
	for start := 0; start < n; start += chunkSize {
		end := min(start+chunkSize, n)
		ranges = append(ranges, [2]int{start, end})
	}
	return ranges
}
