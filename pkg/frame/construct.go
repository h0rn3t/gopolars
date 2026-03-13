package frame

import (
	"fmt"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	"github.com/eugeneshershen/gopolars/pkg/series"
)

type SeriesInput struct {
	Name   string
	Values []any
}

type FromAnyColumnsInput struct {
	Columns []SeriesInput
}

func FromAnyColumns(input FromAnyColumnsInput) (DataFrame, error) {
	out := make([]series.Series, 0, len(input.Columns))
	for _, c := range input.Columns {
		dt, err := inferAnyDataType(c.Values)
		if err != nil {
			return DataFrame{}, err
		}
		s, err := series.New(c.Name, dt, c.Values)
		if err != nil {
			return DataFrame{}, err
		}
		out = append(out, s)
	}
	return New(NewInput{Series: out})
}

func inferAnyDataType(values []any) (dtypes.DataType, error) {
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
