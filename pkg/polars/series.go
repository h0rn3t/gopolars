package polars

import (
	"fmt"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
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
