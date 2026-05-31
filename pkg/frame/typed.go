package frame

import (
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
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(keep))
	}
	df, err := New(NewInput{Series: out})
	return df, true, err
}

// filterKeep evaluates plan over cols and returns the surviving row indices in
// ascending order. ok is false when the caller must fall back to the row-wise
// evaluator: an evaluation error, or a null predicate result (which the
// row-wise Filter treats as a hard error). At or above parallelFilterThreshold
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
		return simd.CompressIndices(mask), true
	}
	return filterKeepParallel(plan, cols, d.height, workers)
}

// filterKeepParallel evaluates plan over height rows across `workers`
// goroutines, each handling a contiguous row range via a zero-copy chunk.View,
// and returns the surviving global row indices in ascending order. Each worker
// writes only its own result slot, so the stitch after the barrier is race-free
// and order-preserving.
func filterKeepParallel(plan *evalbatch.Plan, cols map[string]*chunk.Column, height, workers int) ([]int, bool) {
	ranges := partitionRanges(height, workers)
	type rangeResult struct {
		idx      []int
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
			// CompressIndices yields window-local indices; shift to global.
			idx := simd.CompressIndices(mask)
			for j := range idx {
				idx[j] += start
			}
			results[i].idx = idx
		}(i, rg[0], rg[1])
	}
	wg.Wait()

	total := 0
	for i := range results {
		if results[i].err != nil {
			debugFallback("filter", results[i].err)
			return nil, false
		}
		if results[i].declined {
			debugFallback("filter", "null predicate result")
			return nil, false
		}
		total += len(results[i].idx)
	}
	keep := make([]int, 0, total)
	for i := range results {
		keep = append(keep, results[i].idx...)
	}
	return keep, true
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

	reductions, ok := d.fusedReduce(plan, cols)
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

// fusedReduce evaluates plan over cols and returns the per-column masked
// reduction (one entry per d.order column). ok is false when the caller must
// fall back (predicate eval error or a null predicate result). Above
// parallelFilterThreshold the work is split across GOMAXPROCS workers over
// disjoint contiguous row ranges and the partials are merged in range order.
func (d DataFrame) fusedReduce(plan *evalbatch.Plan, cols map[string]*chunk.Column) ([]colReduction, bool) {
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
		reductions := make([]colReduction, len(d.order))
		for j, name := range d.order {
			f64, _ := cols[name].Float64s()
			s, mn, mx, c := simd.MaskedReduceFloat64(f64, mask, cols[name].Nulls())
			reductions[j] = colReduction{sum: s, min: mn, max: mx, count: c}
		}
		return reductions, true
	}
	return d.fusedReduceParallel(plan, cols, workers)
}

// fusedReduceParallel is the worker-partitioned form of fusedReduce. Each
// worker evaluates the predicate over a zero-copy window and masked-reduces
// every column for that window into its own result slot; the partials are
// merged after the barrier in ascending range order.
func (d DataFrame) fusedReduceParallel(plan *evalbatch.Plan, cols map[string]*chunk.Column, workers int) ([]colReduction, bool) {
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
			red := make([]colReduction, len(d.order))
			for j, name := range d.order {
				c := view[name]
				f64, _ := c.Float64s()
				s, mn, mx, cnt := simd.MaskedReduceFloat64(f64, mask, c.Nulls())
				red[j] = colReduction{sum: s, min: mn, max: mx, count: cnt}
			}
			results[i].red = red
		}(i, rg[0], rg[1])
	}
	wg.Wait()

	reductions := make([]colReduction, len(d.order))
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
