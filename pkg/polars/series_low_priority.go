package polars

import (
	"fmt"
	"math"
	"math/bits"
	"slices"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	iseries "github.com/h0rn3t/gopolars/pkg/series"
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
	values, err := mapValues(s, "truncate", "string", func(str string) any {
		runes := []rune(str)
		if len(runes) > maxLen {
			str = string(runes[:maxLen])
		}
		return str
	})
	if err != nil {
		return nil, err
	}
	return NewSeries(NewSeriesInput{Name: s.Name(), DType: dtypes.String, Values: values})
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
		f, ok := expr.ToFloat(v)
		if !ok {
			return nil, fmt.Errorf("round_sig_figs: non-numeric value at %d", i)
		}
		values[i] = roundSigFigs(f, sigFigs)
	}
	return NewSeries(NewSeriesInput{Name: s.Name(), DType: dtypes.Float64, Values: values})
}

func (s seriesFacade) Clip(lower, upper float64) Series {
	return s.unaryNumeric(func(f float64) float64 {
		if f < lower {
			f = lower
		}
		if f > upper {
			f = upper
		}
		return f
	})
}

func (s seriesFacade) Cot() Series {
	return s.unaryNumeric(func(f float64) float64 {
		t := math.Tan(f)
		if t == 0 {
			return math.Copysign(math.Inf(1), f)
		}
		return 1 / t
	})
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
	return NewSeries(NewSeriesInput{Name: s.Name(), DType: dtypes.Int64, Values: intVals})
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
	return NewSeries(NewSeriesInput{Name: s.Name(), DType: s.DataType(), Values: combined})
}

func (s seriesFacade) GetChunks() ([]Series, error) {
	return []Series{s.Clone()}, nil
}

func (s seriesFacade) Flags() map[string]bool {
	return map[string]bool{
		"sorted":       s.IsSorted(),
		"has_nulls":    s.HasNulls(),
		"has_validity": s.HasValidity(),
	}
}

func (s seriesFacade) MaxBy(by Series) (Series, error) {
	return s.extremeBy(by, true)
}

func (s seriesFacade) MinBy(by Series) (Series, error) {
	return s.extremeBy(by, false)
}

func (s seriesFacade) extremeBy(by Series, wantMax bool) (Series, error) {
	if s.Len() != by.Len() {
		return nil, fmt.Errorf("max_by/min_by: length mismatch")
	}
	order := make([]string, 0)
	best := make(map[string]any)

	for i := 0; i < s.Len(); i++ {
		k := valueKey(by.Value(i))
		cur := s.Value(i)
		prev, ok := best[k]
		if !ok {
			order = append(order, k)
			best[k] = cur
			continue
		}
		if cur == nil {
			continue
		}
		if prev == nil {
			best[k] = cur
			continue
		}
		cmp := compareForSeriesOrder(cur, prev)
		if wantMax && cmp > 0 {
			best[k] = cur
		}
		if !wantMax && cmp < 0 {
			best[k] = cur
		}
	}
	vals := make([]any, 0, len(order))
	for _, k := range order {
		vals = append(vals, best[k])
	}
	return NewSeries(NewSeriesInput{Name: s.Name() + "_by", DType: s.DataType(), Values: vals})
}

func sortPermBy(by Series, descending bool) []int {
	idx := make([]int, by.Len())
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int {
		c := compareForSeriesOrder(by.Value(a), by.Value(b))
		if descending {
			return -c
		}
		return c
	})
	return idx
}

func (s seriesFacade) gatherValues(perm []int) []any {
	out := make([]any, len(perm))
	for i, o := range perm {
		out[i] = s.Value(o)
	}
	return out
}

// applySortedBy runs fn over s reordered by ascending by, then scatters fn's
// result back to s's row order as Float64. op prefixes the length error.
func (s seriesFacade) applySortedBy(by Series, op string, fn func(sorted seriesFacade) Series) (Series, error) {
	if s.Len() != by.Len() {
		return nil, fmt.Errorf("%s: length mismatch", op)
	}
	perm := sortPermBy(by, false)
	tmp, err := iseries.New(s.Name()+"_sorted", s.DataType(), s.gatherValues(perm))
	if err != nil {
		return nil, err
	}
	result := fn(seriesFacade{value: tmp})
	outVals := make([]any, s.Len())
	for pos, orig := range perm {
		outVals[orig] = result.Value(pos)
	}
	return NewSeries(NewSeriesInput{Name: s.Name(), DType: dtypes.Float64, Values: outVals})
}

