package polars

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
	iseries "github.com/h0rn3t/gopolars/pkg/series"
	"github.com/h0rn3t/gopolars/pkg/simd"
)

type seriesFacade struct {
	value iseries.Series
}

type NewSeriesInput struct {
	Name   string
	DType  dtypes.DataType
	Values []any
}

func NewSeries(input NewSeriesInput) (Series, error) {
	s, err := iseries.New(input.Name, input.DType, input.Values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: s}, nil
}

func fromInternalSeries(s iseries.Series) Series {
	return seriesFacade{value: s}
}

func toInternalSeries(s Series) (iseries.Series, error) {
	v, ok := s.(seriesFacade)
	if !ok {
		return iseries.Series{}, fmt.Errorf("unsupported series implementation")
	}
	return v.value, nil
}

func (s seriesFacade) Name() string {
	return s.value.Name()
}

func (s seriesFacade) DataType() dtypes.DataType {
	return s.value.DataType()
}

func (s seriesFacade) Len() int {
	return s.value.Len()
}

func (s seriesFacade) Value(i int) any {
	return s.value.Value(i)
}

func (s seriesFacade) ToList() []any {
	values := make([]any, s.value.Len())
	for i := 0; i < s.value.Len(); i++ {
		values[i] = s.value.Value(i)
	}
	return values
}

func (s seriesFacade) NullCount() int {
	// O(1) after the first call via the chunk's cached null count.
	if c := s.value.Column(); c != nil {
		return c.NullCount()
	}
	return 0
}

func (s seriesFacade) IsNull() Series {
	n := s.value.Len()
	out := make([]bool, n)
	// Only a column with real nulls needs the validity copy; a null-free column
	// (NullCount cached at 0) leaves the all-false zero value untouched.
	if c := s.value.Column(); c != nil && c.NullCount() != 0 {
		copy(out, c.Nulls())
	}
	// The result mask itself is never null.
	return seriesFacade{value: iseries.FromBool(s.value.Name(), out, nil)}
}

func (s seriesFacade) IsNotNull() Series {
	n := s.value.Len()
	out := make([]bool, n)
	if c := s.value.Column(); c != nil && c.NullCount() != 0 {
		// Mixed validity: invert the validity mask element-wise.
		nulls := c.Nulls()
		for i := range n {
			out[i] = !nulls[i]
		}
		return seriesFacade{value: iseries.FromBool(s.value.Name(), out, nil)}
	}
	// Null-free (or no backing column): every row is non-null. Fill all-true at
	// memmove throughput via copy doubling so is_not_null matches is_null's fast
	// path instead of running ~7x slower on a per-element store loop.
	fillTrue(out)
	return seriesFacade{value: iseries.FromBool(s.value.Name(), out, nil)}
}

// fillTrue sets every element of b to true using exponential copy doubling: it
// seeds one true element, then doubles the filled prefix with copy (memmove)
// until the whole slice is set. This runs at memmove throughput rather than a
// per-byte store loop.
func fillTrue(b []bool) {
	if len(b) == 0 {
		return
	}
	b[0] = true
	for j := 1; j < len(b); j <<= 1 {
		copy(b[j:], b[:j])
	}
}

func (s seriesFacade) FillNull(value any) (Series, error) {
	// Typed fast path: float64 column filled with a float64 literal.
	if fv, ok := value.(float64); ok {
		if col, ok2 := s.value.Column().FillNullFloat64(fv); ok2 {
			return seriesFacade{value: iseries.FromColumn(s.value.Name(), col)}, nil
		}
	}
	values := make([]any, s.value.Len())
	for i := 0; i < s.value.Len(); i++ {
		if s.value.IsNull(i) {
			values[i] = value
		} else {
			values[i] = s.value.Value(i)
		}
	}
	out, err := iseries.New(s.value.Name(), s.value.DataType(), values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) FillNan(value float64) (Series, error) {
	// Typed fast path for float64 columns (preserves nulls, fills NaN values).
	if col, ok := s.value.Column().FillNaNFloat64(value); ok {
		return seriesFacade{value: iseries.FromColumn(s.value.Name(), col)}, nil
	}
	values := make([]any, s.value.Len())
	for i := 0; i < s.value.Len(); i++ {
		v := s.value.Value(i)
		if f, ok := v.(float64); ok && math.IsNaN(f) {
			values[i] = value
			continue
		}
		values[i] = v
	}
	out, err := iseries.New(s.value.Name(), s.value.DataType(), values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) DropNans() Series {
	// Typed fast path: filter out non-null NaN rows directly from the chunk.
	if col, ok := s.value.Column().DropNaNFloat64(); ok {
		return seriesFacade{value: iseries.FromColumn(s.value.Name(), col)}
	}
	values := make([]any, 0, s.value.Len())
	for i := 0; i < s.value.Len(); i++ {
		v := s.value.Value(i)
		if f, ok := v.(float64); ok && math.IsNaN(f) {
			continue
		}
		values = append(values, v)
	}
	out, _ := iseries.New(s.value.Name(), s.value.DataType(), values)
	return seriesFacade{value: out}
}

func (s seriesFacade) DropNulls() Series {
	values := make([]any, 0, s.value.Len())
	for i := 0; i < s.value.Len(); i++ {
		if s.value.IsNull(i) {
			continue
		}
		values = append(values, s.value.Value(i))
	}
	out, _ := iseries.New(s.value.Name(), s.value.DataType(), values)
	return seriesFacade{value: out}
}

func (s seriesFacade) RollingMean(window int) Series {
	return s.rolling(window, "mean")
}

func (s seriesFacade) RollingSum(window int) Series {
	return s.rolling(window, "sum")
}

func (s seriesFacade) RollingMin(window int) Series {
	return s.rolling(window, "min")
}

func (s seriesFacade) RollingMax(window int) Series {
	return s.rolling(window, "max")
}

func (s seriesFacade) RollingStd(window int) Series {
	return s.rolling(window, "std")
}

func (s seriesFacade) RollingVar(window int) Series {
	return s.rolling(window, "var")
}

func (s seriesFacade) RollingMedian(window int) Series {
	return s.rolling(window, "median")
}

func (s seriesFacade) RollingQuantile(window int, q float64) Series {
	return s.rollingQuantile(window, q)
}

func (s seriesFacade) Abs() Series {
	return s.unaryNumeric("abs")
}

func (s seriesFacade) Exp() Series {
	return s.unaryNumeric("exp")
}

func (s seriesFacade) Log() Series {
	return s.unaryNumeric("log")
}

func (s seriesFacade) Sqrt() Series {
	return s.unaryNumeric("sqrt")
}

func (s seriesFacade) Sin() Series {
	return s.unaryNumeric("sin")
}

func (s seriesFacade) Cos() Series {
	return s.unaryNumeric("cos")
}

func (s seriesFacade) Tan() Series {
	return s.unaryNumeric("tan")
}

func (s seriesFacade) Sinh() Series {
	return s.unaryNumeric("sinh")
}

func (s seriesFacade) Cosh() Series {
	return s.unaryNumeric("cosh")
}

func (s seriesFacade) Tanh() Series {
	return s.unaryNumeric("tanh")
}

func (s seriesFacade) Arcsin() Series {
	return s.unaryNumeric("arcsin")
}

func (s seriesFacade) Arccos() Series {
	return s.unaryNumeric("arccos")
}

func (s seriesFacade) Arctan() Series {
	return s.unaryNumeric("arctan")
}

func (s seriesFacade) Arcsinh() Series {
	return s.unaryNumeric("arcsinh")
}

func (s seriesFacade) Arccosh() Series {
	return s.unaryNumeric("arccosh")
}

func (s seriesFacade) Arctanh() Series {
	return s.unaryNumeric("arctanh")
}

func (s seriesFacade) Cbrt() Series {
	return s.unaryNumeric("cbrt")
}

func (s seriesFacade) Ceil() Series {
	return s.unaryNumeric("ceil")
}

func (s seriesFacade) Floor() Series {
	return s.unaryNumeric("floor")
}

func (s seriesFacade) Degrees() Series {
	return s.unaryNumeric("degrees")
}

func (s seriesFacade) Sign() Series {
	return s.unaryNumeric("sign")
}

func (s seriesFacade) Log10() Series {
	return s.unaryNumeric("log10")
}

func (s seriesFacade) Log1p() Series {
	return s.unaryNumeric("log1p")
}

func (s seriesFacade) Round() Series {
	return s.unaryNumeric("round")
}

func (s seriesFacade) Pow(power float64) Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		f, ok := toFloat64(v)
		if !ok {
			values[i] = nil
			continue
		}
		values[i] = math.Pow(f, power)
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Float64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Shift(periods int) Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		src := i - periods
		if src >= 0 && src < s.Len() {
			values[i] = s.Value(src)
		}
	}
	out, _ := iseries.New(s.value.Name(), s.value.DataType(), values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Reverse() Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		values[i] = s.Value(s.Len() - 1 - i)
	}
	out, _ := iseries.New(s.value.Name(), s.value.DataType(), values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Sum() float64 {
	vals := s.numericValues(true)
	return simd.SumFloat64(vals)
}

func (s seriesFacade) Std() float64 {
	vals := s.numericValues(true)
	if len(vals) < 2 {
		return 0
	}
	mean := meanFloatSlice(vals)
	sumSq := 0.0
	for _, v := range vals {
		d := v - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(vals)-1))
}

func (s seriesFacade) Max() float64 {
	// Ignore NaN, but a series whose only non-null values are NaN reduces to NaN
	// (matching Polars). Empty input keeps the historical 0 default.
	all := s.numericValues(false)
	vals := s.numericValues(true)
	if len(vals) == 0 {
		if len(all) > 0 {
			return math.NaN()
		}
		return simd.MaxFloat64(vals)
	}
	return simd.MaxFloat64(vals)
}

func (s seriesFacade) Min() float64 {
	all := s.numericValues(false)
	vals := s.numericValues(true)
	if len(vals) == 0 {
		if len(all) > 0 {
			return math.NaN()
		}
		return simd.MinFloat64(vals)
	}
	return simd.MinFloat64(vals)
}

func (s seriesFacade) Mean() float64 {
	vals := s.numericValues(true)
	if len(vals) == 0 {
		return 0
	}
	return meanFloatSlice(vals)
}

func (s seriesFacade) Median() float64 {
	vals := s.numericValues(true)
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	mid := len(vals) / 2
	if len(vals)%2 == 0 {
		return (vals[mid-1] + vals[mid]) / 2
	}
	return vals[mid]
}

func (s seriesFacade) Var() float64 {
	vals := s.numericValues(true)
	if len(vals) < 2 {
		return 0
	}
	mean := meanFloatSlice(vals)
	sumSq := 0.0
	for _, v := range vals {
		d := v - mean
		sumSq += d * d
	}
	return sumSq / float64(len(vals)-1)
}

func (s seriesFacade) NUnique() int {
	seen := map[string]struct{}{}
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if v == nil {
			continue
		}
		seen[valueKey(v)] = struct{}{}
	}
	return len(seen)
}

