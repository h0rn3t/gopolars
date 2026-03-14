package frame

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"strconv"
	"strings"
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

func (d DataFrame) CollectSchema() dtypes.Schema {
	out := make(dtypes.Schema, len(d.schema))
	copy(out, d.schema)
	return out
}

func (d DataFrame) Height() int {
	return d.height
}

func (d DataFrame) IsEmpty() bool {
	return d.height == 0
}

func (d DataFrame) EstimatedSize() int64 {
	size := int64(0)
	for _, name := range d.order {
		s := d.cols[name]
		switch s.DataType() {
		case dtypes.Int64, dtypes.Float64, dtypes.Datetime:
			size += int64(s.Len() * 8)
		case dtypes.Boolean:
			size += int64(s.Len())
		default:
			for i := 0; i < s.Len(); i++ {
				v := s.Value(i)
				switch t := v.(type) {
				case string:
					size += int64(len(t))
				case []any:
					size += int64(len(t) * 8)
				case map[string]any:
					size += int64(len(t) * 16)
				default:
					size += 8
				}
			}
		}
	}
	return size
}

func (d DataFrame) Width() int {
	return len(d.order)
}

func (d DataFrame) Columns() []string {
	out := make([]string, len(d.order))
	copy(out, d.order)
	return out
}

func (d DataFrame) Dtypes() []dtypes.DataType {
	out := make([]dtypes.DataType, 0, len(d.schema))
	for _, f := range d.schema {
		out = append(out, f.Type)
	}
	return out
}

func (d DataFrame) ToDicts() []map[string]any {
	out := make([]map[string]any, 0, d.height)
	for row := 0; row < d.height; row++ {
		record := make(map[string]any, len(d.order))
		for _, name := range d.order {
			record[name] = d.cols[name].Value(row)
		}
		out = append(out, record)
	}
	return out
}

func (d DataFrame) NullCount() map[string]int {
	out := make(map[string]int, len(d.order))
	for _, name := range d.order {
		s := d.cols[name]
		c := 0
		for i := 0; i < d.height; i++ {
			if s.IsNull(i) {
				c++
			}
		}
		out[name] = c
	}
	return out
}

func (d DataFrame) Count() map[string]int {
	out := make(map[string]int, len(d.order))
	for _, name := range d.order {
		s := d.cols[name]
		c := 0
		for i := 0; i < d.height; i++ {
			if !s.IsNull(i) {
				c++
			}
		}
		out[name] = c
	}
	return out
}

func (d DataFrame) NUnique(columns ...string) (int, error) {
	keys := columns
	if len(keys) == 0 {
		keys = d.order
	}
	seen := map[string]struct{}{}
	for row := 0; row < d.height; row++ {
		key := ""
		for _, c := range keys {
			s, ok := d.cols[c]
			if !ok {
				return 0, fmt.Errorf("column %s not found", c)
			}
			key += fmt.Sprintf("|%v", s.Value(row))
		}
		seen[key] = struct{}{}
	}
	return len(seen), nil
}

func (d DataFrame) ApproxNUnique(columns ...string) (int, error) {
	return d.NUnique(columns...)
}

func (d DataFrame) Series(name string) (series.Series, bool) {
	v, ok := d.cols[name]
	return v, ok
}

func (d DataFrame) GetColumn(name string) (series.Series, error) {
	v, ok := d.cols[name]
	if !ok {
		return series.Series{}, fmt.Errorf("column %s not found", name)
	}
	return v.Clone(), nil
}

func (d DataFrame) GetColumnIndex(name string) int {
	for i, col := range d.order {
		if col == name {
			return i
		}
	}
	return -1
}

func (d DataFrame) GetColumns() []series.Series {
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Clone())
	}
	return out
}

func (d DataFrame) Flags() map[string]map[string]bool {
	out := make(map[string]map[string]bool, len(d.order))
	for _, name := range d.order {
		out[name] = map[string]bool{
			"sorted_asc":   isSeriesSortedAsc(d.cols[name]),
			"has_nulls":    hasSeriesNulls(d.cols[name]),
			"has_nan":      hasSeriesNaN(d.cols[name]),
			"is_monotonic": isSeriesSortedAsc(d.cols[name]),
		}
	}
	return out
}

