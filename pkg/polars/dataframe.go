package polars

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/exec"
	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
	icsv "github.com/h0rn3t/gopolars/pkg/io/csv"
	idatabase "github.com/h0rn3t/gopolars/pkg/io/database"
	iipc "github.com/h0rn3t/gopolars/pkg/io/ipc"
	ijson "github.com/h0rn3t/gopolars/pkg/io/json"
	iparquet "github.com/h0rn3t/gopolars/pkg/io/parquet"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
	"github.com/h0rn3t/gopolars/pkg/series"
)

type df struct {
	value frame.DataFrame
}

func NewDataFrame(input NewDataFrameInput) (DataFrame, error) {
	return fromFrame(frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: input.Columns}))
}

func NewDataFrameFromArrow(table iarrow.Table) (DataFrame, error) {
	return fromFrame(iarrow.FromTable(table))
}

type NewDataFrameInput struct {
	Columns []frame.SeriesInput
}

func fromFrame(v frame.DataFrame, err error) (DataFrame, error) {
	if err != nil {
		return nil, err
	}
	return &df{value: v}, nil
}

func (d *df) Schema() dtypes.Schema {
	return d.value.Schema()
}

func (d *df) CollectSchema() dtypes.Schema {
	return d.value.CollectSchema()
}

func (d *df) Height() int {
	return d.value.Height()
}

func (d *df) IsEmpty() bool {
	return d.value.IsEmpty()
}

func (d *df) EstimatedSize() int64 {
	return d.value.EstimatedSize()
}

func (d *df) Width() int {
	return d.value.Width()
}

func (d *df) Columns() []string {
	return d.value.Columns()
}

func (d *df) IterColumns() []Series {
	return d.GetColumns()
}

func (d *df) IterRows() []map[string]any {
	return d.ToDicts()
}

func (d *df) IterSlices(size int) []DataFrame {
	if size <= 0 {
		size = d.Height()
	}
	out := make([]DataFrame, 0, (d.Height()+size-1)/size)
	for start := 0; start < d.Height(); start += size {
		length := size
		if start+length > d.Height() {
			length = d.Height() - start
		}
		out = append(out, d.Slice(start, length))
	}
	return out
}

func (d *df) Dtypes() []dtypes.DataType {
	return d.value.Dtypes()
}

func (d *df) ToDicts() []map[string]any {
	return d.value.ToDicts()
}

func (d *df) Item(row int, column string) (any, error) {
	if row < 0 || row >= d.Height() {
		return nil, fmt.Errorf("row out of bounds")
	}
	s, ok := d.value.Series(column)
	if !ok {
		return nil, fmt.Errorf("column %s not found", column)
	}
	return s.Value(row), nil
}

func (d *df) GetColumn(name string) (Series, error) {
	s, err := d.value.GetColumn(name)
	if err != nil {
		return nil, err
	}
	return fromInternalSeries(s), nil
}

func (d *df) SubSelectColumns(columns ...string) (DataFrame, error) {
	if len(columns) == 0 {
		return d.Clone(), nil
	}
	exprs := make([]Expr, 0, len(columns))
	for _, c := range columns {
		exprs = append(exprs, Col(c))
	}
	return d.Select(exprs...)
}

func (d *df) GetColumnIndex(name string) int {
	return d.value.GetColumnIndex(name)
}

func (d *df) GetColumns() []Series {
	cols := d.value.GetColumns()
	out := make([]Series, 0, len(cols))
	for _, c := range cols {
		out = append(out, fromInternalSeries(c))
	}
	return out
}

func (d *df) NullCount() map[string]int {
	return d.value.NullCount()
}

func (d *df) Count() map[string]int {
	return d.value.Count()
}

func (d *df) NUnique(columns ...string) (int, error) {
	return d.value.NUnique(columns...)
}

func (d *df) ApproxNUnique(columns ...string) (int, error) {
	return d.value.ApproxNUnique(columns...)
}

func (d *df) IsDuplicated() Series {
	rows := d.ToDicts()
	counts := map[string]int{}
	keys := make([]string, len(rows))
	for i, r := range rows {
		k := fmt.Sprintf("%v", r)
		keys[i] = k
		counts[k]++
	}
	values := make([]any, len(rows))
	for i, k := range keys {
		values[i] = counts[k] > 1
	}
	s, _ := NewSeries(NewSeriesInput{Name: "is_duplicated", DType: Boolean, Values: values})
	return s
}

func (d *df) IsUnique() Series {
	dup := d.IsDuplicated()
	values := make([]any, dup.Len())
	for i := 0; i < dup.Len(); i++ {
		values[i] = !dup.Value(i).(bool)
	}
	s, _ := NewSeries(NewSeriesInput{Name: "is_unique", DType: Boolean, Values: values})
	return s
}