func (s seriesFacade) Mode() any {
	counts := map[string]int{}
	values := map[string]any{}
	order := make([]string, 0, s.Len())
	bestKey := ""
	bestCount := 0
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if v == nil {
			continue
		}
		key := valueKey(v)
		if _, ok := values[key]; !ok {
			values[key] = v
			order = append(order, key)
		}
		counts[key]++
		if counts[key] > bestCount {
			bestCount = counts[key]
			bestKey = key
		}
	}
	if bestKey == "" && len(order) > 0 {
		bestKey = order[0]
	}
	return values[bestKey]
}

func (s seriesFacade) Kurtosis() float64 {
	vals := s.numericValues(true)
	if len(vals) < 2 {
		return 0
	}
	mean := meanFloatSlice(vals)
	m2 := 0.0
	m4 := 0.0
	for _, v := range vals {
		d := v - mean
		d2 := d * d
		m2 += d2
		m4 += d2 * d2
	}
	m2 /= float64(len(vals))
	if m2 == 0 {
		return 0
	}
	m4 /= float64(len(vals))
	return m4/(m2*m2) - 3
}

func (s seriesFacade) Skew() float64 {
	vals := s.numericValues(true)
	if len(vals) < 2 {
		return 0
	}
	mean := meanFloatSlice(vals)
	m2 := 0.0
	m3 := 0.0
	for _, v := range vals {
		d := v - mean
		d2 := d * d
		m2 += d2
		m3 += d2 * d
	}
	m2 /= float64(len(vals))
	if m2 == 0 {
		return 0
	}
	m3 /= float64(len(vals))
	return m3 / math.Pow(m2, 1.5)
}

func (s seriesFacade) Quantile(q float64) float64 {
	vals := s.numericValues(true)
	if len(vals) == 0 {
		return 0
	}
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}
	sort.Float64s(vals)
	if len(vals) == 1 {
		return vals[0]
	}
	idx := float64(len(vals)-1) * q
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper {
		return vals[lower]
	}
	weight := idx - float64(lower)
	return vals[lower]*(1-weight) + vals[upper]*weight
}

func (s seriesFacade) Product() float64 {
	vals := s.numericValues(true)
	if len(vals) == 0 {
		return 0
	}
	product := 1.0
	for _, v := range vals {
		product *= v
	}
	return product
}

// NanMax propagates NaN: any NaN among the non-null values yields NaN.
func (s seriesFacade) NanMax() float64 {
	vals := s.numericValues(false)
	for _, v := range vals {
		if math.IsNaN(v) {
			return math.NaN()
		}
	}
	return simd.MaxFloat64(vals)
}

// NanMin propagates NaN: any NaN among the non-null values yields NaN.
func (s seriesFacade) NanMin() float64 {
	vals := s.numericValues(false)
	for _, v := range vals {
		if math.IsNaN(v) {
			return math.NaN()
		}
	}
	return simd.MinFloat64(vals)
}

func (s seriesFacade) Alias(name string) Series {
	return seriesFacade{value: s.value.Rename(name)}
}

func (s seriesFacade) Clone() Series {
	return seriesFacade{value: s.value.Clone()}
}

func (s seriesFacade) Clear() Series {
	out, _ := iseries.New(s.value.Name(), s.value.DataType(), []any{})
	return seriesFacade{value: out}
}

func (s seriesFacade) Head(n int) Series {
	return s.Slice(0, n)
}

func (s seriesFacade) Tail(n int) Series {
	if n >= s.Len() {
		return s.Clone()
	}
	if n <= 0 {
		return s.Clear()
	}
	return s.Slice(s.Len()-n, n)
}

func (s seriesFacade) Limit(n int) Series {
	return s.Head(n)
}

