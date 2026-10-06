package frame

import (
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/series"
)

type SeriesInput struct {
	Name   string
	Values []any
	// DType optionally pins the column dtype. When empty, the dtype is inferred
	// from Values; an explicit DType lets all-null or empty columns be typed
	// (the inference path cannot determine a dtype with no non-null values).
	DType dtypes.DataType
}

type FromAnyColumnsInput struct {
	Columns []SeriesInput
}

func FromAnyColumns(input FromAnyColumnsInput) (DataFrame, error) {
	out := make([]series.Series, 0, len(input.Columns))
	for _, c := range input.Columns {
		dt := c.DType
		if dt == "" {
			var err error
			dt, err = inferDataType(c.Values)
			if err != nil {
				return DataFrame{}, err
			}
		}
		s, err := series.New(c.Name, dt, c.Values)
		if err != nil {
			return DataFrame{}, err
		}
		out = append(out, s)
	}
	return New(NewInput{Series: out})
}
