package frame

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	"github.com/eugeneshershen/gopolars/pkg/expr"
	"github.com/eugeneshershen/gopolars/pkg/series"
)

type DataFrame struct {
	schema dtypes.Schema
	cols   map[string]series.Series
	order  []string
	height int
}

func New(input NewInput) (DataFrame, error) {
	if len(input.Series) == 0 {
		return DataFrame{cols: map[string]series.Series{}}, nil
	}
	height := input.Series[0].Len()
	cols := make(map[string]series.Series, len(input.Series))
	order := make([]string, 0, len(input.Series))
	schema := make(dtypes.Schema, 0, len(input.Series))
	for _, s := range input.Series {
		if s.Len() != height {
			return DataFrame{}, fmt.Errorf("all series must have same length")
		}
		cols[s.Name()] = s
		order = append(order, s.Name())
		schema = append(schema, dtypes.Field{Name: s.Name(), Type: s.DataType()})
	}
	return DataFrame{schema: schema, cols: cols, order: order, height: height}, nil
}

func (d DataFrame) Schema() dtypes.Schema {
	return d.schema
}

func (d DataFrame) Height() int {
	return d.height
}

func (d DataFrame) Width() int {
	return len(d.order)
}

func (d DataFrame) Columns() []string {
	out := make([]string, len(d.order))
	copy(out, d.order)
	return out
}

func (d DataFrame) Series(name string) (series.Series, bool) {
	v, ok := d.cols[name]
	return v, ok
}

func (d DataFrame) Select(exprs ...expr.Expr) (DataFrame, error) {
	if len(exprs) == 0 {
		return d, nil
	}
	outSeries := make([]series.Series, 0, len(exprs))
	for _, ex := range exprs {
		col, err := d.evalExprAsSeriesVectorized(ex)
		if err != nil {
			return DataFrame{}, err
		}
		outSeries = append(outSeries, col)
	}
	return New(NewInput{Series: outSeries})
}

func (d DataFrame) Filter(predicate expr.Expr) (DataFrame, error) {
	mask := make([]bool, d.height)
	if err := d.parallelForRows(func(start int, end int) error {
		for i := start; i < end; i++ {
			v, err := expr.Eval(predicate, rowAccessor{df: d, row: i})
			if err != nil {
				return err
			}
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("filter predicate must return bool")
			}
			mask[i] = b
		}
		return nil
	}); err != nil {
		return DataFrame{}, err
	}
	keep := make([]int, 0, d.height)
	for i, keepRow := range mask {
		if keepRow {
			keep = append(keep, i)
		}
	}
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(keep))
	}
	return New(NewInput{Series: out})
}

func (d DataFrame) WithColumns(exprs ...expr.Expr) (DataFrame, error) {
	out := d.clone()
	for _, ex := range exprs {
		col, err := d.evalExprAsSeriesVectorized(ex)
		if err != nil {
			return DataFrame{}, err
		}
		if _, ok := out.cols[col.Name()]; !ok {
			out.order = append(out.order, col.Name())
			out.schema = append(out.schema, dtypes.Field{Name: col.Name(), Type: col.DataType()})
		}
		out.cols[col.Name()] = col
	}
	return out, nil
}

func (d DataFrame) Sort(input SortInput) (DataFrame, error) {
	if len(input.By) == 0 {
		return d, nil
	}
	indexes := make([]int, d.height)
	for i := range indexes {
		indexes[i] = i
	}
	sortSeries := make([]series.Series, 0, len(input.By))
	for _, by := range input.By {
		s, ok := d.cols[by]
		if !ok {
			return DataFrame{}, fmt.Errorf("column %s not found", by)
		}
		sortSeries = append(sortSeries, s)
	}
	sort.Slice(indexes, func(i, j int) bool {
		for colIdx, s := range sortSeries {
			li := s.Value(indexes[i])
			rj := s.Value(indexes[j])
			cmp := compareSortValues(li, rj, input.NullsLast)
			if cmp == 0 {
				continue
			}
			desc := colIdx < len(input.Descending) && input.Descending[colIdx]
			if desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(indexes))
	}
	return New(NewInput{Series: out})
}

func (d DataFrame) Limit(n int) DataFrame {
	if n >= d.height {
		return d
	}
	if n < 0 {
		n = 0
	}
	indexes := make([]int, n)
	for i := 0; i < n; i++ {
		indexes[i] = i
	}
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(indexes))
	}
	df, _ := New(NewInput{Series: out})
	return df
}

