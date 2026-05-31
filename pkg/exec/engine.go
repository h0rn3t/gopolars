package exec

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
	"github.com/h0rn3t/gopolars/pkg/plan/optimizer"
	"github.com/h0rn3t/gopolars/pkg/plan/physical"
	"github.com/h0rn3t/gopolars/pkg/series"
)

type Engine struct{}

func New() Engine {
	return Engine{}
}

func (e Engine) Execute(ctx context.Context, source frame.DataFrame, nodes []logical.Node) (frame.DataFrame, error) {
	_ = ctx
	optimized := optimizer.Optimize(nodes)
	return executeOptimized(source, optimized)
}

func (e Engine) ExecuteStreaming(ctx context.Context, source frame.DataFrame, nodes []logical.Node, chunkSize int) (frame.DataFrame, error) {
	_ = ctx
	if chunkSize <= 0 || source.Height() <= chunkSize {
		return e.Execute(ctx, source, nodes)
	}
	optimized := optimizer.Optimize(nodes)
	if hasStatefulNode(optimized) {
		return executeOptimized(source, optimized)
	}
	streamNodes, finalLimit := extractGlobalLimit(optimized)
	parts := splitFrames(source, chunkSize)
	streamed := make([]frame.DataFrame, 0, len(parts))
	for _, part := range parts {
		next, err := executeOptimized(part, streamNodes)
		if err != nil {
			return frame.DataFrame{}, err
		}
		streamed = append(streamed, next)
	}
	merged, err := concatFrames(streamed)
	if err != nil {
		return frame.DataFrame{}, err
	}
	if finalLimit >= 0 {
		merged = merged.Limit(finalLimit)
	}
	return merged, nil
}

