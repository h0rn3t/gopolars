package frame

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

type DataFrame struct {
	schema dtypes.Schema
	cols   map[string]series.Series
	order  []string
	height int
	// context holds columns made referenceable by LazyFrame.with_context but not
	// part of this frame's own columns. They may have a different length and are
	// resolved only as a fallback (typically via an aggregation like .first()).
	context map[string]series.Series
}

// WithContextColumns returns a copy of d that also resolves the given columns as
// context (LazyFrame.with_context). They are not added to the frame's own columns.
func (d DataFrame) WithContextColumns(cols map[string]series.Series) DataFrame {
	if len(cols) == 0 {
		return d
	}
	merged := make(map[string]series.Series, len(d.context)+len(cols))
	for k, v := range d.context {
		merged[k] = v
	}
	for k, v := range cols {
		merged[k] = v
	}
	out := d
	out.context = merged
	return out
}

// contextColumn resolves a name against the context columns.
func (d DataFrame) contextColumn(name string) (series.Series, bool) {
	if d.context == nil {
		return series.Series{}, false
	}
	s, ok := d.context[name]
	return s, ok
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
	keyColumns := make([]*chunk.Column, len(keys))
	for j, c := range keys {
		s, ok := d.cols[c]
		if !ok {
			return 0, fmt.Errorf("column %s not found", c)
		}
		keyColumns[j] = s.Column()
	}
	_, firstRow := chunk.GroupIDs(keyColumns, d.height)
	return len(firstRow), nil
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
	fmt.Fprintf(&b, "rows=%d cols=%d\n", d.height, len(d.order))
	for _, f := range d.schema {
		fmt.Fprintf(&b, "%s:%s ", f.Name, f.Type)
	}
	b.WriteString("\n")
	for row := 0; row < maxRows; row++ {
		fmt.Fprintf(&b, "[%d] ", row)
		for _, col := range d.order {
			fmt.Fprintf(&b, "%s=%v ", col, d.cols[col].Value(row))
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

// expandColExprs rewrites multi-column selectors and struct wildcards into one
// concrete expr per output column. Handles: regex pl.col("^...$"), pl.all(),
// pl.col("a","b"), pl.exclude(...), <selector>.exclude(...), and a struct
// wildcard pl.col("s").struct.field("*") (expanded to one field column each). All
// other exprs pass through unchanged.
func (d DataFrame) expandColExprs(exprs []expr.Expr) ([]expr.Expr, error) {
	needsExpand := false
	for _, e := range exprs {
		if _, ok := expr.SelectorColumns(e, d.order); ok {
			needsExpand = true
			break
		}
		if d.isStructWildcard(e) {
			needsExpand = true
			break
		}
	}
	if !needsExpand {
		return exprs, nil
	}
	out := make([]expr.Expr, 0, len(exprs))
	for _, e := range exprs {
		if cols, ok := expr.SelectorColumns(e, d.order); ok {
			for _, name := range cols {
				out = append(out, expr.Col(name))
			}
			continue
		}
		if d.isStructWildcard(e) {
			target := e.Target()
			// pl.struct([...]).struct.field("*") unpacks back to the source columns.
			if target.Op() == "struct_pack" {
				for _, f := range target.Names() {
					out = append(out, expr.Col(f))
				}
				continue
			}
			for _, f := range d.structFieldNames(target.ColName()) {
				out = append(out, target.StructField(f).Alias(f))
			}
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// isStructWildcard reports whether e is <struct>.struct.field("*"), where <struct>
// is either an existing struct column or an inline pl.struct([...]).
func (d DataFrame) isStructWildcard(e expr.Expr) bool {
	if e.Kind() != expr.KindUnary || e.Op() != "struct_field:*" {
		return false
	}
	t := e.Target()
	if t == nil {
		return false
	}
	if t.Op() == "struct_pack" {
		return true
	}
	if t.Kind() != expr.KindCol {
		return false
	}
	_, ok := d.cols[t.ColName()]
	return ok
}

// structFieldNames returns the union of field names in a struct column, sorted
// (gopolars structs are map-backed and don't preserve insertion order).
func (d DataFrame) structFieldNames(column string) []string {
	base, ok := d.cols[column]
	if !ok {
		return nil
	}
	keys := map[string]struct{}{}
	for i := 0; i < d.height; i++ {
		if m, ok := base.Value(i).(map[string]any); ok {
			for k := range m {
				keys[k] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(keys))
	for k := range keys {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// allRowIndices returns [0,1,...,height-1] for whole-frame aggregation.
func (d DataFrame) allRowIndices() []int {
	idx := make([]int, d.height)
	for i := range idx {
		idx[i] = i
	}
	return idx
}

// aggToScalar reduces an aggregation node to a single scalar over the whole frame.
// It backs MapAggregates so aggregations nested in row-wise expressions (e.g.
// pl.col("b") + pl.col("c").first()) precompute to a broadcast constant. Returns
// ok=false for non-aggregation nodes (the caller then recurses into children).
func (d DataFrame) aggToScalar(e expr.Expr) (any, bool, error) {
	if e.Kind() == expr.KindAgg {
		v, err := (GroupBy{df: d}).evalAgg(e, d.allRowIndices())
		return v, true, err
	}
	if e.Kind() == expr.KindUnary && e.Target() != nil {
		switch e.Op() {
		case "first", "last":
			s, err := d.evalExprAsSeriesVectorized(*e.Target())
			if err != nil {
				return nil, false, err
			}
			if s.Len() == 0 {
				return nil, true, nil
			}
			if e.Op() == "first" {
				return s.Value(0), true, nil
			}
			return s.Value(s.Len() - 1), true, nil
		}
	}
	return nil, false, nil
}

// foldAggregates rewrites each expr, replacing any aggregation sub-expression with
// its precomputed scalar (broadcast). allScalar is true when every expr is itself
// a top-level aggregation (so a Select should reduce to a single row, like Polars);
// it is false when any expr is a full-length column or a literal-only projection.
func (d DataFrame) foldAggregates(exprs []expr.Expr) (out []expr.Expr, allScalar bool, err error) {
	out = make([]expr.Expr, len(exprs))
	allScalar = len(exprs) > 0
	for i, e := range exprs {
		if _, isTopAgg, aerr := d.aggToScalar(e); aerr != nil {
			return nil, false, aerr
		} else if !isTopAgg {
			allScalar = false
		}
		folded, ferr := expr.MapAggregates(e, d.aggToScalar)
		if ferr != nil {
			return nil, false, ferr
		}
		out[i] = folded
	}
	return out, allScalar, nil
}

func (d DataFrame) Select(exprs ...expr.Expr) (DataFrame, error) {
	if len(exprs) == 0 {
		return d, nil
	}
	exprs, err := d.expandColExprs(exprs)
	if err != nil {
		return DataFrame{}, err
	}
	exprs, allScalar, err := d.foldAggregates(exprs)
	if err != nil {
		return DataFrame{}, err
	}
	outSeries := make([]series.Series, 0, len(exprs))
	for _, ex := range exprs {
		col, err := d.evalExprAsSeriesVectorized(ex)
		if err != nil {
			return DataFrame{}, err
		}
		// A pure-aggregation select reduces to a single row (Polars semantics):
		// the broadcast scalar columns are sliced to their first element.
		if allScalar && col.Len() > 1 {
			col = col.Slice([]int{0})
		}
		outSeries = append(outSeries, col)
	}
	out, err := New(NewInput{Series: outSeries})
	if err != nil {
		return DataFrame{}, err
	}
	return out.WithContextColumns(d.context), nil
}

func (d DataFrame) Filter(predicate expr.Expr) (DataFrame, error) {
	if useTypedStorage() {
		if df, ok, err := d.filterBatch(predicate); ok {
			return df, err
		}
	}
	mask := make([]bool, d.height)
	if err := d.parallelForRows(func(start int, end int) error {
		for i := start; i < end; i++ {
			v, err := expr.Eval(predicate, rowAccessor{df: d, row: i})
			if err != nil {
				return err
			}
			// A null predicate (e.g. a comparison against a null cell) drops the
			// row, matching Polars: only rows where the predicate is True survive.
			if v == nil {
				mask[i] = false
				continue
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
	return New(NewInput{Series: d.takeColumns(keep)})
}

func (d DataFrame) WithColumns(exprs ...expr.Expr) (DataFrame, error) {
	exprs, err := d.expandColExprs(exprs)
	if err != nil {
		return DataFrame{}, err
	}
	if exprs, _, err = d.foldAggregates(exprs); err != nil {
		return DataFrame{}, err
	}
	out := d.shallowClone()
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
	// Fast path: a single numeric key, no nulls/NaN — radix argsort in O(n)
	// instead of an O(n log n) comparison sort. Falls back below for multi-key,
	// nullable, NaN, or non-numeric sorts (preserving NaN-last / nulls ordering).
	if idx, ok := d.radixArgsort(input); ok {
		return New(NewInput{Series: d.takeColumns(idx)})
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
	// Pre-extract typed comparators so the sort hot path reads chunk buffers
	// directly instead of calling s.Value(i) (boxing) per comparison.
	comparators := buildColumnComparators(sortSeries)
	sort.Slice(indexes, func(i, j int) bool {
		for colIdx, cmp := range comparators {
			c := cmp(indexes[i], indexes[j], input.NullsLast)
			if c == 0 {
				continue
			}
			desc := colIdx < len(input.Descending) && input.Descending[colIdx]
			if desc {
				return c > 0
			}
			return c < 0
		}
		return false
	})
	return New(NewInput{Series: d.takeColumns(indexes)})
}

// radixArgsort returns an index permutation for a sort whose leading key is a
// null-free, NaN-free numeric column, or ok=false to fall back to the comparator
// sort. The leading key is ordered by the O(n) radix (parallel-merged above its
// threshold); for a multi-key sort, ties on the leading key are then resolved by
// the existing stable comparators over the remaining keys. NaN-last and null
// handling are only relevant on the fallback path, which this excludes for the
// leading key.
func (d DataFrame) radixArgsort(input SortInput) ([]int, bool) {
	if len(input.By) == 0 || d.height < radixSortThreshold {
		return nil, false
	}
	s, ok := d.cols[input.By[0]]
	if !ok {
		return nil, false
	}
	col := s.Column()
	if col == nil || col.NullCount() != 0 {
		return nil, false
	}
	// equalLead reports whether two rows share the leading-key value, used to
	// delimit the equal-key runs that multi-key sorts tie-break.
	var idx []int
	var equalLead func(a, b int) bool
	switch col.DataType() {
	case dtypes.Float64:
		f64s, _ := col.Float64s()
		if anyNaN(f64s) {
			return nil, false // NaN ordering differs; use the comparator path
		}
		idx = chunk.ArgsortFloat64Parallel(f64s)
		equalLead = func(a, b int) bool { return f64s[a] == f64s[b] }
	case dtypes.Int64:
		i64s, _ := col.Int64s()
		idx = chunk.ArgsortInt64Parallel(i64s)
		equalLead = func(a, b int) bool { return i64s[a] == i64s[b] }
	default:
		return nil, false
	}
	if len(input.Descending) > 0 && input.Descending[0] {
		for i, j := 0, len(idx)-1; i < j; i, j = i+1, j-1 {
			idx[i], idx[j] = idx[j], idx[i]
		}
	}
	if len(input.By) == 1 {
		return idx, true
	}
	if !d.resolveSecondaryTies(idx, input, equalLead) {
		return nil, false // a secondary key is missing; fall back to report it
	}
	return idx, true
}

// resolveSecondaryTies stable-sorts each maximal run of equal-leading-key rows in
// idx by the remaining sort keys, using the same typed comparators and
// per-key descending flags as the comparison-sort fallback. It returns false if a
// secondary key column is missing (so the caller falls back and surfaces the
// error). idx is already ordered by the leading key, so equal-leading rows are
// contiguous.
func (d DataFrame) resolveSecondaryTies(idx []int, input SortInput, equalLead func(a, b int) bool) bool {
	secSeries := make([]series.Series, 0, len(input.By)-1)
	for _, by := range input.By[1:] {
		s, ok := d.cols[by]
		if !ok {
			return false
		}
		secSeries = append(secSeries, s)
	}
	comps := buildColumnComparators(secSeries)
	// cmp compares two rows by the remaining keys (respecting each key's
	// descending flag); hoisted out of the run loop so per-run stable sorts add no
	// closure/boxing allocations. slices.SortStableFunc sorts the []int run in
	// place without boxing it to any.
	cmp := func(p, q int) int {
		for ci, c := range comps {
			r := c(p, q, input.NullsLast)
			if r == 0 {
				continue
			}
			if ci+1 < len(input.Descending) && input.Descending[ci+1] {
				return -r
			}
			return r
		}
		return 0
	}
	n := len(idx)
	start := 0
	for end := 1; end <= n; end++ {
		if end == n || !equalLead(idx[start], idx[end]) {
			if end-start > 1 {
				slices.SortStableFunc(idx[start:end], cmp)
			}
			start = end
		}
	}
	return true
}

// radixSortThreshold mirrors chunk.radixThreshold: below it a comparison sort
// wins on constant factors.
const radixSortThreshold = 256

// anyNaN reports whether f64s contains a NaN. It gates the radix fast paths
// (rank and sort): the LSD radix key transform is undefined for NaN, so a NaN
// forces the fallback to the comparison path, which sorts NaN last.
func anyNaN(f64s []float64) bool {
	for _, v := range f64s {
		if math.IsNaN(v) {
			return true
		}
	}
	return false
}

// columnComparatorFn compares rows i and j, returning -1, 0, or 1.
type columnComparatorFn func(i, j int, nullsLast bool) int

// buildColumnComparators returns one comparator per sort key. For Float64 and
// Int64 columns the comparator reads the typed backing buffer directly;
// all other dtypes fall back to s.Value(i) boxing.
func buildColumnComparators(cols []series.Series) []columnComparatorFn {
	out := make([]columnComparatorFn, len(cols))
	for k, s := range cols {
		col := s.Column()
		if f64s, ok := col.Float64s(); ok {
			nulls := col.Nulls()
			f64sCopy := f64s
			nullsCopy := nulls
			out[k] = func(i, j int, nullsLast bool) int {
				ni := nullsCopy != nil && nullsCopy[i]
				nj := nullsCopy != nil && nullsCopy[j]
				if ni || nj {
					return compareNulls(ni, nj, nullsLast)
				}
				lv, rv := f64sCopy[i], f64sCopy[j]
				// NaN always sorts last, matching compareSortValues semantics.
				lNaN, rNaN := math.IsNaN(lv), math.IsNaN(rv)
				if lNaN && rNaN {
					return 0
				}
				if lNaN {
					return 1
				}
				if rNaN {
					return -1
				}
				switch {
				case lv < rv:
					return -1
				case lv > rv:
					return 1
				default:
					return 0
				}
			}
			continue
		}
		if i64s, ok := col.Int64s(); ok {
			nulls := col.Nulls()
			i64sCopy := i64s
			nullsCopy := nulls
			out[k] = func(i, j int, nullsLast bool) int {
				ni := nullsCopy != nil && nullsCopy[i]
				nj := nullsCopy != nil && nullsCopy[j]
				if ni || nj {
					return compareNulls(ni, nj, nullsLast)
				}
				switch {
				case i64sCopy[i] < i64sCopy[j]:
					return -1
				case i64sCopy[i] > i64sCopy[j]:
					return 1
				default:
					return 0
				}
			}
			continue
		}
		// Generic fallback for other dtypes.
		sCopy := s
		out[k] = func(i, j int, nullsLast bool) int {
			return compareSortValues(sCopy.Value(i), sCopy.Value(j), nullsLast)
		}
	}
	return out
}

// compareNulls returns the ordering when at least one side is null.
func compareNulls(ni, nj, nullsLast bool) int {
	if ni && nj {
		return 0
	}
	if ni {
		if nullsLast {
			return 1
		}
		return -1
	}
	if nullsLast {
		return -1
	}
	return 1
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
	if length <= 0 || d.height == 0 {
		return d.Limit(0)
	}
	// A negative offset counts from the end; the window [offset, offset+length)
	// is then clamped to [0, height), so a window starting before row 0 keeps
	// only its overlap (matching Polars) rather than sliding forward.
	if offset < 0 {
		offset = d.height + offset
	}
	end := offset + length
	if offset < 0 {
		offset = 0
	}
	if end > d.height {
		end = d.height
	}
	if offset >= d.height || end <= offset {
		return d.Limit(0)
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
	fillF, fillIsFloat := value.(float64)
	out := make([]series.Series, 0, len(d.order))
	for _, f := range d.schema {
		s := d.cols[f.Name]
		col := s.Column()
		// No nulls -> fill is a no-op; reuse the column by pointer (zero-copy).
		if col != nil && col.NullCount() == 0 {
			col.MarkShared()
			out = append(out, series.FromColumn(f.Name, col))
			continue
		}
		// Typed fast path: float64 column filled with a float64 literal.
		if fillIsFloat {
			if c, ok := col.FillNullFloat64(fillF); ok {
				out = append(out, series.FromColumn(f.Name, c))
				continue
			}
		}
		values := make([]any, 0, d.height)
		for i := 0; i < d.height; i++ {
			if s.IsNull(i) {
				values = append(values, value)
				continue
			}
			values = append(values, s.Value(i))
		}
		boxed, err := series.New(f.Name, f.Type, values)
		if err != nil {
			return DataFrame{}, err
		}
		out = append(out, boxed)
	}
	return New(NewInput{Series: out})
}

func (d DataFrame) FillNaN(value float64) (DataFrame, error) {
	out := make([]series.Series, 0, len(d.order))
	for _, f := range d.schema {
		s := d.cols[f.Name]
		col := s.Column()
		// Typed fast path for float64 columns.
		if c, ok := col.FillNaNFloat64(value); ok {
			out = append(out, series.FromColumn(f.Name, c))
			continue
		}
		// Non-float columns cannot contain NaN; reuse them by pointer (zero-copy).
		if col != nil {
			col.MarkShared()
			out = append(out, series.FromColumn(f.Name, col))
			continue
		}
		values := make([]any, 0, d.height)
		for i := 0; i < d.height; i++ {
			v := s.Value(i)
			if fv, ok := v.(float64); ok && math.IsNaN(fv) {
				values = append(values, value)
				continue
			}
			values = append(values, v)
		}
		boxed, err := series.New(f.Name, f.Type, values)
		if err != nil {
			return DataFrame{}, err
		}
		out = append(out, boxed)
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

// dropTargets returns the column names DropNulls/DropNaNs must inspect: the
// named columns that exist (in column order), or every column when none are
// named. Missing names are ignored, matching the prior per-row behavior.
func (d DataFrame) dropTargets(columns []string) []string {
	if len(columns) == 0 {
		return d.order
	}
	out := make([]string, 0, len(columns))
	for _, name := range d.order {
		if slices.Contains(columns, name) {
			out = append(out, name)
		}
	}
	return out
}

// gatherRows materializes the rows at keep across every column, gathering
// columns concurrently when the gather is large enough to amortize it.
func (d DataFrame) gatherRows(keep []int) DataFrame {
	df, _ := New(NewInput{Series: d.takeColumns(keep)})
	return df
}

// keepFromDropped returns the ascending indices of rows whose dropped bit is
// clear.
func keepFromDropped(dropped []bool, height int) []int {
	keep := make([]int, 0, height)
	for row := 0; row < height; row++ {
		if !dropped[row] {
			keep = append(keep, row)
		}
	}
	return keep
}

func (d DataFrame) DropNaNs(columns ...string) DataFrame {
	// Only float64 columns can hold NaN, so the drop mask is built column-at-a-
	// time over the typed backing buffers — no per-row boxing via Value(row).
	var dropped []bool
	for _, name := range d.dropTargets(columns) {
		s := d.cols[name]
		col := s.Column()
		if col != nil {
			f64s, ok := col.Float64s()
			if !ok {
				continue
			}
			nulls := col.Nulls()
			for row, v := range f64s {
				if (nulls == nil || !nulls[row]) && math.IsNaN(v) {
					if dropped == nil {
						dropped = make([]bool, d.height)
					}
					dropped[row] = true
				}
			}
			continue
		}
		// Defensive fallback for a column without a typed chunk.
		for row := 0; row < d.height; row++ {
			if fv, ok := s.Value(row).(float64); ok && math.IsNaN(fv) {
				if dropped == nil {
					dropped = make([]bool, d.height)
				}
				dropped[row] = true
			}
		}
	}
	if dropped == nil {
		return d // no NaN in scope: share the existing columns
	}
	return d.gatherRows(keepFromDropped(dropped, d.height))
}

func (d DataFrame) DropNulls(columns ...string) DataFrame {
	// A null-free column drops nothing, so it is skipped via the cached null
	// count; columns with nulls contribute their validity mask to the drop set
	// in a single column-at-a-time pass instead of a per-row IsNull probe.
	var dropped []bool
	for _, name := range d.dropTargets(columns) {
		s := d.cols[name]
		col := s.Column()
		if col != nil {
			if col.NullCount() == 0 {
				continue
			}
			for row, isNull := range col.Nulls() {
				if isNull {
					if dropped == nil {
						dropped = make([]bool, d.height)
					}
					dropped[row] = true
				}
			}
			continue
		}
		// Defensive fallback for a column without a typed chunk.
		for row := 0; row < d.height; row++ {
			if s.IsNull(row) {
				if dropped == nil {
					dropped = make([]bool, d.height)
				}
				dropped[row] = true
			}
		}
	}
	if dropped == nil {
		return d // no nulls in scope: share the existing columns
	}
	return d.gatherRows(keepFromDropped(dropped, d.height))
}

func (d DataFrame) Unique(columns ...string) (DataFrame, error) {
	keys := columns
	if len(keys) == 0 {
		keys = d.order
	}
	keyColumns := make([]*chunk.Column, len(keys))
	for j, c := range keys {
		s, ok := d.cols[c]
		if !ok {
			return DataFrame{}, fmt.Errorf("column %s not found", c)
		}
		keyColumns[j] = s.Column()
	}
	// firstRow holds the first-seen row index per distinct key, in encounter
	// order — exactly the rows kept by unique() — with no per-row boxing.
	_, keep := chunk.GroupIDs(keyColumns, d.height)
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
		_, _ = fmt.Fprintf(h, "%d", seed)
		for _, col := range d.order {
			_, _ = fmt.Fprintf(h, "|%v", d.cols[col].Value(row))
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
	// Emit field columns in a deterministic (sorted) order; map-backed structs
	// don't preserve insertion order.
	keyList := make([]string, 0, len(keys))
	for k := range keys {
		keyList = append(keyList, k)
	}
	sort.Strings(keyList)
	out := make([]series.Series, 0, len(d.order)+len(keys))
	for _, name := range d.order {
		if name == column {
			continue
		}
		s, _ := d.Series(name)
		out = append(out, s.Clone())
	}
	for _, key := range keyList {
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

func (d DataFrame) WithRowIndex(name string, offset int64) (DataFrame, error) {
	return d.WithRowCount(name, offset)
}

func (d DataFrame) Shift(periods int) (DataFrame, error) {
	if periods == 0 {
		return d.clone(), nil
	}
	out := make([]series.Series, 0, len(d.order))
	for _, name := range d.order {
		out = append(out, d.cols[name].Shift(periods))
	}
	return New(NewInput{Series: out})
}

func (d DataFrame) SetSorted(by string) (DataFrame, error) {
	return d, nil // Logical metadata operation, in this context just return the frame
}

func (d DataFrame) Unnest(columns ...string) (DataFrame, error) {
	return d.FlattenStruct(columns[0], "") // simplified unnest
}

func (d DataFrame) Unpivot(idVars []string, valueVars []string, variableCol string, valueCol string) (DataFrame, error) {
	return d.Melt(idVars, valueVars, variableCol, valueCol)
}

func (d DataFrame) Update(other DataFrame) (DataFrame, error) {
	// naive update implementation replacing matching columns
	out := d.clone()
	for _, name := range other.Columns() {
		if _, ok := out.cols[name]; ok {
			out.cols[name] = other.cols[name].Clone()
		}
	}
	return out, nil
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

// shallowClone copies the frame's structure (schema, order, column map) in
// O(columns) without deep-copying any column buffer. Each reused column is
// marked shared so the copy-on-write contract guards it against any future
// in-place mutation. Use this for derivations (e.g. WithColumns) that reuse
// unchanged columns by pointer; use clone() only when callers need independent
// column buffers.
func (d DataFrame) shallowClone() DataFrame {
	out := DataFrame{
		schema:  make(dtypes.Schema, len(d.schema)),
		cols:    make(map[string]series.Series, len(d.cols)),
		order:   make([]string, len(d.order)),
		height:  d.height,
		context: d.context,
	}
	copy(out.schema, d.schema)
	copy(out.order, d.order)
	for k, v := range d.cols {
		if c := v.Column(); c != nil {
			c.MarkShared()
		}
		out.cols[k] = v
	}
	return out
}

func (d DataFrame) evalExprAsSeries(e expr.Expr) (series.Series, error) {
	if e.Kind() == expr.KindCol {
		s, ok := d.cols[e.ColName()]
		if !ok {
			// Fall back to a with_context column (kept at its own length).
			if cs, cok := d.contextColumn(e.ColName()); cok {
				return cs.Rename(e.Name()), nil
			}
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
		case e.Op() == "reverse":
			return d.evalReverse(*e.Target(), e.Name())
		case strings.HasPrefix(e.Op(), "shift:"):
			return d.evalShift(*e.Target(), strings.TrimPrefix(e.Op(), "shift:"), e.Name())
		case strings.HasPrefix(e.Op(), "rolling_"):
			return d.evalRolling(*e.Target(), e.Op(), e.Name())
		case strings.HasPrefix(e.Op(), "over:"):
			return d.evalOver(*e.Target(), strings.TrimPrefix(e.Op(), "over:"), e.Name())
		}
	}
	if useTypedStorage() {
		if s, ok := d.batchEvalColumn(e, e.Name()); ok {
			return s, nil
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
	col := base.Column()
	n := base.Len()

	// cum_count: running count of non-null rows; works for any dtype via the
	// validity mask and writes a typed []int64 directly.
	if mode == "count" {
		out := make([]int64, n)
		nulls := col.Nulls()
		count := int64(0)
		for i := 0; i < n; i++ {
			if nulls == nil || !nulls[i] {
				count++
			}
			out[i] = count
		}
		return series.FromInt64(name, out, nil), nil
	}

	// cum_sum: typed running sum into a preallocated []float64 (no per-row box).
	// A null contributes nothing and carries the prior cumulative forward; a NaN
	// propagates, matching the prior row-wise behavior.
	if f64s, ok := col.Float64s(); ok {
		out := make([]float64, n)
		nulls := col.Nulls()
		sum := float64(0)
		for i := 0; i < n; i++ {
			if nulls != nil && nulls[i] {
				out[i] = sum
				continue
			}
			sum += f64s[i]
			out[i] = sum
		}
		return series.FromFloat64(name, out, nil), nil
	}
	if i64s, ok := col.Int64s(); ok {
		out := make([]float64, n)
		nulls := col.Nulls()
		sum := float64(0)
		for i := 0; i < n; i++ {
			if nulls != nil && nulls[i] {
				out[i] = sum
				continue
			}
			sum += float64(i64s[i])
			out[i] = sum
		}
		return series.FromFloat64(name, out, nil), nil
	}

	// Fallback for non-numeric dtypes (preserves prior semantics).
	values := make([]any, n)
	sum := float64(0)
	for i := 0; i < n; i++ {
		switch t := base.Value(i).(type) {
		case int64:
			sum += float64(t)
		case float64:
			sum += t
		}
		values[i] = sum
	}
	return series.New(name, dtypes.Float64, values)
}

func (d DataFrame) evalRank(target expr.Expr, name string) (series.Series, error) {
	base, err := d.evalExprAsSeriesVectorized(target)
	if err != nil {
		return series.Series{}, err
	}
	n := base.Len()
	col := base.Column()
	// Radix argsort fast path: an ordinal rank is the inverse of a stable sort
	// permutation, and chunk.ArgsortFloat64/Int64 is a stable O(n) LSD radix, so
	// for a null-free, NaN-free numeric column above the radix threshold it yields
	// the same permutation (hence the same ranks) as the stable comparison sort
	// below — turning rank from O(n log n) into O(n). Anything outside the gate
	// falls through to the comparison path so the order matches the row-wise
	// compareAny path exactly.
	if col != nil && col.NullCount() == 0 && n >= radixSortThreshold {
		if f64s, ok := col.Float64s(); ok && !anyNaN(f64s) {
			return rankSeries(name, chunk.ArgsortFloat64(f64s), n), nil
		}
		if i64s, ok := col.Int64s(); ok {
			return rankSeries(name, chunk.ArgsortInt64(i64s), n), nil
		}
	}
	indexes := make([]int, n)
	for i := range indexes {
		indexes[i] = i
	}
	// Typed comparison fast path: read the backing slice directly (no per-comparison
	// interface boxing) and write a typed []int64 ordinal rank. Restricted to
	// null-free columns so the ordering matches the row-wise compareAny path
	// exactly (compareAny treats <,> normally and everything else as equal).
	if col != nil && col.NullCount() == 0 {
		if f64s, ok := col.Float64s(); ok {
			sort.SliceStable(indexes, func(i, j int) bool { return f64s[indexes[i]] < f64s[indexes[j]] })
			return rankSeries(name, indexes, n), nil
		}
		if i64s, ok := col.Int64s(); ok {
			sort.SliceStable(indexes, func(i, j int) bool { return i64s[indexes[i]] < i64s[indexes[j]] })
			return rankSeries(name, indexes, n), nil
		}
		if strs, ok := col.Strings(); ok {
			sort.SliceStable(indexes, func(i, j int) bool { return strs[indexes[i]] < strs[indexes[j]] })
			return rankSeries(name, indexes, n), nil
		}
	}
	sort.SliceStable(indexes, func(i, j int) bool {
		return compareAny(base.Value(indexes[i]), base.Value(indexes[j])) < 0
	})
	return rankSeries(name, indexes, n), nil
}

// rankSeries assigns 1-based ordinal ranks from a sorted index permutation,
// writing a typed []int64 (no []any round-trip).
func rankSeries(name string, indexes []int, n int) series.Series {
	out := make([]int64, n)
	for rank, idx := range indexes {
		out[idx] = int64(rank + 1)
	}
	return series.FromInt64(name, out, nil)
}

func (d DataFrame) evalReverse(target expr.Expr, name string) (series.Series, error) {
	base, err := d.evalExprAsSeriesVectorized(target)
	if err != nil {
		return series.Series{}, err
	}
	out := make([]any, d.height)
	for i := 0; i < d.height; i++ {
		out[i] = base.Value(d.height - 1 - i)
	}
	return series.New(name, base.DataType(), out)
}

// evalShift is the column-wise shift for Select/WithColumns. The row-wise eval
// path treats "shift:" as identity (a window op can't be done per row), so the
// vectorized evaluator dispatches here to reuse the typed Series shift, which
// fills the vacated head/tail with nulls.
func (d DataFrame) evalShift(target expr.Expr, spec string, name string) (series.Series, error) {
	periods, err := strconv.Atoi(spec)
	if err != nil {
		return series.Series{}, fmt.Errorf("invalid shift periods %q: %w", spec, err)
	}
	base, err := d.evalExprAsSeriesVectorized(target)
	if err != nil {
		return series.Series{}, err
	}
	return base.Shift(periods).Rename(name), nil
}

// rollingFloat64Input extracts a []float64 view and validity mask from a Series
// for the linear rolling kernels. Float64 columns are returned directly; Int64
// columns are converted; other dtypes return ok=false so the caller falls back
// to the row-wise path.
func rollingFloat64Input(s series.Series) (vals []float64, nulls []bool, ok bool) {
	col := s.Column()
	if col == nil {
		return nil, nil, false
	}
	if f64s, ok2 := col.Float64s(); ok2 {
		return f64s, col.Nulls(), true
	}
	if i64s, ok2 := col.Int64s(); ok2 {
		out := make([]float64, len(i64s))
		for i, v := range i64s {
			out[i] = float64(v)
		}
		return out, col.Nulls(), true
	}
	return nil, nil, false
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
	// O(n) typed fast path for sum/mean/min/max: read the typed backing slice,
	// run the linear kernel, and write a single typed output buffer (no per-step
	// slice, no []any boxing). skipNaN is false to match the prior expression
	// path, which included NaN observations in the aggregate.
	switch mode {
	case "rolling_sum", "rolling_mean", "rolling_min", "rolling_max":
		if vals, nulls, ok := rollingFloat64Input(base); ok {
			var outVals []float64
			var outNulls []bool
			switch mode {
			case "rolling_sum":
				outVals, outNulls = chunk.RollingSum(vals, nulls, window, 1, false)
			case "rolling_mean":
				outVals, outNulls = chunk.RollingMean(vals, nulls, window, 1, false)
			case "rolling_min":
				outVals, outNulls = chunk.RollingMin(vals, nulls, window, 1)
			case "rolling_max":
				outVals, outNulls = chunk.RollingMax(vals, nulls, window, 1)
			}
			return series.FromFloat64(name, outVals, outNulls), nil
		}
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
		case "rolling_var":
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
			out[i] = variance / float64(len(nums))
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
	n := base.Len()
	baseCol := base.Column()
	if len(partitions) == 0 {
		// No partition: the window spans the whole frame — return base unchanged
		// (zero-copy, only renamed).
		return series.FromColumn(name, baseCol), nil
	}
	// Typed partition ids via chunk.GroupIDs (no per-row fmt.Sprintf / boxing).
	partCols := make([]*chunk.Column, len(partitions))
	for j, c := range partitions {
		s, ok := d.cols[c]
		if !ok {
			return series.Series{}, fmt.Errorf("partition column %s not found", c)
		}
		partCols[j] = s.Column()
	}
	ids, first := chunk.GroupIDs(partCols, n)
	ngroups := len(first)

	if target.Kind() == expr.KindUnary && target.Op() == "cum_sum" {
		if f64s, ok := baseCol.Float64s(); ok {
			nulls := baseCol.Nulls()
			out := make([]float64, n)
			sums := make([]float64, ngroups)
			for i := 0; i < n; i++ {
				g := ids[i]
				if nulls == nil || !nulls[i] {
					sums[g] += f64s[i]
				}
				out[i] = sums[g]
			}
			return series.FromFloat64(name, out, nil), nil
		}
	}
	if target.Kind() == expr.KindUnary && target.Op() == "cum_count" {
		nulls := baseCol.Nulls()
		out := make([]int64, n)
		counts := make([]int64, ngroups)
		for i := 0; i < n; i++ {
			g := ids[i]
			if nulls == nil || !nulls[i] {
				counts[g]++
			}
			out[i] = counts[g]
		}
		return series.FromInt64(name, out, nil), nil
	}
	if target.Kind() == expr.KindUnary && target.Op() == "rank" {
		buckets := make([][]int, ngroups)
		for i, g := range ids {
			buckets[g] = append(buckets[g], i)
		}
		out := make([]int64, n)
		i64s, isInt := baseCol.Int64s()
		f64s, isFloat := baseCol.Float64s()
		// A partition's ordinal rank is the inverse of a stable sort over its
		// values. For null-free numeric partitions above the radix threshold the
		// O(n) stable radix (chunk.ArgsortInt64/Float64) over a gathered value
		// slice produces the same tie order as the comparison sort below; smaller
		// partitions and other dtypes keep the stable comparison path. buckets are
		// built in row order, so a gathered slice is in encounter order and the
		// radix stays stable across ties exactly like sort.SliceStable.
		floatRadixOK := isFloat && !anyNaN(f64s)
		for _, idxs := range buckets {
			switch {
			case isInt && len(idxs) >= radixSortThreshold:
				vals := make([]int64, len(idxs))
				for k, gi := range idxs {
					vals[k] = i64s[gi]
				}
				for r, p := range chunk.ArgsortInt64(vals) {
					out[idxs[p]] = int64(r + 1)
				}
			case floatRadixOK && len(idxs) >= radixSortThreshold:
				vals := make([]float64, len(idxs))
				for k, gi := range idxs {
					vals[k] = f64s[gi]
				}
				for r, p := range chunk.ArgsortFloat64(vals) {
					out[idxs[p]] = int64(r + 1)
				}
			default:
				if isInt {
					sort.SliceStable(idxs, func(a, b int) bool { return i64s[idxs[a]] < i64s[idxs[b]] })
				} else {
					sort.SliceStable(idxs, func(a, b int) bool {
						return compareAny(base.Value(idxs[a]), base.Value(idxs[b])) < 0
					})
				}
				for rank, idx := range idxs {
					out[idx] = int64(rank + 1)
				}
			}
		}
		return series.FromInt64(name, out, nil), nil
	}
	return series.FromColumn(name, baseCol), nil
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
		// with_context column: resolve at the current row if it is long enough,
		// otherwise treat as null (mismatched lengths are caught when materialized).
		if cs, cok := r.df.contextColumn(name); cok {
			if r.row < cs.Len() {
				return cs.Value(r.row), true
			}
			return nil, true
		}
		return nil, false
	}
	return s.Value(r.row), true
}

func (r rowAccessor) RowIndex() int { return r.row }

func (r rowAccessor) NumRows() int { return r.df.height }

func (r rowAccessor) ValueAt(row int, column string) (any, bool) {
	s, ok := r.df.cols[column]
	if !ok || row < 0 || row >= r.df.height {
		return nil, false
	}
	return s.Value(row), true
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