func (d DataFrame) Glimpse(maxRows int) string {
	if maxRows <= 0 {
		maxRows = 10
	}
	if maxRows > d.height {
		maxRows = d.height
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("rows=%d cols=%d\n", d.height, len(d.order)))
	for _, f := range d.schema {
		b.WriteString(fmt.Sprintf("%s:%s ", f.Name, f.Type))
	}
	b.WriteString("\n")
	for row := 0; row < maxRows; row++ {
		b.WriteString(fmt.Sprintf("[%d] ", row))
		for _, col := range d.order {
			b.WriteString(fmt.Sprintf("%s=%v ", col, d.cols[col].Value(row)))
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
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

func (d DataFrame) WithRowCount(name string, offset int64) (DataFrame, error) {
	if name == "" {
		name = "row_nr"
	}
	values := make([]any, d.height)
	for i := 0; i < d.height; i++ {
		values[i] = offset + int64(i)
	}
	rowSeries, err := series.New(name, dtypes.Int64, values)
	if err != nil {
		return DataFrame{}, err
	}
	out := make([]series.Series, 0, len(d.order)+1)
	out = append(out, rowSeries)
	for _, col := range d.order {
		out = append(out, d.cols[col].Clone())
	}
	return New(NewInput{Series: out})
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

func (d DataFrame) Tail(n int) DataFrame {
	if n >= d.height {
		return d
	}
	if n < 0 {
		n = 0
	}
	start := d.height - n
	indexes := make([]int, n)
	for i := 0; i < n; i++ {
		indexes[i] = start + i
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

func (d DataFrame) Drop(columns ...string) (DataFrame, error) {
	dropSet := map[string]struct{}{}
	for _, c := range columns {
		dropSet[c] = struct{}{}
	}
	outSeries := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		if _, ok := dropSet[name]; ok {
			continue
		}
		outSeries = append(outSeries, d.cols[name].Clone())
	}
	return New(NewInput{Series: outSeries})
}

func (d DataFrame) Reverse() DataFrame {
	indexes := make([]int, d.height)
	for i := 0; i < d.height; i++ {
		indexes[i] = d.height - 1 - i
	}
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(indexes))
	}
	df, _ := New(NewInput{Series: out})
	return df
}

func (d DataFrame) Rename(mapping map[string]string) (DataFrame, error) {
	if len(mapping) == 0 {
		return d, nil
	}
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		if newName, ok := mapping[name]; ok {
			out = append(out, d.cols[name].Rename(newName))
		} else {
			out = append(out, d.cols[name].Clone())
		}
	}
	return New(NewInput{Series: out})
}

func (d DataFrame) GatherEvery(step int, offset int) DataFrame {
	if step <= 0 {
		step = 1
	}
	if offset < 0 {
		offset = 0
	}
	indexes := make([]int, 0, d.height)
	for i := offset; i < d.height; i += step {
		indexes = append(indexes, i)
	}
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(indexes))
	}
	df, _ := New(NewInput{Series: out})
	return df
}

func (d DataFrame) BottomK(k int, by string) (DataFrame, error) {
	if k <= 0 {
		return d.Limit(0), nil
	}
	sorted, err := d.Sort(SortInput{By: []string{by}})
	if err != nil {
		return DataFrame{}, err
	}
	return sorted.Limit(k), nil
}

func (d DataFrame) Clear() DataFrame {
	out := make([]series.Series, 0, len(d.order))
	for _, f := range d.schema {
		s, _ := series.New(f.Name, f.Type, []any{})
		out = append(out, s)
	}
	df, _ := New(NewInput{Series: out})
	return df
}

func (d DataFrame) Clone() DataFrame {
	return d.clone()
}

func (d DataFrame) DropInPlace(column string) (DataFrame, error) {
	if _, ok := d.cols[column]; !ok {
		return DataFrame{}, fmt.Errorf("column %s not found", column)
	}
	out := make([]series.Series, 0, len(d.order)-1)
	for _, name := range d.order {
		if name == column {
			continue
		}
		out = append(out, d.cols[name].Clone())
	}
	return New(NewInput{Series: out})
}