func executeOptimized(source frame.DataFrame, optimized []logical.Node) (frame.DataFrame, error) {
	physicalPlan := physical.Build(optimized)
	scheduler := NewScheduler()
	current := source
	err := scheduler.Run(physicalPlan, func(op physical.Operator) error {
		n := op.Node
		switch n.Type {
		case logical.NodeScan:
			return nil
		case logical.NodeSelect:
			next, err := current.Select(n.Exprs...)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeFilter:
			if len(n.Exprs) == 0 {
				return fmt.Errorf("filter node has no expression")
			}
			next, err := current.Filter(n.Exprs[0])
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeWithCols:
			next, err := current.WithColumns(n.Exprs...)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeSort:
			next, err := current.Sort(frame.SortInput{By: n.Columns, Descending: n.Descending})
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeLimit:
			current = current.Limit(n.IntValue)
			return nil
		case logical.NodeTail:
			current = current.Tail(n.IntValue)
			return nil
		case logical.NodeSlice:
			length := 0
			if len(n.Strings) > 0 {
				if parsed, err := strconv.Atoi(n.Strings[0]); err == nil {
					length = parsed
				}
			}
			current = current.Slice(n.IntValue, length)
			return nil
		case logical.NodeGatherEvery:
			step := 1
			if len(n.Strings) > 0 {
				if parsed, err := strconv.Atoi(n.Strings[0]); err == nil {
					step = parsed
				}
			}
			current = current.GatherEvery(step, n.IntValue)
			return nil
		case logical.NodeReverse:
			current = current.Reverse()
			return nil
		case logical.NodeRename:
			mapping := map[string]string{}
			for i := 0; i < len(n.Strings)-1; i += 2 {
				mapping[n.Strings[i]] = n.Strings[i+1]
			}
			next, err := current.Rename(mapping)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeUnique:
			next, err := current.Unique(n.Columns...)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeFillNull:
			if len(n.Exprs) == 0 {
				return fmt.Errorf("fill_null node has no value expression")
			}
			next, err := current.FillNull(n.Exprs[0].Value())
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeDropNulls:
			current = current.DropNulls(n.Columns...)
			return nil
		case logical.NodeDropNans:
			current = current.DropNaNs(n.Columns...)
			return nil
		case logical.NodeDrop:
			next, err := current.Drop(n.Columns...)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeWindow:
			next, err := applyWindows(current, n.Windows)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeExplode:
			next, err := current.Explode(n.Columns...)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeFlatten:
			if len(n.Columns) == 0 {
				return fmt.Errorf("flatten node has no target column")
			}
			next, err := current.FlattenStruct(n.Columns[0], n.Prefix)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeUnnest:
			next, err := current.Unnest(n.Columns...)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeMelt, logical.NodeUnpivot:
			if len(n.Strings) < 3 {
				return fmt.Errorf("melt/unpivot node metadata is incomplete")
			}
			idCount, err := strconv.Atoi(n.Strings[2])
			if err != nil {
				return err
			}
			if idCount < 0 || idCount > len(n.Columns) {
				return fmt.Errorf("invalid melt/unpivot id count")
			}
			var next frame.DataFrame
			if n.Type == logical.NodeMelt {
				next, err = current.Melt(n.Columns[:idCount], n.Columns[idCount:], n.Strings[0], n.Strings[1])
			} else {
				next, err = current.Unpivot(n.Columns[:idCount], n.Columns[idCount:], n.Strings[0], n.Strings[1])
			}
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeWithRowIdx:
			if len(n.Strings) < 1 {
				return fmt.Errorf("with_row_index node missing name")
			}
			offset := int64(0)
			if len(n.Strings) > 1 {
				if parsed, err := strconv.ParseInt(n.Strings[1], 10, 64); err == nil {
					offset = parsed
				}
			}
			next, err := current.WithRowIndex(n.Strings[0], offset)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeShift:
			next, err := current.Shift(n.IntValue)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeSetSorted:
			if len(n.Columns) < 1 {
				return fmt.Errorf("set_sorted node missing column")
			}
			next, err := current.SetSorted(n.Columns[0])
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeCast:
			mapping := map[string]dtypes.DataType{}
			for i := 0; i < len(n.Strings)-1; i += 2 {
				mapping[n.Strings[i]] = dtypes.DataType(n.Strings[i+1])
			}
			next, err := current.Cast(mapping)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeFillNaN:
			val := 0.0
			if len(n.Strings) > 0 {
				if parsed, err := strconv.ParseFloat(n.Strings[0], 64); err == nil {
					val = parsed
				}
			}
			next, err := current.FillNaN(val)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeInterpolate:
			next, err := current.Interpolate(n.Columns...)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeFrameAgg:
			if len(n.Strings) < 1 {
				return fmt.Errorf("frame_agg node missing aggregation type")
			}
			next, err := aggregateFrame(current, n.Strings[0], n.Strings[1:])
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeUpdate:
			if len(n.Plan) == 0 {
				return fmt.Errorf("update node missing plan")
			}
			other, err := executeOptimized(source, n.Plan)
			if err != nil {
				return err
			}
			next, err := current.Update(other)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodePivot:
			if len(n.Columns) < 3 {
				return fmt.Errorf("pivot node columns are incomplete")
			}
			agg := ""
			if len(n.Strings) > 0 {
				agg = n.Strings[0]
			}
			next, err := current.Pivot([]string{n.Columns[0]}, n.Columns[1], n.Columns[2], agg)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeRolling:
			if len(n.Columns) < 3 || len(n.Strings) < 2 {
				return fmt.Errorf("rolling node metadata is incomplete")
			}
			windowNS, err := strconv.ParseInt(n.Strings[0], 10, 64)
			if err != nil {
				return err
			}
			minRows, err := strconv.Atoi(n.Strings[1])
			if err != nil {
				return err
			}
			closed := ""
			if len(n.Strings) > 2 {
				closed = n.Strings[2]
			}
			next, err := current.RollingMean(n.Columns[0], n.Columns[1], time.Duration(windowNS), minRows, n.Columns[2], closed)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeDynamic:
			if len(n.Columns) < 2 || len(n.Strings) < 5 || len(n.Exprs) == 0 {
				return fmt.Errorf("group_by_dynamic node metadata is incomplete")
			}
			everyNS, err := strconv.ParseInt(n.Strings[0], 10, 64)
			if err != nil {
				return err
			}
			periodNS, err := strconv.ParseInt(n.Strings[1], 10, 64)
			if err != nil {
				return err
			}
			offsetNS, err := strconv.ParseInt(n.Strings[2], 10, 64)
			if err != nil {
				return err
			}
			next, err := current.GroupByDynamic(
				n.Columns[0],
				time.Duration(everyNS),
				time.Duration(periodNS),
				time.Duration(offsetNS),
				n.Strings[3],
				n.Strings[4],
				n.Columns[1],
				n.Exprs[0],
			)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeAggregate:
			next, err := current.GroupBy(n.Columns...).Agg(n.Exprs...)
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeJoin:
			if n.Join == nil {
				return fmt.Errorf("join node is missing join payload")
			}
			next, err := current.Join(frame.JoinInput{
				Other:         n.Join.Other,
				LeftOn:        n.Join.LeftOn,
				RightOn:       n.Join.RightOn,
				How:           n.Join.How,
				Suffix:        n.Join.Suffix,
				AsofDirection: n.Join.AsofDirection,
				AsofTolerance: n.Join.AsofTolerance,
			})
			if err != nil {
				return err
			}
			current = next
			return nil
		case logical.NodeSetOp:
			if len(n.Strings) == 0 {
				return fmt.Errorf("set_op node is missing operation")
			}
			right, err := executeOptimized(source, n.Plan)
			if err != nil {
				return err
			}
			next, err := applySetOp(current, right, n.Strings[0])
			if err != nil {
				return err
			}
			current = next
			return nil
		default:
			return fmt.Errorf("unsupported node type %s", n.Type)
		}
	})
	if err != nil {
		return frame.DataFrame{}, err
	}
	return current, nil
}

