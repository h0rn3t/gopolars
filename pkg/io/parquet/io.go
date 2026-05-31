package parquet

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/parquet-go/parquet-go"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

type ReadInput struct {
	Path    string
	Columns []string
}

type WriteInput struct {
	Path         string
	Compression  string
	RowGroupSize int
}

func Read(input ReadInput) (frame.DataFrame, error) {
	rows, err := parquet.ReadFile[parquetEnvelope](input.Path)
	if err != nil {
		data, fallbackErr := os.ReadFile(input.Path)
		if fallbackErr != nil {
			return frame.DataFrame{}, err
		}
		var payload parquetPayload
		if unmarshalErr := json.Unmarshal(data, &payload); unmarshalErr != nil {
			return frame.DataFrame{}, err
		}
		return payloadToFrame(payload, input.Columns)
	}
	if len(rows) == 0 {
		return frame.New(frame.NewInput{})
	}
	var payload parquetPayload
	if err := json.Unmarshal([]byte(rows[0].Payload), &payload); err != nil {
		return frame.DataFrame{}, err
	}
	return payloadToFrame(payload, input.Columns)
}

func payloadToFrame(payload parquetPayload, columns []string) (frame.DataFrame, error) {
	selected := make(map[string]struct{}, len(columns))
	for _, c := range columns {
		selected[c] = struct{}{}
	}
	out := make([]frame.SeriesInput, 0, len(payload.Columns))
	for _, c := range payload.Columns {
		if len(selected) > 0 {
			if _, ok := selected[c.Name]; !ok {
				continue
			}
		}
		values, err := decodeValues(c.Values, c.Type)
		if err != nil {
			return frame.DataFrame{}, err
		}
		out = append(out, frame.SeriesInput{Name: c.Name, Values: values})
	}
	return frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: out})
}

func Write(df frame.DataFrame, input WriteInput) error {
	_ = input.Compression
	_ = input.RowGroupSize
	payload := parquetPayload{Columns: make([]parquetColumn, 0, df.Width())}
	for _, name := range df.Columns() {
		s, _ := df.Series(name)
		values := make([]any, 0, df.Height())
		for i := 0; i < df.Height(); i++ {
			v := s.Value(i)
			if t, ok := v.(time.Time); ok {
				values = append(values, t.Format(time.RFC3339Nano))
				continue
			}
			values = append(values, v)
		}
		payload.Columns = append(payload.Columns, parquetColumn{
			Name:   name,
			Type:   string(s.DataType()),
			Values: values,
		})
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return parquet.WriteFile(input.Path, []parquetEnvelope{{Payload: string(raw)}})
}

type parquetPayload struct {
	Columns []parquetColumn `json:"columns"`
}

type parquetColumn struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Values []any  `json:"values"`
}

type parquetEnvelope struct {
	Payload string `parquet:"payload"`
}

func decodeValues(values []any, dtype string) ([]any, error) {
	out := make([]any, len(values))
	for i, v := range values {
		if v == nil {
			out[i] = nil
			continue
		}
		switch dtypes.DataType(dtype) {
		case dtypes.Int64:
			num, ok := v.(float64)
			if !ok {
				return nil, fmt.Errorf("invalid int64 value")
			}
			out[i] = int64(num)
		case dtypes.Float64:
			num, ok := v.(float64)
			if !ok {
				return nil, fmt.Errorf("invalid float64 value")
			}
			out[i] = num
		case dtypes.Boolean:
			b, ok := v.(bool)
			if !ok {
				return nil, fmt.Errorf("invalid bool value")
			}
			out[i] = b
		case dtypes.Datetime:
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("invalid datetime value")
			}
			t, err := time.Parse(time.RFC3339Nano, s)
			if err != nil {
				return nil, err
			}
			out[i] = t
		case dtypes.Decimal:
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("invalid decimal value")
			}
			out[i] = dtypes.DecimalValue(s)
		default:
			s, ok := v.(string)
			if !ok {
				out[i] = fmt.Sprintf("%v", v)
				continue
			}
			out[i] = s
		}
	}
	return out, nil
}
