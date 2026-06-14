package frame

import (
	"fmt"
	"math"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
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

	rightKeyCols, err := joinKeyColumns(input.Other, input.RightOn)
	if err != nil {
		return DataFrame{}, err
	}
	leftKeyCols, err := joinKeyColumns(left, input.LeftOn)
	if err != nil {
		return DataFrame{}, err
	}

	// Build the probe index with typed, collision-resistant keys (no per-row
	// fmt.Sprintf): a reused scratch buffer encodes each key row's typed values.
	rightIndex := make(map[string][]int, input.Other.height)
	var scratch []byte
	for i := 0; i < input.Other.height; i++ {
		scratch = chunk.AppendRowKey(scratch[:0], rightKeyCols, i)
		rightIndex[string(scratch)] = append(rightIndex[string(scratch)], i)
	}

	// matchedRight is only consulted to emit the unmatched right rows of a
	// right/full join, so it is allocated only then — and as a bounded []bool
	// over the right height rather than a map that grows with match count. For
	// inner/left/semi/anti joins no match-tracking buffer is allocated at all.
	needMatched := input.How == JoinTypeRight || input.How == JoinTypeFull
	var matchedRight []bool
	if needMatched {
		matchedRight = make([]bool, input.Other.height)
	}

	pairs := make([]pair, 0, left.height)
	for i := 0; i < left.height; i++ {
		scratch = chunk.AppendRowKey(scratch[:0], leftKeyCols, i)
		rightRows := rightIndex[string(scratch)]
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
			continue
		}
		if input.How == JoinTypeAnti {
			continue
		}
		for _, rr := range rightRows {
			if needMatched {
				matchedRight[rr] = true
			}
			pairs = append(pairs, pair{left: i, right: rr})
		}
	}
	if needMatched {
		for rr := 0; rr < input.Other.height; rr++ {
			if matchedRight[rr] {
				continue
			}
			pairs = append(pairs, pair{left: -1, right: rr})
		}
	}

	rightIncluded := input.How != JoinTypeSemi && input.How != JoinTypeAnti
	return materializeJoin(left, input.Other, pairs, input.Suffix, rightIncluded)
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

// materializeJoin builds the joined output columns by a single typed gather per
// column over the collected (left,right) index pairs, null-filling unmatched
// (-1) indices. This replaces the prior map[string][]any per-cell assembly,
// cutting allocations from O(rows×columns) to O(columns).
func materializeJoin(left, other DataFrame, pairs []pair, suffix string, rightIncluded bool) (DataFrame, error) {
	if suffix == "" {
		suffix = "_right"
	}
	leftIdx := make([]int, len(pairs))
	rightIdx := make([]int, len(pairs))
	for i, p := range pairs {
		leftIdx[i] = p.left
		rightIdx[i] = p.right
	}

	outSeries := make([]series.Series, 0, len(left.order)+len(other.order))
	for _, name := range left.order {
		col := left.cols[name].Column().Gather(leftIdx)
		outSeries = append(outSeries, series.FromColumn(name, col))
	}
	if rightIncluded {
		for _, name := range other.order {
			outName := name
			if _, exists := left.cols[name]; exists {
				outName = name + suffix
			}
			col := other.cols[name].Column().Gather(rightIdx)
			outSeries = append(outSeries, series.FromColumn(outName, col))
		}
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
