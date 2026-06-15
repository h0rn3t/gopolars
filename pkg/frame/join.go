package frame

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// parallelJoinThreshold is the left-row count at or above which the equi-join
// probes across multiple workers, and the right-row count at or above which the
// probe table is built sharded. Below it the goroutine/stitch overhead outweighs
// the work and the join stays single-threaded — the small-frame regime where
// gopolars already beats Polars.
//
// Measured basis (Apple M4 Pro, profile-tune-join-and-rowop-headroom §1–§2): a
// 1M×1M high-cardinality join builds ~5.6x faster sharded than single-goroutine
// (25 ms vs 127 ms), while a small-right join (right ≈ 1000) stays sequential and
// cheap and the 1K join stays fully sequential and unregressed. A single
// constant across build/probe/gather holds because their crossovers sit in the
// same band; 1<<15 sits comfortably between the small-right (sequential, fast)
// and 1M (parallel, 5.6x) regimes.
const parallelJoinThreshold = 1 << 15 // 32768 rows

func join(left DataFrame, input JoinInput) (DataFrame, error) {
	if input.How == JoinTypeCross {
		return crossJoin(left, input)
	}
	if len(input.LeftOn) == 0 || len(input.RightOn) == 0 {
		return DataFrame{}, fmt.Errorf("join keys are empty")
	}
	if len(input.LeftOn) != len(input.RightOn) {
		return DataFrame{}, fmt.Errorf("join keys length mismatch")
	}
	if input.How == "" {
		input.How = JoinTypeInner
	}
	if input.Suffix == "" {
		input.Suffix = "_right"
	}
	if input.How == JoinTypeAsof {
		return asofJoin(left, input)
	}

	rightKeyCols, err := joinKeyColumns(input.Other, input.RightOn)
	if err != nil {
		return DataFrame{}, err
	}
	leftKeyCols, err := joinKeyColumns(left, input.LeftOn)
	if err != nil {
		return DataFrame{}, err
	}

	// Build the probe table (typed packed key with no per-row string allocation
	// where possible, byte-encoded fallback otherwise), then probe the left side
	// into (leftIdx, rightIdx) int32 pair buffers. Both build and probe run
	// across workers above the size thresholds; below them the sequential path is
	// used unchanged. Emitted order matches the prior implementation: left-row
	// order, posting-order right matches within a left row, unmatched-right rows
	// appended last in right-row order.
	workers := runtime.GOMAXPROCS(0)
	pt := buildProbeTable(leftKeyCols, rightKeyCols, input.Other.height, workers)
	leftIdx, rightIdx := pt.probeAll(input.How, left.height, input.Other.height, workers)

	rightIncluded := input.How != JoinTypeSemi && input.How != JoinTypeAnti
	return materializeJoinIdx(left, input.Other, leftIdx, rightIdx, input.Suffix, rightIncluded)
}

// joinKeyColumns resolves the typed key columns for the given key names.
func joinKeyColumns(df DataFrame, keys []string) ([]*chunk.Column, error) {
	cols := make([]*chunk.Column, len(keys))
	for j, k := range keys {
		s, ok := df.cols[k]
		if !ok {
			return nil, fmt.Errorf("join key %s not found", k)
		}
		cols[j] = s.Column()
	}
	return cols, nil
}

// probeTable indexes the right (build) side of an equi-join by join key, storing
// int32 right-row indices. For a single fixed-width key column it keys a sharded
// map[uint64] over the packed key (no per-row string allocation); otherwise it
// falls back to a single map[string] over the byte-encoded AppendRowKey. Null
// keys match other null keys (preserving the prior behavior): in the packed path
// they live in nullRows; in the byte path the null-tagged key keeps them in the
// map alongside non-null keys (a non-null value never produces the null tag). A
// non-null packed key never collides with nullRows, so null and value matches
// never cross. For a large right side the build is sharded across workers (shard
// s owns keys with k%shards==s, each worker writing only its own map) — measured
// ~5.6x faster at 1M×1M than a single-goroutine build, since it parallelizes the
// per-key posting-slice allocation. After build the maps are read-only and safe
// for concurrent probes.
type probeTable struct {
	packed   bool
	shards   []map[uint64][]int32 // packed: len >= 1; probe at shards[k % len]
	nullRows []int32              // packed: right rows whose key is null
	byteMap  map[string][]int32   // !packed: byte-encoded key -> right rows

	// Probe-side (left) key material, prepared once at build time.
	leftKeyAt   func(int) uint64 // packed: per-row key (dtype switch hoisted; no O(rows) buffer)
	leftNulls   []bool           // packed: left null mask (nil == no nulls)
	leftKeyCols []*chunk.Column  // !packed: left key columns for AppendRowKey
}

