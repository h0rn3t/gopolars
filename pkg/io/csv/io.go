package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

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

// writeBlockSize is the output accumulation size: rows are appended into one
// reused buffer and handed to the file in blocks this large, so a 1M-row write
// costs a few hundred write syscalls rather than one per row.
const writeBlockSize = 1 << 16 // 64 KiB

func Write(df frame.DataFrame, input WriteInput) error {
	comma := ','
	if input.Separator != 0 {
		comma = input.Separator
	}
	// Validate the delimiter up front, as encoding/csv did for us before.
	if comma == 0 || comma == '"' || comma == '\r' || comma == '\n' ||
		comma == utf8.RuneError || !utf8.ValidRune(comma) {
		return fmt.Errorf("write csv: invalid separator %q", comma)
	}

	f, err := os.Create(input.Path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	names := df.Columns()
	// Resolve each column's typed backing once (not once per cell) and bind a
	// type-specialized appender, so the inner loop avoids per-cell Series lookups,
	// interface boxing, reflection, and — unlike a formatter returning string —
	// any per-cell allocation.
	appenders := make([]func(dst []byte, row int) []byte, len(names))
	for j, name := range names {
		s, _ := df.Series(name)
		appenders[j] = cellAppender(s, comma)
	}

	buf := make([]byte, 0, writeBlockSize+4096)
	if input.IncludeHeader {
		for j, name := range names {
			if j > 0 {
				buf = utf8.AppendRune(buf, comma)
			}
			buf = appendCSVField(buf, []byte(name), comma)
		}
		buf = append(buf, '\n')
	}

	h := df.Height()
	if workers := runtime.GOMAXPROCS(0); workers > 1 && h >= parallelWriteThreshold {
		return writeRowsParallel(f, df, names, comma, buf, h, workers)
	}

	for row := 0; row < h; row++ {
		for j := range appenders {
			if j > 0 {
				buf = utf8.AppendRune(buf, comma)
			}
			buf = appenders[j](buf, row)
		}
		buf = append(buf, '\n')
		if len(buf) >= writeBlockSize {
			if _, err := f.Write(buf); err != nil {
				return err
			}
			buf = buf[:0]
		}
	}
	if len(buf) > 0 {
		if _, err := f.Write(buf); err != nil {
			return err
		}
	}
	return nil
}

// parallelWriteThreshold is the row count above which formatting is spread across
// workers. Below it the goroutine coordination outweighs the gain.
const parallelWriteThreshold = 1 << 15 // 32768 rows

// csvChunkRows is how many rows one worker formats per turn. It bounds peak
// memory to workers x csvChunkRows x row width rather than the whole output.
const csvChunkRows = 1 << 14 // 16384 rows

// writeRowsParallel formats row ranges concurrently and writes the resulting
// blocks in row order, so the output is byte-identical to the sequential path.
//
// Rows are independent — each cell reads only its own column slot — so the only
// shared state is the read-only column set. Each worker owns its own appenders
// (whose scratch buffers must not be shared) and its own output buffer, reused
// across waves. pending carries any bytes already staged by the caller (the
// header) so it is emitted before the first chunk.
func writeRowsParallel(f *os.File, df frame.DataFrame, names []string, comma rune, pending []byte, h, workers int) error {
	cols := make([]series.Series, len(names))
	for j, name := range names {
		cols[j], _ = df.Series(name)
	}
	perWorker := make([][]func(dst []byte, row int) []byte, workers)
	bufs := make([][]byte, workers)
	for w := range perWorker {
		apps := make([]func(dst []byte, row int) []byte, len(cols))
		for j := range cols {
			apps[j] = cellAppender(cols[j], comma)
		}
		perWorker[w] = apps
		// Size for a whole chunk up front (~32 B/row is typical for a handful of
		// numeric/short-string columns): the buffer is reused across waves, so this
		// trades one allocation for the repeated doubling a small cap would cause.
		bufs[w] = make([]byte, 0, csvChunkRows*32)
	}

	if len(pending) > 0 {
		if _, err := f.Write(pending); err != nil {
			return err
		}
	}

	stride := workers * csvChunkRows
	for start := 0; start < h; start += stride {
		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			lo := start + w*csvChunkRows
			if lo >= h {
				bufs[w] = bufs[w][:0]
				continue
			}
			hi := min(lo+csvChunkRows, h)
			wg.Add(1)
			go func(w, lo, hi int) {
				defer wg.Done()
				apps := perWorker[w]
				buf := bufs[w][:0]
				for row := lo; row < hi; row++ {
					for j := range apps {
						if j > 0 {
							buf = utf8.AppendRune(buf, comma)
						}
						buf = apps[j](buf, row)
					}
					buf = append(buf, '\n')
				}
				bufs[w] = buf
			}(w, lo, hi)
		}
		wg.Wait()
		// Write the wave's blocks in row order.
		for w := 0; w < workers; w++ {
			if len(bufs[w]) == 0 {
				continue
			}
			if _, err := f.Write(bufs[w]); err != nil {
				return err
			}
		}
	}
	return nil
}

