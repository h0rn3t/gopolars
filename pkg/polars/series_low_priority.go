package polars

import (
	"fmt"
	"math"
	"math/bits"
	"sort"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	iseries "github.com/eugeneshershen/gopolars/pkg/series"
)

func (s seriesFacade) Count() int {
	return s.Len() - s.NullCount()
}

func (s seriesFacade) ApproxNUnique() int {
	return s.NUnique()
}

func (s seriesFacade) Equals(other Series) (bool, error) {
	o, ok := other.(seriesFacade)
	if !ok {
		return false, fmt.Errorf("equals: unsupported series implementation")
	}
	if s.Len() != o.Len() || s.DataType() != o.DataType() {
		return false, nil
	}
	for i := 0; i < s.Len(); i++ {
		if !valueEquals(s.Value(i), o.Value(i)) {
			return false, nil
		}
	}
	return true, nil
}

func (s seriesFacade) EstimatedSize() int64 {
	n := int64(s.Len())
	if n == 0 {
		return 0
	}
	var unit int64 = 8
	switch s.DataType() {
	case dtypes.Boolean:
		unit = 1
	case dtypes.String:
		unit = 16
	case dtypes.List, dtypes.Struct:
		unit = 24
	}
	return n*unit + n
}

func (s seriesFacade) Filter(mask Series) (Series, error) {
	if mask.Len() != s.Len() {
		return nil, fmt.Errorf("filter: mask length %d != series length %d", mask.Len(), s.Len())
	}
	if mask.DataType() != dtypes.Boolean {
		return nil, fmt.Errorf("filter: expected boolean mask, got %s", mask.DataType())
	}
	mInt, err := toInternalSeries(mask)
	if err != nil {
		return nil, fmt.Errorf("filter: %w", err)
	}
	idx := make([]int, 0)
	for i := 0; i < mInt.Len(); i++ {
		if mInt.IsNull(i) {
			continue
		}
		b, ok := mInt.Value(i).(bool)
		if ok && b {
			idx = append(idx, i)
		}
	}
	return s.Gather(idx), nil
}

func (s seriesFacade) Truncate(maxLen int) (Series, error) {
	if maxLen < 0 {
		return nil, fmt.Errorf("truncate: maxLen must be >= 0")
	}
	if s.DataType() != dtypes.String {
		return nil, fmt.Errorf("truncate: expected string dtype, got %s", s.DataType())
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("truncate: value at %d is not string", i)
		}
		runes := []rune(str)
		if len(runes) > maxLen {
			str = string(runes[:maxLen])
		}
		values[i] = str
	}
	out, err := iseries.New(s.Name(), dtypes.String, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func roundSigFigs(x float64, sig int) float64 {
	if sig <= 0 || math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}
	if x == 0 {
		return 0
	}
	sign := 1.0
	if x < 0 {
		sign = -1
		x = -x
	}
	exp := math.Floor(math.Log10(x))
	scale := math.Pow(10, float64(sig-1)-exp)
	return sign * math.Round(x*scale) / scale
}

func (s seriesFacade) RoundSigFigs(sigFigs int) (Series, error) {
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		f, ok := toFloat64(v)
		if !ok {
			return nil, fmt.Errorf("round_sig_figs: non-numeric value at %d", i)
		}
		values[i] = roundSigFigs(f, sigFigs)
	}
	out, err := iseries.New(s.Name(), dtypes.Float64, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) Clip(lower, upper float64) Series {
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
		if f < lower {
			f = lower
		}
		if f > upper {
			f = upper
		}
		values[i] = f
	}
	out, _ := iseries.New(s.Name(), dtypes.Float64, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) Cot() Series {
	return s.unaryNumeric("cot")
}

func (s seriesFacade) ShrinkDType() (Series, error) {
	if s.DataType() != dtypes.Float64 {
		return s.Clone(), nil
	}
	intVals := make([]any, s.Len())
	allWholeInInt64 := true
	for i := 0; i < s.Len(); i++ {
		v := s.Value(i)
		if v == nil {
			intVals[i] = nil
			continue
		}
		f, ok := v.(float64)
		if !ok || math.IsNaN(f) || math.IsInf(f, 0) {
			allWholeInInt64 = false
			break
		}
		if f != math.Trunc(f) || f > float64(math.MaxInt64) || f < float64(math.MinInt64) {
			allWholeInInt64 = false
			break
		}
		intVals[i] = int64(f)
	}
	if !allWholeInInt64 {
		return s.Clone(), nil
	}
	out, err := iseries.New(s.Name(), dtypes.Int64, intVals)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) List() SeriesArrNS {
	return SeriesArrNS{s: s}
}

