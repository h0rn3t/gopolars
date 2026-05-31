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

// Lower нижній регістр рядків (string dtype).
func (n SeriesStrNS) Lower() (Series, error) {
	if err := requireDType(n.s, dtypes.String, "str"); err != nil {
		return nil, err
	}
	values := make([]any, n.s.Len())
	for i := 0; i < n.s.Len(); i++ {
		v := n.s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("str.Lower: value at %d is not string", i)
		}
		values[i] = strings.ToLower(str)
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name(), DType: dtypes.String, Values: values})
}

// Upper верхній регістр рядків.
func (n SeriesStrNS) Upper() (Series, error) {
	if err := requireDType(n.s, dtypes.String, "str"); err != nil {
		return nil, err
	}
	values := make([]any, n.s.Len())
	for i := 0; i < n.s.Len(); i++ {
		v := n.s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("str.Upper: value at %d is not string", i)
		}
		values[i] = strings.ToUpper(str)
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name(), DType: dtypes.String, Values: values})
}

// Len довжина рядка (string dtype).
func (n SeriesStrNS) Len() (Series, error) {
	if err := requireDType(n.s, dtypes.String, "str"); err != nil {
		return nil, err
	}
	values := make([]any, n.s.Len())
	for i := 0; i < n.s.Len(); i++ {
		v := n.s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("str.Len: value at %d is not string", i)
		}
		values[i] = int64(len(str))
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name() + "_str_len", DType: dtypes.Int64, Values: values})
}

// ListLen довжина списку в кожному рядку (list dtype).
func (n SeriesArrNS) ListLen() (Series, error) {
	if err := requireDType(n.s, dtypes.List, "arr"); err != nil {
		return nil, err
	}
	values := make([]any, n.s.Len())
	for i := 0; i < n.s.Len(); i++ {
		v := n.s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		list, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("arr.ListLen: value at %d is not list", i)
		}
		values[i] = int64(len(list))
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name() + "_list_len", DType: dtypes.Int64, Values: values})
}

// Year витягує рік з datetime.
func (n SeriesDtNS) Year() (Series, error) {
	if err := requireDType(n.s, dtypes.Datetime, "dt"); err != nil {
		return nil, err
	}
	values := make([]any, n.s.Len())
	for i := 0; i < n.s.Len(); i++ {
		v := n.s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		t, ok := v.(time.Time)
		if !ok {
			return nil, fmt.Errorf("dt.Year: value at %d is not time.Time", i)
		}
		values[i] = int64(t.Year())
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
	values := make([]any, n.s.Len())
	for i := 0; i < n.s.Len(); i++ {
		v := n.s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("struct.Field: value at %d is not map[string]any", i)
		}
		inner := m[name]
		switch t := inner.(type) {
		case nil:
			values[i] = nil
		case string:
			values[i] = t
		case int, int32, int64, float32, float64, bool:
			values[i] = fmt.Sprint(t)
		default:
			values[i] = fmt.Sprint(t)
		}
	}
	return NewSeries(NewSeriesInput{Name: n.s.Name() + "_" + name, DType: dtypes.String, Values: values})
}

// Codes повертає цілочисельні коди категорій у порядку першої появи рядка.
func (n SeriesCatNS) Codes() (Series, error) {
	if err := requireDType(n.s, dtypes.Categorical, "cat"); err != nil {
		return nil, err
	}
	next := 0
	keyToCode := map[string]int64{}
	values := make([]any, n.s.Len())
	for i := 0; i < n.s.Len(); i++ {
		v := n.s.Value(i)
		if v == nil {
			values[i] = nil
			continue
		}
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("cat.Codes: value at %d is not string", i)
		}
		code, ok := keyToCode[str]
		if !ok {
			code = int64(next)
			next++
			keyToCode[str] = code
		}
		values[i] = code
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