// buildProbeTable constructs the right-side probe table and prepares the
// left-side key material. The packed path applies when both sides are a single
// key column of the same fixed-width dtype; everything else (string, composite,
// or boxed keys, or a dtype mismatch) uses the byte-encoded fallback.
func buildProbeTable(leftKeyCols, rightKeyCols []*chunk.Column, rightHeight, workers int) *probeTable {
	pt := &probeTable{}
	if len(rightKeyCols) == 1 && len(leftKeyCols) == 1 &&
		chunk.CanPackJoinKey(rightKeyCols[0]) &&
		rightKeyCols[0].DataType() == leftKeyCols[0].DataType() {
		pt.packed = true
		rightKeyAt, _ := chunk.PackKeyFunc(rightKeyCols[0])
		rightNulls := rightKeyCols[0].Nulls()
		pt.leftKeyAt, _ = chunk.PackKeyFunc(leftKeyCols[0])
		pt.leftNulls = leftKeyCols[0].Nulls()
		pt.buildPacked(rightKeyAt, rightNulls, rightHeight, workers)
		return pt
	}

	// Byte-encoded fallback: sequential build. Nulls live under the null-tagged
	// key, matching the prior behavior (null keys match other null keys). A
	// reused scratch buffer avoids per-row fmt formatting; the only per-key
	// allocation is the map's owned string key on first insert of each distinct
	// key (O(distinct keys), not O(rows)).
	pt.leftKeyCols = leftKeyCols
	pt.byteMap = make(map[string][]int32, rightHeight)
	var scratch []byte
	for i := range rightHeight {
		scratch = chunk.AppendRowKey(scratch[:0], rightKeyCols, i)
		pt.byteMap[string(scratch)] = append(pt.byteMap[string(scratch)], int32(i))
	}
	return pt
}

// buildPacked fills the packed shard maps from the right-side packed keys. Above
// the threshold it shards the key space across workers so each worker owns the
// keys with k % shards == s and writes only its own map — no shared writes, and
// the maps are read-only after this returns. This parallelizes the per-key
// posting-slice allocation, which dominates a high-cardinality build (measured
// ~5.6x faster than a single-goroutine build at 1M×1M). Null right rows are
// collected into nullRows in a single pass by a dedicated goroutine.
func (pt *probeTable) buildPacked(keyAt func(int) uint64, nulls []bool, height, workers int) {
	shards := 1
	if height >= parallelJoinThreshold && workers > 1 {
		shards = workers
	}
	pt.shards = make([]map[uint64][]int32, shards)

	if shards == 1 {
		m := make(map[uint64][]int32, height)
		for i := range height {
			if nulls != nil && nulls[i] {
				pt.nullRows = append(pt.nullRows, int32(i))
				continue
			}
			k := keyAt(i)
			m[k] = append(m[k], int32(i))
		}
		pt.shards[0] = m
		return
	}

	var wg sync.WaitGroup
	us := uint64(shards)
	for s := range shards {
		wg.Go(func() {
			m := make(map[uint64][]int32)
			for i := range height {
				if nulls != nil && nulls[i] {
					continue
				}
				k := keyAt(i)
				if int(k%us) == s {
					m[k] = append(m[k], int32(i))
				}
			}
			pt.shards[s] = m
		})
	}
	if nulls != nil {
		wg.Go(func() {
			for i := range height {
				if nulls[i] {
					pt.nullRows = append(pt.nullRows, int32(i))
				}
			}
		})
	}
	wg.Wait()
}

// lookup returns the right rows matching the left row's join key. scratch is a
// per-caller reusable byte buffer (used only on the byte-encoded path); each
// probe worker owns its own, so lookup is safe to call concurrently.
func (pt *probeTable) lookup(leftRow int, scratch *[]byte) []int32 {
	if pt.packed {
		if pt.leftNulls != nil && pt.leftNulls[leftRow] {
			return pt.nullRows
		}
		k := pt.leftKeyAt(leftRow)
		return pt.shards[int(k%uint64(len(pt.shards)))][k]
	}
	*scratch = chunk.AppendRowKey((*scratch)[:0], pt.leftKeyCols, leftRow)
	return pt.byteMap[string(*scratch)]
}

