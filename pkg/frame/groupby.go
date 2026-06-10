package frame

import (
	"fmt"
	"math"
	"sort"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

type GroupBy struct {
	df   DataFrame
	keys []string
}

func (g GroupBy) Agg(exprs ...expr.Expr) (DataFrame, error) {
	if len(g.keys) == 0 {
		return DataFrame{}, fmt.Errorf("group keys are empty")
	}
	// Build group buckets from the typed backing slices of the key columns via
	// chunk.GroupIDs — no per-row interface boxing or fmt.Sprintf. Allocation
	// scales with the number of distinct groups, not the row count.
	keyColumns := make([]*chunk.Column, len(g.keys))
	for j, key := range g.keys {
		s, ok := g.df.cols[key]
		if !ok {
			return DataFrame{}, fmt.Errorf("group key %s not found", key)
		}
		keyColumns[j] = s.Column()
	}
	// Parallel typed fast path for large frames with associative aggregates;
	// declines (ok=false) to the sequential path below for small frames,
	// non-associative aggregates (n_unique), or disabled typed storage.
	if df, ok, err := g.aggParallel(keyColumns, exprs); err != nil {
		return DataFrame{}, err
	} else if ok {
		return df, nil
	}
	ids, firstRow := chunk.GroupIDs(keyColumns, g.df.height)
	ngroups := len(firstRow)

	counts := make([]int, ngroups)
	for _, gid := range ids {
		counts[gid]++
	}
	buckets := make([][]int, ngroups)
	for gi := range buckets {
		buckets[gi] = make([]int, 0, counts[gi])
	}
	for row, gid := range ids {
		buckets[gid] = append(buckets[gid], row)
	}

	aggCols := make([][]any, len(exprs))
	for gi := range ngroups {
		idxs := buckets[gi]
		for i, aggExpr := range exprs {
			v, err := g.evalAgg(aggExpr, idxs)
			if err != nil {
				return DataFrame{}, err
			}
			aggCols[i] = append(aggCols[i], v)
		}
	}

	out := make([]series.Series, 0, len(g.keys)+len(exprs))
	// Key columns are materialized by a typed gather of the representative
	// (first-seen) row per group — no boxing.
	for j, key := range g.keys {
		out = append(out, series.FromColumn(key, keyColumns[j].Gather(firstRow)))
	}
	for i, aggExpr := range exprs {
		dt, err := g.aggType(aggExpr)
		if err != nil {
			return DataFrame{}, err
		}
		s, err := series.New(aggExpr.Name(), dt, aggCols[i])
		if err != nil {
			return DataFrame{}, err
		}
		out = append(out, s)
	}
	return New(NewInput{Series: out})
}

func (g GroupBy) evalAgg(aggExpr expr.Expr, idxs []int) (any, error) {
	if aggExpr.Kind() != expr.KindAgg {
		return nil, fmt.Errorf("expression %s is not aggregate", aggExpr.Name())
	}
	switch aggExpr.Op() {
	case "count":
		return int64(len(idxs)), nil
	case "sum":
		target := aggExpr.Target()
		if target == nil {
			return nil, fmt.Errorf("sum target is nil")
		}
		return g.sum(*target, idxs)
	case "mean":
		target := aggExpr.Target()
		if target == nil {
			return nil, fmt.Errorf("mean target is nil")
		}
		sum, count, err := g.sumAndCount(*target, idxs)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, nil
		}
		switch s := sum.(type) {
		case int64:
			return float64(s) / float64(count), nil
		case float64:
			return s / float64(count), nil
		}
	case "min":
		return g.extreme(aggExpr, idxs, true)
	case "max":
		return g.extreme(aggExpr, idxs, false)
	case "n_unique":
		target := aggExpr.Target()
		if target == nil {
			return nil, fmt.Errorf("n_unique target is nil")
		}
		seen := map[string]struct{}{}
		for _, idx := range idxs {
			v, err := expr.Eval(*target, rowAccessor{df: g.df, row: idx})
			if err != nil {
				return nil, err
			}
			seen[fmt.Sprintf("%v", v)] = struct{}{}
		}
		return int64(len(seen)), nil
	case "count_distinct":
		// SQL COUNT(DISTINCT col): distinct non-null values.
		target := aggExpr.Target()
		if target == nil {
			return nil, fmt.Errorf("count_distinct target is nil")
		}
		seen := map[string]struct{}{}
		for _, idx := range idxs {
			v, err := expr.Eval(*target, rowAccessor{df: g.df, row: idx})
			if err != nil {
				return nil, err
			}
			if v == nil {
				continue
			}
			seen[fmt.Sprintf("%v", v)] = struct{}{}
		}
		return int64(len(seen)), nil
	case "std", "var", "median":
		target := aggExpr.Target()
		if target == nil {
			return nil, fmt.Errorf("%s target is nil", aggExpr.Op())
		}
		values := make([]float64, 0, len(idxs))
		for _, idx := range idxs {
			v, err := expr.Eval(*target, rowAccessor{df: g.df, row: idx})
			if err != nil {
				return nil, err
			}
			if v == nil {
				continue
			}
			f, ok := toFloat(v)
			if !ok {
				return nil, fmt.Errorf("%s expects numeric values", aggExpr.Op())
			}
			if math.IsNaN(f) {
				continue
			}
			values = append(values, f)
		}
		if aggExpr.Op() == "median" {
			return medianOf(values), nil
		}
		// Sample variance (ddof=1, the Polars default); null below two values.
		if len(values) < 2 {
			return nil, nil
		}
		mean := 0.0
		for _, v := range values {
			mean += v
		}
		mean /= float64(len(values))
		ss := 0.0
		for _, v := range values {
			d := v - mean
			ss += d * d
		}
		variance := ss / float64(len(values)-1)
		if aggExpr.Op() == "var" {
			return variance, nil
		}
		return math.Sqrt(variance), nil
	case "first", "last":
		target := aggExpr.Target()
		if target == nil {
			return nil, fmt.Errorf("%s target is nil", aggExpr.Op())
		}
		if len(idxs) == 0 {
			return nil, nil
		}
		idx := idxs[0]
		if aggExpr.Op() == "last" {
			idx = idxs[len(idxs)-1]
		}
		return expr.Eval(*target, rowAccessor{df: g.df, row: idx})
	}
	return nil, fmt.Errorf("unsupported aggregate %s", aggExpr.Op())
}

