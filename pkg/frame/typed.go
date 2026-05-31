package frame

import (
	"log"
	"os"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/expr/evalbatch"
	"github.com/h0rn3t/gopolars/pkg/series"
	"github.com/h0rn3t/gopolars/pkg/simd"
)

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
	mask, valid, err := plan.EvalBool(cols, d.height)
	if err != nil {
		debugFallback("filter", err)
		return DataFrame{}, false, nil
	}
	for _, v := range valid {
		if !v {
			debugFallback("filter", "null predicate result")
			return DataFrame{}, false, nil
		}
	}
	keep := simd.CompressIndices(mask)
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(keep))
	}
	df, err := New(NewInput{Series: out})
	return df, true, err
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