func applySetOp(left frame.DataFrame, right frame.DataFrame, op string) (frame.DataFrame, error) {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "union":
		return unionFrames(left, right, false)
	case "union all":
		return unionFrames(left, right, true)
	case "intersect":
		return intersectFrames(left, right)
	case "except":
		return exceptFrames(left, right)
	default:
		return frame.DataFrame{}, fmt.Errorf("unsupported set operation %s", op)
	}
}

func unionFrames(left frame.DataFrame, right frame.DataFrame, all bool) (frame.DataFrame, error) {
	merged, err := frame.ConcatVertical(left, right)
	if err != nil {
		return frame.DataFrame{}, err
	}
	if all {
		return merged, nil
	}
	return merged.Unique()
}

func intersectFrames(left frame.DataFrame, right frame.DataFrame) (frame.DataFrame, error) {
	rightSet := map[string]struct{}{}
	for row := 0; row < right.Height(); row++ {
		rightSet[rowKey(right, row)] = struct{}{}
	}
	idx := make([]int, 0, left.Height())
	for row := 0; row < left.Height(); row++ {
		if _, ok := rightSet[rowKey(left, row)]; ok {
			idx = append(idx, row)
		}
	}
	return selectRows(left, idx)
}

func exceptFrames(left frame.DataFrame, right frame.DataFrame) (frame.DataFrame, error) {
	rightSet := map[string]struct{}{}
	for row := 0; row < right.Height(); row++ {
		rightSet[rowKey(right, row)] = struct{}{}
	}
	idx := make([]int, 0, left.Height())
	for row := 0; row < left.Height(); row++ {
		if _, ok := rightSet[rowKey(left, row)]; !ok {
			idx = append(idx, row)
		}
	}
	return selectRows(left, idx)
}

func selectRows(df frame.DataFrame, idx []int) (frame.DataFrame, error) {
	out := make([]series.Series, 0, df.Width())
	for _, name := range df.Columns() {
		s, _ := df.Series(name)
		out = append(out, s.Slice(idx))
	}
	return frame.New(frame.NewInput{Series: out})
}