func (d DataFrame) Slice(offset int, length int) DataFrame {
	if offset < 0 {
		offset = 0
	}
	if length < 0 {
		length = 0
	}
	if offset >= d.height {
		return d.Limit(0)
	}
	end := offset + length
	if end > d.height {
		end = d.height
	}
	indexes := make([]int, 0, end-offset)
	for i := offset; i < end; i++ {
		indexes = append(indexes, i)
	}
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(indexes))
	}
	df, _ := New(NewInput{Series: out})
	return df
}

func (d DataFrame) FillNull(value any) (DataFrame, error) {
	out := make([]series.Series, 0, len(d.order))
	for _, f := range d.schema {
		s := d.cols[f.Name]
		values := make([]any, 0, d.height)
		for i := 0; i < d.height; i++ {
			if s.IsNull(i) {
				values = append(values, value)
				continue
			}
			values = append(values, s.Value(i))
		}
		col, err := series.New(f.Name, f.Type, values)
		if err != nil {
			return DataFrame{}, err
		}
		out = append(out, col)
	}
	return New(NewInput{Series: out})
}

func (d DataFrame) DropNulls(columns ...string) DataFrame {
	targets := map[string]struct{}{}
	for _, c := range columns {
		targets[c] = struct{}{}
	}
	keep := make([]int, 0, d.height)
	for row := 0; row < d.height; row++ {
		drop := false
		for _, name := range d.order {
			if len(targets) > 0 {
				if _, ok := targets[name]; !ok {
					continue
				}
			}
			if d.cols[name].IsNull(row) {
				drop = true
				break
			}
		}
		if !drop {
			keep = append(keep, row)
		}
	}
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(keep))
	}
	df, _ := New(NewInput{Series: out})
	return df
}

func (d DataFrame) Unique(columns ...string) (DataFrame, error) {
	keys := columns
	if len(keys) == 0 {
		keys = d.order
	}
	seen := map[string]struct{}{}
	keep := make([]int, 0, d.height)
	for row := 0; row < d.height; row++ {
		key := ""
		for _, c := range keys {
			s, ok := d.cols[c]
			if !ok {
				return DataFrame{}, fmt.Errorf("column %s not found", c)
			}
			key += fmt.Sprintf("|%v", s.Value(row))
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keep = append(keep, row)
	}
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(keep))
	}
	return New(NewInput{Series: out})
}

func (d DataFrame) Explode(columns ...string) (DataFrame, error) {
	if len(columns) == 0 {
		return d, nil
	}
	targetSet := map[string]struct{}{}
	for _, c := range columns {
		targetSet[c] = struct{}{}
		if _, ok := d.cols[c]; !ok {
			return DataFrame{}, fmt.Errorf("column %s not found", c)
		}
	}
	outRows := make([]map[string]any, 0, d.height)
	for row := 0; row < d.height; row++ {
		repeat := 1
		for c := range targetSet {
			v := d.cols[c].Value(row)
			if v == nil {
				continue
			}
			list, ok := v.([]any)
			if !ok {
				return DataFrame{}, fmt.Errorf("explode expects list column %s", c)
			}
			if len(list) > repeat {
				repeat = len(list)
			}
		}
		for i := 0; i < repeat; i++ {
			r := map[string]any{}
			for _, name := range d.order {
				if _, ok := targetSet[name]; !ok {
					r[name] = d.cols[name].Value(row)
					continue
				}
				v := d.cols[name].Value(row)
				if v == nil {
					r[name] = nil
					continue
				}
				list := v.([]any)
				if i >= len(list) {
					r[name] = nil
					continue
				}
				r[name] = list[i]
			}
			outRows = append(outRows, r)
		}
	}
	seriesOut := make([]series.Series, 0, len(d.order))
	for _, f := range d.schema {
		values := make([]any, 0, len(outRows))
		dt := f.Type
		if _, ok := targetSet[f.Name]; ok {
			for _, r := range outRows {
				values = append(values, r[f.Name])
			}
			if inferred, err := inferDataType(values); err == nil {
				dt = inferred
			}
		} else {
			for _, r := range outRows {
				values = append(values, r[f.Name])
			}
		}
		s, err := series.New(f.Name, dt, values)
		if err != nil {
			return DataFrame{}, err
		}
		seriesOut = append(seriesOut, s)
	}
	return New(NewInput{Series: seriesOut})
}

