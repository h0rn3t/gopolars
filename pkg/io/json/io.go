package json

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
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

// Read reads a JSON array of objects, or with NDJSON one object per line, into
// a DataFrame with one column per key found in any record, in the order keys
// first appear. A record without a key is null in that column; Columns keeps
// only the named keys without reordering them.
func Read(input ReadInput) (frame.DataFrame, error) {
	f, err := os.Open(input.Path)
	if err != nil {
		return frame.DataFrame{}, err
	}
	defer func() { _ = f.Close() }()
	rows, keys, err := readRecords(json.NewDecoder(f), input.NDJSON)
	if errors.Is(err, io.EOF) {
		// Between records the decoder stops at EOF without an error, so an EOF
		// here means the input ended inside a record or the array.
		err = io.ErrUnexpectedEOF
	}
	if err != nil {
		return frame.DataFrame{}, err
	}
	return fromRows(rows, keys, input.Schema, input.Columns)
}

func Write(df frame.DataFrame, input WriteInput) (err error) {
	rows := toRows(df)
	f, err := os.Create(input.Path)
	if err != nil {
		return err
	}
	// A failed close can lose buffered data, so it is reported when nothing
	// failed before it.
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
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

// readRecords decodes a JSON array of objects, or with ndjson a stream of
// objects, and returns the records with their keys in order of first
// appearance across all records. A null record has no keys.
func readRecords(dec *json.Decoder, ndjson bool) ([]map[string]any, []string, error) {
	var rows []map[string]any
	var keys []string
	seen := map[string]bool{}
	if !ndjson {
		tok, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		if tok != json.Delim('[') {
			return nil, nil, fmt.Errorf("json: document starts with %v, want an array of objects", tok)
		}
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		row := map[string]any{}
		rows = append(rows, row)
		if tok == nil {
			continue
		}
		if tok != json.Delim('{') {
			return nil, nil, fmt.Errorf("json: record %d is %v, want an object", len(rows)-1, tok)
		}
		for dec.More() {
			tok, err := dec.Token()
			if err != nil {
				return nil, nil, err
			}
			key := tok.(string) // inside an object Token returns each key as a string
			var value any
			if err := dec.Decode(&value); err != nil {
				return nil, nil, err
			}
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
			row[key] = value
		}
		if _, err := dec.Token(); err != nil { // the record's '}'
			return nil, nil, err
		}
	}
	if !ndjson {
		if _, err := dec.Token(); err != nil { // the array's ']'
			return nil, nil, err
		}
	}
	// Like json.Unmarshal of the whole input, nothing may follow the records.
	tok, err := dec.Token()
	if err == nil {
		return nil, nil, fmt.Errorf("json: unexpected %v after the records", tok)
	}
	if !errors.Is(err, io.EOF) {
		return nil, nil, err
	}
	return rows, keys, nil
}

// fromRows builds one column per key, in the order of keys; a record without
// the key contributes a null.
func fromRows(rows []map[string]any, keys []string, schema dtypes.Schema, columns []string) (frame.DataFrame, error) {
	if len(rows) == 0 {
		return frame.New(frame.NewInput{})
	}
	if len(columns) > 0 {
		keys = slices.DeleteFunc(keys, func(c string) bool { return !slices.Contains(columns, c) })
	}
	out := make([]series.Series, 0, len(keys))
	for _, c := range keys {
		values := make([]any, len(rows))
		for i, row := range rows {
			values[i] = normalizeValue(row[c])
		}
		s, err := series.New(c, inferType(values, schema, c), values)
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