func (d *df) Flags() map[string]map[string]bool {
	return d.value.Flags()
}

func (d *df) Glimpse(maxRows int) string {
	return d.value.Glimpse(maxRows)
}

func (d *df) Series(name string) (Series, bool) {
	s, ok := d.value.Series(name)
	if !ok {
		return nil, false
	}
	return fromInternalSeries(s), true
}

func (d *df) Select(exprs ...Expr) (DataFrame, error) {
	next, err := d.value.Select(exprs...)
	return fromFrame(next, err)
}

func (d *df) Filter(predicate Expr) (DataFrame, error) {
	next, err := d.value.Filter(predicate)
	return fromFrame(next, err)
}

// FilterAggregateDirect evaluates pred and computes op over cols in a single
// masked pass, returning a column→value map. It is the eager fused fast path:
// it builds no lazy plan and materializes no filtered DataFrame, surviving-index
// slice, or sliced column — only the predicate bitmap and the result map. op is
// one of "sum", "min", "max", "mean", "count"; an empty cols aggregates every
// column. A zero-survivor predicate is gated to skip the reduction kernel and
// reports 0 for each requested column. Results match
// df.Lazy().Filter(pred).<reduce>().Collect(ctx) for supported float64 columns.
func (d *df) FilterAggregateDirect(pred Expr, op string, cols []string) (map[string]float64, error) {
	return d.value.FilterAggregateDirect(pred, op, cols)
}

func (d *df) WithColumns(exprs ...Expr) (DataFrame, error) {
	next, err := d.value.WithColumns(exprs...)
	return fromFrame(next, err)
}

func (d *df) WithRowCount(name string, offset int64) (DataFrame, error) {
	next, err := d.value.WithRowCount(name, offset)
	return fromFrame(next, err)
}

func (d *df) WithRowIndex(name string, offset int64) (DataFrame, error) {
	next, err := d.value.WithRowCount(name, offset)
	return fromFrame(next, err)
}

func (d *df) GroupBy(keys ...string) GroupBy {
	return groupBy{value: d.value.GroupBy(keys...)}
}

func (d *df) GroupByDynamic(input DynamicGroupInput) (DataFrame, error) {
	next, err := d.value.GroupByDynamic(
		input.By,
		input.Every,
		input.Period,
		input.Offset,
		input.Closed,
		input.Label,
		input.WindowColumn,
		input.AggExpr,
	)
	return fromFrame(next, err)
}

func (d *df) Join(input JoinInput) (DataFrame, error) {
	other, ok := input.Other.(*df)
	if !ok {
		return nil, fmt.Errorf("unsupported dataframe implementation")
	}
	next, err := d.value.Join(frame.JoinInput{
		Other:         other.value,
		LeftOn:        input.LeftOn,
		RightOn:       input.RightOn,
		How:           frame.JoinType(input.How),
		Suffix:        input.Suffix,
		AsofDirection: input.AsofDirection,
		AsofTolerance: input.AsofTolerance,
	})
	return fromFrame(next, err)
}

func (d *df) JoinAsof(input JoinInput) (DataFrame, error) {
	if input.How == "" {
		input.How = "asof"
	}
	return d.Join(input)
}

func (d *df) JoinWhere(predicate Expr) (DataFrame, error) {
	return d.Filter(predicate)
}

func (d *df) Sort(input SortInput) (DataFrame, error) {
	next, err := d.value.Sort(frame.SortInput{
		By:            input.By,
		Descending:    input.Descending,
		NullsLast:     input.NullsLast,
		MaintainOrder: input.MaintainOrder,
	})
	return fromFrame(next, err)
}

func (d *df) Concat(input ConcatInput) (DataFrame, error) {
	others := make([]frame.DataFrame, 0, len(input.Others))
	for _, other := range input.Others {
		next, ok := other.(*df)
		if !ok {
			return nil, fmt.Errorf("unsupported dataframe implementation")
		}
		others = append(others, next.value)
	}
	if input.How == "horizontal" {
		return fromFrame(frame.ConcatHorizontal(d.value, others...))
	}
	return fromFrame(frame.ConcatVertical(d.value, others...))
}

func (d *df) Sample(n int, seed int64) DataFrame {
	return &df{value: d.value.Sample(n, seed)}
}

func (d *df) GatherEvery(step int, offset int) DataFrame {
	return &df{value: d.value.GatherEvery(step, offset)}
}

func (d *df) Rechunk() DataFrame {
	return d.Clone()
}

func (d *df) ToNumpy() [][]any {
	rows := d.value.ToDicts()
	out := make([][]any, 0, len(rows))
	columns := d.value.Columns()
	for _, row := range rows {
		values := make([]any, 0, len(columns))
		for _, c := range columns {
			values = append(values, row[c])
		}
		out = append(out, values)
	}
	return out
}

