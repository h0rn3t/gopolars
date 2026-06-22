package csv

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/series"
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
	defer func() { _ = f.Close() }()
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
	defer func() { _ = f.Close() }()
	bw := bufio.NewWriter(f)
	w := csv.NewWriter(bw)
	if input.Separator != 0 {
		w.Comma = input.Separator
	}
	names := df.Columns()
	if input.IncludeHeader {
		if err := w.Write(names); err != nil {
			return err
		}
	}
	// Resolve each column's typed backing once (not once per cell) and build a
	// type-specialized formatter, so the inner loop avoids per-cell Series
	// lookups, interface boxing, and fmt.Sprintf reflection.
	format := make([]func(i int) string, len(names))
	for j, name := range names {
		s, _ := df.Series(name)
		format[j] = cellFormatter(s)
	}
	record := make([]string, len(names))
	for row, h := 0, df.Height(); row < h; row++ {
		for j := range format {
			record[j] = format[j](row)
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return bw.Flush()
}

// cellFormatter returns a per-row string formatter for a single column, bound
// once to the column's typed backing slice. Numeric/bool/string/time columns
// format via strconv / time.Format (no fmt.Sprintf, no per-cell boxing); other
// dtypes (e.g. Decimal) fall back to the boxed Value path, preserving the
// previous output exactly. Null cells render as an empty field.
func cellFormatter(s series.Series) func(i int) string {
	col := s.Column()
	nulls := col.Nulls()
	null := func(i int) bool { return nulls != nil && nulls[i] }

	if v, ok := col.Int64s(); ok {
		return func(i int) string {
			if null(i) {
				return ""
			}
			return strconv.FormatInt(v[i], 10)
		}
	}
	if v, ok := col.Float64s(); ok {
		return func(i int) string {
			if null(i) {
				return ""
			}
			return strconv.FormatFloat(v[i], 'g', -1, 64)
		}
	}
	if v, ok := col.Bools(); ok {
		return func(i int) string {
			if null(i) {
				return ""
			}
			return strconv.FormatBool(v[i])
		}
	}
	if v, ok := col.Strings(); ok {
		return func(i int) string {
			if null(i) {
				return ""
			}
			return v[i]
		}
	}
	if v, ok := col.Times(); ok {
		return func(i int) string {
			if null(i) {
				return ""
			}
			return v[i].Format(time.RFC3339Nano)
		}
	}
	// Boxed fallback (Decimal and any non-primitive dtype): identical to the
	// previous row-at-a-time writer.
	return func(i int) string {
		v := s.Value(i)
		if v == nil {
			return ""
		}
		if t, ok := v.(time.Time); ok {
			return t.Format(time.RFC3339Nano)
		}
		return fmt.Sprintf("%v", v)
	}
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