func (s seriesFacade) Append(other Series) (Series, error) {
	o, ok := other.(seriesFacade)
	if !ok {
		return nil, fmt.Errorf("append: unsupported series implementation")
	}
	if s.DataType() != o.DataType() {
		return nil, fmt.Errorf("append: dtype mismatch %s vs %s", s.DataType(), o.DataType())
	}
	combined := append(s.ToList(), o.ToList()...)
	out, err := iseries.New(s.Name(), s.DataType(), combined)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) GetChunks() ([]Series, error) {
	return []Series{s.Clone()}, nil
}

func (s seriesFacade) Flags() map[string]bool {
	return map[string]bool{
		"sorted":     s.IsSorted(),
		"has_nulls":  s.HasNulls(),
		"has_validity": s.HasValidity(),
	}
}

func (s seriesFacade) MaxBy(by Series) (Series, error) {
	return s.extremeBy(by, true)
}

func (s seriesFacade) MinBy(by Series) (Series, error) {
	return s.extremeBy(by, false)
}

func (s seriesFacade) extremeBy(by Series, max bool) (Series, error) {
	if s.Len() != by.Len() {
		return nil, fmt.Errorf("max_by/min_by: length mismatch")
	}
	order := make([]string, 0)
	seen := map[string]struct{}{}
	best := make(map[string]any)
	set := make(map[string]bool)

	for i := 0; i < s.Len(); i++ {
		k := valueKey(by.Value(i))
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			order = append(order, k)
		}
		cur := s.Value(i)
		if !set[k] {
			best[k] = cur
			set[k] = true
			continue
		}
		prev := best[k]
		if cur == nil {
			continue
		}
		if prev == nil {
			best[k] = cur
			continue
		}
		cmp := compareForSeriesOrder(cur, prev)
		if max && cmp > 0 {
			best[k] = cur
		}
		if !max && cmp < 0 {
			best[k] = cur
		}
	}
	vals := make([]any, 0, len(order))
	for _, k := range order {
		vals = append(vals, best[k])
	}
	out, err := iseries.New(s.Name()+"_by", s.DataType(), vals)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func sortPermBy(by Series) []int {
	n := by.Len()
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		return compareForSeriesOrder(by.Value(idx[i]), by.Value(idx[j])) < 0
	})
	return idx
}

func permInverse(perm []int) []int {
	inv := make([]int, len(perm))
	for pos, orig := range perm {
		inv[orig] = pos
	}
	return inv
}

func (s seriesFacade) gatherValues(perm []int) []any {
	out := make([]any, len(perm))
	for i, o := range perm {
		out[i] = s.Value(o)
	}
	return out
}

func (s seriesFacade) rollingScatterBy(by Series, window int, mode string) (Series, error) {
	if s.Len() != by.Len() {
		return nil, fmt.Errorf("rolling_by: length mismatch")
	}
	perm := sortPermBy(by)
	sortedVals := s.gatherValues(perm)
	tmp, err := iseries.New(s.Name()+"_sorted", s.DataType(), sortedVals)
	if err != nil {
		return nil, err
	}
	inner := seriesFacade{value: tmp}
	rolled := inner.rolling(window, mode)
	rf, ok := rolled.(seriesFacade)
	if !ok {
		return nil, fmt.Errorf("rolling_by: internal rolling failed")
	}
	inv := permInverse(perm)
	outVals := make([]any, s.Len())
	for orig := 0; orig < s.Len(); orig++ {
		pos := inv[orig]
		outVals[orig] = rf.value.Value(pos)
	}
	out, err := iseries.New(s.Name(), dtypes.Float64, outVals)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) rollingQuantileScatterBy(by Series, window int, q float64) (Series, error) {
	if s.Len() != by.Len() {
		return nil, fmt.Errorf("rolling_quantile_by: length mismatch")
	}
	perm := sortPermBy(by)
	sortedVals := s.gatherValues(perm)
	tmp, err := iseries.New(s.Name()+"_sorted", s.DataType(), sortedVals)
	if err != nil {
		return nil, err
	}
	inner := seriesFacade{value: tmp}
	rolled := inner.rollingQuantile(window, q)
	rf, ok := rolled.(seriesFacade)
	if !ok {
		return nil, fmt.Errorf("rolling_quantile_by: internal rolling failed")
	}
	inv := permInverse(perm)
	outVals := make([]any, s.Len())
	for orig := 0; orig < s.Len(); orig++ {
		outVals[orig] = rf.value.Value(inv[orig])
	}
	out, err := iseries.New(s.Name(), dtypes.Float64, outVals)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) RollingMeanBy(by Series, window int) (Series, error) {
	return s.rollingScatterBy(by, window, "mean")
}

