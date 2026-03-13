package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/series"
)

type ReadInput struct {
	Path      string
	HasHeader bool
	Separator rune
	Schema    dtypes.Schema
	Columns   []string
}

type WriteInput struct {
	Path          string
	IncludeHeader bool
	Separator     rune
}

func Read(input ReadInput) (frame.DataFrame, error) {
	f, err := os.Open(input.Path)
	if err != nil {
		return frame.DataFrame{}, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	if input.Separator != 0 {
		r.Comma = input.Separator
	}
	rows, err := r.ReadAll()
	if err != nil {
		return frame.DataFrame{}, err
	}
	if len(rows) == 0 {
		return frame.New(frame.NewInput{})
	}
	header := make([]string, len(rows[0]))
	start := 0
	if input.HasHeader {
		copy(header, rows[0])
		start = 1
	} else {
		for i := range header {
			header[i] = fmt.Sprintf("column_%d", i+1)
		}
	}
	colValues := make([][]string, len(header))
	for i := start; i < len(rows); i++ {
		for j := range header {
			if j < len(rows[i]) {
				colValues[j] = append(colValues[j], rows[i][j])
			} else {
				colValues[j] = append(colValues[j], "")
			}
		}
	}
	selected := map[string]struct{}{}
	for _, c := range input.Columns {
		selected[c] = struct{}{}
	}
	sr := make([]series.Series, 0, len(header))
	for i, name := range header {
		if len(selected) > 0 {
			if _, ok := selected[name]; !ok {
				continue
			}
		}
		values, dt := inferColumn(colValues[i], input.Schema, name)
		s, err := series.New(name, dt, values)
		if err != nil {
			return frame.DataFrame{}, err
		}
		sr = append(sr, s)
	}
	return frame.New(frame.NewInput{Series: sr})
}

func Write(df frame.DataFrame, input WriteInput) error {
	f, err := os.Create(input.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if input.Separator != 0 {
		w.Comma = input.Separator
	}
	if input.IncludeHeader {
		if err := w.Write(df.Columns()); err != nil {
			return err
		}
	}
	for row := 0; row < df.Height(); row++ {
		out := make([]string, 0, df.Width())
		for _, name := range df.Columns() {
			s, _ := df.Series(name)
			v := s.Value(row)
			if v == nil {
				out = append(out, "")
				continue
			}
			switch t := v.(type) {
			case time.Time:
				out = append(out, t.Format(time.RFC3339Nano))
			default:
				out = append(out, fmt.Sprintf("%v", t))
			}
		}
		if err := w.Write(out); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func inferColumn(values []string, schema dtypes.Schema, name string) ([]any, dtypes.DataType) {
	if idx := schema.IndexOf(name); idx >= 0 {
		return parseWithType(values, schema[idx].Type), schema[idx].Type
	}
	isInt := true
	isFloat := true
	isBool := true
	isTime := true
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, err := strconv.ParseInt(v, 10, 64); err != nil {
			isInt = false
		}
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			isFloat = false
		}
		if _, err := strconv.ParseBool(v); err != nil {
			isBool = false
		}
		if _, err := time.Parse(time.RFC3339, v); err != nil {
			isTime = false
		}
	}
	switch {
	case isInt:
		return parseWithType(values, dtypes.Int64), dtypes.Int64
	case isFloat:
		return parseWithType(values, dtypes.Float64), dtypes.Float64
	case isBool:
		return parseWithType(values, dtypes.Boolean), dtypes.Boolean
	case isTime:
		return parseWithType(values, dtypes.Datetime), dtypes.Datetime
	default:
		return parseWithType(values, dtypes.String), dtypes.String
	}
}

func parseWithType(values []string, dt dtypes.DataType) []any {
	out := make([]any, len(values))
	for i, v := range values {
		if v == "" {
			out[i] = nil
			continue
		}
		switch dt {
		case dtypes.Int64:
			n, _ := strconv.ParseInt(v, 10, 64)
			out[i] = n
		case dtypes.Float64:
			n, _ := strconv.ParseFloat(v, 64)
			out[i] = n
		case dtypes.Boolean:
			n, _ := strconv.ParseBool(v)
			out[i] = n
		case dtypes.Datetime:
			n, _ := time.Parse(time.RFC3339, v)
			out[i] = n
		default:
			out[i] = v
		}
	}
	return out
}
