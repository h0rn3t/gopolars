package json

import (
	"bufio"
	"encoding/json"
	"os"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/series"
)

type ReadInput struct {
	Path    string
	NDJSON  bool
	Schema  dtypes.Schema
	Columns []string
}

type WriteInput struct {
	Path   string
	NDJSON bool
	Pretty bool
}

func Read(input ReadInput) (frame.DataFrame, error) {
	if input.NDJSON {
		return readNDJSON(input)
	}
	data, err := os.ReadFile(input.Path)
	if err != nil {
		return frame.DataFrame{}, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return frame.DataFrame{}, err
	}
	return fromRows(rows, input.Schema, input.Columns)
}

func Write(df frame.DataFrame, input WriteInput) error {
	rows := toRows(df)
	f, err := os.Create(input.Path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if input.NDJSON {
		w := bufio.NewWriter(f)
		for _, row := range rows {
			raw, err := json.Marshal(row)
			if err != nil {
				return err
			}
			if _, err := w.Write(append(raw, '\n')); err != nil {
				return err
			}
		}
		return w.Flush()
	}
	var raw []byte
	if input.Pretty {
		raw, err = json.MarshalIndent(rows, "", "  ")
	} else {
		raw, err = json.Marshal(rows)
	}
	if err != nil {
		return err
	}
	_, err = f.Write(raw)
	return err
}

func readNDJSON(input ReadInput) (frame.DataFrame, error) {
	f, err := os.Open(input.Path)
	if err != nil {
		return frame.DataFrame{}, err
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	rows := make([]map[string]any, 0)
	for scanner.Scan() {
		line := scanner.Bytes()
		var row map[string]any
		if err := json.Unmarshal(line, &row); err != nil {
			return frame.DataFrame{}, err
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return frame.DataFrame{}, err
	}
	return fromRows(rows, input.Schema, input.Columns)
}

func fromRows(rows []map[string]any, schema dtypes.Schema, columns []string) (frame.DataFrame, error) {
	if len(rows) == 0 {
		return frame.New(frame.NewInput{})
	}
	order := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		order = append(order, k)
	}
	selected := map[string]struct{}{}
	for _, c := range columns {
		selected[c] = struct{}{}
	}
	valuesByCol := map[string][]any{}
	for _, c := range order {
		if len(selected) > 0 {
			if _, ok := selected[c]; !ok {
				continue
			}
		}
		valuesByCol[c] = make([]any, 0, len(rows))
	}
	for _, row := range rows {
		for _, c := range order {
			if len(selected) > 0 {
				if _, ok := selected[c]; !ok {
					continue
				}
			}
			valuesByCol[c] = append(valuesByCol[c], normalizeValue(row[c]))
		}
	}
	out := make([]series.Series, 0, len(order))
	for _, c := range order {
		if len(selected) > 0 {
			if _, ok := selected[c]; !ok {
				continue
			}
		}
		dt := inferType(valuesByCol[c], schema, c)
		s, err := series.New(c, dt, valuesByCol[c])
		if err != nil {
			return frame.DataFrame{}, err
		}
		out = append(out, s)
	}
	return frame.New(frame.NewInput{Series: out})
}

func toRows(df frame.DataFrame) []map[string]any {
	rows := make([]map[string]any, 0, df.Height())
	cols := df.Columns()
	for i := 0; i < df.Height(); i++ {
		row := map[string]any{}
		for _, c := range cols {
			s, _ := df.Series(c)
			v := s.Value(i)
			if t, ok := v.(time.Time); ok {
				row[c] = t.Format(time.RFC3339Nano)
				continue
			}
			row[c] = v
		}
		rows = append(rows, row)
	}
	return rows
}

func normalizeValue(v any) any {
	switch t := v.(type) {
	case float64:
		if float64(int64(t)) == t {
			return int64(t)
		}
		return t
	default:
		return t
	}
}

func inferType(values []any, schema dtypes.Schema, name string) dtypes.DataType {
	if idx := schema.IndexOf(name); idx >= 0 {
		return schema[idx].Type
	}
	for _, v := range values {
		if v == nil {
			continue
		}
		switch v.(type) {
		case int64:
			return dtypes.Int64
		case float64:
			return dtypes.Float64
		case bool:
			return dtypes.Boolean
		case string:
			return dtypes.String
		}
	}
	return dtypes.String
}
