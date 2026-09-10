package polars

import (
	"fmt"
	"strings"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// SeriesStrNS — простір імен string для Series (аналог Python Series.str.*).
type SeriesStrNS struct{ s Series }

// SeriesArrNS — простір імен list/array для Series.
type SeriesArrNS struct{ s Series }

// SeriesDtNS — простір імен datetime для Series.
type SeriesDtNS struct{ s Series }

// SeriesStructNS — простір імен struct для Series.
type SeriesStructNS struct{ s Series }

// SeriesCatNS — простір імен categorical для Series.
type SeriesCatNS struct{ s Series }

// SeriesBinNS — простір імен binary/bitwise для Series.
type SeriesBinNS struct{ s Series }

func (s seriesFacade) Str() SeriesStrNS { return SeriesStrNS{s: s} }
func (s seriesFacade) Arr() SeriesArrNS { return SeriesArrNS{s: s} }
func (s seriesFacade) Dt() SeriesDtNS   { return SeriesDtNS{s: s} }
func (s seriesFacade) Struct() SeriesStructNS {
	return SeriesStructNS{s: s}
}
func (s seriesFacade) Cat() SeriesCatNS { return SeriesCatNS{s: s} }
func (s seriesFacade) Bin() SeriesBinNS { return SeriesBinNS{s: s} }

func requireDType(s Series, want dtypes.DataType, ns string) error {
	if s.DataType() != want {
		return fmt.Errorf("%s namespace: expected dtype %s, got %s", ns, want, s.DataType())
	}
	return nil
}

// mapValues applies fn to every non-null row of s, asserting the value to T
// first. Nulls stay null. op names the calling namespace method and want the
// expected type in the error a mistyped row raises.
func mapValues[T any](s Series, op string, want string, fn func(T) any) ([]any, error) {
	values := make([]any, s.Len())
	for i := range values {
		v := s.Value(i)
		if v == nil {
			continue
		}
		x, ok := v.(T)
		if !ok {
			return nil, fmt.Errorf("%s: value at %d is not %s", op, i, want)
		}
		values[i] = fn(x)
	}
	return values, nil
}

// Lower нижній регістр рядків (string dtype).
func (n SeriesStrNS) Lower() (Series, error) {
	if err := requireDType(n.s, dtypes.String, "str"); err != nil {
		return nil, err
	}
	values, err := mapValues(n.s, "str.Lower", "string", func(s string) any { return strings.ToLower(s) })
	if err != nil {
		return nil, err
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name(), DType: dtypes.String, Values: values})
}

// Upper верхній регістр рядків.
func (n SeriesStrNS) Upper() (Series, error) {
	if err := requireDType(n.s, dtypes.String, "str"); err != nil {
		return nil, err
	}
	values, err := mapValues(n.s, "str.Upper", "string", func(s string) any { return strings.ToUpper(s) })
	if err != nil {
		return nil, err
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name(), DType: dtypes.String, Values: values})
}

// Len довжина рядка (string dtype).
func (n SeriesStrNS) Len() (Series, error) {
	if err := requireDType(n.s, dtypes.String, "str"); err != nil {
		return nil, err
	}
	values, err := mapValues(n.s, "str.Len", "string", func(s string) any { return int64(len(s)) })
	if err != nil {
		return nil, err
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name() + "_str_len", DType: dtypes.Int64, Values: values})
}

// ListLen довжина списку в кожному рядку (list dtype).
func (n SeriesArrNS) ListLen() (Series, error) {
	if err := requireDType(n.s, dtypes.List, "arr"); err != nil {
		return nil, err
	}
	values, err := mapValues(n.s, "arr.ListLen", "list", func(l []any) any { return int64(len(l)) })
	if err != nil {
		return nil, err
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name() + "_list_len", DType: dtypes.Int64, Values: values})
}

// Year витягує рік з datetime.
func (n SeriesDtNS) Year() (Series, error) {
	if err := requireDType(n.s, dtypes.Datetime, "dt"); err != nil {
		return nil, err
	}
	values, err := mapValues(n.s, "dt.Year", "time.Time", func(t time.Time) any { return int64(t.Year()) })
	if err != nil {
		return nil, err
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name() + "_year", DType: dtypes.Int64, Values: values})
}

// Field витягує поле struct (struct dtype).
func (n SeriesStructNS) Field(name string) (Series, error) {
	if err := requireDType(n.s, dtypes.Struct, "struct"); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("struct.Field: empty field name")
	}
	values, err := mapValues(n.s, "struct.Field", "map[string]any", func(m map[string]any) any {
		// Preserve the field's native value/dtype (matching Polars) instead of
		// stringifying. Normalize narrow numeric types to the canonical int64/
		// float64 the series constructor expects.
		switch t := m[name].(type) {
		case int:
			return int64(t)
		case int32:
			return int64(t)
		case float32:
			return float64(t)
		default:
			return t
		}
	})
	if err != nil {
		return nil, err
	}
	dt := inferDataTypeFromValues(values, dtypes.String)
	return NewSeries(NewSeriesInput{Name: n.s.Name() + "_" + name, DType: dt, Values: values})
}

// Codes повертає цілочисельні коди категорій у порядку першої появи рядка.
func (n SeriesCatNS) Codes() (Series, error) {
	if err := requireDType(n.s, dtypes.Categorical, "cat"); err != nil {
		return nil, err
	}
	keyToCode := map[string]int64{}
	values, err := mapValues(n.s, "cat.Codes", "string", func(s string) any {
		code, ok := keyToCode[s]
		if !ok {
			code = int64(len(keyToCode))
			keyToCode[s] = code
		}
		return code
	})
	if err != nil {
		return nil, err
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name() + "_codes", DType: dtypes.Int64, Values: values})
}

// CountOnes кількість одиничних бітів для int64 (аналог bitwise_count_ones).
func (n SeriesBinNS) CountOnes() (Series, error) {
	if err := requireDType(n.s, dtypes.Int64, "bin"); err != nil {
		return nil, err
	}
	f, ok := n.s.(seriesFacade)
	if !ok {
		return nil, fmt.Errorf("bin.CountOnes: unsupported series implementation")
	}
	return f.BitwiseCountOnes()
}