func (d DataFrame) Extend(other DataFrame) (DataFrame, error) {
	return ConcatVertical(d, other)
}

func (d DataFrame) Hstack(columns ...series.Series) (DataFrame, error) {
	if len(columns) == 0 {
		return d, nil
	}
	out := d.clone()
	for _, c := range columns {
		if c.Len() != out.height {
			return DataFrame{}, fmt.Errorf("column %s has invalid length", c.Name())
		}
		if _, exists := out.cols[c.Name()]; !exists {
			out.order = append(out.order, c.Name())
			out.schema = append(out.schema, dtypes.Field{Name: c.Name(), Type: c.DataType()})
		}
		out.cols[c.Name()] = c.Clone()
	}
	return out, nil
}

func (d DataFrame) InsertColumn(index int, column series.Series) (DataFrame, error) {
	if column.Len() != d.height {
		return DataFrame{}, fmt.Errorf("column %s has invalid length", column.Name())
	}
	if index < 0 {
		index = 0
	}
	if index > len(d.order) {
		index = len(d.order)
	}
	out := d.clone()
	if old, ok := out.cols[column.Name()]; ok && old.Len() == out.height {
		for i, name := range out.order {
			if name == column.Name() {
				out.order = append(out.order[:i], out.order[i+1:]...)
				break
			}
		}
		for i, f := range out.schema {
			if f.Name == column.Name() {
				out.schema = append(out.schema[:i], out.schema[i+1:]...)
				break
			}
		}
	}
	out.order = append(out.order[:index], append([]string{column.Name()}, out.order[index:]...)...)
	out.schema = append(out.schema[:index], append([]dtypes.Field{{Name: column.Name(), Type: column.DataType()}}, out.schema[index:]...)...)
	out.cols[column.Name()] = column.Clone()
	return out, nil
}

func (d DataFrame) Sample(n int, seed int64) DataFrame {
	if n <= 0 || d.height == 0 {
		return d.Limit(0)
	}
	if n > d.height {
		n = d.height
	}
	idx := make([]int, d.height)
	for i := 0; i < d.height; i++ {
		idx[i] = i
	}
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(idx), func(i, j int) {
		idx[i], idx[j] = idx[j], idx[i]
	})
	selected := idx[:n]
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Slice(selected))
	}
	df, _ := New(NewInput{Series: out})
	return df
}

