package frame

import (
	"fmt"
	"math"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/series"
)

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

	rightIndex := map[string][]int{}
	for i := 0; i < input.Other.height; i++ {
		k, err := makeJoinKey(input.Other, input.RightOn, i)
		if err != nil {
			return DataFrame{}, err
		}
		rightIndex[k] = append(rightIndex[k], i)
	}

	pairs := make([]pair, 0, left.height)
	matchedRight := map[int]struct{}{}
	for i := 0; i < left.height; i++ {
		k, err := makeJoinKey(left, input.LeftOn, i)
		if err != nil {
			return DataFrame{}, err
		}
		rightRows := rightIndex[k]
		if len(rightRows) == 0 {
			if input.How == JoinTypeAnti {
				pairs = append(pairs, pair{left: i, right: -1})
				continue
			}
			if input.How == JoinTypeLeft || input.How == JoinTypeFull {
				pairs = append(pairs, pair{left: i, right: -1})
			}
			continue
		}
		if input.How == JoinTypeSemi {
			pairs = append(pairs, pair{left: i, right: rightRows[0]})
			for _, rr := range rightRows {
				matchedRight[rr] = struct{}{}
			}
			continue
		}
		if input.How == JoinTypeAnti {
			continue
		}
		for _, rr := range rightRows {
			matchedRight[rr] = struct{}{}
			pairs = append(pairs, pair{left: i, right: rr})
		}
	}
	if input.How == JoinTypeRight || input.How == JoinTypeFull {
		for rr := 0; rr < input.Other.height; rr++ {
			if _, ok := matchedRight[rr]; ok {
				continue
			}
			pairs = append(pairs, pair{left: -1, right: rr})
		}
	}

	rightIncluded := input.How != JoinTypeSemi && input.How != JoinTypeAnti
	outOrder := make([]string, 0, len(left.order)+len(input.Other.order))
	outSchema := make(dtypes.Schema, 0, len(left.schema)+len(input.Other.schema))
	outCols := map[string][]any{}

	for _, name := range left.order {
		outOrder = append(outOrder, name)
		outSchema = append(outSchema, dtypes.Field{Name: name, Type: left.cols[name].DataType()})
		outCols[name] = make([]any, 0, len(pairs))
	}
	if rightIncluded {
		for _, name := range input.Other.order {
			outName := name
			if _, exists := left.cols[name]; exists {
				outName = name + input.Suffix
			}
			outOrder = append(outOrder, outName)
			outSchema = append(outSchema, dtypes.Field{Name: outName, Type: input.Other.cols[name].DataType()})
			outCols[outName] = make([]any, 0, len(pairs))
		}
	}

	for _, p := range pairs {
		for _, name := range left.order {
			if p.left == -1 {
				outCols[name] = append(outCols[name], nil)
				continue
			}
			outCols[name] = append(outCols[name], left.cols[name].Value(p.left))
		}
		if rightIncluded {
			for _, name := range input.Other.order {
				outName := name
				if _, exists := left.cols[name]; exists {
					outName = name + input.Suffix
				}
				if p.right == -1 {
					outCols[outName] = append(outCols[outName], nil)
					continue
				}
				outCols[outName] = append(outCols[outName], input.Other.cols[name].Value(p.right))
			}
		}
	}

	outSeries := make([]series.Series, 0, len(outOrder))
	for _, f := range outSchema {
		s, err := series.New(f.Name, f.Type, outCols[f.Name])
		if err != nil {
			return DataFrame{}, err
		}
		outSeries = append(outSeries, s)
	}
	return New(NewInput{Series: outSeries})
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
	for i := 0; i < left.height; i++ {
		for j := 0; j < input.Other.height; j++ {
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
	outOrder := make([]string, 0, len(left.order)+len(input.Other.order))
	outSchema := make(dtypes.Schema, 0, len(left.schema)+len(input.Other.schema))
	outCols := map[string][]any{}
	for _, name := range left.order {
		outOrder = append(outOrder, name)
		outSchema = append(outSchema, dtypes.Field{Name: name, Type: left.cols[name].DataType()})
		outCols[name] = make([]any, 0, len(pairs))
	}
	if rightIncluded {
		for _, name := range input.Other.order {
			outName := name
			if _, exists := left.cols[name]; exists {
				outName = name + input.Suffix
			}
			outOrder = append(outOrder, outName)
			outSchema = append(outSchema, dtypes.Field{Name: outName, Type: input.Other.cols[name].DataType()})
			outCols[outName] = make([]any, 0, len(pairs))
		}
	}
	for _, p := range pairs {
		for _, name := range left.order {
			if p.left == -1 {
				outCols[name] = append(outCols[name], nil)
				continue
			}
			outCols[name] = append(outCols[name], left.cols[name].Value(p.left))
		}
		if rightIncluded {
			for _, name := range input.Other.order {
				outName := name
				if _, exists := left.cols[name]; exists {
					outName = name + input.Suffix
				}
				if p.right == -1 {
					outCols[outName] = append(outCols[outName], nil)
					continue
				}
				outCols[outName] = append(outCols[outName], input.Other.cols[name].Value(p.right))
			}
		}
	}
	outSeries := make([]series.Series, 0, len(outOrder))
	for _, f := range outSchema {
		s, err := series.New(f.Name, f.Type, outCols[f.Name])
		if err != nil {
			return DataFrame{}, err
		}
		outSeries = append(outSeries, s)
	}
	return New(NewInput{Series: outSeries})
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

func makeJoinKey(df DataFrame, keys []string, row int) (string, error) {
	out := ""
	for _, k := range keys {
		s, ok := df.cols[k]
		if !ok {
			return "", fmt.Errorf("join key %s not found", k)
		}
		col := s.Column()
		// Read typed buffers directly to avoid per-element interface boxing.
		if col.IsNull(row) {
			out += "|<null>"
			continue
		}
		if i64s, ok2 := col.Int64s(); ok2 {
			out += fmt.Sprintf("|%d", i64s[row])
			continue
		}
		if strs, ok2 := col.Strings(); ok2 {
			out += "|" + strs[row]
			continue
		}
		if f64s, ok2 := col.Float64s(); ok2 {
			out += fmt.Sprintf("|%g", f64s[row])
			continue
		}
		out += fmt.Sprintf("|%v", s.Value(row))
	}
	return out, nil
}