func (d DataFrame) FlattenStruct(column string, prefix string) (DataFrame, error) {
	base, ok := d.cols[column]
	if !ok {
		return DataFrame{}, fmt.Errorf("column %s not found", column)
	}
	keys := map[string]struct{}{}
	for i := 0; i < d.height; i++ {
		v := base.Value(i)
		if v == nil {
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			return DataFrame{}, fmt.Errorf("flatten expects struct column %s", column)
		}
		for k := range m {
			keys[k] = struct{}{}
		}
	}
	out := make([]series.Series, 0, len(d.order)+len(keys))
	for _, name := range d.order {
		if name == column {
			continue
		}
		s, _ := d.Series(name)
		out = append(out, s.Clone())
	}
	for key := range keys {
		values := make([]any, d.height)
		for i := 0; i < d.height; i++ {
			v := base.Value(i)
			if v == nil {
				values[i] = nil
				continue
			}
			values[i] = v.(map[string]any)[key]
		}
		dt, err := inferDataType(values)
		if err != nil {
			dt = dtypes.String
		}
		name := key
		if prefix != "" {
			name = prefix + key
		}
		s, err := series.New(name, dt, values)
		if err != nil {
			return DataFrame{}, err
		}
		out = append(out, s)
	}
	return New(NewInput{Series: out})
}

func (d DataFrame) GroupBy(keys ...string) GroupBy {
	return GroupBy{df: d, keys: keys}
}

func (d DataFrame) Join(input JoinInput) (DataFrame, error) {
	return join(d, input)
}

func (d DataFrame) RollingMean(by string, value string, window time.Duration, minRows int, output string, closed string) (DataFrame, error) {
	if output == "" {
		output = "rolling_mean"
	}
	if minRows <= 0 {
		minRows = 1
	}
	if window <= 0 {
		return DataFrame{}, fmt.Errorf("rolling window must be positive")
	}
	bySeries, ok := d.cols[by]
	if !ok {
		return DataFrame{}, fmt.Errorf("time column %s not found", by)
	}
	valueSeries, ok := d.cols[value]
	if !ok {
		return DataFrame{}, fmt.Errorf("value column %s not found", value)
	}
	outValues := make([]any, d.height)
	for i := 0; i < d.height; i++ {
		end, ok := bySeries.Value(i).(time.Time)
		if !ok {
			return DataFrame{}, fmt.Errorf("rolling expects datetime order column")
		}
		start := end.Add(-window)
		sum := float64(0)
		cnt := 0
		for j := 0; j <= i; j++ {
			ts, ok := bySeries.Value(j).(time.Time)
			if !ok {
				return DataFrame{}, fmt.Errorf("rolling expects datetime order column")
			}
			if !inTemporalBounds(ts, start, end, closed) {
				continue
			}
			switch v := valueSeries.Value(j).(type) {
			case int64:
				sum += float64(v)
				cnt++
			case float64:
				sum += v
				cnt++
			}
		}
		if cnt < minRows || cnt == 0 {
			outValues[i] = nil
			continue
		}
		outValues[i] = sum / float64(cnt)
	}
	outSeries := make([]series.Series, 0, len(d.order)+1)
	for _, name := range d.order {
		outSeries = append(outSeries, d.cols[name].Clone())
	}
	roll, err := series.New(output, dtypes.Float64, outValues)
	if err != nil {
		return DataFrame{}, err
	}
	outSeries = append(outSeries, roll)
	return New(NewInput{Series: outSeries})
}

