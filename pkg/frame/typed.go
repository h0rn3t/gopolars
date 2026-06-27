package frame

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/expr/evalbatch"
	"github.com/h0rn3t/gopolars/pkg/series"
	"github.com/h0rn3t/gopolars/pkg/simd"
)

// parallelFilterThreshold is the column height at or above which filterBatch
// evaluates the predicate across multiple worker goroutines. Below it the
// goroutine/stitch overhead outweighs the work and we stay single-threaded —
// this is also the regime where gopolars already beats Polars on small data.
// Calibrated with BenchmarkCrossFilterSum on arm64 (Apple M4 Pro): 10K rows
// stay sequential, 100K+ parallelize.
const parallelFilterThreshold = 1 << 15 // 32768 rows

// useTypedStorage reports whether the vectorized typed-chunk execution paths
// are enabled. Set GOPOLARS_TYPED_STORAGE=0 to roll back to the row-wise
// evaluator for Filter / WithColumns / Select.
func useTypedStorage() bool {
	return os.Getenv("GOPOLARS_TYPED_STORAGE") != "0"
}

// debugFallback emits a diagnostic when a batch path declines and the engine
// falls back to row-wise evaluation. Enabled with GOPOLARS_DEBUG set.
func debugFallback(ctx string, reason any) {
	if os.Getenv("GOPOLARS_DEBUG") == "" {
		return
	}
	log.Printf("gopolars: batch eval fallback (%s): %v", ctx, reason)
}

// chunkColumns returns the typed chunk backing for every column, or nil if any
// column lacks a typed chunk (which forces the row-wise fallback).
func (d DataFrame) chunkColumns() map[string]*chunk.Column {
	m := make(map[string]*chunk.Column, len(d.order))
	for name, s := range d.cols {
		c := s.Column()
		if c == nil {
			return nil
		}
		m[name] = c
	}
	return m
}

// filterBatch attempts to evaluate predicate as a single vectorized bool mask
// and gather the surviving rows. The bool result reports whether the batch path
// handled the predicate; when false the caller must fall back to row-wise eval.
//
// It deliberately declines (falls back) whenever the predicate result carries a
// null, because the row-wise Filter treats a null predicate as a hard error and
// the fallback reproduces that behavior exactly.
func (d DataFrame) filterBatch(predicate expr.Expr) (DataFrame, bool, error) {
	plan, ok := evalbatch.Compile(predicate)
	if !ok {
		debugFallback("filter", "unsupported predicate")
		return DataFrame{}, false, nil
	}
	cols := d.chunkColumns()
	if cols == nil {
		return DataFrame{}, false, nil
	}
	keep, ok := d.filterKeep(plan, cols)
	if !ok {
		return DataFrame{}, false, nil
	}
	df, err := New(NewInput{Series: d.takeColumns(keep)})
	return df, true, err
}

// takeColumns gathers the rows at keep across every column, in column order, so
// the otherwise-serial materialization tail of filter/drop/sort/unique uses the
// idle cores left over after the (already parallel) predicate evaluation. Below
// the threshold, for a single column, or with one CPU it stays sequential to
// avoid goroutine overhead — the regime where gopolars already beats Polars on
// small data.
//
// Two parallel regimes split on column count vs worker count:
//   - At least as many columns as workers: one column per worker (round-robin)
//     already saturates every core, so each column gathers serially and the
//     per-column goroutine fan-out is avoided.
//   - Fewer columns than workers: round-robin would leave cores idle while the
//     frame's most expensive column (a pointer-heavy string gather whose GC
//     write barriers serialize) runs alone, so each column's gather is instead
//     sharded across all workers in turn.
func (d DataFrame) takeColumns(keep []int) []series.Series {
	out := make([]series.Series, len(d.order))
	workers := runtime.GOMAXPROCS(0)
	if len(d.order) <= 1 || len(keep) < parallelFilterThreshold || workers <= 1 {
		for i, name := range d.order {
			out[i] = d.cols[name].Slice(keep)
		}
		return out
	}
	if len(d.order) >= workers {
		w := min(workers, len(d.order))
		var wg sync.WaitGroup
		wg.Add(w)
		for k := range w {
			go func(k int) {
				defer wg.Done()
				for i := k; i < len(d.order); i += w {
					out[i] = d.cols[d.order[i]].Slice(keep)
				}
			}(k)
		}
		wg.Wait()
		return out
	}
	for i, name := range d.order {
		out[i] = d.cols[name].SliceParallel(keep, workers)
	}
	return out
}

