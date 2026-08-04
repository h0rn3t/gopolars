package chunk

import "github.com/h0rn3t/gopolars/pkg/dtypes"

// groupSampleSize is the head sample used to estimate a key column's
// cardinality before committing to the sharded group build.
const groupSampleSize = 1 << 13 // 8192 rows

// groupCardinalityDivisor bounds how distinct the sample may be for the sharded
// build to be worth it: a sample with more than groupSampleSize/divisor distinct
// keys is treated as high-cardinality. The sharded build holds one map per
// worker, so its peak memory is workers x distinct keys; for a key that is
// nearly unique per row that is both slower (merge dominates) and far heavier
// than the single-map sequential build.
const groupCardinalityDivisor = 8

// GroupIDsUnordered assigns a group id to every row across the given key
// columns, sharding the hashing across workers, and returns the ids plus the
// number of distinct groups.
//
// Unlike GroupIDs it does NOT number groups in encounter order: the numbering
// follows an unspecified order that may change with the worker count. Use it
// only where the numbering itself is not observable — for example a windowed
// aggregation that accumulates into a per-id slot, where only the partitioning
// matters. Callers that expose group order (GroupBy, Unique) MUST keep using
// GroupIDs.
//
// Null keys form a single group, matching GroupIDs. The row-to-group mapping is
// identical to GroupIDs' up to a renumbering of the group ids.
func GroupIDsUnordered(cols []*Column, n int) (ids []int, ngroups int) {
	if len(cols) == 1 {
		// ponytail: Int64 and String cover the realistic partition keys; Float64
		// (whose ids must key on canonical NaN bits, so -0.0 and 0.0 stay distinct)
		// and the boxed dtypes fall through to the sequential build rather than
		// growing a second generic path for a case partitioning rarely uses.
		c := cols[0]
		if c != nil && n > 0 {
			switch c.dtype {
			case dtypes.Int64:
				if out, count, ok := groupIDsSharded(c.i64[:n], c.nulls, n); ok {
					return out, count
				}
			case dtypes.String, dtypes.Categorical, dtypes.Enum:
				if out, count, ok := groupIDsSharded(c.str[:n], c.nulls, n); ok {
					return out, count
				}
			}
		}
	}
	seqIDs, firstRow := GroupIDs(cols, n)
	return seqIDs, len(firstRow)
}

// groupIDsSharded builds group ids in three passes: each worker numbers the keys
// in its own row range against a shard-local map, the shard-local keys are then
// merged into one global numbering, and a final parallel pass rewrites the ids.
// ok is false when the build is not worth sharding (single worker, or a
// high-cardinality key), leaving the caller on the sequential path.
//
// Nulls are marked with a negative id in the first pass so that a shard's key
// ids stay dense and index its key slice directly; the merge maps them onto one
// shared null group.
func groupIDsSharded[K comparable](vals []K, nulls []bool, n int) (ids []int, ngroups int, ok bool) {
	bounds := shardBounds(n)
	workers := len(bounds) - 1
	if workers <= 1 {
		return nil, 0, false
	}

	sample := min(groupSampleSize, n)
	seen := make(map[K]struct{}, sample)
	for row := 0; row < sample; row++ {
		if nulls != nil && nulls[row] {
			continue
		}
		seen[vals[row]] = struct{}{}
	}
	if len(seen)*groupCardinalityDivisor > sample {
		return nil, 0, false
	}

	ids = make([]int, n)
	localKeys := make([][]K, workers)
	sawNull := make([]bool, workers)
	forEachBound(bounds, func(w, lo, hi int) {
		local := make(map[K]int, len(seen))
		keys := make([]K, 0, len(seen))
		for row := lo; row < hi; row++ {
			if nulls != nil && nulls[row] {
				ids[row] = -1
				sawNull[w] = true
				continue
			}
			v := vals[row]
			g, hit := local[v]
			if !hit {
				g = len(keys)
				local[v] = g
				keys = append(keys, v)
			}
			ids[row] = g
		}
		localKeys[w] = keys
	})

	global := make(map[K]int, len(seen))
	remaps := make([][]int, workers)
	anyNull := false
	for w := 0; w < workers; w++ {
		remap := make([]int, len(localKeys[w]))
		for localID, k := range localKeys[w] {
			g, hit := global[k]
			if !hit {
				g = len(global)
				global[k] = g
			}
			remap[localID] = g
		}
		remaps[w] = remap
		anyNull = anyNull || sawNull[w]
	}
	ngroups = len(global)
	nullGroup := -1
	if anyNull {
		nullGroup = ngroups
		ngroups++
	}

	forEachBound(bounds, func(w, lo, hi int) {
		remap := remaps[w]
		for row := lo; row < hi; row++ {
			if ids[row] < 0 {
				ids[row] = nullGroup
				continue
			}
			ids[row] = remap[ids[row]]
		}
	})
	return ids, ngroups, true
}