func rowKey(df frame.DataFrame, row int) string {
	key := ""
	for _, c := range df.Columns() {
		s, _ := df.Series(c)
		key += fmt.Sprintf("|%v", s.Value(row))
	}
	return key
}

func hasStatefulNode(nodes []logical.Node) bool {
	for _, n := range nodes {
		switch n.Type {
		case logical.NodeSort, logical.NodeJoin, logical.NodeAggregate, logical.NodeWindow, logical.NodePivot, logical.NodeSetOp, logical.NodeRolling, logical.NodeDynamic:
			return true
		}
	}
	return false
}

func splitFrames(source frame.DataFrame, chunkSize int) []frame.DataFrame {
	if source.Height() == 0 {
		return []frame.DataFrame{source}
	}
	out := make([]frame.DataFrame, 0, (source.Height()+chunkSize-1)/chunkSize)
	for start := 0; start < source.Height(); start += chunkSize {
		end := start + chunkSize
		if end > source.Height() {
			end = source.Height()
		}
		idx := make([]int, 0, end-start)
		for i := start; i < end; i++ {
			idx = append(idx, i)
		}
		cols := make([]series.Series, 0, source.Width())
		for _, name := range source.Columns() {
			s, _ := source.Series(name)
			cols = append(cols, s.Slice(idx))
		}
		part, _ := frame.New(frame.NewInput{Series: cols})
		out = append(out, part)
	}
	return out
}

func concatFrames(parts []frame.DataFrame) (frame.DataFrame, error) {
	if len(parts) == 0 {
		return frame.DataFrame{}, nil
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	// Use chunk-level concat to avoid per-element []any boxing.
	others := parts[1:]
	return frame.ConcatVertical(parts[0], others...)
}

func extractGlobalLimit(nodes []logical.Node) ([]logical.Node, int) {
	limit := -1
	out := make([]logical.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Type != logical.NodeLimit {
			out = append(out, n)
			continue
		}
		if limit == -1 || n.IntValue < limit {
			limit = n.IntValue
		}
	}
	return out, limit
}

func applyWindows(df frame.DataFrame, windows []logical.WindowSpec) (frame.DataFrame, error) {
	current := df
	for _, w := range windows {
		values := make([]any, current.Height())
		partitions := map[string][]int{}
		for i := 0; i < current.Height(); i++ {
			key := ""
			for _, c := range w.PartitionBy {
				s, ok := current.Series(c)
				if !ok {
					return frame.DataFrame{}, fmt.Errorf("window partition column %s not found", c)
				}
				key += fmt.Sprintf("|%v", s.Value(i))
			}
			partitions[key] = append(partitions[key], i)
		}
		for _, idxs := range partitions {
			ordered := make([]int, len(idxs))
			copy(ordered, idxs)
			if len(w.OrderBy) > 0 {
				sort.Slice(ordered, func(i, j int) bool {
					li := ordered[i]
					rj := ordered[j]
					for k, col := range w.OrderBy {
						s, _ := current.Series(col)
						lv := s.Value(li)
						rv := s.Value(rj)
						cmp := compareForOrder(lv, rv)
						if cmp == 0 {
							continue
						}
						desc := k < len(w.Descending) && w.Descending[k]
						if desc {
							return cmp > 0
						}
						return cmp < 0
					}
					return li < rj
				})
			}
			switch w.Func {
			case "row_number":
				for i, row := range ordered {
					values[row] = int64(i + 1)
				}
			default:
				agg, err := computePartitionAgg(current, ordered, w.Func, w.Target)
				if err != nil {
					return frame.DataFrame{}, err
				}
				for _, row := range ordered {
					values[row] = agg
				}
			}
		}
		dt := dtypes.Float64
		switch w.Func {
		case "count", "row_number":
			dt = dtypes.Int64
		case "sum", "min", "max":
			if s, ok := current.Series(w.Target); ok {
				dt = s.DataType()
			}
		}
		col, err := series.New(w.Alias, dt, values)
		if err != nil {
			return frame.DataFrame{}, err
		}
		current, err = appendSeries(current, col)
		if err != nil {
			return frame.DataFrame{}, err
		}
	}
	return current, nil
}

func computePartitionAgg(df frame.DataFrame, rows []int, fn string, target string) (any, error) {
	if fn == "count" {
		if target == "*" || target == "" {
			return int64(len(rows)), nil
		}
		s, ok := df.Series(target)
		if !ok {
			return nil, fmt.Errorf("window target column %s not found", target)
		}
		count := int64(0)
		for _, r := range rows {
			if s.Value(r) != nil {
				count++
			}
		}
		return count, nil
	}
	s, ok := df.Series(target)
	if !ok {
		return nil, fmt.Errorf("window target column %s not found", target)
	}
	switch fn {
	case "sum", "mean":
		total := float64(0)
		count := 0
		for _, r := range rows {
			v := s.Value(r)
			if v == nil {
				continue
			}
			f, ok := toFloat(v)
			if !ok {
				return nil, fmt.Errorf("window %s expects numeric values", fn)
			}
			total += f
			count++
		}
		if fn == "sum" {
			if s.DataType() == dtypes.Int64 {
				return int64(total), nil
			}
			return total, nil
		}
		if count == 0 {
			return nil, nil
		}
		return total / float64(count), nil
	case "min", "max":
		var best any
		first := true
		for _, r := range rows {
			v := s.Value(r)
			if v == nil {
				continue
			}
			if first {
				best = v
				first = false
				continue
			}
			c := compareForOrder(v, best)
			if (fn == "min" && c < 0) || (fn == "max" && c > 0) {
				best = v
			}
		}
		return best, nil
	}
	return nil, fmt.Errorf("unsupported window function %s", fn)
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case int64:
		return float64(t), true
	case float64:
		return t, true
	default:
		return 0, false
	}
}

