package frame

import (
	"fmt"
	"math"
	"strings"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	"github.com/eugeneshershen/gopolars/pkg/expr"
	"github.com/eugeneshershen/gopolars/pkg/series"
)

type GroupBy struct {
	df   DataFrame
	keys []string
}

func (g GroupBy) Agg(exprs ...expr.Expr) (DataFrame, error) {
	if len(g.keys) == 0 {
		return DataFrame{}, fmt.Errorf("group keys are empty")
	}
	buckets := map[string][]int{}
	for i := 0; i < g.df.height; i++ {
		parts := make([]string, len(g.keys))
		for j, key := range g.keys {
			s, ok := g.df.cols[key]
			if !ok {
				return DataFrame{}, fmt.Errorf("group key %s not found", key)
			}
			parts[j] = fmt.Sprintf("%v", s.Value(i))
		}
		hash := strings.Join(parts, "|")
		buckets[hash] = append(buckets[hash], i)
	}
	keyCols := make([][]any, len(g.keys))
	aggCols := make([][]any, len(exprs))
	for _, idxs := range buckets {
		for i, key := range g.keys {
			keyCols[i] = append(keyCols[i], g.df.cols[key].Value(idxs[0]))
		}
		for i, aggExpr := range exprs {
			v, err := g.evalAgg(aggExpr, idxs)
			if err != nil {
				return DataFrame{}, err
			}
			aggCols[i] = append(aggCols[i], v)
		}
	}
	out := make([]series.Series, 0, len(g.keys)+len(exprs))
	for i, key := range g.keys {
		dt := g.df.cols[key].DataType()
		s, err := series.New(key, dt, keyCols[i])
		if err != nil {
			return DataFrame{}, err
		}
		out = append(out, s)
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
	}
	return nil, fmt.Errorf("unsupported aggregate %s", aggExpr.Op())
}

func (g GroupBy) sum(target expr.Expr, idxs []int) (any, error) {
	sum, _, err := g.sumAndCount(target, idxs)
	return sum, err
}

func (g GroupBy) sumAndCount(target expr.Expr, idxs []int) (any, int, error) {
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
	if aggExpr.Op() == "count" || aggExpr.Op() == "n_unique" {
		return dtypes.Int64, nil
	}
	if aggExpr.Op() == "mean" {
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