// probeRange probes left rows [start,end) into freshly allocated (leftIdx,
// rightIdx) pair buffers, pre-sized from estPerRow. The per-row emission rules
// reproduce the prior sequential join exactly for every mode; right/full
// unmatched-right rows are appended later by probeAll.
func (pt *probeTable) probeRange(how JoinType, start, end int, estPerRow float64) (lIdx, rIdx []int32) {
	capHint := int(float64(end-start)*estPerRow) + 1
	lIdx = make([]int32, 0, capHint)
	rIdx = make([]int32, 0, capHint)
	var scratch []byte
	for i := start; i < end; i++ {
		rightRows := pt.lookup(i, &scratch)
		if len(rightRows) == 0 {
			switch how {
			case JoinTypeAnti, JoinTypeLeft, JoinTypeFull:
				lIdx = append(lIdx, int32(i))
				rIdx = append(rIdx, -1)
			}
			continue
		}
		switch how {
		case JoinTypeSemi:
			lIdx = append(lIdx, int32(i))
			rIdx = append(rIdx, rightRows[0])
		case JoinTypeAnti:
			// Matched left row: anti-join emits nothing.
		default:
			for _, rr := range rightRows {
				lIdx = append(lIdx, int32(i))
				rIdx = append(rIdx, rr)
			}
		}
	}
	return lIdx, rIdx
}

// sampleMatchRate estimates output rows per left row by probing a bounded prefix,
// so probeRange can pre-size its pair buffers. A floor keeps the estimate from
// under-sizing to near-zero on a sparse prefix.
func (pt *probeTable) sampleMatchRate(how JoinType, height int) float64 {
	sample := min(height, 1024)
	if sample == 0 {
		return 1
	}
	var scratch []byte
	total := 0
	for i := range sample {
		m := len(pt.lookup(i, &scratch))
		switch how {
		case JoinTypeSemi:
			if m > 0 {
				total++
			}
		case JoinTypeAnti:
			if m == 0 {
				total++
			}
		case JoinTypeLeft, JoinTypeFull:
			if m == 0 {
				total++
			} else {
				total += m
			}
		default: // inner, right
			total += m
		}
	}
	rate := float64(total) / float64(sample)
	if rate < 0.5 {
		rate = 0.5
	}
	return rate
}

// probeAll probes the whole left side, in parallel over contiguous left-row
// shards above the threshold, and returns the concatenated (leftIdx, rightIdx)
// pair buffers in left-row order. For right/full joins it then appends the
// unmatched right rows last, in right-row order, with the left columns
// null-filled. The matched-right set is derived from the collected pairs, so the
// parallel probe needs no shared per-right matched buffer and stays race-free.
func (pt *probeTable) probeAll(how JoinType, leftHeight, otherHeight, workers int) (lIdx, rIdx []int32) {
	rate := pt.sampleMatchRate(how, leftHeight)

	if leftHeight < parallelJoinThreshold || workers <= 1 {
		lIdx, rIdx = pt.probeRange(how, 0, leftHeight, rate)
	} else {
		ranges := partitionRanges(leftHeight, workers)
		type part struct{ l, r []int32 }
		parts := make([]part, len(ranges))
		var wg sync.WaitGroup
		for i, rg := range ranges {
			wg.Go(func() {
				parts[i].l, parts[i].r = pt.probeRange(how, rg[0], rg[1], rate)
			})
		}
		wg.Wait()
		total := 0
		for _, p := range parts {
			total += len(p.l)
		}
		lIdx = make([]int32, 0, total)
		rIdx = make([]int32, 0, total)
		for _, p := range parts {
			lIdx = append(lIdx, p.l...)
			rIdx = append(rIdx, p.r...)
		}
	}

	if how == JoinTypeRight || how == JoinTypeFull {
		matched := make([]bool, otherHeight)
		for _, rr := range rIdx {
			if rr >= 0 {
				matched[rr] = true
			}
		}
		for rr := range otherHeight {
			if !matched[rr] {
				lIdx = append(lIdx, -1)
				rIdx = append(rIdx, int32(rr))
			}
		}
	}
	return lIdx, rIdx
}

