package frame

import (
	"math"
	"runtime"
	"sort"
	"sync"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// parallelGroupByThreshold is the row count at or above which GroupBy.Agg builds
// shard-local group tables across GOMAXPROCS workers and merges them, instead of
// the single-threaded GroupIDs + bucket path. It reuses the Filter parallelism
// gate so small frames (where gopolars already wins) stay sequential.
const parallelGroupByThreshold = parallelFilterThreshold

// aggKind enumerates the aggregates the parallel path folds as an associative
// running reduction over a typed column. Anything else (e.g. n_unique) forces
// the sequential bucket path.
type aggKind int

const (
	aggCount aggKind = iota
	aggSum
	aggMean
	aggMin
	aggMax
)

// aggSpec is a resolved aggregate for the parallel path: its kind, output name
// and dtype, and the typed backing of its target column (nil for count, which
// reads no column).
type aggSpec struct {
	kind     aggKind
	name     string
	dtype    dtypes.DataType
	colFloat bool // target column is Float64 (else Int64); ignored for count
	f64s     []float64
	i64s     []int64
	nulls    []bool
}

// acc is the running accumulator for one group under one aggregate. Which fields
// are live depends on the kind and column type; finalizeAgg reads only the
// meaningful one. cnt is the contributing-value count (sum/mean/min/max) or the
// group row count (count, which — like the sequential path — includes nulls).
type acc struct {
	cnt  int64
	fval float64
	ival int64
}

// foldAgg folds the value at row into a, matching the sequential typed semantics
// exactly: count tallies every row; sum/mean/min/max skip nulls and NaN.
func foldAgg(spec *aggSpec, a *acc, row int) {
	if spec.kind == aggCount {
		a.cnt++
		return
	}
	if spec.nulls != nil && spec.nulls[row] {
		return
	}
	if spec.colFloat {
		v := spec.f64s[row]
		if math.IsNaN(v) {
			return
		}
		switch spec.kind {
		case aggSum, aggMean:
			a.fval += v
		case aggMin:
			if a.cnt == 0 || v < a.fval {
				a.fval = v
			}
		case aggMax:
			if a.cnt == 0 || v > a.fval {
				a.fval = v
			}
		}
	} else {
		v := spec.i64s[row]
		switch spec.kind {
		case aggSum, aggMean:
			a.ival += v
		case aggMin:
			if a.cnt == 0 || v < a.ival {
				a.ival = v
			}
		case aggMax:
			if a.cnt == 0 || v > a.ival {
				a.ival = v
			}
		}
	}
	a.cnt++
}

// mergeAgg reduces a shard's partial accumulator src into the global dst. Every
// aggregate here is associative, so merging in shard order is order-independent
// for the value (group ordering is fixed separately by first-seen row).
func mergeAgg(kind aggKind, colFloat bool, dst *acc, src acc) {
	switch kind {
	case aggCount, aggSum, aggMean:
		dst.cnt += src.cnt
		dst.fval += src.fval
		dst.ival += src.ival
	case aggMin:
		if src.cnt > 0 {
			if colFloat {
				if dst.cnt == 0 || src.fval < dst.fval {
					dst.fval = src.fval
				}
			} else {
				if dst.cnt == 0 || src.ival < dst.ival {
					dst.ival = src.ival
				}
			}
		}
		dst.cnt += src.cnt
	case aggMax:
		if src.cnt > 0 {
			if colFloat {
				if dst.cnt == 0 || src.fval > dst.fval {
					dst.fval = src.fval
				}
			} else {
				if dst.cnt == 0 || src.ival > dst.ival {
					dst.ival = src.ival
				}
			}
		}
		dst.cnt += src.cnt
	}
}

// finalizeAgg produces the boxed output value for a group, matching the
// sequential evalAgg: an empty contributing set yields nil (a null result) for
// sum/mean/min/max; count always yields its int64 row count.
func finalizeAgg(spec *aggSpec, a acc) any {
	switch spec.kind {
	case aggCount:
		return a.cnt
	case aggSum:
		if a.cnt == 0 {
			return nil
		}
		if spec.colFloat {
			return a.fval
		}
		return a.ival
	case aggMean:
		if a.cnt == 0 {
			return nil
		}
		if spec.colFloat {
			return a.fval / float64(a.cnt)
		}
		return float64(a.ival) / float64(a.cnt)
	case aggMin, aggMax:
		if a.cnt == 0 {
			return nil
		}
		if spec.colFloat {
			return a.fval
		}
		return a.ival
	}
	return nil
}

// resolveParallelAggs maps the aggregate expressions to typed specs. ok is false
// (the caller falls back to the sequential path) for any aggregate that is not an
// associative running reduction over a typed numeric column: n_unique, a
// non-column target, or a non-numeric target column.
func (g GroupBy) resolveParallelAggs(exprs []expr.Expr) ([]aggSpec, bool) {
	specs := make([]aggSpec, len(exprs))
	for i, e := range exprs {
		if e.Kind() != expr.KindAgg {
			return nil, false
		}
		spec := aggSpec{name: e.Name()}
		switch e.Op() {
		case "count":
			spec.kind = aggCount
			spec.dtype = dtypes.Int64
		case "sum", "mean", "min", "max":
			target := e.Target()
			if target == nil || target.Kind() != expr.KindCol {
				return nil, false
			}
			s, ok := g.df.cols[target.ColName()]
			if !ok {
				return nil, false
			}
			col := s.Column()
			if col == nil {
				return nil, false
			}
			if f64s, ok := col.Float64s(); ok {
				spec.colFloat = true
				spec.f64s = f64s
			} else if i64s, ok := col.Int64s(); ok {
				spec.i64s = i64s
			} else {
				return nil, false // non-numeric target: sequential path
			}
			spec.nulls = col.Nulls()
			switch e.Op() {
			case "sum":
				spec.kind = aggSum
			case "mean":
				spec.kind = aggMean
			case "min":
				spec.kind = aggMin
			case "max":
				spec.kind = aggMax
			}
			if e.Op() == "mean" {
				spec.dtype = dtypes.Float64
			} else if spec.colFloat {
				spec.dtype = dtypes.Float64
			} else {
				spec.dtype = dtypes.Int64
			}
		default:
			return nil, false // n_unique and anything else: sequential path
		}
		specs[i] = spec
	}
	return specs, true
}

// shardTable is one worker's group table over a contiguous row range: a typed
// key map, the first-seen global row per local group, and per-aggregate
// accumulators. nullLG is the local group holding null keys (-1 until seen); for
// the composite path nulls are encoded in the key bytes so nullLG stays -1.
type shardTable[K comparable] struct {
	m        map[K]int
	keys     []K
	isNull   []bool
	firstRow []int
	accs     [][]acc
	nullLG   int
}

// shardGroupHint presizes a shard's group slices and map so small-cardinality
// group-bys (the common case) allocate each backing array once instead of
// growing it repeatedly; larger cardinalities grow from here amortized.
const shardGroupHint = 64

func newShardTable[K comparable](naggs int) *shardTable[K] {
	st := &shardTable[K]{
		m:        make(map[K]int, shardGroupHint),
		nullLG:   -1,
		keys:     make([]K, 0, shardGroupHint),
		isNull:   make([]bool, 0, shardGroupHint),
		firstRow: make([]int, 0, shardGroupHint),
		accs:     make([][]acc, naggs),
	}
	for a := 0; a < naggs; a++ {
		st.accs[a] = make([]acc, 0, shardGroupHint)
	}
	return st
}

func (st *shardTable[K]) add(key K, isNull bool, row, naggs int) int {
	gi := len(st.firstRow)
	st.firstRow = append(st.firstRow, row)
	st.keys = append(st.keys, key)
	st.isNull = append(st.isNull, isNull)
	for a := 0; a < naggs; a++ {
		st.accs[a] = append(st.accs[a], acc{})
	}
	return gi
}

// scanShardTyped builds a shard table for a single typed key column. keyAt
// returns the typed key and false when the row's key is null (routed to nullLG).
func scanShardTyped[K comparable](start, end int, keyAt func(int) (K, bool), specs []aggSpec) *shardTable[K] {
	naggs := len(specs)
	st := newShardTable[K](naggs)
	for row := start; row < end; row++ {
		k, ok := keyAt(row)
		var lg int
		if !ok {
			if st.nullLG == -1 {
				st.nullLG = st.add(k, true, row, naggs)
			}
			lg = st.nullLG
		} else {
			g, seen := st.m[k]
			if !seen {
				g = st.add(k, false, row, naggs)
				st.m[k] = g
			}
			lg = g
		}
		for a := range specs {
			foldAgg(&specs[a], &st.accs[a][lg], row)
		}
	}
	return st
}

// scanShardComposite builds a shard table for multi-key or non-(int/string)
// single-key inputs, using the same byte encoding as chunk.GroupIDs' composite
// path (nulls and canonical NaN are baked into the key bytes). The string(scratch)
// map index reuses the scratch buffer without allocating on lookups.
func scanShardComposite(start, end int, keyCols []*chunk.Column, specs []aggSpec) *shardTable[string] {
	naggs := len(specs)
	st := newShardTable[string](naggs)
	var scratch []byte
	for row := start; row < end; row++ {
		scratch = scratch[:0]
		scratch = chunk.AppendRowKey(scratch, keyCols, row)
		lg, seen := st.m[string(scratch)]
		if !seen {
			key := string(scratch)
			lg = st.add(key, false, row, naggs)
			st.m[key] = lg
		}
		for a := range specs {
			foldAgg(&specs[a], &st.accs[a][lg], row)
		}
	}
	return st
}

// mergedGroups holds the merged global groups: the minimum first-seen row per
// group and the reduced accumulators. Group ordering is resolved later by sorting
// on firstRow.
type mergedGroups struct {
	firstRow []int
	accs     [][]acc
}

// mergeShardsTyped combines shard tables by typed key, reducing partial
// aggregates associatively and tracking each group's minimum first-seen row.
// Shards are visited in ascending range order, so a group is created at its
// earliest-seen shard; the explicit min keeps firstRow correct regardless.
func mergeShardsTyped[K comparable](shards []*shardTable[K], specs []aggSpec) mergedGroups {
	naggs := len(specs)
	global := make(map[K]int, shardGroupHint)
	globalNull := -1
	firstRow := make([]int, 0, shardGroupHint)
	accs := make([][]acc, naggs)
	for a := 0; a < naggs; a++ {
		accs[a] = make([]acc, 0, shardGroupHint)
	}
	addGlobal := func(fr int) int {
		gi := len(firstRow)
		firstRow = append(firstRow, fr)
		for a := 0; a < naggs; a++ {
			accs[a] = append(accs[a], acc{})
		}
		return gi
	}
	for _, sh := range shards {
		if sh == nil {
			continue
		}
		for lg := range sh.firstRow {
			var gi int
			if sh.isNull[lg] {
				if globalNull == -1 {
					globalNull = addGlobal(sh.firstRow[lg])
				}
				gi = globalNull
			} else {
				k := sh.keys[lg]
				g, ok := global[k]
				if !ok {
					g = addGlobal(sh.firstRow[lg])
					global[k] = g
				}
				gi = g
			}
			if sh.firstRow[lg] < firstRow[gi] {
				firstRow[gi] = sh.firstRow[lg]
			}
			for a := 0; a < naggs; a++ {
				mergeAgg(specs[a].kind, specs[a].colFloat, &accs[a][gi], sh.accs[a][lg])
			}
		}
	}
	return mergedGroups{firstRow: firstRow, accs: accs}
}

// aggParallel is the parallel group-by fast path. It returns ok=true when it
// produced the result, ok=false to fall back to the sequential typed path
// (disabled storage, below threshold, single worker, or a non-associative
// aggregate), or an error.
func (g GroupBy) aggParallel(keyColumns []*chunk.Column, exprs []expr.Expr) (DataFrame, bool, error) {
	if !useTypedStorage() {
		return DataFrame{}, false, nil
	}
	height := g.df.height
	workers := runtime.GOMAXPROCS(0)
	if height < parallelGroupByThreshold || workers <= 1 {
		return DataFrame{}, false, nil
	}
	specs, ok := g.resolveParallelAggs(exprs)
	if !ok {
		return DataFrame{}, false, nil
	}
	ranges := partitionRanges(height, workers)
	merged := runShardedGroupBy(keyColumns, specs, ranges)
	df, err := g.buildAggOutput(keyColumns, specs, merged)
	if err != nil {
		return DataFrame{}, false, err
	}
	return df, true, nil
}

// runShardedGroupBy scans the row ranges in parallel into shard tables and merges
// them. A single Int64 or String key uses a typed-key map; everything else
// (Float64, Boolean, Datetime, boxed, or multi-key) uses the composite encoder,
// which matches chunk.GroupIDs' grouping for those inputs.
func runShardedGroupBy(keyColumns []*chunk.Column, specs []aggSpec, ranges [][2]int) mergedGroups {
	if len(keyColumns) == 1 {
		col := keyColumns[0]
		switch col.DataType() {
		case dtypes.Int64:
			i64s, _ := col.Int64s()
			nulls := col.Nulls()
			keyAt := func(row int) (int64, bool) {
				if nulls != nil && nulls[row] {
					return 0, false
				}
				return i64s[row], true
			}
			return mergeShardsTyped(parScan(ranges, func(start, end int) *shardTable[int64] {
				return scanShardTyped(start, end, keyAt, specs)
			}), specs)
		case dtypes.String, dtypes.Categorical, dtypes.Enum:
			strs, _ := col.Strings()
			nulls := col.Nulls()
			keyAt := func(row int) (string, bool) {
				if nulls != nil && nulls[row] {
					return "", false
				}
				return strs[row], true
			}
			return mergeShardsTyped(parScan(ranges, func(start, end int) *shardTable[string] {
				return scanShardTyped(start, end, keyAt, specs)
			}), specs)
		}
	}
	return mergeShardsTyped(parScan(ranges, func(start, end int) *shardTable[string] {
		return scanShardComposite(start, end, keyColumns, specs)
	}), specs)
}

// parScan runs build over each range concurrently, one worker per range, each
// writing only its own shard slot — no shared mutable state.
func parScan[K comparable](ranges [][2]int, build func(start, end int) *shardTable[K]) []*shardTable[K] {
	shards := make([]*shardTable[K], len(ranges))
	var wg sync.WaitGroup
	for i, rg := range ranges {
		wg.Add(1)
		go func(i, start, end int) {
			defer wg.Done()
			shards[i] = build(start, end)
		}(i, rg[0], rg[1])
	}
	wg.Wait()
	return shards
}

// buildAggOutput orders groups by ascending first-seen row (matching the
// sequential encounter order), materializes the key columns via a typed Gather of
// the representative rows, and finalizes each aggregate.
func (g GroupBy) buildAggOutput(keyColumns []*chunk.Column, specs []aggSpec, m mergedGroups) (DataFrame, error) {
	ngroups := len(m.firstRow)
	order := make([]int, ngroups)
	for i := range order {
		order[i] = i
	}
	// First-seen rows are distinct across groups, so this is a total order.
	sort.Slice(order, func(i, j int) bool { return m.firstRow[order[i]] < m.firstRow[order[j]] })

	reps := make([]int, ngroups)
	for i, gi := range order {
		reps[i] = m.firstRow[gi]
	}
	out := make([]series.Series, 0, len(g.keys)+len(specs))
	for j, key := range g.keys {
		out = append(out, series.FromColumn(key, keyColumns[j].Gather(reps)))
	}
	for a := range specs {
		vals := make([]any, ngroups)
		for i, gi := range order {
			vals[i] = finalizeAgg(&specs[a], m.accs[a][gi])
		}
		s, err := series.New(specs[a].name, specs[a].dtype, vals)
		if err != nil {
			return DataFrame{}, err
		}
		out = append(out, s)
	}
	return New(NewInput{Series: out})
}