func (d *df) ToPandas() []map[string]any {
	return d.ToDicts()
}

func (d *df) PartitionBy(columns ...string) ([]DataFrame, error) {
	if len(columns) == 0 {
		return []DataFrame{d.Clone()}, nil
	}
	records := d.ToDicts()
	groups := map[string][]int{}
	for i, rec := range records {
		key := ""
		for _, c := range columns {
			key += fmt.Sprintf("|%v", rec[c])
		}
		groups[key] = append(groups[key], i)
	}
	out := make([]DataFrame, 0, len(groups))
	for _, idxs := range groups {
		seriesOut := make([]series.Series, 0, len(d.value.Columns()))
		for _, name := range d.value.Columns() {
			s, _ := d.value.Series(name)
			seriesOut = append(seriesOut, s.Slice(idxs))
		}
		part, err := frame.New(frame.NewInput{Series: seriesOut})
		if err != nil {
			return nil, err
		}
		out = append(out, &df{value: part})
	}
	return out, nil
}

func (d *df) MatchToSchema(schema dtypes.Schema) (DataFrame, error) {
	out := d.Clone()
	current := map[string]dtypes.DataType{}
	for _, f := range out.Schema() {
		current[f.Name] = f.Type
	}
	for _, f := range schema {
		if _, ok := current[f.Name]; !ok {
			values := make([]any, out.Height())
			col, err := NewSeries(NewSeriesInput{Name: f.Name, DType: f.Type, Values: values})
			if err != nil {
				return nil, err
			}
			out, err = out.Hstack(col)
			if err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func (d *df) MergeSorted(other DataFrame, by string) (DataFrame, error) {
	return d.Concat(ConcatInput{Others: []DataFrame{other}, How: "vertical"})
}

func (d *df) SelectSeq(exprs ...Expr) (DataFrame, error) {
	return d.Select(exprs...)
}

func (d *df) ReplaceColumn(index int, column Series) (DataFrame, error) {
	if index < 0 || index >= d.Width() {
		return nil, fmt.Errorf("index out of bounds")
	}
	internal, err := toInternalSeries(column)
	if err != nil {
		return nil, err
	}
	out := make([]series.Series, 0, d.Width())
	for i, name := range d.Columns() {
		if i == index {
			out = append(out, internal.Clone())
			continue
		}
		s, _ := d.value.Series(name)
		out = append(out, s.Clone())
	}
	return fromFrame(frame.New(frame.NewInput{Series: out}))
}

func (d *df) Reverse() DataFrame {
	indexes := make([]int, d.Height())
	for i := 0; i < d.Height(); i++ {
		indexes[i] = d.Height() - 1 - i
	}
	out := make([]series.Series, 0, d.Width())
	for _, name := range d.Columns() {
		s, _ := d.value.Series(name)
		out = append(out, s.Slice(indexes))
	}
	next, err := frame.New(frame.NewInput{Series: out})
	if err != nil {
		return d
	}
	return &df{value: next}
}

func (d *df) Rolling(by string, value string, window time.Duration, output string) (DataFrame, error) {
	return d.RollingMean(RollingMeanInput{By: by, Value: value, Window: window, Output: output})
}

func (d *df) Row(index int) (map[string]any, error) {
	if index < 0 || index >= d.Height() {
		return nil, fmt.Errorf("row out of bounds")
	}
	rows := d.ToDicts()
	return rows[index], nil
}

func (d *df) Rows() []map[string]any {
	return d.ToDicts()
}

func (d *df) RowsByKey(column string) map[any][]map[string]any {
	out := map[any][]map[string]any{}
	for _, row := range d.ToDicts() {
		key := row[column]
		out[key] = append(out[key], row)
	}
	return out
}

func (d *df) NChunks() int {
	if d.Height() == 0 || d.Width() == 0 {
		return 0
	}
	return 1
}

func (d *df) Pipe(fn func(DataFrame) (DataFrame, error)) (DataFrame, error) {
	if fn == nil {
		return d, nil
	}
	return fn(d)
}

func (d *df) MapColumns(fn func(name string, s Series) (Series, error)) (DataFrame, error) {
	if fn == nil {
		return d, nil
	}
	out := make([]series.Series, 0, d.Width())
	for _, name := range d.Columns() {
		current, _ := d.GetColumn(name)
		next, err := fn(name, current)
		if err != nil {
			return nil, err
		}
		internal, err := toInternalSeries(next)
		if err != nil {
			return nil, err
		}
		out = append(out, internal)
	}
	return fromFrame(frame.New(frame.NewInput{Series: out}))
}

func (d *df) MapRows(fn func(row map[string]any) (map[string]any, error)) (DataFrame, error) {
	if fn == nil {
		return d, nil
	}
	rows := d.ToDicts()
	mapped := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		next, err := fn(r)
		if err != nil {
			return nil, err
		}
		mapped = append(mapped, next)
	}
	if len(mapped) == 0 {
		return d.Limit(0), nil
	}
	keys := make([]string, 0, len(mapped[0]))
	for k := range mapped[0] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	cols := make([]frame.SeriesInput, 0, len(keys))
	for _, k := range keys {
		values := make([]any, len(mapped))
		for i, row := range mapped {
			values[i] = row[k]
		}
		cols = append(cols, frame.SeriesInput{Name: k, Values: values})
	}
	return NewDataFrame(NewDataFrameInput{Columns: cols})
}

func (d *df) Plot(x string, y string) map[string]any {
	return map[string]any{
		"type": "line",
		"x":    x,
		"y":    y,
		"rows": d.Height(),
	}
}

func (d *df) Serialize() ([]byte, error) {
	return json.Marshal(d.ToDicts())
}

func (d *df) SetSorted(by string) (DataFrame, error) {
	return d.Sort(SortInput{By: []string{by}})
}

func (d *df) Shape() [2]int {
	return [2]int{d.Height(), d.Width()}
}

func (d *df) Shift(periods int) (DataFrame, error) {
	out := make([]series.Series, 0, d.Width())
	for _, name := range d.Columns() {
		s, _ := d.value.Series(name)
		values := make([]any, d.Height())
		for i := 0; i < d.Height(); i++ {
			src := i - periods
			if src >= 0 && src < d.Height() {
				values[i] = s.Value(src)
			}
		}
		next, err := series.New(name, s.DataType(), values)
		if err != nil {
			return nil, err
		}
		out = append(out, next)
	}
	return fromFrame(frame.New(frame.NewInput{Series: out}))
}

func (d *df) Show(maxRows int) string {
	return d.Glimpse(maxRows)
}

func (d *df) ShrinkToFit() DataFrame {
	return d.Clone()
}

func (d *df) SQL(ctx context.Context, query string) (LazyFrame, error) {
	return SQLFromDataFrame(ctx, d, query, "self")
}

func (d *df) Sql(ctx context.Context, query string) (LazyFrame, error) {
	return d.SQL(ctx, query)
}

func (d *df) Style() string {
	return d.ToInitRepr()
}

func (d *df) Upsample(by string, every time.Duration) (DataFrame, error) {
	if every <= 0 {
		return d, nil
	}
	return d.Sort(SortInput{By: []string{by}})
}

func (d *df) Limit(n int) DataFrame {
	return &df{value: d.value.Limit(n)}
}

func (d *df) Slice(offset int, length int) DataFrame {
	return &df{value: d.value.Slice(offset, length)}
}

func (d *df) Head(n int) DataFrame {
	return d.Limit(n)
}

func (d *df) Tail(n int) DataFrame {
	if n >= d.value.Height() {
		return d
	}
	if n <= 0 {
		return d.Limit(0)
	}
	start := d.value.Height() - n
	indexes := make([]int, 0, n)
	for i := start; i < d.value.Height(); i++ {
		indexes = append(indexes, i)
	}
	out := make([]series.Series, 0, len(d.value.Columns()))
	for _, name := range d.value.Columns() {
		s, _ := d.value.Series(name)
		out = append(out, s.Slice(indexes))
	}
	next, err := frame.New(frame.NewInput{Series: out})
	if err != nil {
		return d.Limit(0)
	}
	return &df{value: next}
}

func (d *df) BottomK(k int, by string) (DataFrame, error) {
	next, err := d.value.BottomK(k, by)
	return fromFrame(next, err)
}

func (d *df) Unique(columns ...string) (DataFrame, error) {
	next, err := d.value.Unique(columns...)
	return fromFrame(next, err)
}

func (d *df) Cast(mapping map[string]dtypes.DataType) (DataFrame, error) {
	next, err := d.value.Cast(mapping)
	return fromFrame(next, err)
}

func (d *df) Clear() DataFrame {
	return &df{value: d.value.Clear()}
}

func (d *df) Clone() DataFrame {
	return &df{value: d.value.Clone()}
}

func (d *df) DropInPlace(column string) (DataFrame, error) {
	next, err := d.value.DropInPlace(column)
	return fromFrame(next, err)
}

func (d *df) Equals(other DataFrame) (bool, error) {
	otherDF, ok := other.(*df)
	if !ok {
		return false, fmt.Errorf("unsupported dataframe implementation")
	}
	return d.value.Equals(otherDF.value)
}

func (d *df) Extend(other DataFrame) (DataFrame, error) {
	otherDF, ok := other.(*df)
	if !ok {
		return nil, fmt.Errorf("unsupported dataframe implementation")
	}
	next, err := d.value.Extend(otherDF.value)
	return fromFrame(next, err)
}

func (d *df) Hstack(columns ...Series) (DataFrame, error) {
	internal := make([]series.Series, 0, len(columns))
	for _, c := range columns {
		v, err := toInternalSeries(c)
		if err != nil {
			return nil, err
		}
		internal = append(internal, v)
	}
	next, err := d.value.Hstack(internal...)
	return fromFrame(next, err)
}

func (d *df) InsertColumn(index int, column Series) (DataFrame, error) {
	internal, err := toInternalSeries(column)
	if err != nil {
		return nil, err
	}
	next, err := d.value.InsertColumn(index, internal)
	return fromFrame(next, err)
}

func (d *df) FillNull(value any) (DataFrame, error) {
	next, err := d.value.FillNull(value)
	return fromFrame(next, err)
}

func (d *df) FillNaN(value float64) (DataFrame, error) {
	next, err := d.value.FillNaN(value)
	return fromFrame(next, err)
}

func (d *df) Interpolate(columns ...string) (DataFrame, error) {
	next, err := d.value.Interpolate(columns...)
	return fromFrame(next, err)
}

func (d *df) DropNaNs(columns ...string) DataFrame {
	return &df{value: d.value.DropNaNs(columns...)}
}

func (d *df) DropNulls(columns ...string) DataFrame {
	return &df{value: d.value.DropNulls(columns...)}
}

func (d *df) Fold(op string, columns []string, alias string) (DataFrame, error) {
	next, err := d.value.Fold(op, columns, alias)
	return fromFrame(next, err)
}

func (d *df) HashRows(seed uint64) ([]uint64, error) {
	return d.value.HashRows(seed)
}

func (d *df) Corr(columnA string, columnB string) (float64, error) {
	return d.value.Corr(columnA, columnB)
}

func (d *df) Describe() (DataFrame, error) {
	next, err := d.value.Describe()
	return fromFrame(next, err)
}

func (d *df) Max() map[string]any {
	out := map[string]any{}
	for _, col := range d.Columns() {
		s, _ := d.value.Series(col)
		var best any
		has := false
		for i := 0; i < s.Len(); i++ {
			v := s.Value(i)
			if v == nil {
				continue
			}
			if !has || compareAnyLocal(v, best) > 0 {
				best = v
				has = true
			}
		}
		out[col] = best
	}
	return out
}

func (d *df) Min() map[string]any {
	out := map[string]any{}
	for _, col := range d.Columns() {
		s, _ := d.value.Series(col)
		var best any
		has := false
		for i := 0; i < s.Len(); i++ {
			v := s.Value(i)
			if v == nil {
				continue
			}
			if !has || compareAnyLocal(v, best) < 0 {
				best = v
				has = true
			}
		}
		out[col] = best
	}
	return out
}

func (d *df) Mean() map[string]float64 {
	out := map[string]float64{}
	for _, col := range d.Columns() {
		s, _ := d.value.Series(col)
		sum := float64(0)
		cnt := 0
		for i := 0; i < s.Len(); i++ {
			if v, ok := toFloatLocal(s.Value(i)); ok {
				sum += v
				cnt++
			}
		}
		if cnt > 0 {
			out[col] = sum / float64(cnt)
		}
	}
	return out
}

func (d *df) Median() map[string]float64 {
	out := map[string]float64{}
	for _, col := range d.Columns() {
		s, _ := d.value.Series(col)
		vals := make([]float64, 0, s.Len())
		for i := 0; i < s.Len(); i++ {
			if v, ok := toFloatLocal(s.Value(i)); ok {
				vals = append(vals, v)
			}
		}
		if len(vals) == 0 {
			continue
		}
		sort.Float64s(vals)
		mid := len(vals) / 2
		if len(vals)%2 == 0 {
			out[col] = (vals[mid-1] + vals[mid]) / 2
		} else {
			out[col] = vals[mid]
		}
	}
	return out
}

func (d *df) Product() map[string]float64 {
	out := map[string]float64{}
	for _, col := range d.Columns() {
		s, _ := d.value.Series(col)
		prod := float64(1)
		cnt := 0
		for i := 0; i < s.Len(); i++ {
			if v, ok := toFloatLocal(s.Value(i)); ok {
				prod *= v
				cnt++
			}
		}
		if cnt > 0 {
			out[col] = prod
		}
	}
	return out
}

func (d *df) Quantile(q float64) map[string]float64 {
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}
	out := map[string]float64{}
	for _, col := range d.Columns() {
		s, _ := d.value.Series(col)
		vals := make([]float64, 0, s.Len())
		for i := 0; i < s.Len(); i++ {
			if v, ok := toFloatLocal(s.Value(i)); ok {
				vals = append(vals, v)
			}
		}
		if len(vals) == 0 {
			continue
		}
		sort.Float64s(vals)
		idx := int(math.Round(q * float64(len(vals)-1)))
		out[col] = vals[idx]
	}
	return out
}

func (d *df) Std() map[string]float64 {
	out := map[string]float64{}
	for _, col := range d.Columns() {
		s, _ := d.value.Series(col)
		vals := make([]float64, 0, s.Len())
		for i := 0; i < s.Len(); i++ {
			if v, ok := toFloatLocal(s.Value(i)); ok {
				vals = append(vals, v)
			}
		}
		if len(vals) < 2 {
			continue
		}
		mean := 0.0
		for _, v := range vals {
			mean += v
		}
		mean /= float64(len(vals))
		sumSq := 0.0
		for _, v := range vals {
			diff := v - mean
			sumSq += diff * diff
		}
		out[col] = math.Sqrt(sumSq / float64(len(vals)-1))
	}
	return out
}

func (d *df) Var() map[string]float64 {
	out := map[string]float64{}
	for k, v := range d.Std() {
		out[k] = v * v
	}
	return out
}

func (d *df) Sum() map[string]float64 {
	out := map[string]float64{}
	for _, col := range d.Columns() {
		s, _ := d.value.Series(col)
		sum := 0.0
		has := false
		for i := 0; i < s.Len(); i++ {
			if v, ok := toFloatLocal(s.Value(i)); ok {
				sum += v
				has = true
			}
		}
		if has {
			out[col] = sum
		}
	}
	return out
}

func (d *df) MaxHorizontal(alias string) (DataFrame, error) {
	return d.horizontalAgg(alias, "max")
}

func (d *df) MinHorizontal(alias string) (DataFrame, error) {
	return d.horizontalAgg(alias, "min")
}

func (d *df) MeanHorizontal(alias string) (DataFrame, error) {
	return d.horizontalAgg(alias, "mean")
}

func (d *df) SumHorizontal(alias string) (DataFrame, error) {
	return d.horizontalAgg(alias, "sum")
}

func (d *df) Remove(column string) (DataFrame, error) {
	return d.Drop(column)
}

func (d *df) Explode(columns ...string) (DataFrame, error) {
	next, err := d.value.Explode(columns...)
	return fromFrame(next, err)
}

func (d *df) FlattenStruct(column string, prefix string) (DataFrame, error) {
	next, err := d.value.FlattenStruct(column, prefix)
	return fromFrame(next, err)
}

func (d *df) Melt(input MeltInput) (DataFrame, error) {
	next, err := d.value.Melt(input.IDVars, input.ValueVars, input.VariableCol, input.ValueCol)
	return fromFrame(next, err)
}

func (d *df) Unpivot(input MeltInput) (DataFrame, error) {
	return d.Melt(input)
}

func (d *df) Unstack(by string) ([]DataFrame, error) {
	return d.PartitionBy(by)
}

func (d *df) Unnest(columns ...string) (DataFrame, error) {
	return d.Clone(), nil
}

func (d *df) Pivot(input PivotInput) (DataFrame, error) {
	idx := []string{}
	if input.Index != "" {
		idx = append(idx, input.Index)
	}
	next, err := d.value.Pivot(idx, input.Columns, input.Values, input.Agg)
	if err != nil {
		return nil, err
	}
	if input.ValueName != "" {
		return (&df{value: next}).Rename(map[string]string{input.Values: input.ValueName})
	}
	return fromFrame(next, nil)
}

func (d *df) Transpose() (DataFrame, error) {
	rows := d.Rows()
	if len(rows) == 0 {
		return d.Limit(0), nil
	}
	values := make([]frame.SeriesInput, 0, len(rows))
	cols := d.Columns()
	for i, col := range cols {
		rowVals := make([]any, 0, len(rows))
		for _, r := range rows {
			rowVals = append(rowVals, r[col])
		}
		values = append(values, frame.SeriesInput{Name: fmt.Sprintf("row_%d", i), Values: rowVals})
	}
	return NewDataFrame(NewDataFrameInput{Columns: values})
}

func (d *df) ToSeries(column string) (Series, error) {
	return d.GetColumn(column)
}

func (d *df) ToStruct() map[string]any {
	return map[string]any{
		"schema": d.Schema(),
		"rows":   d.Rows(),
	}
}

func (d *df) ToTorch() [][]float64 {
	return d.ToJax()
}

func (d *df) TopK(k int, by string) (DataFrame, error) {
	sorted, err := d.Sort(SortInput{By: []string{by}, Descending: []bool{true}})
	if err != nil {
		return nil, err
	}
	return sorted.Limit(k), nil
}

func (d *df) VStack(other DataFrame) (DataFrame, error) {
	return d.Concat(ConcatInput{Others: []DataFrame{other}, How: "vertical"})
}

func (d *df) Vstack(other DataFrame) (DataFrame, error) {
	return d.VStack(other)
}

func (d *df) Update(other DataFrame) (DataFrame, error) {
	return d.VStack(other)
}

func (d *df) WithColumnsSeq(exprs ...Expr) (DataFrame, error) {
	return d.WithColumns(exprs...)
}

func (d *df) RollingMean(input RollingMeanInput) (DataFrame, error) {
	next, err := d.value.RollingMean(input.By, input.Value, input.Window, input.MinRows, input.Output, input.Closed)
	return fromFrame(next, err)
}

func (d *df) Deserialize(payload []byte) (DataFrame, error) {
	next, err := d.value.Deserialize(payload)
	return fromFrame(next, err)
}

func (d *df) Drop(columns ...string) (DataFrame, error) {
	dropSet := make(map[string]struct{}, len(columns))
	for _, c := range columns {
		dropSet[c] = struct{}{}
	}
	out := make([]series.Series, 0, d.value.Width())
	for _, name := range d.value.Columns() {
		if _, shouldDrop := dropSet[name]; shouldDrop {
			continue
		}
		s, _ := d.value.Series(name)
		out = append(out, s.Clone())
	}
	return fromFrame(frame.New(frame.NewInput{Series: out}))
}

func (d *df) Rename(mapping map[string]string) (DataFrame, error) {
	if len(mapping) == 0 {
		return d, nil
	}
	out := make([]series.Series, 0, d.value.Width())
	for _, name := range d.value.Columns() {
		s, _ := d.value.Series(name)
		if nextName, ok := mapping[name]; ok && nextName != "" {
			out = append(out, s.Rename(nextName))
			continue
		}
		out = append(out, s.Clone())
	}
	return fromFrame(frame.New(frame.NewInput{Series: out}))
}

func (d *df) Lazy() LazyFrame {
	return &lf{
		source: d.value,
		engine: exec.New(),
		nodes:  []logical.Node{},
	}
}

func (d *df) WriteCSV(input WriteCSVInput) error {
	return icsv.Write(d.value, icsv.WriteInput{
		Path:          input.Path,
		IncludeHeader: input.IncludeHeader,
		Separator:     input.Separator,
	})
}

func (d *df) WriteCsv(input WriteCSVInput) error {
	return d.WriteCSV(input)
}

func (d *df) WriteJSON(input WriteJSONInput) error {
	return ijson.Write(d.value, ijson.WriteInput{
		Path:   input.Path,
		Pretty: input.Pretty,
		NDJSON: input.NDJSON,
	})
}

func (d *df) WriteJson(input WriteJSONInput) error {
	return d.WriteJSON(input)
}

func (d *df) WriteParquet(input WriteParquetInput) error {
	return iparquet.Write(d.value, iparquet.WriteInput{
		Path:         input.Path,
		Compression:  input.Compression,
		RowGroupSize: input.RowGroupSize,
	})
}

func (d *df) WriteIPC(input WriteIPCInput) error {
	return iipc.Write(d.value, iipc.WriteInput{Path: input.Path})
}

func (d *df) WriteIpc(input WriteIPCInput) error {
	return d.WriteIPC(input)
}

func (d *df) WriteIpcStream(input WriteIPCInput) error {
	return d.WriteIPC(input)
}

func (d *df) WriteNDJSON(input WriteJSONInput) error {
	input.NDJSON = true
	return d.WriteJSON(input)
}

func (d *df) WriteNdjson(input WriteJSONInput) error {
	return d.WriteNDJSON(input)
}

func (d *df) WriteAvro(path string) error {
	return fmt.Errorf("write_avro not supported: %s", path)
}

func (d *df) WriteClipboard() error {
	return fmt.Errorf("write_clipboard not supported")
}

func (d *df) WriteDatabase(input WriteDatabaseInput) (int64, error) {
	return idatabase.Write(context.Background(), d.value, idatabase.WriteInput{
		TableName:     input.TableName,
		IfTableExists: idatabase.IfTableExists(input.IfTableExists),
		Conn:          input.Conn,
		DriverName:    input.DriverName,
		DriverOptions: input.DriverOptions,
		BatchSize:     input.BatchSize,
	})
}

func (d *df) WriteDelta(path string) error {
	return fmt.Errorf("write_delta not supported: %s", path)
}

func (d *df) WriteExcel(path string) error {
	return fmt.Errorf("write_excel not supported: %s", path)
}

func (d *df) WriteIceberg(path string) error {
	return fmt.Errorf("write_iceberg not supported: %s", path)
}

func (d *df) ToArrow(input ToArrowInput) (iarrow.Table, error) {
	_ = input
	return iarrow.ToTable(d.value), nil
}

func (d *df) ToDict() map[string][]any {
	out := map[string][]any{}
	for _, name := range d.Columns() {
		s, _ := d.value.Series(name)
		values := make([]any, d.Height())
		for i := 0; i < d.Height(); i++ {
			values[i] = s.Value(i)
		}
		out[name] = values
	}
	return out
}

func (d *df) ToDummies(columns ...string) (DataFrame, error) {
	targets := map[string]struct{}{}
	if len(columns) == 0 {
		for _, c := range d.Columns() {
			targets[c] = struct{}{}
		}
	} else {
		for _, c := range columns {
			targets[c] = struct{}{}
		}
	}
	base := d.ToDict()
	newCols := make([]frame.SeriesInput, 0)
	for _, name := range d.Columns() {
		values := base[name]
		if _, ok := targets[name]; !ok {
			newCols = append(newCols, frame.SeriesInput{Name: name, Values: anySlice(values)})
			continue
		}
		uniq := map[string]struct{}{}
		for _, v := range values {
			uniq[fmt.Sprintf("%v", v)] = struct{}{}
		}
		keys := make([]string, 0, len(uniq))
		for k := range uniq {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			dummyVals := make([]any, len(values))
			for i, v := range values {
				dummyVals[i] = fmt.Sprintf("%v", v) == k
			}
			colName := name + "_" + strings.ReplaceAll(k, " ", "_")
			newCols = append(newCols, frame.SeriesInput{Name: colName, Values: dummyVals})
		}
	}
	return NewDataFrame(NewDataFrameInput{Columns: newCols})
}

func (d *df) ToInitRepr() string {
	return fmt.Sprintf("DataFrame(shape=%v, columns=%v)", d.Shape(), d.Columns())
}

func (d *df) ToJax() [][]float64 {
	out := make([][]float64, d.Height())
	for i := 0; i < d.Height(); i++ {
		row := make([]float64, 0, d.Width())
		for _, c := range d.Columns() {
			s, _ := d.value.Series(c)
			if v, ok := toFloatLocal(s.Value(i)); ok {
				row = append(row, v)
			} else {
				row = append(row, 0)
			}
		}
		out[i] = row
	}
	return out
}

func (d *df) horizontalAgg(alias string, mode string) (DataFrame, error) {
	if alias == "" {
		alias = mode + "_horizontal"
	}
	values := make([]any, d.Height())
	for row := 0; row < d.Height(); row++ {
		nums := make([]float64, 0, d.Width())
		for _, col := range d.Columns() {
			s, _ := d.value.Series(col)
			if v, ok := toFloatLocal(s.Value(row)); ok {
				nums = append(nums, v)
			}
		}
		if len(nums) == 0 {
			values[row] = nil
			continue
		}
		switch mode {
		case "max":
			best := nums[0]
			for _, n := range nums[1:] {
				if n > best {
					best = n
				}
			}
			values[row] = best
		case "min":
			best := nums[0]
			for _, n := range nums[1:] {
				if n < best {
					best = n
				}
			}
			values[row] = best
		case "sum":
			sum := 0.0
			for _, n := range nums {
				sum += n
			}
			values[row] = sum
		default:
			sum := 0.0
			for _, n := range nums {
				sum += n
			}
			values[row] = sum / float64(len(nums))
		}
	}
	seriesOut, err := NewSeries(NewSeriesInput{Name: alias, DType: Float64, Values: values})
	if err != nil {
		return nil, err
	}
	return d.Hstack(seriesOut)
}

func anySlice(values []any) []any {
	out := make([]any, len(values))
	copy(out, values)
	return out
}

func toFloatLocal(v any) (float64, bool) {
	switch t := v.(type) {
	case int64:
		return float64(t), true
	case float64:
		if math.IsNaN(t) {
			return 0, false
		}
		return t, true
	default:
		return 0, false
	}
}

func compareAnyLocal(left any, right any) int {
	lf, lok := toFloatLocal(left)
	rf, rok := toFloatLocal(right)
	if lok && rok {
		switch {
		case lf < rf:
			return -1
		case lf > rf:
			return 1
		default:
			return 0
		}
	}
	ls := fmt.Sprintf("%v", left)
	rs := fmt.Sprintf("%v", right)
	switch {
	case ls < rs:
		return -1
	case ls > rs:
		return 1
	default:
		return 0
	}
}