func compareForOrder(left any, right any) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return -1
	}
	if right == nil {
		return 1
	}
	switch l := left.(type) {
	case int64:
		r, ok := right.(int64)
		if !ok {
			return 0
		}
		if l < r {
			return -1
		}
		if l > r {
			return 1
		}
	case float64:
		r, ok := right.(float64)
		if !ok {
			return 0
		}
		if l < r {
			return -1
		}
		if l > r {
			return 1
		}
	case string:
		r, ok := right.(string)
		if !ok {
			return 0
		}
		if l < r {
			return -1
		}
		if l > r {
			return 1
		}
	case time.Time:
		r, ok := right.(time.Time)
		if !ok {
			return 0
		}
		if l.Before(r) {
			return -1
		}
		if l.After(r) {
			return 1
		}
	}
	return 0
}

func appendSeries(df frame.DataFrame, s series.Series) (frame.DataFrame, error) {
	cols := make([]series.Series, 0, df.Width()+1)
	replaced := false
	for _, name := range df.Columns() {
		col, _ := df.Series(name)
		if name == s.Name() {
			cols = append(cols, s)
			replaced = true
			continue
		}
		cols = append(cols, col.Clone())
	}
	if !replaced {
		cols = append(cols, s)
	}
	return frame.New(frame.NewInput{Series: cols})
}