// appendCSVField appends field to dst with the same quoting encoding/csv's
// Writer produces (UseCRLF disabled): a field is quoted when it is exactly `\.`,
// when it contains the delimiter, a double quote, CR or LF, or when it starts
// with a space — and with the delimiter set to a space, when it contains any
// space at all. Inside quotes, each double quote is doubled.
func appendCSVField(dst, field []byte, comma rune) []byte {
	if !fieldNeedsQuotes(field, comma) {
		return append(dst, field...)
	}
	dst = append(dst, '"')
	for {
		i := bytes.IndexByte(field, '"')
		if i < 0 {
			break
		}
		dst = append(dst, field[:i]...)
		dst = append(dst, '"', '"')
		field = field[i+1:]
	}
	dst = append(dst, field...)
	return append(dst, '"')
}

// fieldNeedsQuotes mirrors encoding/csv's (*Writer).fieldNeedsQuotes exactly so
// the output stays byte-identical to the previous writer. Kept structurally
// identical to the upstream function, including the ASCII-delimiter byte scan,
// so a future divergence is easy to spot.
func fieldNeedsQuotes(field []byte, comma rune) bool {
	if len(field) == 0 {
		return false
	}
	if string(field) == `\.` {
		return true
	}
	if comma < utf8.RuneSelf {
		for i := 0; i < len(field); i++ {
			c := field[i]
			if c == '\n' || c == '\r' || c == '"' || c == byte(comma) {
				return true
			}
		}
	} else {
		if bytes.ContainsRune(field, comma) || bytes.ContainsAny(field, "\"\r\n") {
			return true
		}
	}
	r1, _ := utf8.DecodeRune(field)
	return unicode.IsSpace(r1)
}

// cellAppender returns a per-row appender for a single column, bound once to the
// column's typed backing slice. Numeric/bool/string/time columns append via
// strconv.Append* / time.AppendFormat (no fmt.Sprintf, no per-cell boxing, and
// no per-cell string); other dtypes (e.g. Decimal) fall back to the boxed Value
// path, preserving the previous output exactly. Null cells render as an empty
// field.
//
// Numeric and time cells are staged through a scratch buffer owned by the
// closure so they can go through the same quoting check as strings: a delimiter
// may legally be a digit, a minus or a colon, in which case even a number needs
// quoting to stay byte-identical to encoding/csv.
func cellAppender(s series.Series, comma rune) func(dst []byte, row int) []byte {
	col := s.Column()
	nulls := col.Nulls()
	null := func(i int) bool { return nulls != nil && nulls[i] }
	var scratch []byte

	if v, ok := col.Int64s(); ok {
		return func(dst []byte, i int) []byte {
			if null(i) {
				return dst
			}
			scratch = strconv.AppendInt(scratch[:0], v[i], 10)
			return appendCSVField(dst, scratch, comma)
		}
	}
	if v, ok := col.Float64s(); ok {
		return func(dst []byte, i int) []byte {
			if null(i) {
				return dst
			}
			scratch = strconv.AppendFloat(scratch[:0], v[i], 'g', -1, 64)
			return appendCSVField(dst, scratch, comma)
		}
	}
	if v, ok := col.Bools(); ok {
		return func(dst []byte, i int) []byte {
			if null(i) {
				return dst
			}
			scratch = strconv.AppendBool(scratch[:0], v[i])
			return appendCSVField(dst, scratch, comma)
		}
	}
	if v, ok := col.Strings(); ok {
		return func(dst []byte, i int) []byte {
			if null(i) {
				return dst
			}
			// unsafe-free: appendCSVField only reads the bytes it is handed.
			scratch = append(scratch[:0], v[i]...)
			return appendCSVField(dst, scratch, comma)
		}
	}
	if v, ok := col.Times(); ok {
		return func(dst []byte, i int) []byte {
			if null(i) {
				return dst
			}
			scratch = v[i].AppendFormat(scratch[:0], time.RFC3339Nano)
			return appendCSVField(dst, scratch, comma)
		}
	}
	// Boxed fallback (Decimal and any non-primitive dtype): identical output to
	// the previous row-at-a-time writer.
	return func(dst []byte, i int) []byte {
		v := s.Value(i)
		if v == nil {
			return dst
		}
		var text string
		if t, ok := v.(time.Time); ok {
			text = t.Format(time.RFC3339Nano)
		} else {
			text = fmt.Sprintf("%v", v)
		}
		scratch = append(scratch[:0], text...)
		return appendCSVField(dst, scratch, comma)
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