func (d DataFrame) GroupByDynamic(by string, every time.Duration, period time.Duration, offset time.Duration, closed string, label string, windowColumn string, aggExpr expr.Expr) (DataFrame, error) {
	if by == "" {
		return DataFrame{}, fmt.Errorf("dynamic group column is empty")
	}
	if every <= 0 {
		return DataFrame{}, fmt.Errorf("dynamic group every must be positive")
	}
	if period <= 0 {
		period = every
	}
	if windowColumn == "" {
		windowColumn = "dynamic_window"
	}
	bySeries, ok := d.cols[by]
	if !ok {
		return DataFrame{}, fmt.Errorf("dynamic group column %s not found", by)
	}
	windows := make([]any, d.height)
	for i := 0; i < d.height; i++ {
		ts, ok := bySeries.Value(i).(time.Time)
		if !ok {
			return DataFrame{}, fmt.Errorf("dynamic group expects datetime column")
		}
		base := ts.Truncate(every).Add(offset)
		if ts.Before(base) {
			base = base.Add(-every)
		}
		end := base.Add(period)
		if !inTemporalBounds(ts, base, end, closed) {
			if ts.Before(base) {
				base = base.Add(-every)
				end = base.Add(period)
			} else {
				base = base.Add(every)
				end = base.Add(period)
			}
		}
		if label == "right" {
			windows[i] = end
		} else {
			windows[i] = base
		}
	}
	withWindow := make([]series.Series, 0, len(d.order)+1)
	for _, name := range d.order {
		withWindow = append(withWindow, d.cols[name].Clone())
	}
	windowSeries, err := series.New(windowColumn, dtypes.Datetime, windows)
	if err != nil {
		return DataFrame{}, err
	}
	withWindow = append(withWindow, windowSeries)
	prepared, err := New(NewInput{Series: withWindow})
	if err != nil {
		return DataFrame{}, err
	}
	return prepared.GroupBy(windowColumn).Agg(aggExpr)
}

func (d DataFrame) Melt(idVars []string, valueVars []string, variableCol string, valueCol string) (DataFrame, error) {
	if variableCol == "" {
		variableCol = "variable"
	}
	if valueCol == "" {
		valueCol = "value"
	}
	if len(valueVars) == 0 {
		for _, c := range d.order {
			found := false
			for _, id := range idVars {
				if c == id {
					found = true
					break
				}
			}
			if !found {
				valueVars = append(valueVars, c)
			}
		}
	}
	outCols := map[string][]any{}
	outOrder := make([]string, 0, len(idVars)+2)
	for _, id := range idVars {
		if _, ok := d.cols[id]; !ok {
			return DataFrame{}, fmt.Errorf("column %s not found", id)
		}
		outOrder = append(outOrder, id)
		outCols[id] = make([]any, 0, d.height*len(valueVars))
	}
	outOrder = append(outOrder, variableCol, valueCol)
	outCols[variableCol] = make([]any, 0, d.height*len(valueVars))
	outCols[valueCol] = make([]any, 0, d.height*len(valueVars))
	for row := 0; row < d.height; row++ {
		for _, valCol := range valueVars {
			s, ok := d.cols[valCol]
			if !ok {
				return DataFrame{}, fmt.Errorf("column %s not found", valCol)
			}
			for _, id := range idVars {
				outCols[id] = append(outCols[id], d.cols[id].Value(row))
			}
			outCols[variableCol] = append(outCols[variableCol], valCol)
			outCols[valueCol] = append(outCols[valueCol], s.Value(row))
		}
	}
	seriesOut := make([]series.Series, 0, len(outOrder))
	for _, name := range outOrder {
		dt := dtypes.String
		if name == valueCol {
			if inferred, err := inferDataType(outCols[name]); err == nil {
				dt = inferred
			}
		} else if s, ok := d.cols[name]; ok {
			dt = s.DataType()
		}
		s, err := series.New(name, dt, outCols[name])
		if err != nil {
			return DataFrame{}, err
		}
		seriesOut = append(seriesOut, s)
	}
	return New(NewInput{Series: seriesOut})
}