func (s seriesFacade) Slice(offset int, length int) Series {
	if length <= 0 || s.Len() == 0 {
		return s.Clear()
	}
	// A negative offset counts from the end; the window [offset, offset+length)
	// is clamped to [0, len) so it keeps only its overlap (matching Polars).
	if offset < 0 {
		offset = s.Len() + offset
	}
	end := offset + length
	if offset < 0 {
		offset = 0
	}
	if end > s.Len() {
		end = s.Len()
	}
	if offset >= s.Len() || end <= offset {
		return s.Clear()
	}
	values := make([]any, 0, end-offset)
	for i := offset; i < end; i++ {
		values = append(values, s.Value(i))
	}
	out, _ := iseries.New(s.value.Name(), s.value.DataType(), values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Sort(descending bool) Series {
	indexes := make([]int, s.Len())
	for i := range indexes {
		indexes[i] = i
	}
	sort.SliceStable(indexes, func(i int, j int) bool {
		left := s.Value(indexes[i])
		right := s.Value(indexes[j])
		// Nulls sort first regardless of direction (Polars default nulls_last=False).
		lnull := left == nil
		rnull := right == nil
		if lnull || rnull {
			if lnull && rnull {
				return false
			}
			return lnull
		}
		cmp := compareForSeriesOrder(left, right)
		if descending {
			return cmp > 0
		}
		return cmp < 0
	})
	return s.Gather(indexes)
}

func (s seriesFacade) Unique() Series {
	values := make([]any, 0, s.Len())
	seen := map[string]struct{}{}
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		key := valueKey(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		values = append(values, v)
	}
	out, _ := iseries.New(s.value.Name(), s.value.DataType(), values)
	return seriesFacade{value: out}
}

func (s seriesFacade) ArgSort() Series {
	indexes := make([]int, s.Len())
	for i := range indexes {
		indexes[i] = i
	}
	sort.SliceStable(indexes, func(i int, j int) bool {
		return compareForSeriesOrder(s.Value(indexes[i]), s.Value(indexes[j])) < 0
	})
	return newIndexSeries(s.value.Name()+"_arg_sort", indexes)
}

func (s seriesFacade) ArgMax() int {
	if s.Len() == 0 {
		return -1
	}
	bestIdx := -1
	var best any
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if v == nil {
			continue
		}
		if bestIdx == -1 || compareForSeriesOrder(v, best) > 0 {
			bestIdx = i
			best = v
		}
	}
	return bestIdx
}

func (s seriesFacade) ArgMin() int {
	if s.Len() == 0 {
		return -1
	}
	bestIdx := -1
	var best any
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if v == nil {
			continue
		}
		if bestIdx == -1 || compareForSeriesOrder(v, best) < 0 {
			bestIdx = i
			best = v
		}
	}
	return bestIdx
}

func (s seriesFacade) ArgUnique() Series {
	seen := map[string]struct{}{}
	indexes := make([]int, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		key := valueKey(s.Value(i))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		indexes = append(indexes, i)
	}
	return newIndexSeries(s.value.Name()+"_arg_unique", indexes)
}

func (s seriesFacade) ArgTrue() Series {
	indexes := make([]int, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if flag, ok := s.Value(i).(bool); ok && flag {
			indexes = append(indexes, i)
		}
	}
	return newIndexSeries(s.value.Name()+"_arg_true", indexes)
}

func (s seriesFacade) Gather(indices []int) Series {
	values := make([]any, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= s.Len() {
			continue
		}
		values = append(values, s.Value(idx))
	}
	out, _ := iseries.New(s.value.Name(), s.value.DataType(), values)
	return seriesFacade{value: out}
}

func (s seriesFacade) GatherEvery(step int) Series {
	if step <= 0 {
		return s.Clone()
	}
	indexes := make([]int, 0, (s.Len()+step-1)/step)
	for i := 0; i < s.Len(); i += step {
		indexes = append(indexes, i)
	}
	return s.Gather(indexes)
}

func (s seriesFacade) Sample(n int, seed int64) Series {
	if n <= 0 || s.Len() == 0 {
		return s.Clear()
	}
	if n > s.Len() {
		n = s.Len()
	}
	rng := rand.New(rand.NewSource(seed))
	indexes := rng.Perm(s.Len())[:n]
	return s.Gather(indexes)
}

func (s seriesFacade) Shuffle(seed int64) Series {
	if s.Len() == 0 {
		return s.Clear()
	}
	rng := rand.New(rand.NewSource(seed))
	indexes := rng.Perm(s.Len())
	return s.Gather(indexes)
}

func (s seriesFacade) Rechunk() Series {
	return s.Clone()
}

func (s seriesFacade) ShrinkToFit() Series {
	return s.Clone()
}

func (s seriesFacade) Shape() [1]int {
	return [1]int{s.Len()}
}

func (s seriesFacade) IsEmpty() bool {
	return s.Len() == 0
}

func (s seriesFacade) IsSorted() bool {
	for i := 1; i < s.Len(); i++ {
		if compareForSeriesOrder(s.Value(i-1), s.Value(i)) > 0 {
			return false
		}
	}
	return true
}

func (s seriesFacade) HasNulls() bool {
	return s.NullCount() > 0
}

func (s seriesFacade) HasValidity() bool {
	return s.HasNulls()
}

func (s seriesFacade) NChunks() int {
	return 1
}

func (s seriesFacade) ChunkLengths() []int {
	return []int{s.Len()}
}

func (s seriesFacade) All() bool {
	for i := 0; i < s.Len(); i++ {
		flag, ok := s.Value(i).(bool)
		if !ok || !flag {
			return false
		}
	}
	return s.Len() > 0
}

func (s seriesFacade) Any() bool {
	for i := 0; i < s.Len(); i++ {
		if flag, ok := s.Value(i).(bool); ok && flag {
			return true
		}
	}
	return false
}