// filterKeep evaluates plan over cols and returns the surviving row indices in
// ascending order. ok is false when the caller must fall back to the row-wise
// evaluator: an evaluation error, or a null predicate result (the row-wise
// Filter drops null-predicate rows, matching Polars). At or above parallelFilterThreshold
// the predicate is evaluated across GOMAXPROCS workers over disjoint contiguous
// row ranges; the surviving indices are stitched back in ascending range order
// so the result is identical to a single-threaded evaluation.
func (d DataFrame) filterKeep(plan *evalbatch.Plan, cols map[string]*chunk.Column) (keep []int, ok bool) {
	workers := runtime.GOMAXPROCS(0)
	if d.height < parallelFilterThreshold || workers <= 1 {
		// mask and nulls are read-only views into the evaluated predicate chunk.
		mask, nulls, err := plan.EvalBool(cols, d.height)
		if err != nil {
			debugFallback("filter", err)
			return nil, false
		}
		for _, isNull := range nulls {
			if isNull {
				debugFallback("filter", "null predicate result")
				return nil, false
			}
		}
		return simd.CompressIndices(mask, d.height), true
	}
	return filterKeepParallel(plan, cols, d.height, workers)
}

// filterKeepParallel evaluates plan over height rows across `workers`
// goroutines, each handling a contiguous, 64-bit-word-aligned row range via a
// zero-copy chunk.View, and returns the surviving global row indices in
// ascending order. Because every range starts on a word boundary, each worker's
// local predicate Bitmap maps onto a disjoint span of words in a single shared
// global Bitmap and is placed there with a plain word copy — no per-worker index
// slice and no stitching allocation. The global Bitmap is compressed to indices
// once, after the barrier.
func filterKeepParallel(plan *evalbatch.Plan, cols map[string]*chunk.Column, height, workers int) ([]int, bool) {
	ranges := partitionRangesAligned(height, workers)
	global := simd.BitmapNew(height)
	type rangeResult struct {
		declined bool
		err      error
	}
	results := make([]rangeResult, len(ranges))
	var wg sync.WaitGroup
	for i, rg := range ranges {
		wg.Add(1)
		go func(i, start, end int) {
			defer wg.Done()
			view := make(map[string]*chunk.Column, len(cols))
			for name, c := range cols {
				view[name] = c.View(start, end)
			}
			mask, nulls, err := plan.EvalBool(view, end-start)
			if err != nil {
				results[i].err = err
				return
			}
			for _, isNull := range nulls {
				if isNull {
					results[i].declined = true
					return
				}
			}
			// start is word-aligned, so window word k is global word start/64+k.
			// Ranges are disjoint on word boundaries, so workers never share a
			// destination word and this copy is race-free.
			copy(global[start>>6:], mask)
		}(i, rg[0], rg[1])
	}
	wg.Wait()

	for i := range results {
		if results[i].err != nil {
			debugFallback("filter", results[i].err)
			return nil, false
		}
		if results[i].declined {
			debugFallback("filter", "null predicate result")
			return nil, false
		}
	}
	return simd.CompressIndices(global, height), true
}

// partitionRanges splits [0,n) into up to `workers` contiguous [start,end)
// ranges, mirroring the chunking in DataFrame.parallelForRows.
func partitionRanges(n, workers int) [][2]int {
	workers = max(1, min(workers, n))
	chunkSize := (n + workers - 1) / workers
	ranges := make([][2]int, 0, workers)
	for start := 0; start < n; start += chunkSize {
		end := min(start+chunkSize, n)
		ranges = append(ranges, [2]int{start, end})
	}
	return ranges
}