func aggregateFrame(df frame.DataFrame, op string, args []string) (frame.DataFrame, error) {
	out := make([]series.Series, 0, df.Width())
	for _, name := range df.Columns() {
		s, _ := df.Series(name)
		var val any
		var dt = dtypes.Float64
		switch op {
		case "max":
			var best any
			has := false
			for i := 0; i < s.Len(); i++ {
				v := s.Value(i)
				if v == nil {
					continue
				}
				if !has || compareForOrder(v, best) > 0 {
					best = v
					has = true
				}
			}
			val = best
			dt = s.DataType()
		case "min":
			var best any
			has := false
			for i := 0; i < s.Len(); i++ {
				v := s.Value(i)
				if v == nil {
					continue
				}
				if !has || compareForOrder(v, best) < 0 {
					best = v
					has = true
				}
			}
			val = best
			dt = s.DataType()
		case "count":
			val = int64(s.Len())
			for i := 0; i < s.Len(); i++ {
				if s.IsNull(i) {
					val = val.(int64) - 1
				}
			}
			dt = dtypes.Int64
		case "null_count":
			val = int64(0)
			for i := 0; i < s.Len(); i++ {
				if s.IsNull(i) {
					val = val.(int64) + 1
				}
			}
			dt = dtypes.Int64
		case "sum":
			sum := float64(0)
			dt = dtypes.Float64
			allInt := true
			has := false
			for i := 0; i < s.Len(); i++ {
				if s.IsNull(i) {
					continue
				}
				has = true
				v := s.Value(i)
				switch t := v.(type) {
				case int64:
					sum += float64(t)
				case float64:
					sum += t
					allInt = false
				}
			}
			if !has {
				val = nil
			} else if allInt && s.DataType() == dtypes.Int64 {
				val = int64(sum)
				dt = dtypes.Int64
			} else {
				val = sum
			}
		case "mean":
			sum := float64(0)
			count := 0
			for i := 0; i < s.Len(); i++ {
				if s.IsNull(i) {
					continue
				}
				v := s.Value(i)
				switch t := v.(type) {
				case int64:
					sum += float64(t)
					count++
				case float64:
					sum += t
					count++
				}
			}
			if count == 0 {
				val = nil
			} else {
				val = sum / float64(count)
			}
		case "median":
			nums := make([]float64, 0, s.Len())
			for i := 0; i < s.Len(); i++ {
				if s.IsNull(i) {
					continue
				}
				v := s.Value(i)
				switch t := v.(type) {
				case int64:
					nums = append(nums, float64(t))
				case float64:
					nums = append(nums, t)
				}
			}
			if len(nums) == 0 {
				val = nil
			} else {
				sort.Float64s(nums)
				mid := len(nums) / 2
				if len(nums)%2 == 0 {
					val = (nums[mid-1] + nums[mid]) / 2.0
				} else {
					val = nums[mid]
				}
			}
		case "std", "var":
			sum := float64(0)
			count := 0
			for i := 0; i < s.Len(); i++ {
				if s.IsNull(i) {
					continue
				}
				v := s.Value(i)
				switch t := v.(type) {
				case int64:
					sum += float64(t)
					count++
				case float64:
					sum += t
					count++
				}
			}
			if count == 0 {
				val = nil
			} else {
				mean := sum / float64(count)
				variance := float64(0)
				for i := 0; i < s.Len(); i++ {
					if s.IsNull(i) {
						continue
					}
					v := s.Value(i)
					diff := float64(0)
					switch t := v.(type) {
					case int64:
						diff = float64(t) - mean
					case float64:
						diff = t - mean
					}
					variance += diff * diff
				}
				variance /= float64(count)
				if op == "std" {
					val = math.Sqrt(variance)
				} else {
					val = variance
				}
			}
		case "quantile":
			q := 0.5
			if len(args) > 0 {
				if parsed, err := strconv.ParseFloat(args[0], 64); err == nil {
					q = parsed
				}
			}
			nums := make([]float64, 0, s.Len())
			for i := 0; i < s.Len(); i++ {
				if s.IsNull(i) {
					continue
				}
				v := s.Value(i)
				switch t := v.(type) {
				case int64:
					nums = append(nums, float64(t))
				case float64:
					nums = append(nums, t)
				}
			}
			if len(nums) == 0 {
				val = nil
			} else {
				sort.Float64s(nums)
				idx := float64(len(nums)-1) * q
				lower := int(math.Floor(idx))
				upper := int(math.Ceil(idx))
				if lower == upper {
					val = nums[lower]
				} else {
					weight := idx - float64(lower)
					val = nums[lower]*(1.0-weight) + nums[upper]*weight
				}
			}
		}
		newS, err := series.New(name, dt, []any{val})
		if err != nil {
			return frame.DataFrame{}, err
		}
		out = append(out, newS)
	}
	return frame.New(frame.NewInput{Series: out})
}
