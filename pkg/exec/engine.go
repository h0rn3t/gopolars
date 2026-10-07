package exec

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
	"github.com/h0rn3t/gopolars/pkg/plan/optimizer"
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
	if chunkSize <= 0 || source.Height() <= chunkSize {
		return e.Execute(ctx, source, nodes)
	}
	optimized := optimizer.Optimize(nodes)
	if !streamable(optimized) {
		return executeOptimized(source, optimized)
	}
	streamNodes, finalLimit := extractGlobalLimit(optimized)
	parts := splitFrames(source, chunkSize)
	streamed := make([]frame.DataFrame, 0, len(parts))
	for _, part := range parts {
		next, err := executeOptimized(part, streamNodes)
		if err != nil || (len(streamed) > 0 && !slices.Equal(next.Schema(), streamed[0].Schema())) {
			// Output dtypes are inferred from the values, so a chunk can fail or
			// get another schema where the whole input does not (a computed
			// column that is all null in that chunk, struct fields missing from
			// it); concatenating such chunks would lose values. The whole input
			// decides.
			return executeOptimized(source, optimized)
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

// fuseFilterFrameAgg rewrites a NodeFilter immediately followed by a plain
// full-frame NodeFrameAgg into a single NodeFrameAgg that carries the filter
// predicate in Exprs[0]. The executor then runs the fused single-pass masked
// reduce (frame.DataFrame.FilterAggregate) instead of materializing the
// filtered frame and aggregating it. Only plain FrameAgg nodes (no existing
// Exprs) are fused; everything else is left untouched.
func fuseFilterFrameAgg(nodes []logical.Node) []logical.Node {
	out := make([]logical.Node, 0, len(nodes))
	for i := 0; i < len(nodes); i++ {
		if i+1 < len(nodes) &&
			nodes[i].Type == logical.NodeFilter &&
			len(nodes[i].Exprs) == 1 &&
			nodes[i+1].Type == logical.NodeFrameAgg &&
			len(nodes[i+1].Exprs) == 0 {
			fused := nodes[i+1]
			fused.Exprs = nodes[i].Exprs
			out = append(out, fused)
			i++ // also consume the FrameAgg
			continue
		}
		out = append(out, nodes[i])
	}
	return out
}

func executeOptimized(source frame.DataFrame, optimized []logical.Node) (frame.DataFrame, error) {
	optimized = fuseFilterFrameAgg(optimized)
	current := source
	runNode := func(n logical.Node) (frame.DataFrame, error) {
		switch n.Type {
		case logical.NodeScan:
			return current, nil
		case logical.NodeSelect:
			return current.Select(n.Exprs...)
		case logical.NodeFilter:
			if len(n.Exprs) == 0 {
				return frame.DataFrame{}, fmt.Errorf("filter node has no expression")
			}
			return current.Filter(n.Exprs[0])
		case logical.NodeWithCols:
			return current.WithColumns(n.Exprs...)
		case logical.NodeSort:
			return current.Sort(frame.SortInput{By: n.Columns, Descending: n.Descending, NullsLast: n.NullsLast})
		case logical.NodeLimit:
			return current.Limit(n.IntValue), nil
		case logical.NodeTail:
			return current.Tail(n.IntValue), nil
		case logical.NodeSlice:
			length := 0
			if len(n.Strings) > 0 {
				if parsed, err := strconv.Atoi(n.Strings[0]); err == nil {
					length = parsed
				}
			}
			return current.Slice(n.IntValue, length), nil
		case logical.NodeGatherEvery:
			step := 1
			if len(n.Strings) > 0 {
				if parsed, err := strconv.Atoi(n.Strings[0]); err == nil {
					step = parsed
				}
			}
			return current.GatherEvery(step, n.IntValue), nil
		case logical.NodeReverse:
			return current.Reverse(), nil
		case logical.NodeRename:
			mapping := map[string]string{}
			for i := 0; i < len(n.Strings)-1; i += 2 {
				mapping[n.Strings[i]] = n.Strings[i+1]
			}
			return current.Rename(mapping)
		case logical.NodeUnique:
			return current.Unique(n.Columns...)
		case logical.NodeFillNull:
			if len(n.Exprs) == 0 {
				return frame.DataFrame{}, fmt.Errorf("fill_null node has no value expression")
			}
			return current.FillNull(n.Exprs[0].Value())
		case logical.NodeDropNulls:
			return current.DropNulls(n.Columns...), nil
		case logical.NodeDropNans:
			return current.DropNaNs(n.Columns...), nil
		case logical.NodeDrop:
			return current.Drop(n.Columns...)
		case logical.NodeWindow:
			return applyWindows(current, n.Windows)
		case logical.NodeExplode:
			return current.Explode(n.Columns...)
		case logical.NodeFlatten:
			if len(n.Columns) == 0 {
				return frame.DataFrame{}, fmt.Errorf("flatten node has no target column")
			}
			return current.FlattenStruct(n.Columns[0], n.Prefix)
		case logical.NodeUnnest:
			return current.Unnest(n.Columns...)
		case logical.NodeMelt, logical.NodeUnpivot:
			if len(n.Strings) < 3 {
				return frame.DataFrame{}, fmt.Errorf("melt/unpivot node metadata is incomplete")
			}
			idCount, err := strconv.Atoi(n.Strings[2])
			if err != nil {
				return frame.DataFrame{}, err
			}
			if idCount < 0 || idCount > len(n.Columns) {
				return frame.DataFrame{}, fmt.Errorf("invalid melt/unpivot id count")
			}
			if n.Type == logical.NodeMelt {
				return current.Melt(n.Columns[:idCount], n.Columns[idCount:], n.Strings[0], n.Strings[1])
			}
			return current.Unpivot(n.Columns[:idCount], n.Columns[idCount:], n.Strings[0], n.Strings[1])
		case logical.NodeWithRowIdx:
			if len(n.Strings) < 1 {
				return frame.DataFrame{}, fmt.Errorf("with_row_index node missing name")
			}
			offset := int64(0)
			if len(n.Strings) > 1 {
				if parsed, err := strconv.ParseInt(n.Strings[1], 10, 64); err == nil {
					offset = parsed
				}
			}
			return current.WithRowIndex(n.Strings[0], offset)
		case logical.NodeShift:
			return current.Shift(n.IntValue)
		case logical.NodeSetSorted:
			if len(n.Columns) < 1 {
				return frame.DataFrame{}, fmt.Errorf("set_sorted node missing column")
			}
			return current.SetSorted(n.Columns[0])
		case logical.NodeCast:
			mapping := map[string]dtypes.DataType{}
			for i := 0; i < len(n.Strings)-1; i += 2 {
				mapping[n.Strings[i]] = dtypes.DataType(n.Strings[i+1])
			}
			return current.Cast(mapping)
		case logical.NodeFillNaN:
			if len(n.Strings) < 1 {
				return frame.DataFrame{}, fmt.Errorf("fill_nan node missing value")
			}
			val, err := strconv.ParseFloat(n.Strings[0], 64)
			if err != nil {
				return frame.DataFrame{}, fmt.Errorf("fill_nan value: %w", err)
			}
			return current.FillNaN(val)
		case logical.NodeInterpolate:
			return current.Interpolate(n.Columns...)
		case logical.NodeFrameAgg:
			if len(n.Strings) < 1 {
				return frame.DataFrame{}, fmt.Errorf("frame_agg node missing aggregation type")
			}
			if len(n.Exprs) == 0 {
				return aggregateFrame(current, n.Strings[0], n.Strings[1:])
			}
			// Fused filter+aggregate: try the single-pass masked path; fall
			// back to materialize-then-aggregate when it declines.
			fused, ok, err := current.FilterAggregate(n.Exprs[0], n.Strings[0], n.Strings[1:])
			if err != nil {
				return frame.DataFrame{}, err
			}
			if ok {
				return fused, nil
			}
			filtered, err := current.Filter(n.Exprs[0])
			if err != nil {
				return frame.DataFrame{}, err
			}
			return aggregateFrame(filtered, n.Strings[0], n.Strings[1:])
		case logical.NodeUpdate:
			other, err := otherFrame(n)
			if err != nil {
				return frame.DataFrame{}, err
			}
			return current.Update(other)
		case logical.NodePivot:
			if len(n.Columns) < 3 {
				return frame.DataFrame{}, fmt.Errorf("pivot node columns are incomplete")
			}
			agg := ""
			if len(n.Strings) > 0 {
				agg = n.Strings[0]
			}
			return current.Pivot([]string{n.Columns[0]}, n.Columns[1], n.Columns[2], agg)
		case logical.NodeRolling:
			if len(n.Columns) < 3 || len(n.Strings) < 2 {
				return frame.DataFrame{}, fmt.Errorf("rolling node metadata is incomplete")
			}
			windowNS, err := strconv.ParseInt(n.Strings[0], 10, 64)
			if err != nil {
				return frame.DataFrame{}, err
			}
			minRows, err := strconv.Atoi(n.Strings[1])
			if err != nil {
				return frame.DataFrame{}, err
			}
			closed := ""
			if len(n.Strings) > 2 {
				closed = n.Strings[2]
			}
			return current.RollingMean(n.Columns[0], n.Columns[1], time.Duration(windowNS), minRows, n.Columns[2], closed)
		case logical.NodeDynamic:
			if len(n.Columns) < 2 || len(n.Strings) < 5 || len(n.Exprs) == 0 {
				return frame.DataFrame{}, fmt.Errorf("group_by_dynamic node metadata is incomplete")
			}
			everyNS, err := strconv.ParseInt(n.Strings[0], 10, 64)
			if err != nil {
				return frame.DataFrame{}, err
			}
			periodNS, err := strconv.ParseInt(n.Strings[1], 10, 64)
			if err != nil {
				return frame.DataFrame{}, err
			}
			offsetNS, err := strconv.ParseInt(n.Strings[2], 10, 64)
			if err != nil {
				return frame.DataFrame{}, err
			}
			return current.GroupByDynamic(
				n.Columns[0],
				time.Duration(everyNS),
				time.Duration(periodNS),
				time.Duration(offsetNS),
				n.Strings[3],
				n.Strings[4],
				n.Columns[1],
				n.Exprs[0],
			)
		case logical.NodeAggregate:
			return current.GroupBy(n.Columns...).Agg(n.Exprs...)
		case logical.NodeJoin:
			if n.Join == nil {
				return frame.DataFrame{}, fmt.Errorf("join node is missing join payload")
			}
			return current.Join(frame.JoinInput{
				Other:         n.Join.Other,
				LeftOn:        n.Join.LeftOn,
				RightOn:       n.Join.RightOn,
				How:           n.Join.How,
				Suffix:        n.Join.Suffix,
				AsofDirection: n.Join.AsofDirection,
				AsofTolerance: n.Join.AsofTolerance,
			})
		case logical.NodeSetOp:
			if len(n.Strings) == 0 {
				return frame.DataFrame{}, fmt.Errorf("set_op node is missing operation")
			}
			right, err := otherFrame(n)
			if err != nil {
				return frame.DataFrame{}, err
			}
			return applySetOp(current, right, n.Strings[0])
		default:
			return frame.DataFrame{}, fmt.Errorf("unsupported node type %s", n.Type)
		}
	}
	for _, n := range optimized {
		next, err := runNode(n)
		if err != nil {
			return frame.DataFrame{}, err
		}
		current = next
	}
	return current, nil
}

// otherFrame returns the right-hand frame of a NodeUpdate or NodeSetOp node,
// or the error that building the node recorded.
func otherFrame(n logical.Node) (frame.DataFrame, error) {
	if n.Err != nil {
		return frame.DataFrame{}, n.Err
	}
	if n.Other == nil {
		return frame.DataFrame{}, fmt.Errorf("%s node missing other frame", n.Type)
	}
	return *n.Other, nil
}

func applySetOp(left frame.DataFrame, right frame.DataFrame, op string) (frame.DataFrame, error) {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "union":
		return unionFrames(left, right, false)
	case "union all":
		return unionFrames(left, right, true)
	case "intersect":
		return filterRowsBySet(left, right, true)
	case "except":
		return filterRowsBySet(left, right, false)
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

// filterRowsBySet keeps, in order, the rows of left whose rowKey is among the
// row keys of right (keepMatches) or is not (!keepMatches).
func filterRowsBySet(left frame.DataFrame, right frame.DataFrame, keepMatches bool) (frame.DataFrame, error) {
	rightSet := map[string]struct{}{}
	for row := 0; row < right.Height(); row++ {
		rightSet[rowKey(right, row)] = struct{}{}
	}
	idx := make([]int, 0, left.Height())
	for row := 0; row < left.Height(); row++ {
		if _, ok := rightSet[rowKey(left, row)]; ok == keepMatches {
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
	var key []byte
	for _, c := range df.Columns() {
		s, _ := df.Series(c)
		key = fmt.Appendf(key, "|%v", chunk.CanonicalKey(s.Value(row)))
	}
	return string(key)
}

// streamable reports whether running nodes on each chunk of the input and
// concatenating the results equals running them on the whole input: every node
// is row-local, every expression it carries is elementwise, and limits appear
// only as a trailing run, which ExecuteStreaming applies after concatenation.
func streamable(nodes []logical.Node) bool {
	limits := len(nodes)
	for limits > 0 && nodes[limits-1].Type == logical.NodeLimit {
		limits--
	}
	for _, n := range nodes[:limits] {
		switch n.Type {
		case logical.NodeScan, logical.NodeFilter, logical.NodeSelect, logical.NodeWithCols,
			logical.NodeRename, logical.NodeDrop, logical.NodeCast, logical.NodeFillNull,
			logical.NodeFillNaN, logical.NodeDropNulls, logical.NodeDropNans,
			logical.NodeExplode, logical.NodeFlatten, logical.NodeUnnest:
		default:
			return false
		}
		if slices.ContainsFunc(n.Exprs, func(e expr.Expr) bool { return !expr.IsElementwise(e) }) {
			return false
		}
	}
	return true
}

func splitFrames(source frame.DataFrame, chunkSize int) []frame.DataFrame {
	if source.Height() == 0 {
		return []frame.DataFrame{source}
	}
	out := make([]frame.DataFrame, 0, (source.Height()+chunkSize-1)/chunkSize)
	for start := 0; start < source.Height(); start += chunkSize {
		end := min(start+chunkSize, source.Height())
		idx := make([]int, 0, end-start)
		for i := start; i < end; i++ {
			idx = append(idx, i)
		}
		part, _ := selectRows(source, idx)
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

// extractGlobalLimit splits the trailing run of limits off nodes and returns
// the nodes before it with the smallest of those limits, or -1 when nodes do
// not end in a limit.
func extractGlobalLimit(nodes []logical.Node) ([]logical.Node, int) {
	limit := -1
	end := len(nodes)
	for end > 0 && nodes[end-1].Type == logical.NodeLimit {
		end--
		if limit == -1 || nodes[end].IntValue < limit {
			limit = nodes[end].IntValue
		}
	}
	return nodes[:end], limit
}

func applyWindows(df frame.DataFrame, windows []logical.WindowSpec) (frame.DataFrame, error) {
	current := df
	for _, w := range windows {
		values := make([]any, current.Height())
		partitions := map[string][]int{}
		for i := 0; i < current.Height(); i++ {
			var key []byte
			for _, c := range w.PartitionBy {
				s, ok := current.Series(c)
				if !ok {
					return frame.DataFrame{}, fmt.Errorf("window partition column %s not found", c)
				}
				key = fmt.Appendf(key, "|%v", chunk.CanonicalKey(s.Value(i)))
			}
			partitions[string(key)] = append(partitions[string(key)], i)
		}
		for _, idxs := range partitions {
			ordered := slices.Clone(idxs)
			if len(w.OrderBy) > 0 {
				sort.Slice(ordered, func(i, j int) bool {
					li := ordered[i]
					rj := ordered[j]
					for k, col := range w.OrderBy {
						s, _ := current.Series(col)
						lv := s.Value(li)
						rv := s.Value(rj)
						c := compareForOrder(lv, rv)
						if c == 0 {
							continue
						}
						desc := k < len(w.Descending) && w.Descending[k]
						if desc {
							return c > 0
						}
						return c < 0
					}
					return li < rj
				})
			}
			switch w.Func {
			case "row_number":
				for i, row := range ordered {
					values[row] = int64(i + 1)
				}
			case "rank", "dense_rank":
				// Ties (equal ORDER BY keys) share a rank; RANK leaves gaps
				// (1,1,3), DENSE_RANK does not (1,1,2).
				rank := int64(1)
				dense := int64(1)
				for i, row := range ordered {
					if i > 0 && orderKeyChanged(current, w.OrderBy, ordered[i-1], row) {
						rank = int64(i + 1)
						dense++
					}
					if w.Func == "rank" {
						values[row] = rank
					} else {
						values[row] = dense
					}
				}
			case "lag", "lead":
				s, ok := current.Series(w.Target)
				if !ok {
					return frame.DataFrame{}, fmt.Errorf("window target column %s not found", w.Target)
				}
				offset := w.Offset
				if offset == 0 {
					offset = 1
				}
				for i, row := range ordered {
					j := i - offset
					if w.Func == "lead" {
						j = i + offset
					}
					if j >= 0 && j < len(ordered) {
						values[row] = s.Value(ordered[j])
					} else {
						values[row] = w.Default
					}
				}
			case "first_value", "last_value":
				s, ok := current.Series(w.Target)
				if !ok {
					return frame.DataFrame{}, fmt.Errorf("window target column %s not found", w.Target)
				}
				edge := ordered[0]
				if w.Func == "last_value" {
					edge = ordered[len(ordered)-1]
				}
				v := s.Value(edge)
				for _, row := range ordered {
					values[row] = v
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
		case "count", "row_number", "rank", "dense_rank":
			dt = dtypes.Int64
		case "sum", "min", "max", "lag", "lead", "first_value", "last_value":
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

// orderKeyChanged reports whether two rows differ on any ORDER BY column
// (used to detect rank ties).
func orderKeyChanged(df frame.DataFrame, orderBy []string, prev int, row int) bool {
	for _, col := range orderBy {
		s, ok := df.Series(col)
		if !ok {
			continue
		}
		if compareForOrder(s.Value(prev), s.Value(row)) != 0 {
			return true
		}
	}
	return false
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
		// intTotal keeps Int64 sums exact (and wrapping) above 2^53.
		var intTotal int64
		count := 0
		for _, r := range rows {
			v := s.Value(r)
			if v == nil {
				continue
			}
			f, ok := expr.ToFloat(v)
			if !ok {
				return nil, fmt.Errorf("window %s expects numeric values", fn)
			}
			if n, isInt := v.(int64); isInt {
				intTotal += n
			}
			total += f
			count++
		}
		if fn == "sum" {
			if s.DataType() == dtypes.Int64 {
				return intTotal, nil
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

// compareOrdered orders two values of the same comparable type. Any pair that
// is neither less nor greater compares equal — which for float64 includes every
// pair involving a NaN. cmp.Compare is deliberately not used: it orders NaN
// below every other value, which would reorder rows this engine sorts.
func compareOrdered[T cmp.Ordered](l T, r T) int {
	if l < r {
		return -1
	}
	if l > r {
		return 1
	}
	return 0
}

// compareForOrder orders two row values, with nulls first and any mismatched or
// unsupported pair reported as equal.
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
		if r, ok := right.(int64); ok {
			return compareOrdered(l, r)
		}
	case float64:
		if r, ok := right.(float64); ok {
			return compareOrdered(l, r)
		}
	case string:
		if r, ok := right.(string); ok {
			return compareOrdered(l, r)
		}
	case time.Time:
		if r, ok := right.(time.Time); ok {
			// Compare is Before/After/equal in one call, with the same
			// monotonic-clock handling.
			return l.Compare(r)
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
	var q float64
	if op == "quantile" {
		if len(args) < 1 {
			return frame.DataFrame{}, fmt.Errorf("quantile node missing probability")
		}
		var err error
		if q, err = strconv.ParseFloat(args[0], 64); err != nil {
			return frame.DataFrame{}, fmt.Errorf("quantile probability: %w", err)
		}
		// The negated range check also rejects NaN.
		if !(q >= 0 && q <= 1) {
			return frame.DataFrame{}, fmt.Errorf("quantile must be between 0.0 and 1.0")
		}
	}
	out := make([]series.Series, 0, df.Width())
	for _, name := range df.Columns() {
		s, _ := df.Series(name)
		var val any
		var dt = dtypes.Float64
		switch op {
		case "max", "min":
			// replaces is the compareForOrder result that makes a value the new best.
			replaces := 1
			if op == "min" {
				replaces = -1
			}
			var best any
			has := false
			for i := 0; i < s.Len(); i++ {
				v := s.Value(i)
				if v == nil {
					continue
				}
				if !has || compareForOrder(v, best) == replaces {
					best = v
					has = true
				}
			}
			val = best
			dt = s.DataType()
		case "count", "null_count":
			nulls := 0
			for i := 0; i < s.Len(); i++ {
				if s.IsNull(i) {
					nulls++
				}
			}
			if op == "count" {
				val = int64(s.Len() - nulls)
			} else {
				val = int64(nulls)
			}
			dt = dtypes.Int64
		case "sum":
			sum := float64(0)
			// intSum keeps Int64 sums exact (and wrapping) above 2^53.
			var intSum int64
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
					intSum += t
				case float64:
					sum += t
					allInt = false
				}
			}
			if !has {
				val = nil
			} else if allInt && s.DataType() == dtypes.Int64 {
				val = intSum
				dt = dtypes.Int64
			} else {
				val = sum
			}
		case "mean":
			nums := numericValues(s)
			if len(nums) == 0 {
				val = nil
			} else {
				sum := float64(0)
				for _, f := range nums {
					sum += f
				}
				val = sum / float64(len(nums))
			}
		case "median":
			nums := numericValues(s)
			if len(nums) == 0 {
				val = nil
			} else {
				slices.Sort(nums)
				mid := len(nums) / 2
				if len(nums)%2 == 0 {
					val = (nums[mid-1] + nums[mid]) / 2.0
				} else {
					val = nums[mid]
				}
			}
		case "std", "var":
			nums := numericValues(s)
			if len(nums) == 0 {
				val = nil
			} else {
				sum := float64(0)
				for _, f := range nums {
					sum += f
				}
				mean := sum / float64(len(nums))
				variance := float64(0)
				for _, f := range nums {
					diff := f - mean
					variance += diff * diff
				}
				variance /= float64(len(nums))
				if op == "std" {
					val = math.Sqrt(variance)
				} else {
					val = variance
				}
			}
		case "quantile":
			nums := numericValues(s)
			if len(nums) == 0 {
				val = nil
			} else {
				slices.Sort(nums)
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

// numericValues returns the non-null int64 and float64 values of s as float64,
// in row order. Values of any other type are skipped.
func numericValues(s series.Series) []float64 {
	nums := make([]float64, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		switch t := s.Value(i).(type) {
		case int64:
			nums = append(nums, float64(t))
		case float64:
			nums = append(nums, t)
		}
	}
	return nums
}