func (s seriesFacade) Not_() Series {
	values := make([]any, s.Len())
	// On Int64, not_ is the bitwise complement (^v == -v-1), matching Polars; on
	// Boolean it is the logical negation. Nulls are preserved.
	if s.DataType() == dtypes.Int64 {
		for i := 0; i < s.Len(); i++ {
			if s.value.IsNull(i) {
				continue
			}
			if v, ok := s.Value(i).(int64); ok {
				values[i] = ^v
			}
		}
		out, _ := iseries.New(s.value.Name(), dtypes.Int64, values)
		return seriesFacade{value: out}
	}
	for i := 0; i < s.Len(); i++ {
		if flag, ok := s.Value(i).(bool); ok {
			values[i] = !flag
		}
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Boolean, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) IsNan() Series {
	return s.booleanMap(func(v any) bool {
		f, ok := toFloat64(v)
		return ok && math.IsNaN(f)
	})
}

func (s seriesFacade) IsNotNan() Series {
	return s.booleanMap(func(v any) bool {
		f, ok := toFloat64(v)
		return !ok || !math.IsNaN(f)
	})
}

func (s seriesFacade) IsFinite() Series {
	return s.booleanMap(func(v any) bool {
		f, ok := toFloat64(v)
		return ok && !math.IsNaN(f) && !math.IsInf(f, 0)
	})
}

func (s seriesFacade) IsInfinite() Series {
	return s.booleanMap(func(v any) bool {
		f, ok := toFloat64(v)
		return ok && math.IsInf(f, 0)
	})
}

func (s seriesFacade) IsDuplicated() Series {
	counts := s.valueCounts()
	return s.booleanMap(func(v any) bool {
		return counts[valueKey(v)] > 1
	})
}

func (s seriesFacade) IsUnique() Series {
	counts := s.valueCounts()
	return s.booleanMap(func(v any) bool {
		return counts[valueKey(v)] == 1
	})
}

func (s seriesFacade) IsFirstDistinct() Series {
	seen := map[string]struct{}{}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		key := valueKey(s.Value(i))
		_, ok := seen[key]
		values[i] = !ok
		seen[key] = struct{}{}
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Boolean, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) IsLastDistinct() Series {
	seen := map[string]struct{}{}
	values := make([]any, s.Len())
	for i := s.Len() - 1; i >= 0; i-- {
		key := valueKey(s.Value(i))
		_, ok := seen[key]
		values[i] = !ok
		seen[key] = struct{}{}
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Boolean, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) IsBetween(lower float64, upper float64) Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		f, ok := toFloat64(s.Value(i))
		values[i] = ok && f >= lower && f <= upper
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Boolean, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) IsClose(other Series) (Series, error) {
	if s.Len() != other.Len() {
		return nil, fmt.Errorf("series length mismatch")
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		left, lok := toFloat64(s.Value(i))
		right, rok := toFloat64(other.Value(i))
		values[i] = lok && rok && math.Abs(left-right) <= 1e-9
	}
	out, err := iseries.New(s.value.Name(), dtypes.Boolean, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) IsIn(values []any) Series {
	allowed := map[string]struct{}{}
	for _, v := range values {
		allowed[valueKey(v)] = struct{}{}
	}
	return s.booleanMap(func(v any) bool {
		_, ok := allowed[valueKey(v)]
		return ok
	})
}

func (s seriesFacade) EqMissing(other Series) (Series, error) {
	return s.binaryMissing(other, true)
}

func (s seriesFacade) NeMissing(other Series) (Series, error) {
	return s.binaryMissing(other, false)
}

func (s seriesFacade) CumSum() Series {
	return s.cumulative("sum")
}

func (s seriesFacade) CumMax() Series {
	return s.cumulative("max")
}

func (s seriesFacade) CumMin() Series {
	return s.cumulative("min")
}

func (s seriesFacade) CumProd() Series {
	return s.cumulative("prod")
}

func (s seriesFacade) CumCount() Series {
	return s.cumulative("count")
}

func (s seriesFacade) EwmMean(alpha float64) Series {
	return s.ewm(alpha, "mean")
}

func (s seriesFacade) EwmStd(alpha float64) Series {
	return s.ewm(alpha, "std")
}

func (s seriesFacade) EwmVar(alpha float64) Series {
	return s.ewm(alpha, "var")
}

func (s seriesFacade) Diff(n int) Series {
	values := make([]any, s.Len())
	if n <= 0 {
		n = 1
	}
	for i := 0; i < s.Len(); i++ {
		if i < n {
			values[i] = nil
			continue
		}
		current, cok := toFloat64(s.Value(i))
		prev, pok := toFloat64(s.Value(i - n))
		if !cok || !pok {
			values[i] = nil
			continue
		}
		values[i] = current - prev
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Float64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) PctChange(n int) Series {
	values := make([]any, s.Len())
	if n <= 0 {
		n = 1
	}
	for i := 0; i < s.Len(); i++ {
		if i < n {
			values[i] = nil
			continue
		}
		current, cok := toFloat64(s.Value(i))
		prev, pok := toFloat64(s.Value(i - n))
		if !cok || !pok || prev == 0 {
			values[i] = nil
			continue
		}
		values[i] = (current - prev) / prev
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Float64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Dot(other Series) (float64, error) {
	if s.Len() != other.Len() {
		return 0, fmt.Errorf("series length mismatch")
	}
	acc := 0.0
	for i := 0; i < s.Len(); i++ {
		left, lok := toFloat64(s.Value(i))
		right, rok := toFloat64(other.Value(i))
		if !lok || !rok {
			return 0, fmt.Errorf("dot requires numeric series")
		}
		acc += left * right
	}
	return acc, nil
}

func (s seriesFacade) Entropy() float64 {
	counts := s.valueCounts()
	if len(counts) == 0 {
		return 0
	}
	total := float64(s.Len())
	entropy := 0.0
	for _, count := range counts {
		p := float64(count) / total
		entropy -= p * math.Log(p)
	}
	return entropy
}

func (s seriesFacade) ValueCounts() (DataFrame, error) {
	values := make([]any, 0, s.Len())
	counts := make([]any, 0, s.Len())
	seen := map[string]struct{}{}
	valueMap := map[string]any{}
	freq := s.valueCounts()
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		key := valueKey(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		valueMap[key] = v
		values = append(values, v)
		counts = append(counts, int64(freq[key]))
	}
	return NewDataFrame(NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "value", Values: values},
			{Name: "count", Values: counts},
		},
	})
}

func (s seriesFacade) UniqueCounts() Series {
	seen := map[string]struct{}{}
	freq := s.valueCounts()
	values := make([]any, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		key := valueKey(s.Value(i))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		values = append(values, int64(freq[key]))
	}
	out, _ := iseries.New(s.value.Name()+"_unique_counts", dtypes.Int64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) TopK(k int) Series {
	if k <= 0 {
		return s.Clear()
	}
	return s.Sort(true).Head(k)
}

func (s seriesFacade) TopKBy(by Series, k int) (Series, error) {
	return s.kBy(by, k, true)
}

func (s seriesFacade) BottomK(k int) Series {
	if k <= 0 {
		return s.Clear()
	}
	return s.Sort(false).Head(k)
}

func (s seriesFacade) BottomKBy(by Series, k int) (Series, error) {
	return s.kBy(by, k, false)
}

func (s seriesFacade) PeakMax() Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		if i == 0 || i == s.Len()-1 {
			values[i] = false
			continue
		}
		curr, cok := toFloat64(s.Value(i))
		prev, pok := toFloat64(s.Value(i - 1))
		next, nok := toFloat64(s.Value(i + 1))
		values[i] = cok && pok && nok && curr > prev && curr > next
	}
	out, _ := iseries.New(s.value.Name()+"_peak_max", dtypes.Boolean, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) PeakMin() Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		if i == 0 || i == s.Len()-1 {
			values[i] = false
			continue
		}
		curr, cok := toFloat64(s.Value(i))
		prev, pok := toFloat64(s.Value(i - 1))
		next, nok := toFloat64(s.Value(i + 1))
		values[i] = cok && pok && nok && curr < prev && curr < next
	}
	out, _ := iseries.New(s.value.Name()+"_peak_min", dtypes.Boolean, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) InterpolateBy(by Series) (Series, error) {
	if s.Len() != by.Len() {
		return nil, fmt.Errorf("series length mismatch")
	}
	return s.Interpolate(), nil
}

func (s seriesFacade) ForwardFill() Series {
	values := s.ToList()
	var last any
	for i := 0; i < len(values); i++ {
		if values[i] != nil {
			last = values[i]
			continue
		}
		values[i] = last
	}
	out, _ := iseries.New(s.value.Name(), s.value.DataType(), values)
	return seriesFacade{value: out}
}

func (s seriesFacade) BackwardFill() Series {
	values := s.ToList()
	var next any
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] != nil {
			next = values[i]
			continue
		}
		values[i] = next
	}
	out, _ := iseries.New(s.value.Name(), s.value.DataType(), values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Cut(breaks []float64) (Series, error) {
	if len(breaks) == 0 {
		return nil, fmt.Errorf("cut requires at least one break")
	}
	sortedBreaks := append([]float64(nil), breaks...)
	sort.Float64s(sortedBreaks)
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		f, ok := toFloat64(s.Value(i))
		if !ok {
			values[i] = nil
			continue
		}
		label := fmt.Sprintf("> %v", sortedBreaks[len(sortedBreaks)-1])
		for _, b := range sortedBreaks {
			if f <= b {
				label = fmt.Sprintf("<= %v", b)
				break
			}
		}
		values[i] = label
	}
	out, err := iseries.New(s.value.Name()+"_cut", dtypes.String, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) QCut(q int) (Series, error) {
	if q <= 1 {
		return nil, fmt.Errorf("qcut requires q > 1")
	}
	breaks := make([]float64, 0, q-1)
	seen := map[string]struct{}{}
	for i := 1; i < q; i++ {
		b := s.Quantile(float64(i) / float64(q))
		key := fmt.Sprintf("%.12f", b)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		breaks = append(breaks, b)
	}
	if len(breaks) == 0 {
		breaks = append(breaks, s.Max())
	}
	return s.Cut(breaks)
}

func (s seriesFacade) Rle() (DataFrame, error) {
	if s.Len() == 0 {
		return NewDataFrame(NewDataFrameInput{
			Columns: []frame.SeriesInput{
				{Name: "value", Values: []any{}},
				{Name: "count", Values: []any{}},
			},
		})
	}
	values := make([]any, 0, s.Len())
	counts := make([]any, 0, s.Len())
	current := s.Value(0)
	run := int64(1)
	for i := 1; i < s.Len(); i++ {
		v := s.Value(i)
		if valueEquals(v, current) {
			run++
			continue
		}
		values = append(values, current)
		counts = append(counts, run)
		current = v
		run = 1
	}
	values = append(values, current)
	counts = append(counts, run)
	return NewDataFrame(NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "value", Values: values},
			{Name: "count", Values: counts},
		},
	})
}

