package series

import (
	"fmt"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
)

type Series struct {
	name  string
	dtype dtypes.DataType
	data  []any
	nulls []bool
}

func New(name string, dtype dtypes.DataType, values []any) (Series, error) {
	s := Series{
		name:  name,
		dtype: dtype,
		data:  make([]any, len(values)),
		nulls: make([]bool, len(values)),
	}
	for i, v := range values {
		if v == nil {
			s.nulls[i] = true
			continue
		}
		if err := validateType(dtype, v); err != nil {
			return Series{}, fmt.Errorf("invalid value for series %s: %w", name, err)
		}
		s.data[i] = v
	}
	return s, nil
}

func (s Series) Name() string {
	return s.name
}

func (s Series) DataType() dtypes.DataType {
	return s.dtype
}

func (s Series) Len() int {
	return len(s.data)
}

func (s Series) IsNull(i int) bool {
	return s.nulls[i]
}

func (s Series) Value(i int) any {
	if s.nulls[i] {
		return nil
	}
	return s.data[i]
}

func (s Series) Clone() Series {
	data := make([]any, len(s.data))
	copy(data, s.data)
	nulls := make([]bool, len(s.nulls))
	copy(nulls, s.nulls)
	return Series{name: s.name, dtype: s.dtype, data: data, nulls: nulls}
}

func (s Series) Rename(name string) Series {
	c := s.Clone()
	c.name = name
	return c
}

func (s Series) Slice(indexes []int) Series {
	out := Series{
		name:  s.name,
		dtype: s.dtype,
		data:  make([]any, 0, len(indexes)),
		nulls: make([]bool, 0, len(indexes)),
	}
	for _, idx := range indexes {
		out.data = append(out.data, s.data[idx])
		out.nulls = append(out.nulls, s.nulls[idx])
	}
	return out
}

func validateType(dt dtypes.DataType, v any) error {
	switch dt {
	case dtypes.Int64:
		if _, ok := v.(int64); !ok {
			return fmt.Errorf("expected int64")
		}
	case dtypes.Float64:
		if _, ok := v.(float64); !ok {
			return fmt.Errorf("expected float64")
		}
	case dtypes.Decimal:
		if _, ok := v.(dtypes.DecimalValue); ok {
			return nil
		}
		if _, ok := v.(string); ok {
			return nil
		}
		return fmt.Errorf("expected decimal value")
	case dtypes.String:
		if _, ok := v.(string); !ok {
			return fmt.Errorf("expected string")
		}
	case dtypes.Categorical, dtypes.Enum:
		if _, ok := v.(string); !ok {
			return fmt.Errorf("expected string")
		}
	case dtypes.Boolean:
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("expected bool")
		}
	case dtypes.Datetime:
		if _, ok := v.(time.Time); !ok {
			return fmt.Errorf("expected time.Time")
		}
	case dtypes.List:
		if _, ok := v.([]any); !ok {
			return fmt.Errorf("expected []any")
		}
	case dtypes.Struct:
		if _, ok := v.(map[string]any); !ok {
			return fmt.Errorf("expected map[string]any")
		}
	default:
		return fmt.Errorf("unsupported data type %s", dt)
	}
	return nil
}