func (s seriesFacade) RollingSumBy(by Series, window int) (Series, error) {
	return s.rollingScatterBy(by, window, "sum")
}

func (s seriesFacade) RollingMinBy(by Series, window int) (Series, error) {
	return s.rollingScatterBy(by, window, "min")
}

func (s seriesFacade) RollingMaxBy(by Series, window int) (Series, error) {
	return s.rollingScatterBy(by, window, "max")
}

func (s seriesFacade) RollingStdBy(by Series, window int) (Series, error) {
	return s.rollingScatterBy(by, window, "std")
}

func (s seriesFacade) RollingVarBy(by Series, window int) (Series, error) {
	return s.rollingScatterBy(by, window, "var")
}

func (s seriesFacade) RollingMedianBy(by Series, window int) (Series, error) {
	return s.rollingScatterBy(by, window, "median")
}

func (s seriesFacade) RollingQuantileBy(by Series, window int, q float64) (Series, error) {
	return s.rollingQuantileScatterBy(by, window, q)
}

func (s seriesFacade) RollingRank(window int) Series {
	return s.rolling(window, "rank")
}

func (s seriesFacade) RollingRankBy(by Series, window int) (Series, error) {
	return s.rollingScatterBy(by, window, "rank")
}

func (s seriesFacade) RollingSkew(window int) Series {
	return s.rolling(window, "skew")
}

func (s seriesFacade) RollingKurtosis(window int) Series {
	return s.rolling(window, "kurtosis")
}