func (d DataFrame) Pivot(index []string, columns string, values string, agg string) (DataFrame, error) {
	if len(index) == 0 {
		return DataFrame{}, fmt.Errorf("pivot requires index")
	}
	indexVals := make([][]any, d.height)
	colSeries, ok := d.cols[columns]
	if !ok {
		return DataFrame{}, fmt.Errorf("column %s not found", columns)
	}
	valSeries, ok := d.cols[values]
	if !ok {
		return DataFrame{}, fmt.Errorf("column %s not found", values)
	}
	uniqPivot := []string{}
	pivotSet := map[string]struct{}{}
	type bucket struct {
		idxVals []any
		data    map[string][]any
	}
	groups := map[string]*bucket{}
	rowOrder := []string{}
	for row := 0; row < d.height; row++ {
		key := ""
		idx := make([]any, 0, len(index))
		for _, c := range index {
			s, ok := d.cols[c]
			if !ok {
				return DataFrame{}, fmt.Errorf("column %s not found", c)
			}
			v := s.Value(row)
			idx = append(idx, v)
			key += fmt.Sprintf("|%v", v)
		}
		indexVals[row] = idx
		pv := fmt.Sprintf("%v", colSeries.Value(row))
		if _, ok := pivotSet[pv]; !ok {
			pivotSet[pv] = struct{}{}
			uniqPivot = append(uniqPivot, pv)
		}
		if _, ok := groups[key]; !ok {
			groups[key] = &bucket{idxVals: idx, data: map[string][]any{}}
			rowOrder = append(rowOrder, key)
		}
		groups[key].data[pv] = append(groups[key].data[pv], valSeries.Value(row))
	}
	outCols := map[string][]any{}
	outOrder := make([]string, 0, len(index)+len(uniqPivot))
	for _, c := range index {
		outOrder = append(outOrder, c)
		outCols[c] = make([]any, 0, len(rowOrder))
	}
	for _, p := range uniqPivot {
		outOrder = append(outOrder, p)
		outCols[p] = make([]any, 0, len(rowOrder))
	}
	for _, key := range rowOrder {
		group := groups[key]
		for i, c := range index {
			outCols[c] = append(outCols[c], group.idxVals[i])
		}
		for _, p := range uniqPivot {
			valuesSet := group.data[p]
			outCols[p] = append(outCols[p], aggregateValues(valuesSet, agg))
		}
	}
	seriesOut := make([]series.Series, 0, len(outOrder))
	for _, c := range outOrder {
		dt := dtypes.String
		if src, ok := d.cols[c]; ok {
			dt = src.DataType()
		} else if inferred, err := inferDataType(outCols[c]); err == nil {
			dt = inferred
		}
		s, err := series.New(c, dt, outCols[c])
		if err != nil {
			return DataFrame{}, err
		}
		seriesOut = append(seriesOut, s)
	}
	return New(NewInput{Series: seriesOut})
}

func (d DataFrame) clone() DataFrame {
	out := DataFrame{
		schema: make(dtypes.Schema, len(d.schema)),
		cols:   make(map[string]series.Series, len(d.cols)),
		order:  make([]string, len(d.order)),
		height: d.height,
	}
	copy(out.schema, d.schema)
	copy(out.order, d.order)
	for k, v := range d.cols {
		out.cols[k] = v.Clone()
	}
	return out
}

func (d DataFrame) evalExprAsSeries(e expr.Expr) (series.Series, error) {
	if e.Kind() == expr.KindCol {
		s, ok := d.cols[e.ColName()]
		if !ok {
			return series.Series{}, fmt.Errorf("column %s not found", e.ColName())
		}
		return s.Rename(e.Name()), nil
	}
	values := make([]any, d.height)
	for i := 0; i < d.height; i++ {
		v, err := expr.Eval(e, rowAccessor{df: d, row: i})
		if err != nil {
			return series.Series{}, err
		}
		values[i] = v
	}
	dt, err := inferDataType(values)
	if err != nil {
		return series.Series{}, err
	}
	return series.New(e.Name(), dt, values)
}

func (d DataFrame) evalExprAsSeriesVectorized(e expr.Expr) (series.Series, error) {
	if e.Kind() == expr.KindCol {
		return d.evalExprAsSeries(e)
	}
	values := make([]any, d.height)
	if err := d.parallelForRows(func(start int, end int) error {
		for i := start; i < end; i++ {
			v, err := expr.Eval(e, rowAccessor{df: d, row: i})
			if err != nil {
				return err
			}
			values[i] = v
		}
		return nil
	}); err != nil {
		return series.Series{}, err
	}
	dt, err := inferDataType(values)
	if err != nil {
		return series.Series{}, err
	}
	return series.New(e.Name(), dt, values)
}