// medianOf returns the median of values (mean of the middle two for even
// counts), or nil for an empty slice.
func medianOf(values []float64) any {
	if len(values) == 0 {
		return nil
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func (g GroupBy) sum(target expr.Expr, idxs []int) (any, error) {
	sum, _, err := g.sumAndCount(target, idxs)
	return sum, err
}

func (g GroupBy) sumAndCount(target expr.Expr, idxs []int) (any, int, error) {
	// Fast path: simple column reference — read typed chunk directly.
	if target.Kind() == expr.KindCol {
		s, ok := g.df.cols[target.ColName()]
		if ok {
			col := s.Column()
			if f64s, ok2 := col.Float64s(); ok2 {
				nulls := col.Nulls()
				var sum float64
				count := 0
				for _, idx := range idxs {
					if nulls != nil && nulls[idx] {
						continue
					}
					v := f64s[idx]
					if math.IsNaN(v) {
						continue
					}
					sum += v
					count++
				}
				if count == 0 {
					return nil, 0, nil
				}
				return sum, count, nil
			}
			if i64s, ok2 := col.Int64s(); ok2 {
				nulls := col.Nulls()
				var sum int64
				count := 0
				for _, idx := range idxs {
					if nulls != nil && nulls[idx] {
						continue
					}
					sum += i64s[idx]
					count++
				}
				if count == 0 {
					return nil, 0, nil
				}
				return sum, count, nil
			}
		}
	}
	// Slow path: row-wise expression eval.
	var intSum int64
	var floatSum float64
	isFloat := false
	count := 0
	for _, idx := range idxs {
		v, err := expr.Eval(target, rowAccessor{df: g.df, row: idx})
		if err != nil {
			return nil, 0, err
		}
		if v == nil {
			continue
		}
		switch t := v.(type) {
		case int64:
			intSum += t
			count++
		case float64:
			if math.IsNaN(t) {
				continue
			}
			isFloat = true
			floatSum += t
			count++
		default:
			return nil, 0, fmt.Errorf("sum supports int64/float64")
		}
	}
	if count == 0 {
		return nil, 0, nil
	}
	if isFloat {
		return floatSum + float64(intSum), count, nil
	}
	return intSum, count, nil
}

func (g GroupBy) extreme(aggExpr expr.Expr, idxs []int, isMin bool) (any, error) {
	target := aggExpr.Target()
	if target == nil {
		return nil, fmt.Errorf("aggregate target is nil")
	}
	// Fast path: simple column reference on typed float64/int64 chunk.
	if target.Kind() == expr.KindCol {
		s, ok := g.df.cols[target.ColName()]
		if ok {
			col := s.Column()
			if f64s, ok2 := col.Float64s(); ok2 {
				nulls := col.Nulls()
				hasBest := false
				var best float64
				for _, idx := range idxs {
					if nulls != nil && nulls[idx] {
						continue
					}
					v := f64s[idx]
					if math.IsNaN(v) {
						continue
					}
					if !hasBest || (isMin && v < best) || (!isMin && v > best) {
						best = v
						hasBest = true
					}
				}
				if !hasBest {
					return nil, nil
				}
				return best, nil
			}
			if i64s, ok2 := col.Int64s(); ok2 {
				nulls := col.Nulls()
				hasBest := false
				var best int64
				for _, idx := range idxs {
					if nulls != nil && nulls[idx] {
						continue
					}
					v := i64s[idx]
					if !hasBest || (isMin && v < best) || (!isMin && v > best) {
						best = v
						hasBest = true
					}
				}
				if !hasBest {
					return nil, nil
				}
				return best, nil
			}
		}
	}
	// Slow path: row-wise eval.
	var best any
	hasBest := false
	for _, idx := range idxs {
		v, err := expr.Eval(*target, rowAccessor{df: g.df, row: idx})
		if err != nil {
			return nil, err
		}
		if v == nil {
			continue
		}
		if fv, ok := v.(float64); ok && math.IsNaN(fv) {
			continue
		}
		if !hasBest {
			best = v
			hasBest = true
			continue
		}
		if isMin && lessAny(v, best) {
			best = v
		}
		if !isMin && lessAny(best, v) {
			best = v
		}
	}
	if !hasBest {
		return nil, nil
	}
	return best, nil
}

func (g GroupBy) aggType(aggExpr expr.Expr) (dtypes.DataType, error) {
	if aggExpr.Op() == "count" || aggExpr.Op() == "n_unique" || aggExpr.Op() == "count_distinct" {
		return dtypes.Int64, nil
	}
	if aggExpr.Op() == "mean" || aggExpr.Op() == "std" || aggExpr.Op() == "var" || aggExpr.Op() == "median" {
		return dtypes.Float64, nil
	}
	target := aggExpr.Target()
	if target == nil {
		return "", fmt.Errorf("aggregate target is nil")
	}
	if target.Kind() == expr.KindCol {
		return g.df.cols[target.ColName()].DataType(), nil
	}
	return dtypes.Float64, nil
}
