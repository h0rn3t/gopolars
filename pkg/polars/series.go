package polars

import (
	"fmt"
	"math"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	"github.com/eugeneshershen/gopolars/pkg/frame"
	iseries "github.com/eugeneshershen/gopolars/pkg/series"
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
	count := 0
	for i := 0; i < s.value.Len(); i++ {
		if s.value.IsNull(i) {
			count++
		}
	}
	return count
}

func (s seriesFacade) IsNull() Series {
	values := make([]any, s.value.Len())
	for i := 0; i < s.value.Len(); i++ {
		values[i] = s.value.IsNull(i)
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Boolean, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) IsNotNull() Series {
	values := make([]any, s.value.Len())
	for i := 0; i < s.value.Len(); i++ {
		values[i] = !s.value.IsNull(i)
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Boolean, values)
	return seriesFacade{value: out}
}

func (s seriesFacade) FillNull(value any) (Series, error) {
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
	acc := 0.0
	for i := 0; i < s.Len(); i++ {
		if v, ok := toFloat64(s.Value(i)); ok {
			acc += v
		}
	}
	return acc
}

func (s seriesFacade) Std() float64 {
	vals := make([]float64, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if v, ok := toFloat64(s.Value(i)); ok {
			vals = append(vals, v)
		}
	}
	if len(vals) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))
	sumSq := 0.0
	for _, v := range vals {
		d := v - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(vals)-1))
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

func (s seriesFacade) rolling(window int, mode string) Series {
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
			switch t := s.value.Value(j).(type) {
			case int64:
				nums = append(nums, float64(t))
			case float64:
				if !math.IsNaN(t) {
					nums = append(nums, t)
				}
			}
		}
		if len(nums) == 0 {
			values[i] = nil
			continue
		}
		switch mode {
		case "sum":
			acc := float64(0)
			for _, n := range nums {
				acc += n
			}
			values[i] = acc
		case "mean":
			acc := float64(0)
			for _, n := range nums {
				acc += n
			}
			values[i] = acc / float64(len(nums))
		case "min":
			best := nums[0]
			for _, n := range nums[1:] {
				if n < best {
					best = n
				}
			}
			values[i] = best
		case "max":
			best := nums[0]
			for _, n := range nums[1:] {
				if n > best {
					best = n
				}
			}
			values[i] = best
		}
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
		}
	}
	out, _ := iseries.New(s.value.Name(), dtypes.Float64, values)
	return seriesFacade{value: out}
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