func (d DataFrame) parallelForRows(run func(start int, end int) error) error {
	if d.height == 0 {
		return nil
	}
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > d.height {
		workers = d.height
	}
	chunk := (d.height + workers - 1) / workers
	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	for start := 0; start < d.height; start += chunk {
		end := start + chunk
		if end > d.height {
			end = d.height
		}
		wg.Add(1)
		go func(s int, e int) {
			defer wg.Done()
			if err := run(s, e); err != nil {
				errCh <- err
			}
		}(start, end)
	}
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func inferDataType(values []any) (dtypes.DataType, error) {
	for _, v := range values {
		if v == nil {
			continue
		}
		switch v.(type) {
		case int64:
			return dtypes.Int64, nil
		case float64:
			return dtypes.Float64, nil
		case dtypes.DecimalValue:
			return dtypes.Decimal, nil
		case string:
			return dtypes.String, nil
		case bool:
			return dtypes.Boolean, nil
		case time.Time:
			return dtypes.Datetime, nil
		case []any:
			return dtypes.List, nil
		case map[string]any:
			return dtypes.Struct, nil
		}
	}
	return "", fmt.Errorf("cannot infer data type")
}

func aggregateValues(values []any, agg string) any {
	if len(values) == 0 {
		return nil
	}
	switch agg {
	case "", "first":
		return values[0]
	case "sum", "mean":
		total := float64(0)
		count := 0
		allInt := true
		for _, v := range values {
			switch t := v.(type) {
			case int64:
				total += float64(t)
				count++
			case float64:
				total += t
				count++
				allInt = false
			}
		}
		if count == 0 {
			return nil
		}
		if agg == "mean" {
			return total / float64(count)
		}
		if allInt {
			return int64(total)
		}
		return total
	case "count":
		return int64(len(values))
	case "min", "max":
		best := values[0]
		for i := 1; i < len(values); i++ {
			if compareAny(values[i], best) < 0 && agg == "min" {
				best = values[i]
			}
			if compareAny(values[i], best) > 0 && agg == "max" {
				best = values[i]
			}
		}
		return best
	default:
		return values[0]
	}
}

func compareAny(left any, right any) int {
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
	}
	return 0
}

func inTemporalBounds(ts time.Time, start time.Time, end time.Time, closed string) bool {
	switch closed {
	case "left", "":
		return (ts.Equal(start) || ts.After(start)) && ts.Before(end)
	case "right":
		return ts.After(start) && (ts.Before(end) || ts.Equal(end))
	case "both":
		return (ts.Equal(start) || ts.After(start)) && (ts.Before(end) || ts.Equal(end))
	case "none":
		return ts.After(start) && ts.Before(end)
	default:
		return (ts.Equal(start) || ts.After(start)) && ts.Before(end)
	}
}

type rowAccessor struct {
	df  DataFrame
	row int
}

func (r rowAccessor) ValueByName(name string) (any, bool) {
	s, ok := r.df.cols[name]
	if !ok {
		return nil, false
	}
	return s.Value(r.row), true
}

func lessAny(left any, right any) bool {
	switch l := left.(type) {
	case int64:
		r, ok := right.(int64)
		if !ok {
			return false
		}
		return l < r
	case float64:
		r, ok := right.(float64)
		if !ok {
			return false
		}
		return l < r
	case string:
		r, ok := right.(string)
		if !ok {
			return false
		}
		return l < r
	case bool:
		r, ok := right.(bool)
		if !ok {
			return false
		}
		return !l && r
	default:
		return false
	}
}

func compareSortValues(left any, right any, nullsLast bool) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		if nullsLast {
			return 1
		}
		return -1
	}
	if right == nil {
		if nullsLast {
			return -1
		}
		return 1
	}
	lf, lok := left.(float64)
	rf, rok := right.(float64)
	if lok && rok {
		lNaN := math.IsNaN(lf)
		rNaN := math.IsNaN(rf)
		if lNaN && rNaN {
			return 0
		}
		if lNaN {
			return 1
		}
		if rNaN {
			return -1
		}
	}
	if left == right {
		return 0
	}
	if lessAny(left, right) {
		return -1
	}
	if lessAny(right, left) {
		return 1
	}
	return 0
}