// materializeJoinIdx builds the joined output columns by one typed gather per
// column over the (leftIdx, rightIdx) int32 pair buffers, null-filling unmatched
// (-1) indices. Above the threshold the per-column gathers run concurrently —
// each writes a disjoint output slot, so the materialization tail uses the idle
// cores. Allocation is O(columns), not O(rows × columns).
func materializeJoinIdx(left, other DataFrame, leftIdx, rightIdx []int32, suffix string, rightIncluded bool) (DataFrame, error) {
	if suffix == "" {
		suffix = "_right"
	}
	type gatherTask struct {
		outName string
		src     *chunk.Column
		idx     []int32
	}
	tasks := make([]gatherTask, 0, len(left.order)+len(other.order))
	for _, name := range left.order {
		tasks = append(tasks, gatherTask{name, left.cols[name].Column(), leftIdx})
	}
	if rightIncluded {
		for _, name := range other.order {
			outName := name
			if _, exists := left.cols[name]; exists {
				outName = name + suffix
			}
			tasks = append(tasks, gatherTask{outName, other.cols[name].Column(), rightIdx})
		}
	}

	out := make([]series.Series, len(tasks))
	workers := runtime.GOMAXPROCS(0)
	if len(tasks) <= 1 || len(leftIdx) < parallelJoinThreshold || workers <= 1 {
		for t := range tasks {
			out[t] = series.FromColumn(tasks[t].outName, tasks[t].src.GatherInt32(tasks[t].idx))
		}
		return New(NewInput{Series: out})
	}
	if workers > len(tasks) {
		workers = len(tasks)
	}
	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			for t := w; t < len(tasks); t += workers {
				out[t] = series.FromColumn(tasks[t].outName, tasks[t].src.GatherInt32(tasks[t].idx))
			}
		})
	}
	wg.Wait()
	return New(NewInput{Series: out})
}

// materializeJoin builds the joined output from []pair (the cross/asof join
// representation), widening to the int32 pair buffers materializeJoinIdx expects.
func materializeJoin(left, other DataFrame, pairs []pair, suffix string, rightIncluded bool) (DataFrame, error) {
	leftIdx := make([]int32, len(pairs))
	rightIdx := make([]int32, len(pairs))
	for i, p := range pairs {
		leftIdx[i] = int32(p.left)
		rightIdx[i] = int32(p.right)
	}
	return materializeJoinIdx(left, other, leftIdx, rightIdx, suffix, rightIncluded)
}

type pair struct {
	left  int
	right int
}

func crossJoin(left DataFrame, input JoinInput) (DataFrame, error) {
	if input.Suffix == "" {
		input.Suffix = "_right"
	}
	if left.height == 0 || input.Other.height == 0 {
		return materializePairs(left, input, nil, true)
	}
	pairs := make([]pair, 0, left.height*input.Other.height)
	for i := range left.height {
		for j := range input.Other.height {
			pairs = append(pairs, pair{left: i, right: j})
		}
	}
	clone := input
	clone.How = JoinTypeInner
	clone.LeftOn = []string{left.order[0]}
	clone.RightOn = []string{input.Other.order[0]}
	return materializePairs(left, clone, pairs, true)
}

func asofJoin(left DataFrame, input JoinInput) (DataFrame, error) {
	if len(input.LeftOn) != 1 || len(input.RightOn) != 1 {
		return DataFrame{}, fmt.Errorf("asof join requires single key")
	}
	rightKey, ok := input.Other.cols[input.RightOn[0]]
	if !ok {
		return DataFrame{}, fmt.Errorf("join key %s not found", input.RightOn[0])
	}
	leftKey, ok := left.cols[input.LeftOn[0]]
	if !ok {
		return DataFrame{}, fmt.Errorf("join key %s not found", input.LeftOn[0])
	}
	direction := input.AsofDirection
	if direction == "" {
		direction = "backward"
	}
	pairs := make([]pair, 0, left.height)
	for i := 0; i < left.height; i++ {
		lv := leftKey.Value(i)
		best := -1
		bestDiff := int64(math.MaxInt64)
		for j := 0; j < input.Other.height; j++ {
			rv := rightKey.Value(j)
			diff, ok := asofDiff(lv, rv)
			if !ok {
				continue
			}
			if input.AsofTolerance > 0 && abs64(diff) > input.AsofTolerance {
				continue
			}
			if direction == "backward" && diff < 0 {
				continue
			}
			if direction == "forward" && diff > 0 {
				continue
			}
			ad := abs64(diff)
			if ad < bestDiff {
				bestDiff = ad
				best = j
				continue
			}
			if ad == bestDiff && direction == "nearest" && best >= 0 && j < best {
				best = j
			}
		}
		pairs = append(pairs, pair{left: i, right: best})
	}
	return materializePairs(left, input, pairs, true)
}

func materializePairs(left DataFrame, input JoinInput, pairs []pair, rightIncluded bool) (DataFrame, error) {
	return materializeJoin(left, input.Other, pairs, input.Suffix, rightIncluded)
}

func asofDiff(left any, right any) (int64, bool) {
	switch lv := left.(type) {
	case int64:
		rv, ok := right.(int64)
		if !ok {
			return 0, false
		}
		return lv - rv, true
	case float64:
		rv, ok := right.(float64)
		if !ok {
			return 0, false
		}
		return int64(lv - rv), true
	case time.Time:
		rv, ok := right.(time.Time)
		if !ok {
			return 0, false
		}
		return lv.Sub(rv).Nanoseconds(), true
	}
	return 0, false
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
