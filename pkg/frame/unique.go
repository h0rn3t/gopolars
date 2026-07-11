package frame

import (
	"runtime"
	"sort"

	"github.com/h0rn3t/gopolars/pkg/chunk"
)

// parallelUniqueThreshold is the row count at or above which firstRows discovers
// distinct keys across GOMAXPROCS shards instead of the single-threaded
// chunk.FirstRows scan. It reuses the Filter/GroupBy parallelism gate so small
// frames (where gopolars already wins) stay sequential.
const parallelUniqueThreshold = parallelFilterThreshold

// firstRows returns the first-seen row index of each distinct composite key in
// encounter order — the rows Unique/NUnique keep. Below parallelUniqueThreshold
// (or with a single worker) it runs the sequential chunk.FirstRows kernel; above
// it, it shards the scan and merges (see firstRowsParallel).
func (d DataFrame) firstRows(keyColumns []*chunk.Column) []int {
	workers := runtime.GOMAXPROCS(0)
	if d.height < parallelUniqueThreshold || workers <= 1 || !useTypedStorage() {
		return chunk.FirstRows(keyColumns, d.height)
	}
	return d.firstRowsParallel(keyColumns, workers)
}

// firstRowsParallel discovers distinct keys across contiguous row shards in
// parallel and merges them to the deterministic keep-first row set. It reuses the
// sharded group-by machinery with no aggregates: each shard records the first row
// per local key, the merge keeps min(firstRow) per global key, and sorting those
// first-seen rows ascending yields the encounter order (first-seen rows are
// distinct, so ascending index == first-seen order). Identical to the sequential
// chunk.FirstRows result — the composite key encoding matches GroupIDs/FirstRows.
func (d DataFrame) firstRowsParallel(keyColumns []*chunk.Column, workers int) []int {
	ranges := partitionRanges(d.height, workers)
	merged := runShardedGroupBy(keyColumns, nil, ranges)
	keep := make([]int, len(merged.firstRow))
	copy(keep, merged.firstRow)
	sort.Ints(keep)
	return keep
}