func (s seriesFacade) RollingMap(window int, fn func([]float64) float64) (Series, error) {
	if fn == nil {
		return nil, fmt.Errorf("rolling_map: fn is nil")
	}
	if window <= 0 {
		window = 1
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		nums := make([]float64, 0, window)
		for j := start; j <= i; j++ {
			if f, ok := toFloat64(s.Value(j)); ok && !math.IsNaN(f) {
				nums = append(nums, f)
			}
		}
		if len(nums) == 0 {
			values[i] = nil
			continue
		}
		values[i] = fn(nums)
	}
	out, err := iseries.New(s.Name(), dtypes.Float64, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) EwmMeanBy(by Series, alpha float64) (Series, error) {
	if s.Len() != by.Len() {
		return nil, fmt.Errorf("ewm_mean_by: length mismatch")
	}
	perm := sortPermBy(by)
	sortedVals := s.gatherValues(perm)
	tmp, err := iseries.New(s.Name()+"_sorted", s.DataType(), sortedVals)
	if err != nil {
		return nil, err
	}
	inner := seriesFacade{value: tmp}
	ewm := inner.EwmMean(alpha)
	rf, ok := ewm.(seriesFacade)
	if !ok {
		return nil, fmt.Errorf("ewm_mean_by: internal ewm failed")
	}
	inv := permInverse(perm)
	outVals := make([]any, s.Len())
	for orig := 0; orig < s.Len(); orig++ {
		outVals[orig] = rf.value.Value(inv[orig])
	}
	out, err := iseries.New(s.Name(), dtypes.Float64, outVals)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func requireInt64Series(name string, s Series) error {
	if s.DataType() != dtypes.Int64 {
		return fmt.Errorf("%s: expected int64 dtype, got %s", name, s.DataType())
	}
	return nil
}

func (s seriesFacade) BitwiseAnd(other Series) (Series, error) {
	if err := requireInt64Series("bitwise_and", s); err != nil {
		return nil, err
	}
	if err := requireInt64Series("bitwise_and", other); err != nil {
		return nil, err
	}
	if s.Len() != other.Len() {
		return nil, fmt.Errorf("bitwise_and: length mismatch")
	}
	values := make([]any, s.Len())
	o, ok := other.(seriesFacade)
	if !ok {
		return nil, fmt.Errorf("bitwise_and: unsupported series implementation")
	}
	for i := 0; i < s.Len(); i++ {
		if s.value.IsNull(i) || o.value.IsNull(i) {
			values[i] = nil
			continue
		}
		a, _ := s.Value(i).(int64)
		b, _ := o.Value(i).(int64)
		values[i] = a & b
	}
	out, err := iseries.New(s.Name(), dtypes.Int64, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) BitwiseOr(other Series) (Series, error) {
	if err := requireInt64Series("bitwise_or", s); err != nil {
		return nil, err
	}
	if err := requireInt64Series("bitwise_or", other); err != nil {
		return nil, err
	}
	if s.Len() != other.Len() {
		return nil, fmt.Errorf("bitwise_or: length mismatch")
	}
	o, ok := other.(seriesFacade)
	if !ok {
		return nil, fmt.Errorf("bitwise_or: unsupported series implementation")
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.value.IsNull(i) || o.value.IsNull(i) {
			values[i] = nil
			continue
		}
		a, _ := s.Value(i).(int64)
		b, _ := o.Value(i).(int64)
		values[i] = a | b
	}
	out, err := iseries.New(s.Name(), dtypes.Int64, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) BitwiseXor(other Series) (Series, error) {
	if err := requireInt64Series("bitwise_xor", s); err != nil {
		return nil, err
	}
	if err := requireInt64Series("bitwise_xor", other); err != nil {
		return nil, err
	}
	if s.Len() != other.Len() {
		return nil, fmt.Errorf("bitwise_xor: length mismatch")
	}
	o, ok := other.(seriesFacade)
	if !ok {
		return nil, fmt.Errorf("bitwise_xor: unsupported series implementation")
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.value.IsNull(i) || o.value.IsNull(i) {
			values[i] = nil
			continue
		}
		a, _ := s.Value(i).(int64)
		b, _ := o.Value(i).(int64)
		values[i] = a ^ b
	}
	out, err := iseries.New(s.Name(), dtypes.Int64, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) int64UnaryBits(fn func(int64) int64) (Series, error) {
	if err := requireInt64Series("bitwise_op", s); err != nil {
		return nil, err
	}
	values := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.value.IsNull(i) {
			values[i] = nil
			continue
		}
		v, _ := s.Value(i).(int64)
		values[i] = fn(v)
	}
	out, err := iseries.New(s.Name(), dtypes.Int64, values)
	if err != nil {
		return nil, err
	}
	return seriesFacade{value: out}, nil
}

func (s seriesFacade) BitwiseCountOnes() (Series, error) {
	return s.int64UnaryBits(func(v int64) int64 {
		return int64(bits.OnesCount64(uint64(v)))
	})
}

func (s seriesFacade) BitwiseCountZeros() (Series, error) {
	return s.int64UnaryBits(func(v int64) int64 {
		return int64(64 - bits.OnesCount64(uint64(v)))
	})
}

func (s seriesFacade) BitwiseLeadingOnes() (Series, error) {
	return s.int64UnaryBits(func(v int64) int64 {
		return int64(bits.LeadingZeros64(^uint64(v)))
	})
}

func (s seriesFacade) BitwiseLeadingZeros() (Series, error) {
	return s.int64UnaryBits(func(v int64) int64 {
		return int64(bits.LeadingZeros64(uint64(v)))
	})
}

func (s seriesFacade) BitwiseTrailingOnes() (Series, error) {
	return s.int64UnaryBits(func(v int64) int64 {
		return int64(bits.TrailingZeros64(^uint64(v)))
	})
}

func (s seriesFacade) BitwiseTrailingZeros() (Series, error) {
	return s.int64UnaryBits(func(v int64) int64 {
		return int64(bits.TrailingZeros64(uint64(v)))
	})
}

func (s seriesFacade) Reinterpret(dt dtypes.DataType) (Series, error) {
	return nil, fmt.Errorf("reinterpret(%s): not supported in gopolars v1; use Cast or ToPhysical", dt)
}