func (s seriesFacade) RleId() Series {
	values := make([]any, s.Len())
	if s.Len() == 0 {
		out, _ := iseries.New(s.value.Name()+"_rle_id", dtypes.Int64, values)
		return seriesFacade{value: out}
	}
	runID := int64(0)
	values[0] = runID
	prev := s.Value(0)
	for i := 1; i < s.Len(); i++ {
		current := s.Value(i)
		if !valueEquals(current, prev) {
			runID++
		}
		values[i] = runID
		prev = current
	}
	out, _ := iseries.New(s.value.Name()+"_rle_id", dtypes.Int64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Rank() Series {
	indexes := make([]int, s.Len())
	for i := range indexes {
		indexes[i] = i
	}
	sort.SliceStable(indexes, func(i int, j int) bool {
		return compareForSeriesOrder(s.Value(indexes[i]), s.Value(indexes[j])) < 0
	})
	values := make([]any, s.Len())
	for rank, idx := range indexes {
		values[idx] = int64(rank + 1)
	}
	out, _ := iseries.New(s.value.Name()+"_rank", dtypes.Int64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) SearchSorted(value any) int {
	return s.LowerBound(value)
}

func (s seriesFacade) LowerBound(value any) int {
	return sort.Search(s.Len(), func(i int) bool {
		return compareForSeriesOrder(s.Value(i), value) >= 0
	})
}

func (s seriesFacade) UpperBound(value any) int {
	return sort.Search(s.Len(), func(i int) bool {
		return compareForSeriesOrder(s.Value(i), value) > 0
	})
}

func (s seriesFacade) Item(index int) (any, error) {
	if index < 0 || index >= s.Len() {
		return nil, fmt.Errorf("index out of bounds")
	}
	return s.Value(index), nil
}

func (s seriesFacade) First() any {
	if s.Len() == 0 {
		return nil
	}
	return s.Value(0)
}

func (s seriesFacade) Last() any {
	if s.Len() == 0 {
		return nil
	}
	return s.Value(s.Len() - 1)
}

func (s seriesFacade) IndexOf(value any) int {
	for i := 0; i < s.Len(); i++ {
		if valueEquals(s.Value(i), value) {
			return i
		}
	}
	return -1
}

func (s seriesFacade) MapElements(fn func(any) any) (Series, error) {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.Value(i) == nil {
			values[i] = nil
			continue
		}
		values[i] = fn(s.Value(i))
	}
	dt := inferDataTypeFromValues(values, s.value.DataType())
	out, err := iseries.New(s.value.Name(), dt, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) Replace(old any, new any) Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if valueEquals(v, old) {
			values[i] = new
			continue
		}
		values[i] = v
	}
	dt := inferDataTypeFromValues(values, s.value.DataType())
	out, _ := iseries.New(s.value.Name(), dt, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) ReplaceStrict(old any, new any) (Series, error) {
	if s.IndexOf(old) < 0 {
		return nil, fmt.Errorf("value not found")
	}
	return s.Replace(old, new), nil
}

func (s seriesFacade) Reshape(dims ...int) (Series, error) {
	if len(dims) == 0 {
		return s.Clone(), nil
	}
	product := 1
	for _, dim := range dims {
		if dim <= 0 {
			return nil, fmt.Errorf("reshape dimensions must be positive")
		}
		product *= dim
	}
	if product != s.Len() {
		return nil, fmt.Errorf("reshape size mismatch")
	}
	return s.Clone(), nil
}

// RepeatBy returns a List Series where element i is a list containing its value
// repeated n times (matching Polars Expr.repeat_by). n <= 0 yields empty lists.
func (s seriesFacade) RepeatBy(n int) Series {
	if n < 0 {
		n = 0
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		inner := make([]any, n)
		for j := 0; j < n; j++ {
			inner[j] = v
		}
		values[i] = inner
	}
	out, _ := iseries.New(s.value.Name(), dtypes.List, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) SetSorted(descending bool) Series {
	_ = descending
	return s.Clone()
}

func (s seriesFacade) ToFrame() (DataFrame, error) {
	// Build directly from the backing series so the dtype is preserved (e.g. an
	// all-null typed column survives instead of failing dtype inference).
	f, err := frame.New(frame.NewInput{Series: []iseries.Series{s.value}})
	if err != nil {
		return nil, err
	}
	return &df{value: f}, nil
}

func (s seriesFacade) ToDummies() (DataFrame, error) {
	df, err := s.ToFrame()
	if err != nil {
		return nil, err
	}
	return df.ToDummies(s.Name())
}

func (s seriesFacade) ToArrow() (iarrow.Table, error) {
	frameOut, err := s.ToFrame()
	if err != nil {
		return iarrow.Table{}, err
	}
	wrapped, ok := frameOut.(*df)
	if !ok {
		return iarrow.Table{}, fmt.Errorf("unsupported dataframe implementation")
	}
	return iarrow.ToTable(wrapped.value), nil
}

func (s seriesFacade) ToInitRepr() string {
	return fmt.Sprintf("Series(name=%q, len=%d, dtype=%s)", s.Name(), s.Len(), s.DataType())
}

func (s seriesFacade) ToJax() []float64 {
	out := make([]float64, s.Len())
	for i := 0; i < s.Len(); i++ {
		if f, ok := toFloat64(s.Value(i)); ok {
			out[i] = f
		}
	}
	return out
}

func (s seriesFacade) ToTorch() []float64 {
	return s.ToJax()
}

func (s seriesFacade) ToPhysical() Series {
	return s.Clone()
}

func (s seriesFacade) Rename(name string) Series {
	return s.Alias(name)
}

func (s seriesFacade) Serialize() ([]byte, error) {
	payload := map[string]any{
		"name":   s.Name(),
		"dtype":  string(s.DataType()),
		"values": s.ToList(),
	}
	return json.Marshal(payload)
}

func (s seriesFacade) Deserialize(payload []byte) (Series, error) {
	var raw struct {
		Name   string `json:"name"`
		DType  string `json:"dtype"`
		Values []any  `json:"values"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	name := raw.Name
	if name == "" {
		name = s.Name()
	}
	dt := dtypes.DataType(raw.DType)
	if dt == "" {
		dt = inferDataTypeFromValues(raw.Values, s.DataType())
	}
	values, err := coerceValuesForDataType(raw.Values, dt)
	if err != nil {
		return nil, err
	}
	out, err := iseries.New(name, dt, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) Hash(seed uint64) Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		h := fnv.New64a()
		_, _ = fmt.Fprintf(h, "%d:%v", seed, s.Value(i))
		values[i] = int64(h.Sum64())
	}
	out, _ := iseries.New(s.value.Name()+"_hash", dtypes.Int64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Implode() Series {
	values := []any{s.ToList()}
	out, _ := iseries.New(s.value.Name(), dtypes.List, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Explode() Series {
	if s.DataType() != dtypes.List {
		return s.Clone()
	}
	values := make([]any, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		list, ok := s.Value(i).([]any)
		if !ok {
			continue
		}
		values = append(values, list...)
	}
	dt := inferDataTypeFromValues(values, dtypes.String)
	out, _ := iseries.New(s.value.Name(), dt, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Flatten() Series {
	return s.Explode()
}

func (s seriesFacade) Extend(other Series) (Series, error) {
	values := append(append([]any{}, s.ToList()...), other.ToList()...)
	dt := inferDataTypeFromValues(values, s.DataType())
	out, err := iseries.New(s.value.Name(), dt, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) ExtendConstant(value any, n int) Series {
	if n <= 0 {
		return s.Clone()
	}
	values := append([]any{}, s.ToList()...)
	for i := 0; i < n; i++ {
		values = append(values, value)
	}
	dt := inferDataTypeFromValues(values, s.DataType())
	out, _ := iseries.New(s.value.Name(), dt, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) NewFromIndex(index int, n int) (Series, error) {
	value, err := s.Item(index)
	if err != nil {
		return nil, err
	}
	if n <= 0 {
		return s.Clear(), nil
	}
	values := make([]any, n)
	for i := 0; i < n; i++ {
		values[i] = value
	}
	out, err := iseries.New(s.value.Name(), s.DataType(), values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) Scatter(indices []int, values []any) (Series, error) {
	if len(indices) != len(values) {
		return nil, fmt.Errorf("scatter indices and values length mismatch")
	}
	outValues := append([]any{}, s.ToList()...)
	for i, idx := range indices {
		if idx < 0 || idx >= len(outValues) {
			return nil, fmt.Errorf("scatter index out of bounds")
		}
		outValues[idx] = values[i]
	}
	dt := inferDataTypeFromValues(outValues, s.DataType())
	out, err := iseries.New(s.value.Name(), dt, outValues)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) Set(mask Series, value any) (Series, error) {
	if s.Len() != mask.Len() {
		return nil, fmt.Errorf("series length mismatch")
	}
	values := append([]any{}, s.ToList()...)
	for i := 0; i < s.Len(); i++ {
		flag, ok := mask.Value(i).(bool)
		if ok && flag {
			values[i] = value
		}
	}
	dt := inferDataTypeFromValues(values, s.DataType())
	out, err := iseries.New(s.value.Name(), dt, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) ZipWith(mask Series, other Series) (Series, error) {
	if s.Len() != mask.Len() || s.Len() != other.Len() {
		return nil, fmt.Errorf("series length mismatch")
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		flag, ok := mask.Value(i).(bool)
		if ok && flag {
			values[i] = s.Value(i)
			continue
		}
		values[i] = other.Value(i)
	}
	dt := inferDataTypeFromValues(values, s.DataType())
	out, err := iseries.New(s.value.Name(), dt, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) Describe() map[string]any {
	out := map[string]any{
		"len":        s.Len(),
		"null_count": s.NullCount(),
		"dtype":      string(s.DataType()),
	}
	nums := make([]float64, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		switch t := v.(type) {
		case int64:
			nums = append(nums, float64(t))
		case float64:
			if !math.IsNaN(t) {
				nums = append(nums, t)
			}
		}
	}
	if len(nums) > 0 {
		minV, maxV, sum := nums[0], nums[0], float64(0)
		for _, n := range nums {
			sum += n
			if n < minV {
				minV = n
			}
			if n > maxV {
				maxV = n
			}
		}
		out["min"] = minV
		out["max"] = maxV
		out["mean"] = sum / float64(len(nums))
	}
	return out
}

func (s seriesFacade) Hist(bins int) (DataFrame, error) {
	if bins <= 0 {
		bins = 10
	}
	nums := make([]float64, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		switch t := s.Value(i).(type) {
		case int64:
			nums = append(nums, float64(t))
		case float64:
			if !math.IsNaN(t) {
				nums = append(nums, t)
			}
		}
	}
	if len(nums) == 0 {
		return NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
			{Name: "bin", Values: []any{}},
			{Name: "count", Values: []any{}},
		}})
	}
	minV, maxV := nums[0], nums[0]
	for _, n := range nums[1:] {
		if n < minV {
			minV = n
		}
		if n > maxV {
			maxV = n
		}
	}
	width := (maxV - minV) / float64(bins)
	if width == 0 {
		width = 1
	}
	counts := make([]int64, bins)
	labels := make([]any, bins)
	for i := 0; i < bins; i++ {
		start := minV + float64(i)*width
		labels[i] = start
	}
	for _, n := range nums {
		idx := int((n - minV) / width)
		if idx >= bins {
			idx = bins - 1
		}
		if idx < 0 {
			idx = 0
		}
		counts[idx]++
	}
	countVals := make([]any, bins)
	for i, c := range counts {
		countVals[i] = c
	}
	return NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "bin", Values: labels},
		{Name: "count", Values: countVals},
	}})
}

func (s seriesFacade) Interpolate() Series {
	values := s.ToList()
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
			l, lok := toFloat64(values[left])
			r, rok := toFloat64(values[right])
			if lok && rok {
				ratio := float64(i-left) / float64(right-left)
				values[i] = l + (r-l)*ratio
			}
		case left >= 0:
			values[i] = values[left]
		case right >= 0:
			values[i] = values[right]
		}
	}
	out, err := iseries.New(s.value.Name(), s.value.DataType(), values)
	if err != nil {
		return s
	}
	return seriesFacade{value: out}
}

func (s seriesFacade) ToNumpy() []any {
	return s.ToList()
}

func (s seriesFacade) ToPandas() []any {
	return s.ToList()
}

func (s seriesFacade) Cast(dt dtypes.DataType) (Series, error) {
	values := make([]any, s.value.Len())
	for i := 0; i < s.value.Len(); i++ {
		if s.value.IsNull(i) {
			values[i] = nil
			continue
		}
		v, err := castAny(s.value.Value(i), dt)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}
	out, err := iseries.New(s.value.Name(), dt, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) Add(other Series) (Series, error) { return s.binaryNumeric(other, "add") }
func (s seriesFacade) Sub(other Series) (Series, error) { return s.binaryNumeric(other, "sub") }
func (s seriesFacade) Mul(other Series) (Series, error) { return s.binaryNumeric(other, "mul") }
func (s seriesFacade) Div(other Series) (Series, error) { return s.binaryNumeric(other, "div") }
func (s seriesFacade) Eq(other Series) (Series, error)  { return s.binaryCompare(other, "eq") }
func (s seriesFacade) Ne(other Series) (Series, error)  { return s.binaryCompare(other, "ne") }
func (s seriesFacade) Gt(other Series) (Series, error)  { return s.binaryCompare(other, "gt") }
func (s seriesFacade) Ge(other Series) (Series, error)  { return s.binaryCompare(other, "ge") }
func (s seriesFacade) Lt(other Series) (Series, error)  { return s.binaryCompare(other, "lt") }
func (s seriesFacade) Le(other Series) (Series, error)  { return s.binaryCompare(other, "le") }

func (s seriesFacade) binaryNumeric(other Series, op string) (Series, error) {
	if s.Len() != other.Len() {
		return nil, fmt.Errorf("series length mismatch")
	}
	outType := dtypes.Float64
	if s.DataType() == dtypes.Int64 && other.DataType() == dtypes.Int64 && op != "div" {
		outType = dtypes.Int64
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		lv := s.Value(i)
		rv := other.Value(i)
		if lv == nil || rv == nil {
			values[i] = nil
			continue
		}
		l, lok := toFloat64(lv)
		r, rok := toFloat64(rv)
		if !lok || !rok {
			return nil, fmt.Errorf("numeric operations require numeric values")
		}
		switch op {
		case "add":
			values[i] = l + r
		case "sub":
			values[i] = l - r
		case "mul":
			values[i] = l * r
		case "div":
			values[i] = l / r
		}
	}
	if outType == dtypes.Int64 {
		intVals := make([]any, len(values))
		for i, v := range values {
			if v == nil {
				intVals[i] = nil
				continue
			}
			intVals[i] = int64(v.(float64))
		}
		values = intVals
	}
	next, err := iseries.New(s.Name(), outType, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: next}, nil
}

func (s seriesFacade) binaryCompare(other Series, op string) (Series, error) {
	if s.Len() != other.Len() {
		return nil, fmt.Errorf("series length mismatch")
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		lv := s.Value(i)
		rv := other.Value(i)
		if lv == nil || rv == nil {
			values[i] = nil
			continue
		}
		values[i] = compareAny(lv, rv, op)
	}
	next, err := iseries.New(s.Name(), dtypes.Boolean, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: next}, nil
}

// rollingFloat64Col extracts a []float64 view and validity mask from a column
// for the linear rolling kernels (Int64 columns are converted; other dtypes
// return ok=false so the caller uses the row-wise path).
func rollingFloat64Col(col *chunk.Column) (vals []float64, nulls []bool, ok bool) {
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

func (s seriesFacade) rolling(window int, mode string) Series {
	if window <= 0 {
		window = 1
	}
	// O(n) typed fast path for sum/mean/min/max via the shared linear kernels.
	// skipNaN is true to match the Series facade's prior behavior, which
	// excluded NaN observations from the window aggregate.
	switch mode {
	case "sum", "mean", "min", "max":
		if vals, nulls, ok := rollingFloat64Col(s.value.Column()); ok {
			var outVals []float64
			var outNulls []bool
			// min_periods defaults to the window size (Polars default), so the
			// leading window-1 positions (and any window with too few valid
			// observations) are null.
			switch mode {
			case "sum":
				outVals, outNulls = chunk.RollingSum(vals, nulls, window, window, true)
			case "mean":
				outVals, outNulls = chunk.RollingMean(vals, nulls, window, window, true)
			case "min":
				outVals, outNulls = chunk.RollingMin(vals, nulls, window, window)
			case "max":
				outVals, outNulls = chunk.RollingMax(vals, nulls, window, window)
			}
			return seriesFacade{value: iseries.FromFloat64(s.value.Name(), outVals, outNulls)}
		}
	}
	values := make([]any, s.value.Len())
	for i := 0; i < s.value.Len(); i++ {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		nums := make([]float64, 0, window)
		for j := start; j <= i; j++ {
			switch t := s.value.Value(j).(type) {
			case int64:
				nums = append(nums, float64(t))
			case float64:
				if !math.IsNaN(t) {
					nums = append(nums, t)
				}
			}
		}
		// min_periods defaults to the window size: a window with fewer valid
		// observations than `window` yields null (matches Polars default).
		if len(nums) < window {
			values[i] = nil
			continue
		}
		switch mode {
		case "sum":
			values[i] = simd.SumFloat64(nums)
		case "mean":
			values[i] = simd.SumFloat64(nums) / float64(len(nums))
		case "min":
			values[i] = simd.MinFloat64(nums)
		case "max":
			values[i] = simd.MaxFloat64(nums)
		case "std", "var":
			mean := meanFloatSlice(nums)
			sumSq := 0.0
			for _, n := range nums {
				d := n - mean
				sumSq += d * d
			}
			denominator := float64(len(nums))
			if len(nums) > 1 {
				denominator = float64(len(nums) - 1)
			}
			variance := sumSq / denominator
			if mode == "std" {
				values[i] = math.Sqrt(variance)
			} else {
				values[i] = variance
			}
		case "median":
			sorted := append([]float64(nil), nums...)
			sort.Float64s(sorted)
			mid := len(sorted) / 2
			if len(sorted)%2 == 0 {
				values[i] = (sorted[mid-1] + sorted[mid]) / 2
			} else {
				values[i] = sorted[mid]
			}
		case "skew":
			n := len(nums)
			if n < 2 {
				values[i] = nil
				break
			}
			mean := meanFloatSlice(nums)
			m2, m3 := 0.0, 0.0
			for _, x := range nums {
				d := x - mean
				m2 += d * d
				m3 += d * d * d
			}
			m2 /= float64(n)
			if m2 == 0 {
				values[i] = 0.0
				break
			}
			sigma := math.Sqrt(m2)
			values[i] = m3 / (float64(n) * sigma * sigma * sigma)
		case "kurtosis":
			n := len(nums)
			if n < 4 {
				values[i] = nil
				break
			}
			mean := meanFloatSlice(nums)
			m2, m4 := 0.0, 0.0
			for _, x := range nums {
				d := x - mean
				d2 := d * d
				m2 += d2
				m4 += d2 * d2
			}
			m2 /= float64(n)
			if m2 == 0 {
				values[i] = 0.0
				break
			}
			m4 /= float64(n)
			values[i] = m4/(m2*m2) - 3
		case "rank":
			last := nums[len(nums)-1]
			if math.IsNaN(last) {
				values[i] = nil
				break
			}
			less, eq := 0, 0
			for _, v := range nums {
				if math.IsNaN(v) {
					continue
				}
				if v < last {
					less++
				} else if v == last {
					eq++
				}
			}
			if eq == 0 {
				values[i] = nil
				break
			}
			values[i] = float64(less) + (float64(eq)+1)/2
		}
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Float64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) rollingQuantile(window int, q float64) Series {
	if window <= 0 {
		window = 1
	}
	values := make([]any, s.value.Len())
	for i := 0; i < s.value.Len(); i++ {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		nums := make([]float64, 0, window)
		for j := start; j <= i; j++ {
			if f, ok := toFloat64(s.value.Value(j)); ok && !math.IsNaN(f) {
				nums = append(nums, f)
			}
		}
		// min_periods defaults to the window size (Polars default).
		if len(nums) < window {
			values[i] = nil
			continue
		}
		values[i] = quantileFloatSlice(nums, q)
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Float64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) unaryNumeric(op string) Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		f, ok := toFloat64(v)
		if !ok {
			values[i] = nil
			continue
		}
		switch op {
		case "abs":
			values[i] = math.Abs(f)
		case "exp":
			values[i] = math.Exp(f)
		case "log":
			values[i] = math.Log(f)
		case "sqrt":
			values[i] = math.Sqrt(f)
		case "sin":
			values[i] = math.Sin(f)
		case "cos":
			values[i] = math.Cos(f)
		case "tan":
			values[i] = math.Tan(f)
		case "cot":
			t := math.Tan(f)
			if t == 0 {
				values[i] = math.Copysign(math.Inf(1), f)
			} else {
				values[i] = 1 / t
			}
		case "sinh":
			values[i] = math.Sinh(f)
		case "cosh":
			values[i] = math.Cosh(f)
		case "tanh":
			values[i] = math.Tanh(f)
		case "arcsin":
			values[i] = math.Asin(f)
		case "arccos":
			values[i] = math.Acos(f)
		case "arctan":
			values[i] = math.Atan(f)
		case "arcsinh":
			values[i] = math.Asinh(f)
		case "arccosh":
			values[i] = math.Acosh(f)
		case "arctanh":
			values[i] = math.Atanh(f)
		case "cbrt":
			values[i] = math.Cbrt(f)
		case "ceil":
			values[i] = math.Ceil(f)
		case "floor":
			values[i] = math.Floor(f)
		case "degrees":
			values[i] = f * 180 / math.Pi
		case "sign":
			switch {
			case f < 0:
				values[i] = float64(-1)
			case f > 0:
				values[i] = float64(1)
			default:
				values[i] = float64(0)
			}
		case "log10":
			values[i] = math.Log10(f)
		case "log1p":
			values[i] = math.Log1p(f)
		case "round":
			values[i] = math.Round(f)
		}
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Float64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) numericValues(skipNaN bool) []float64 {
	col := s.value.Column()
	// Float64 fast path: read typed backing directly, skip per-element boxing.
	if f64s, ok := col.Float64s(); ok {
		nulls := col.Nulls()
		out := make([]float64, 0, len(f64s))
		for i, v := range f64s {
			if nulls != nil && nulls[i] {
				continue
			}
			if skipNaN && math.IsNaN(v) {
				continue
			}
			out = append(out, v)
		}
		return out
	}
	// Int64 fast path.
	if i64s, ok := col.Int64s(); ok {
		nulls := col.Nulls()
		out := make([]float64, 0, len(i64s))
		for i, v := range i64s {
			if nulls != nil && nulls[i] {
				continue
			}
			out = append(out, float64(v))
		}
		return out
	}
	// Boolean fast path: reductions treat true as 1.0 and false as 0.0
	// (sum = count of true, min/max follow truthiness), matching Polars.
	if bs, ok := col.Bools(); ok {
		nulls := col.Nulls()
		out := make([]float64, 0, len(bs))
		for i, b := range bs {
			if nulls != nil && nulls[i] {
				continue
			}
			if b {
				out = append(out, 1.0)
			} else {
				out = append(out, 0.0)
			}
		}
		return out
	}
	// Slow path for other dtypes.
	values := make([]float64, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if v == nil {
			continue
		}
		if b, ok := v.(bool); ok {
			if b {
				values = append(values, 1.0)
			} else {
				values = append(values, 0.0)
			}
			continue
		}
		f, ok := toFloat64(v)
		if !ok {
			continue
		}
		if skipNaN && math.IsNaN(f) {
			continue
		}
		values = append(values, f)
	}
	return values
}

func meanFloatSlice(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func quantileFloatSlice(values []float64, q float64) any {
	if len(values) == 0 {
		return nil
	}
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	idx := float64(len(sorted)-1) * q
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper {
		return sorted[lower]
	}
	weight := idx - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func valueKey(v any) string {
	return fmt.Sprintf("%T:%v", v, v)
}

func newIndexSeries(name string, indexes []int) Series {
	values := make([]any, len(indexes))
	for i, idx := range indexes {
		values[i] = int64(idx)
	}
	out, _ := iseries.New(name, dtypes.Int64, values)
	return seriesFacade{value: out}
}

func compareForSeriesOrder(left any, right any) int {
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
		switch {
		case l < r:
			return -1
		case l > r:
			return 1
		default:
			return 0
		}
	case float64:
		r, ok := right.(float64)
		if !ok {
			return 0
		}
		switch {
		case l < r:
			return -1
		case l > r:
			return 1
		default:
			return 0
		}
	case string:
		r, ok := right.(string)
		if !ok {
			return 0
		}
		switch {
		case l < r:
			return -1
		case l > r:
			return 1
		default:
			return 0
		}
	case bool:
		r, ok := right.(bool)
		if !ok {
			return 0
		}
		if l == r {
			return 0
		}
		if !l && r {
			return -1
		}
		return 1
	case time.Time:
		r, ok := right.(time.Time)
		if !ok {
			return 0
		}
		switch {
		case l.Before(r):
			return -1
		case l.After(r):
			return 1
		default:
			return 0
		}
	default:
		return 0
	}
}

func (s seriesFacade) booleanMap(fn func(v any) bool) Series {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		values[i] = fn(s.Value(i))
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Boolean, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) valueCounts() map[string]int {
	counts := map[string]int{}
	for i := 0; i < s.Len(); i++ {
		counts[valueKey(s.Value(i))]++
	}
	return counts
}

func (s seriesFacade) binaryMissing(other Series, equal bool) (Series, error) {
	if s.Len() != other.Len() {
		return nil, fmt.Errorf("series length mismatch")
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		left := s.Value(i)
		right := other.Value(i)
		match := false
		switch {
		case left == nil || right == nil:
			match = left == right
		default:
			match = valueKey(left) == valueKey(right)
		}
		if equal {
			values[i] = match
		} else {
			values[i] = !match
		}
	}
	out, err := iseries.New(s.value.Name(), dtypes.Boolean, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) kBy(by Series, k int, descending bool) (Series, error) {
	if s.Len() != by.Len() {
		return nil, fmt.Errorf("series length mismatch")
	}
	if k <= 0 {
		return s.Clear(), nil
	}
	if k > s.Len() {
		k = s.Len()
	}
	indexes := make([]int, s.Len())
	for i := range indexes {
		indexes[i] = i
	}
	sort.SliceStable(indexes, func(i int, j int) bool {
		cmp := compareForSeriesOrder(by.Value(indexes[i]), by.Value(indexes[j]))
		if descending {
			return cmp > 0
		}
		return cmp < 0
	})
	return s.Gather(indexes[:k]), nil
}

func (s seriesFacade) cumulative(mode string) Series {
	values := make([]any, s.Len())
	sum := 0.0
	count := 0.0
	product := 1.0
	var running any
	for i := 0; i < s.Len(); i++ {
		current := s.Value(i)
		f, ok := toFloat64(current)
		if !ok {
			values[i] = nil
			continue
		}
		switch mode {
		case "sum":
			sum += f
			values[i] = sum
		case "max":
			if i == 0 || compareForSeriesOrder(f, running) > 0 {
				running = f
			}
			values[i] = running
		case "min":
			if i == 0 || compareForSeriesOrder(f, running) < 0 {
				running = f
			}
			values[i] = running
		case "prod":
			product *= f
			values[i] = product
		case "count":
			count++
			values[i] = count
		}
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Float64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) ewm(alpha float64, mode string) Series {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.5
	}
	values := make([]any, s.Len())
	mean := 0.0
	variance := 0.0
	initialized := false
	for i := 0; i < s.Len(); i++ {
		current, ok := toFloat64(s.Value(i))
		if !ok {
			values[i] = nil
			continue
		}
		if !initialized {
			mean = current
			variance = 0
			initialized = true
		} else {
			mean = alpha*current + (1-alpha)*mean
			diff := current - mean
			variance = alpha*diff*diff + (1-alpha)*variance
		}
		switch mode {
		case "mean":
			values[i] = mean
		case "var":
			values[i] = variance
		case "std":
			values[i] = math.Sqrt(variance)
		}
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Float64, values)
	return seriesFacade{value: out}
}

func inferDataTypeFromValues(values []any, fallback dtypes.DataType) dtypes.DataType {
	for _, v := range values {
		if v == nil {
			continue
		}
		switch v.(type) {
		case int64:
			return dtypes.Int64
		case float64:
			return dtypes.Float64
		case string:
			return dtypes.String
		case bool:
			return dtypes.Boolean
		case time.Time:
			return dtypes.Datetime
		case dtypes.DecimalValue:
			return dtypes.Decimal
		case []any:
			return dtypes.List
		case map[string]any:
			return dtypes.Struct
		case []byte:
			return dtypes.Binary
		case time.Duration:
			return dtypes.Duration
		default:
			return fallback
		}
	}
	return fallback
}

func coerceValuesForDataType(values []any, dt dtypes.DataType) ([]any, error) {
	out := make([]any, len(values))
	for i, v := range values {
		if v == nil {
			out[i] = nil
			continue
		}
		cast, err := castAny(v, dt)
		if err != nil {
			return nil, err
		}
		out[i] = cast
	}
	return out, nil
}

func valueEquals(left any, right any) bool {
	switch {
	case left == nil || right == nil:
		return left == right
	default:
		return valueKey(left) == valueKey(right)
	}
}

func toFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case int64:
		return float64(t), true
	case float64:
		return t, true
	default:
		return 0, false
	}
}

func compareAny(left any, right any, op string) bool {
	switch l := left.(type) {
	case int64:
		r, ok := right.(int64)
		if !ok {
			return false
		}
		switch op {
		case "eq":
			return l == r
		case "ne":
			return l != r
		case "gt":
			return l > r
		case "ge":
			return l >= r
		case "lt":
			return l < r
		case "le":
			return l <= r
		}
	case float64:
		r, ok := right.(float64)
		if !ok {
			return false
		}
		switch op {
		case "eq":
			return l == r
		case "ne":
			return l != r
		case "gt":
			return l > r
		case "ge":
			return l >= r
		case "lt":
			return l < r
		case "le":
			return l <= r
		}
	case string:
		r, ok := right.(string)
		if !ok {
			return false
		}
		switch op {
		case "eq":
			return l == r
		case "ne":
			return l != r
		case "gt":
			return l > r
		case "ge":
			return l >= r
		case "lt":
			return l < r
		case "le":
			return l <= r
		}
	case bool:
		r, ok := right.(bool)
		if !ok {
			return false
		}
		switch op {
		case "eq":
			return l == r
		case "ne":
			return l != r
		case "gt":
			return l && !r
		case "ge":
			return l == r || (l && !r)
		case "lt":
			return !l && r
		case "le":
			return l == r || (!l && r)
		}
	}
	return false
}

func castAny(v any, dt dtypes.DataType) (any, error) {
	switch dt {
	case dtypes.Int64:
		switch t := v.(type) {
		case int64:
			return t, nil
		case float64:
			return int64(t), nil
		case bool:
			if t {
				return int64(1), nil
			}
			return int64(0), nil
		case string:
			var x int64
			_, err := fmt.Sscan(t, &x)
			return x, err
		}
	case dtypes.Float64:
		switch t := v.(type) {
		case int64:
			return float64(t), nil
		case float64:
			return t, nil
		case bool:
			if t {
				return float64(1), nil
			}
			return float64(0), nil
		case string:
			var x float64
			_, err := fmt.Sscan(t, &x)
			return x, err
		}
	case dtypes.String:
		return fmt.Sprintf("%v", v), nil
	case dtypes.Boolean:
		switch t := v.(type) {
		case bool:
			return t, nil
		case string:
			return t == "true", nil
		case int64:
			return t != 0, nil
		}
	case dtypes.Datetime:
		switch t := v.(type) {
		case time.Time:
			return t, nil
		case string:
			return time.Parse(time.RFC3339Nano, t)
		}
	case dtypes.Decimal:
		return dtypes.DecimalValue(fmt.Sprintf("%v", v)), nil
	}
	return nil, fmt.Errorf("cannot cast value to %s", dt)
}