// partitionRangesAligned is like partitionRanges but rounds the chunk size up to
// a multiple of 64 so every range (except possibly the last) starts and ends on
// a 64-bit word boundary. This lets a worker's local predicate Bitmap be copied
// word-for-word into a shared global Bitmap without any worker touching another
// worker's words. Only the final range may end off-boundary (at n), and its
// partial last word belongs to it alone.
func partitionRangesAligned(n, workers int) [][2]int {
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

// fusedAggOps are the full-frame reductions the fused filter+reduce path
// supports. Anything else falls back to materialize-then-aggregate.
var fusedAggOps = map[string]bool{
	"sum": true, "min": true, "max": true, "count": true, "mean": true,
}

// colReduction is the single-pass masked reduction of one column over the rows
// that survive a predicate (kept and non-null).
type colReduction struct {
	sum, min, max float64
	count         int
}

// merge folds a per-window partial reduction into the running total. min/max
// seed from the first contributing window (windows are processed in ascending
// range order) and combine with plain < / >, preserving the scalar path's
// NaN semantics.
func (r *colReduction) merge(p colReduction) {
	if p.count == 0 {
		return
	}
	if r.count == 0 {
		r.min, r.max = p.min, p.max
	} else {
		if p.min < r.min {
			r.min = p.min
		}
		if p.max > r.max {
			r.max = p.max
		}
	}
	r.sum += p.sum
	r.count += p.count
}

// FilterAggregate computes op over the rows of d that survive predicate, for
// every column, in a single masked pass per column — without materializing the
// filtered frame or building a surviving-index slice. This is the fused
// filter+reduce path. It returns ok=false (the caller must
// materialize-then-aggregate) when op, the predicate, or any column dtype is
// unsupported, or the predicate yields a null. Supported ops are sum, min, max,
// count, and mean over Float64 columns; the result matches
// exec.aggregateFrame applied to d.Filter(predicate).
func (d DataFrame) FilterAggregate(predicate expr.Expr, op string, args []string) (DataFrame, bool, error) {
	_ = args
	if !fusedAggOps[op] {
		return DataFrame{}, false, nil
	}
	plan, ok := evalbatch.Compile(predicate)
	if !ok {
		debugFallback("filter_agg", "unsupported predicate")
		return DataFrame{}, false, nil
	}
	cols := d.chunkColumns()
	if cols == nil {
		return DataFrame{}, false, nil
	}
	for _, name := range d.order {
		if cols[name].DataType() != dtypes.Float64 {
			debugFallback("filter_agg", "non-float64 column")
			return DataFrame{}, false, nil
		}
	}

	reductions, ok := d.fusedReduceFast(plan, cols, d.order, op)
	if !ok {
		reductions, ok = d.fusedReduce(plan, cols, d.order)
	}
	if !ok {
		return DataFrame{}, false, nil
	}

	out := make([]series.Series, 0, len(d.order))
	for j, name := range d.order {
		val, dt := fusedResult(op, reductions[j])
		s, err := series.New(name, dt, []any{val})
		if err != nil {
			return DataFrame{}, false, err
		}
		out = append(out, s)
	}
	df, err := New(NewInput{Series: out})
	return df, true, err
}

// fusedResult maps a column reduction to the (value, dtype) pair that
// exec.aggregateFrame would produce for op over a Float64 column. A zero
// contributing count yields a null, except count which is always Int64.
func fusedResult(op string, r colReduction) (any, dtypes.DataType) {
	switch op {
	case "count":
		return int64(r.count), dtypes.Int64
	case "mean":
		if r.count == 0 {
			return nil, dtypes.Float64
		}
		return r.sum / float64(r.count), dtypes.Float64
	case "min":
		if r.count == 0 {
			return nil, dtypes.Float64
		}
		return r.min, dtypes.Float64
	case "max":
		if r.count == 0 {
			return nil, dtypes.Float64
		}
		return r.max, dtypes.Float64
	default: // sum
		if r.count == 0 {
			return nil, dtypes.Float64
		}
		return r.sum, dtypes.Float64
	}
}

// directResult is the map-valued counterpart of fusedResult for
// FilterAggregateDirect. Because the return type is a plain float64 map (no null
// sentinel), an empty reduction yields 0 for every op — matching the
// "aggregation key set to 0" contract for a zero-survivor filter and the value a
// caller reads from a null lazy result via float64 zero-value.
func directResult(op string, r colReduction) float64 {
	switch op {
	case "count":
		return float64(r.count)
	case "mean":
		if r.count == 0 {
			return 0
		}
		return r.sum / float64(r.count)
	case "min":
		if r.count == 0 {
			return 0
		}
		return r.min
	case "max":
		if r.count == 0 {
			return 0
		}
		return r.max
	default: // sum
		return r.sum
	}
}

// FilterAggregateDirect evaluates predicate to a packed Bitmap and computes op
// over the requested float64 columns in a single masked pass, returning a
// name→value map. It is the eager fused entry point: unlike the lazy
// Filter().<reduce>().Collect() path it builds no logical plan, and unlike the
// eager Filter() path it materializes no filtered DataFrame, no surviving-index
// []int, and no sliced column buffer — only the predicate bitmap and the result
// map are allocated. op is one of sum, min, max, mean, count; cols names the
// columns to aggregate, or all columns in frame order when empty.
//
// A selectivity gate skips the reduction kernel entirely when the predicate
// matches zero rows. Results match exec.aggregateFrame(op) over
// d.Filter(predicate) — i.e. the lazy fused path — including null exclusion and
// NaN propagation, with an empty result reported as 0 (see directResult).
func (d DataFrame) FilterAggregateDirect(predicate expr.Expr, op string, cols []string) (map[string]float64, error) {
	if !fusedAggOps[op] {
		return nil, fmt.Errorf("frame: FilterAggregateDirect: unsupported op %q", op)
	}
	plan, ok := evalbatch.Compile(predicate)
	if !ok {
		return nil, fmt.Errorf("frame: FilterAggregateDirect: predicate not supported by the batch evaluator")
	}
	chunks := d.chunkColumns()
	if chunks == nil {
		return nil, fmt.Errorf("frame: FilterAggregateDirect: typed column storage unavailable")
	}
	names := cols
	if len(names) == 0 {
		names = d.order
	}
	for _, name := range names {
		c, found := chunks[name]
		if !found {
			return nil, fmt.Errorf("frame: FilterAggregateDirect: column %q not found", name)
		}
		if c.DataType() != dtypes.Float64 {
			return nil, fmt.Errorf("frame: FilterAggregateDirect: column %q is not float64", name)
		}
	}

	// Delegate to the shared fused reduction, which evaluates the predicate and
	// masked-reduces the requested columns in a single pass — and, above the
	// parallelism threshold, splits the predicate scan across workers (the
	// dominant cost at low selectivity, so the empty-filter case stays fast).
	// The sequential branch applies the popcount selectivity gate, skipping the
	// reduction kernel entirely when no rows survive.
	reductions, ok := d.fusedReduceFast(plan, chunks, names, op)
	if !ok {
		reductions, ok = d.fusedReduce(plan, chunks, names)
	}
	if !ok {
		return nil, fmt.Errorf("frame: FilterAggregateDirect: null predicate result or evaluation error")
	}
	out := make(map[string]float64, len(names))
	for j, name := range names {
		out[name] = directResult(op, reductions[j])
	}
	return out, nil
}

// fusedReduce evaluates plan over cols and returns the masked reduction of each
// column in names (one entry per name, in order). ok is false when the caller
// must fall back (predicate eval error or a null predicate result). Above
// parallelFilterThreshold the work is split across GOMAXPROCS workers over
// disjoint contiguous row ranges and the partials are merged in range order —
// so the predicate scan, the dominant cost at low selectivity, is parallel.
func (d DataFrame) fusedReduce(plan *evalbatch.Plan, cols map[string]*chunk.Column, names []string) ([]colReduction, bool) {
	workers := runtime.GOMAXPROCS(0)
	if d.height < parallelFilterThreshold || workers <= 1 {
		mask, nulls, err := plan.EvalBool(cols, d.height)
		if err != nil {
			debugFallback("filter_agg", err)
			return nil, false
		}
		for _, isNull := range nulls {
			if isNull {
				debugFallback("filter_agg", "null predicate result")
				return nil, false
			}
		}
		// Selectivity gate: with no survivors every column reduces to the empty
		// (count 0) result, so skip the per-column MaskedReduceFloat64 kernel
		// entirely and return zeroed reductions directly.
		if simd.BitmapPopcount(mask, d.height) == 0 {
			return make([]colReduction, len(names)), true
		}
		reductions := make([]colReduction, len(names))
		for j, name := range names {
			f64, _ := cols[name].Float64s()
			s, mn, mx, c := simd.MaskedReduceFloat64(f64, mask, cols[name].Nulls())
			reductions[j] = colReduction{sum: s, min: mn, max: mx, count: c}
		}
		return reductions, true
	}
	return d.fusedReduceParallel(plan, cols, names, workers)
}

// fusedReduceParallel is the worker-partitioned form of fusedReduce. Each
// worker evaluates the predicate over a zero-copy window and masked-reduces
// every column for that window into its own result slot; the partials are
// merged after the barrier in ascending range order.
func (d DataFrame) fusedReduceParallel(plan *evalbatch.Plan, cols map[string]*chunk.Column, names []string, workers int) ([]colReduction, bool) {
	ranges := partitionRanges(d.height, workers)
	type winResult struct {
		red      []colReduction
		declined bool
		err      error
	}
	results := make([]winResult, len(ranges))
	var wg sync.WaitGroup
	for i, rg := range ranges {
		wg.Add(1)
		go func(i, start, end int) {
			defer wg.Done()
			view := make(map[string]*chunk.Column, len(cols))
			for name, c := range cols {
				view[name] = c.View(start, end)
			}
			mask, nulls, err := plan.EvalBool(view, end-start)
			if err != nil {
				results[i].err = err
				return
			}
			for _, isNull := range nulls {
				if isNull {
					results[i].declined = true
					return
				}
			}
			red := make([]colReduction, len(names))
			for j, name := range names {
				c := view[name]
				f64, _ := c.Float64s()
				s, mn, mx, cnt := simd.MaskedReduceFloat64(f64, mask, c.Nulls())
				red[j] = colReduction{sum: s, min: mn, max: mx, count: cnt}
			}
			results[i].red = red
		}(i, rg[0], rg[1])
	}
	wg.Wait()

	reductions := make([]colReduction, len(names))
	for i := range results {
		if results[i].err != nil {
			debugFallback("filter_agg", results[i].err)
			return nil, false
		}
		if results[i].declined {
			debugFallback("filter_agg", "null predicate result")
			return nil, false
		}
		for j := range reductions {
			reductions[j].merge(results[i].red[j])
		}
	}
	return reductions, true
}

// fusedReduceFast handles the single-column `col <cmp> lit` float64 predicate
// with a single-pass kernel that builds no predicate bitmap: it scans the value
// array once, selecting passing rows inline. It returns ok=false (so the caller
// falls back to the bitmap fusedReduce path) whenever the predicate shape, the
// column count, or the column dtype is not eligible. Results are identical to
// the bitmap path (sum within float reduction-order tolerance).
func (d DataFrame) fusedReduceFast(plan *evalbatch.Plan, cols map[string]*chunk.Column, names []string, op string) ([]colReduction, bool) {
	if len(names) != 1 {
		return nil, false
	}
	colName, cmp, litF, ok := plan.AsColCmpLit()
	if !ok || colName != names[0] {
		return nil, false
	}
	c, found := cols[colName]
	if !found || c.DataType() != dtypes.Float64 {
		return nil, false
	}
	f64, ok := c.Float64s()
	if !ok {
		return nil, false
	}
	return []colReduction{d.reduceWhere(f64, cmp, litF, c.Nulls(), op)}, true
}

// reduceWhere runs the single-pass kernel for op over vals, parallelizing across
// workers above parallelFilterThreshold and merging the per-range partials. Only
// the reduction op needs computing: sum/mean/count use SumWhereFloat64, min/max
// use MinMaxWhereFloat64.
func (d DataFrame) reduceWhere(vals []float64, cmp simd.Cmp, lit float64, nulls []bool, op string) colReduction {
	workers := runtime.GOMAXPROCS(0)
	if len(vals) < parallelFilterThreshold || workers <= 1 {
		return reduceWhereSeq(vals, cmp, lit, nulls, op)
	}
	ranges := partitionRanges(len(vals), workers)
	partials := make([]colReduction, len(ranges))
	var wg sync.WaitGroup
	for i, rg := range ranges {
		wg.Add(1)
		go func(i, start, end int) {
			defer wg.Done()
			var wn []bool
			if nulls != nil {
				wn = nulls[start:end]
			}
			partials[i] = reduceWhereSeq(vals[start:end], cmp, lit, wn, op)
		}(i, rg[0], rg[1])
	}
	wg.Wait()
	var out colReduction
	for i := range partials {
		out.merge(partials[i])
	}
	return out
}

func reduceWhereSeq(vals []float64, cmp simd.Cmp, lit float64, nulls []bool, op string) colReduction {
	switch op {
	case "min", "max":
		mn, mx, c := simd.MinMaxWhereFloat64(vals, cmp, lit, nulls)
		return colReduction{min: mn, max: mx, count: c}
	default: // sum, mean, count
		s, c := simd.SumWhereFloat64(vals, cmp, lit, nulls)
		return colReduction{sum: s, count: c}
	}
}

// batchEvalColumn attempts to evaluate a projection/with-columns expression to a
// typed chunk in one pass. It returns ok=false (forcing row-wise eval) when the
// expression is unsupported, evaluation errors, or the result is entirely null
// (in which case the row-wise path's Float64 dtype inference must be preserved).
//
// It is invoked only after the existing vectorized fast paths decline, so it
// never changes the semantics of those paths; for the expressions it does handle
// the output matches the row-wise expr.Eval result (verified by conformance
// tests in pkg/expr/evalbatch).
func (d DataFrame) batchEvalColumn(e expr.Expr, name string) (series.Series, bool) {
	plan, ok := evalbatch.Compile(e)
	if !ok {
		return series.Series{}, false
	}
	cols := d.chunkColumns()
	if cols == nil {
		return series.Series{}, false
	}
	rc, err := plan.Eval(cols, d.height)
	if err != nil {
		debugFallback(name, err)
		return series.Series{}, false
	}
	allNull := d.height > 0
	for i := 0; i < d.height; i++ {
		if !rc.IsNull(i) {
			allNull = false
			break
		}
	}
	if allNull {
		debugFallback(name, "all-null result")
		return series.Series{}, false
	}
	return series.FromColumn(name, rc), true
}