func (d DataFrame) Cast(mapping map[string]dtypes.DataType) (DataFrame, error) {
	if len(mapping) == 0 {
		return d, nil
	}
	out := d.clone()
	for name, dt := range mapping {
		s, ok := out.cols[name]
		if !ok {
			return DataFrame{}, fmt.Errorf("column %s not found", name)
		}
		values := make([]any, s.Len())
		for i := 0; i < s.Len(); i++ {
			if s.IsNull(i) {
				values[i] = nil
				continue
			}
			v, err := castValueToType(s.Value(i), dt)
			if err != nil {
				return DataFrame{}, err
			}
			values[i] = v
		}
		castSeries, err := series.New(name, dt, values)
		if err != nil {
			return DataFrame{}, err
		}
		out.cols[name] = castSeries
		for i := range out.schema {
			if out.schema[i].Name == name {
				out.schema[i].Type = dt
			}
		}
	}
	return out, nil
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

func (d DataFrame) FillNaN(value float64) (DataFrame, error) {
	out := make([]series.Series, 0, len(d.order))
	for _, f := range d.schema {
		s := d.cols[f.Name]
		values := make([]any, 0, d.height)
		for i := 0; i < d.height; i++ {
			v := s.Value(i)
			if fv, ok := v.(float64); ok && math.IsNaN(fv) {
				values = append(values, value)
				continue
			}
			values = append(values, v)
		}
		col, err := series.New(f.Name, f.Type, values)
		if err != nil {
			return DataFrame{}, err
		}
		out = append(out, col)
	}
	return New(NewInput{Series: out})
}

func (d DataFrame) Interpolate(columns ...string) (DataFrame, error) {
	targets := map[string]struct{}{}
	if len(columns) == 0 {
		for _, name := range d.order {
			targets[name] = struct{}{}
		}
	} else {
		for _, name := range columns {
			targets[name] = struct{}{}
		}
	}
	out := d.clone()
	for name := range targets {
		s, ok := out.cols[name]
		if !ok {
			return DataFrame{}, fmt.Errorf("column %s not found", name)
		}
		if s.DataType() != dtypes.Float64 && s.DataType() != dtypes.Int64 {
			continue
		}
		values := make([]any, s.Len())
		for i := 0; i < s.Len(); i++ {
			values[i] = s.Value(i)
		}
		for i := 0; i < len(values); i++ {
			if values[i] != nil {
				continue
			}
			left := -1
			for j := i - 1; j >= 0; j-- {
				if values[j] != nil {
					left = j
					break
				}
			}
			right := -1
			for j := i + 1; j < len(values); j++ {
				if values[j] != nil {
					right = j
					break
				}
			}
			switch {
			case left >= 0 && right >= 0:
				lv, lok := toFloat(values[left])
				rv, rok := toFloat(values[right])
				if lok && rok {
					ratio := float64(i-left) / float64(right-left)
					values[i] = lv + (rv-lv)*ratio
				}
			case left >= 0:
				values[i] = values[left]
			case right >= 0:
				values[i] = values[right]
			}
		}
		newSeries, err := series.New(name, s.DataType(), values)
		if err != nil {
			return DataFrame{}, err
		}
		out.cols[name] = newSeries
	}
	return out, nil
}

func (d DataFrame) DropNaNs(columns ...string) DataFrame {
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
			v := d.cols[name].Value(row)
			if fv, ok := v.(float64); ok && math.IsNaN(fv) {
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

func (d DataFrame) Equals(other DataFrame) (bool, error) {
	if d.height != other.height {
		return false, nil
	}
	if len(d.order) != len(other.order) {
		return false, nil
	}
	for i, name := range d.order {
		if other.order[i] != name {
			return false, nil
		}
		left := d.cols[name]
		right, ok := other.cols[name]
		if !ok || left.DataType() != right.DataType() {
			return false, nil
		}
		for row := 0; row < d.height; row++ {
			lv := left.Value(row)
			rv := right.Value(row)
			if !valueEquals(lv, rv) {
				return false, nil
			}
		}
	}
	return true, nil
}

func (d DataFrame) Fold(op string, columns []string, alias string) (DataFrame, error) {
	if len(columns) == 0 {
		columns = d.order
	}
	if alias == "" {
		alias = "fold"
	}
	values := make([]any, d.height)
	for row := 0; row < d.height; row++ {
		accSet := false
		acc := float64(0)
		for _, col := range columns {
			s, ok := d.cols[col]
			if !ok {
				return DataFrame{}, fmt.Errorf("column %s not found", col)
			}
			v := s.Value(row)
			num, ok := toFloat(v)
			if !ok {
				continue
			}
			if !accSet {
				acc = num
				accSet = true
				continue
			}
			switch op {
			case "sum":
				acc += num
			case "max":
				if num > acc {
					acc = num
				}
			case "min":
				if num < acc {
					acc = num
				}
			default:
				acc += num
			}
		}
		if !accSet {
			values[row] = nil
		} else {
			values[row] = acc
		}
	}
	foldSeries, err := series.New(alias, dtypes.Float64, values)
	if err != nil {
		return DataFrame{}, err
	}
	out := d.clone()
	out.cols[alias] = foldSeries
	out.order = append(out.order, alias)
	out.schema = append(out.schema, dtypes.Field{Name: alias, Type: dtypes.Float64})
	return out, nil
}

func (d DataFrame) HashRows(seed uint64) ([]uint64, error) {
	out := make([]uint64, d.height)
	for row := 0; row < d.height; row++ {
		h := fnv.New64a()
		_, _ = h.Write([]byte(fmt.Sprintf("%d", seed)))
		for _, col := range d.order {
			_, _ = h.Write([]byte(fmt.Sprintf("|%v", d.cols[col].Value(row))))
		}
		out[row] = h.Sum64()
	}
	return out, nil
}

func (d DataFrame) Corr(columnA string, columnB string) (float64, error) {
	a, ok := d.cols[columnA]
	if !ok {
		return 0, fmt.Errorf("column %s not found", columnA)
	}
	b, ok := d.cols[columnB]
	if !ok {
		return 0, fmt.Errorf("column %s not found", columnB)
	}
	xs := make([]float64, 0, d.height)
	ys := make([]float64, 0, d.height)
	for i := 0; i < d.height; i++ {
		x, okx := toFloat(a.Value(i))
		y, oky := toFloat(b.Value(i))
		if okx && oky {
			xs = append(xs, x)
			ys = append(ys, y)
		}
	}
	if len(xs) < 2 {
		return 0, nil
	}
	return pearson(xs, ys), nil
}

func (d DataFrame) Describe() (DataFrame, error) {
	stats := []string{"count", "null_count", "mean", "std", "min", "max"}
	cols := make([]series.Series, 0, len(d.order)+1)
	statValues := make([]any, len(stats))
	for i, s := range stats {
		statValues[i] = s
	}
	statSeries, _ := series.New("statistic", dtypes.String, statValues)
	cols = append(cols, statSeries)
	for _, name := range d.order {
		s := d.cols[name]
		values := make([]any, len(stats))
		numeric := make([]float64, 0, d.height)
		nulls := 0
		for i := 0; i < d.height; i++ {
			v := s.Value(i)
			if v == nil {
				nulls++
				continue
			}
			if n, ok := toFloat(v); ok {
				numeric = append(numeric, n)
			}
		}
		values[0] = float64(d.height - nulls)
		values[1] = float64(nulls)
		if len(numeric) > 0 {
			mean := 0.0
			for _, n := range numeric {
				mean += n
			}
			mean /= float64(len(numeric))
			variance := 0.0
			minVal := numeric[0]
			maxVal := numeric[0]
			for _, n := range numeric {
				diff := n - mean
				variance += diff * diff
				if n < minVal {
					minVal = n
				}
				if n > maxVal {
					maxVal = n
				}
			}
			values[2] = mean
			values[3] = math.Sqrt(variance / float64(len(numeric)))
			values[4] = minVal
			values[5] = maxVal
		}
		descSeries, err := series.New(name, dtypes.Float64, values)
		if err != nil {
			return DataFrame{}, err
		}
		cols = append(cols, descSeries)
	}
	return New(NewInput{Series: cols})
}

func (d DataFrame) Deserialize(payload []byte) (DataFrame, error) {
	var records []map[string]any
	if err := json.Unmarshal(payload, &records); err != nil {
		return DataFrame{}, err
	}
	if len(records) == 0 {
		return New(NewInput{Series: []series.Series{}})
	}
	order := make([]string, 0, len(records[0]))
	for k := range records[0] {
		order = append(order, k)
	}
	sort.Strings(order)
	seriesOut := make([]series.Series, 0, len(order))
	for _, col := range order {
		values := make([]any, 0, len(records))
		for _, row := range records {
			values = append(values, row[col])
		}
		dt, err := inferDataType(values)
		if err != nil {
			dt = dtypes.String
		}
		s, err := series.New(col, dt, values)
		if err != nil {
			return DataFrame{}, err
		}
		seriesOut = append(seriesOut, s)
	}
	return New(NewInput{Series: seriesOut})
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
	if e.Kind() == expr.KindUnary && e.Target() != nil {
		switch {
		case e.Op() == "cum_sum":
			return d.evalCumulative(*e.Target(), e.Name(), "sum")
		case e.Op() == "cum_count":
			return d.evalCumulative(*e.Target(), e.Name(), "count")
		case e.Op() == "rank":
			return d.evalRank(*e.Target(), e.Name())
		case strings.HasPrefix(e.Op(), "rolling_"):
			return d.evalRolling(*e.Target(), e.Op(), e.Name())
		case strings.HasPrefix(e.Op(), "over:"):
			return d.evalOver(*e.Target(), strings.TrimPrefix(e.Op(), "over:"), e.Name())
		}
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

func (d DataFrame) evalCumulative(target expr.Expr, name string, mode string) (series.Series, error) {
	base, err := d.evalExprAsSeriesVectorized(target)
	if err != nil {
		return series.Series{}, err
	}
	values := make([]any, d.height)
	sum := float64(0)
	count := int64(0)
	for i := 0; i < d.height; i++ {
		v := base.Value(i)
		if mode == "count" {
			if v != nil {
				count++
			}
			values[i] = count
			continue
		}
		switch t := v.(type) {
		case int64:
			sum += float64(t)
		case float64:
			sum += t
		}
		values[i] = sum
	}
	if mode == "count" {
		return series.New(name, dtypes.Int64, values)
	}
	return series.New(name, dtypes.Float64, values)
}

func (d DataFrame) evalRank(target expr.Expr, name string) (series.Series, error) {
	base, err := d.evalExprAsSeriesVectorized(target)
	if err != nil {
		return series.Series{}, err
	}
	indexes := make([]int, d.height)
	for i := 0; i < d.height; i++ {
		indexes[i] = i
	}
	sort.SliceStable(indexes, func(i, j int) bool {
		return compareAny(base.Value(indexes[i]), base.Value(indexes[j])) < 0
	})
	out := make([]any, d.height)
	for rank, idx := range indexes {
		out[idx] = int64(rank + 1)
	}
	return series.New(name, dtypes.Int64, out)
}

func (d DataFrame) evalRolling(target expr.Expr, op string, name string) (series.Series, error) {
	base, err := d.evalExprAsSeriesVectorized(target)
	if err != nil {
		return series.Series{}, err
	}
	window := 1
	if idx := strings.Index(op, ":"); idx >= 0 && idx+1 < len(op) {
		if w, parseErr := strconv.Atoi(op[idx+1:]); parseErr == nil && w > 0 {
			window = w
		}
	}
	mode := op
	if idx := strings.Index(mode, ":"); idx >= 0 {
		mode = mode[:idx]
	}
	out := make([]any, d.height)
	for i := 0; i < d.height; i++ {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		nums := make([]float64, 0, window)
		for j := start; j <= i; j++ {
			switch t := base.Value(j).(type) {
			case int64:
				nums = append(nums, float64(t))
			case float64:
				nums = append(nums, t)
			}
		}
		if len(nums) == 0 {
			out[i] = nil
			continue
		}
		switch mode {
		case "rolling_sum":
			sum := float64(0)
			for _, n := range nums {
				sum += n
			}
			out[i] = sum
		case "rolling_mean":
			sum := float64(0)
			for _, n := range nums {
				sum += n
			}
			out[i] = sum / float64(len(nums))
		case "rolling_min":
			min := nums[0]
			for _, n := range nums[1:] {
				if n < min {
					min = n
				}
			}
			out[i] = min
		case "rolling_max":
			max := nums[0]
			for _, n := range nums[1:] {
				if n > max {
					max = n
				}
			}
			out[i] = max
		case "rolling_std":
			sum := float64(0)
			for _, n := range nums {
				sum += n
			}
			mean := sum / float64(len(nums))
			variance := float64(0)
			for _, n := range nums {
				diff := n - mean
				variance += diff * diff
			}
			variance /= float64(len(nums))
			out[i] = math.Sqrt(variance)
		default:
			out[i] = nums[len(nums)-1]
		}
	}
	return series.New(name, dtypes.Float64, out)
}

func (d DataFrame) evalOver(target expr.Expr, partitionSpec string, name string) (series.Series, error) {
	partitions := []string{}
	if strings.TrimSpace(partitionSpec) != "" {
		partitions = strings.Split(partitionSpec, ",")
	}
	trimmed := make([]string, 0, len(partitions))
	for _, p := range partitions {
		p = strings.TrimSpace(p)
		if p != "" {
			trimmed = append(trimmed, p)
		}
	}
	partitions = trimmed
	base, err := d.evalExprAsSeriesVectorized(target)
	if err != nil {
		return series.Series{}, err
	}
	if len(partitions) == 0 {
		return series.New(name, base.DataType(), collectSeriesValues(base))
	}
	groups := map[string][]int{}
	for row := 0; row < d.height; row++ {
		key := ""
		for _, c := range partitions {
			s, ok := d.cols[c]
			if !ok {
				return series.Series{}, fmt.Errorf("partition column %s not found", c)
			}
			key += fmt.Sprintf("|%v", s.Value(row))
		}
		groups[key] = append(groups[key], row)
	}
	out := collectSeriesValues(base)
	if target.Kind() == expr.KindUnary && target.Op() == "cum_sum" {
		for _, idxs := range groups {
			sum := float64(0)
			for _, idx := range idxs {
				switch t := base.Value(idx).(type) {
				case int64:
					sum += float64(t)
				case float64:
					sum += t
				}
				out[idx] = sum
			}
		}
		return series.New(name, dtypes.Float64, out)
	}
	if target.Kind() == expr.KindUnary && target.Op() == "cum_count" {
		for _, idxs := range groups {
			count := int64(0)
			for _, idx := range idxs {
				if base.Value(idx) != nil {
					count++
				}
				out[idx] = count
			}
		}
		return series.New(name, dtypes.Int64, out)
	}
	if target.Kind() == expr.KindUnary && target.Op() == "rank" {
		for _, idxs := range groups {
			sort.SliceStable(idxs, func(i, j int) bool {
				return compareAny(base.Value(idxs[i]), base.Value(idxs[j])) < 0
			})
			for rank, idx := range idxs {
				out[idx] = int64(rank + 1)
			}
		}
		return series.New(name, dtypes.Int64, out)
	}
	return series.New(name, base.DataType(), out)
}

func collectSeriesValues(s series.Series) []any {
	out := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		out[i] = s.Value(i)
	}
	return out
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

func isSeriesSortedAsc(s series.Series) bool {
	for i := 1; i < s.Len(); i++ {
		if compareAny(s.Value(i-1), s.Value(i)) > 0 {
			return false
		}
	}
	return true
}

func hasSeriesNulls(s series.Series) bool {
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			return true
		}
	}
	return false
}

func hasSeriesNaN(s series.Series) bool {
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if f, ok := v.(float64); ok && math.IsNaN(f) {
			return true
		}
	}
	return false
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

func castValueToType(v any, dt dtypes.DataType) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch dt {
	case dtypes.Int64:
		switch t := v.(type) {
		case int64:
			return t, nil
		case float64:
			return int64(t), nil
		case string:
			n, err := strconv.ParseInt(t, 10, 64)
			if err != nil {
				return nil, err
			}
			return n, nil
		}
	case dtypes.Float64:
		switch t := v.(type) {
		case int64:
			return float64(t), nil
		case float64:
			return t, nil
		case string:
			n, err := strconv.ParseFloat(t, 64)
			if err != nil {
				return nil, err
			}
			return n, nil
		}
	case dtypes.String:
		return fmt.Sprintf("%v", v), nil
	case dtypes.Boolean:
		switch t := v.(type) {
		case bool:
			return t, nil
		case string:
			if t == "true" {
				return true, nil
			}
			if t == "false" {
				return false, nil
			}
		}
	}
	return nil, fmt.Errorf("cannot cast %T to %s", v, dt)
}

func valueEquals(left any, right any) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	if lf, ok := left.(float64); ok {
		rf, rok := right.(float64)
		if !rok {
			return false
		}
		if math.IsNaN(lf) && math.IsNaN(rf) {
			return true
		}
		return lf == rf
	}
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
}

func pearson(xs []float64, ys []float64) float64 {
	if len(xs) != len(ys) || len(xs) == 0 {
		return 0
	}
	meanX := 0.0
	meanY := 0.0
	for i := range xs {
		meanX += xs[i]
		meanY += ys[i]
	}
	meanX /= float64(len(xs))
	meanY /= float64(len(ys))
	numerator := 0.0
	denomX := 0.0
	denomY := 0.0
	for i := range xs {
		dx := xs[i] - meanX
		dy := ys[i] - meanY
		numerator += dx * dy
		denomX += dx * dx
		denomY += dy * dy
	}
	if denomX == 0 || denomY == 0 {
		return 0
	}
	return numerator / math.Sqrt(denomX*denomY)
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