func (s seriesFacade) rollingScatterBy(by Series, window int, mode string) (Series, error) {
	return s.applySortedBy(by, "rolling_by", func(sorted seriesFacade) Series {
		return sorted.rolling(window, mode)
	})
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
	return s.applySortedBy(by, "rolling_quantile_by", func(sorted seriesFacade) Series {
		return sorted.rollingQuantile(window, q)
	})
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
		nums := s.numericRange(max(i-window+1, 0), i+1)
		if len(nums) == 0 {
			values[i] = nil
			continue
		}
		values[i] = fn(nums)
	}
	return NewSeries(NewSeriesInput{Name: s.Name(), DType: dtypes.Float64, Values: values})
}

func (s seriesFacade) EwmMeanBy(by Series, alpha float64) (Series, error) {
	return s.applySortedBy(by, "ewm_mean_by", func(sorted seriesFacade) Series {
		return sorted.EwmMean(alpha)
	})
}

func requireInt64Series(name string, s Series) error {
	if s.DataType() != dtypes.Int64 {
		return fmt.Errorf("%s: expected int64 dtype, got %s", name, s.DataType())
	}
	return nil
}

// bitwiseBinary applies a bitwise op element-wise. Both operands must share the
// same dtype: Int64 (integer bitwise) or Boolean (logical and/or/xor). Boolean
// inputs yield a Boolean result, matching Polars.
func (s seriesFacade) bitwiseBinary(other Series, name string, intOp func(int64, int64) int64, boolOp func(bool, bool) bool) (Series, error) {
	if s.Len() != other.Len() {
		return nil, fmt.Errorf("%s: length mismatch", name)
	}
	o, ok := other.(seriesFacade)
	if !ok {
		return nil, fmt.Errorf("%s: unsupported series implementation", name)
	}
	dt := s.DataType()
	if dt != other.DataType() {
		return nil, fmt.Errorf("%s: dtype mismatch (%s vs %s)", name, dt, other.DataType())
	}
	values := make([]any, s.Len())
	switch dt {
	case dtypes.Int64:
		for i := 0; i < s.Len(); i++ {
			if s.value.IsNull(i) || o.value.IsNull(i) {
				values[i] = nil
				continue
			}
			a, _ := s.Value(i).(int64)
			b, _ := o.Value(i).(int64)
			values[i] = intOp(a, b)
		}
	case dtypes.Boolean:
		for i := 0; i < s.Len(); i++ {
			if s.value.IsNull(i) || o.value.IsNull(i) {
				values[i] = nil
				continue
			}
			a, _ := s.Value(i).(bool)
			b, _ := o.Value(i).(bool)
			values[i] = boolOp(a, b)
		}
	default:
		return nil, fmt.Errorf("%s: expected int64 or bool dtype, got %s", name, dt)
	}
	return NewSeries(NewSeriesInput{Name: s.Name(), DType: dt, Values: values})
}

func (s seriesFacade) BitwiseAnd(other Series) (Series, error) {
	return s.bitwiseBinary(other, "bitwise_and",
		func(a, b int64) int64 { return a & b },
		func(a, b bool) bool { return a && b })
}

func (s seriesFacade) BitwiseOr(other Series) (Series, error) {
	return s.bitwiseBinary(other, "bitwise_or",
		func(a, b int64) int64 { return a | b },
		func(a, b bool) bool { return a || b })
}

func (s seriesFacade) BitwiseXor(other Series) (Series, error) {
	return s.bitwiseBinary(other, "bitwise_xor",
		func(a, b int64) int64 { return a ^ b },
		func(a, b bool) bool { return a != b })
}

func (s seriesFacade) int64UnaryBits(fn func(int64) int64) (Series, error) {
	if err := requireInt64Series("bitwise_op", s); err != nil {
		return nil, err
	}
	values, err := mapValues(s, "bitwise_op", "int64", func(v int64) any { return fn(v) })
	if err != nil {
		return nil, err
	}
	return NewSeries(NewSeriesInput{Name: s.Name(), DType: dtypes.Int64, Values: values})
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
