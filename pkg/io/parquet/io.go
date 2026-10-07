package parquet

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"time"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	aparquet "github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/compress"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	pqgo "github.com/parquet-go/parquet-go"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

// defaultRowGroupRows bounds a row group when the caller does not specify one.
const defaultRowGroupRows = 128 * 1024

type ReadInput struct {
	Path    string
	Columns []string
}

type WriteInput struct {
	Path         string
	Compression  string
	RowGroupSize int
}

// Write serializes a DataFrame as a standard columnar parquet file: one typed
// parquet column per DataFrame column, written from the typed Arrow arrays
// produced by the native frame<->Arrow bridge (no intermediate whole-frame JSON
// encoding). Compression and row-group size are honored. The output is a
// regular parquet file readable by external readers (pyarrow / Polars).
func Write(df frame.DataFrame, input WriteInput) error {
	rec, err := iarrow.ToArrowRecord(df)
	if err != nil {
		return err
	}
	defer rec.Release()

	rowGroup := int64(input.RowGroupSize)
	if rowGroup <= 0 {
		rowGroup = defaultRowGroupRows
	}
	// A string column the dictionary pays off for is dictionary-encoded here
	// instead of by arrow-go, whose encoder allocates twice for every value it
	// hashes; pqarrow writes a dictionary array's dictionary and indices as is.
	// Every column that is not lowCardinality has dictionary encoding turned
	// off: arrow-go dictionary-encodes every column by default and, for such
	// columns, builds a hash table only to fall back to plain encoding once the
	// dictionary outgrows its page.
	fields := rec.Schema().Fields()
	columns := make([]goarrow.Column, len(fields))
	var dictionaryOff []aparquet.WriterProperty
	for i, field := range fields {
		s, _ := df.Series(field.Name)
		col := s.Column()
		low := lowCardinality(col)
		if !low {
			dictionaryOff = append(dictionaryOff, aparquet.WithDictionaryFor(field.Name, false))
		}
		values, ok := col.Strings()
		if !ok || !low {
			columns[i] = goarrow.NewColumnFromArr(field, rec.Column(i))
			continue
		}
		data := dictionaryChunks(values, col.Nulls(), int(rowGroup))
		fields[i].Type = data.DataType()
		columns[i] = *goarrow.NewColumn(fields[i], data)
		data.Release()
	}
	tbl := array.NewTable(goarrow.NewSchema(fields, nil), columns, int64(df.Height()))
	defer tbl.Release()
	for i := range columns {
		columns[i].Release()
	}

	codec, err := codecFor(input.Compression)
	if err != nil {
		return err
	}
	props := aparquet.NewWriterProperties(append([]aparquet.WriterProperty{
		aparquet.WithCompression(codec),
		aparquet.WithMaxRowGroupLength(rowGroup),
	}, dictionaryOff...)...)

	f, err := os.Create(input.Path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := pqarrow.WriteTable(tbl, f, rowGroup, props, pqarrow.DefaultWriterProps()); err != nil {
		return err
	}
	return nil
}

// dictSampleSize bounds how many non-null values per column are sampled to
// estimate cardinality before choosing dictionary encoding.
const dictSampleSize = 4096

// lowCardinality reports whether at most half of a sample of an Int64,
// Float64, Datetime or string-backed column's values are distinct. Other
// dtypes report true, which keeps the arrow-go default.
func lowCardinality(col *chunk.Column) bool {
	if vals, ok := col.Int64s(); ok {
		return sampledLowCardinality(vals, col.Nulls(), func(v int64) int64 { return v })
	}
	if vals, ok := col.Float64s(); ok {
		return sampledLowCardinality(vals, col.Nulls(), math.Float64bits)
	}
	if vals, ok := col.Times(); ok {
		// Datetimes are written as microsecond timestamps, so that is the
		// value whose cardinality the dictionary sees. Write has already
		// rejected instants outside the microsecond range.
		return sampledLowCardinality(vals, col.Nulls(), time.Time.UnixMicro)
	}
	if vals, ok := col.Strings(); ok {
		return sampledLowCardinality(vals, col.Nulls(), func(v string) string { return v })
	}
	return true
}

// dictionaryChunks dictionary-encodes values one row group of rowGroup rows at
// a time, so that each column chunk's dictionary page holds only the values
// its own rows use, as arrow-go's encoder writes it.
func dictionaryChunks(values []string, nulls []bool, rowGroup int) *goarrow.Chunked {
	dtype := &goarrow.DictionaryType{IndexType: goarrow.PrimitiveTypes.Int32, ValueType: goarrow.BinaryTypes.String}
	indices := array.NewInt32Builder(memory.DefaultAllocator)
	defer indices.Release()
	dictionary := array.NewStringBuilder(memory.DefaultAllocator)
	defer dictionary.Release()
	index := make(map[string]int32)
	var chunks []goarrow.Array
	for start := 0; start < len(values); start += rowGroup {
		end := min(start+rowGroup, len(values))
		clear(index)
		indices.Reserve(end - start)
		for i := start; i < end; i++ {
			if nulls != nil && nulls[i] {
				indices.AppendNull()
				continue
			}
			k, seen := index[values[i]]
			if !seen {
				k = int32(len(index))
				index[values[i]] = k
				dictionary.Append(values[i])
			}
			indices.UnsafeAppend(k)
		}
		ids, dict := indices.NewArray(), dictionary.NewArray()
		chunks = append(chunks, array.NewDictionaryArray(dtype, ids, dict))
		ids.Release()
		dict.Release()
	}
	data := goarrow.NewChunked(dtype, chunks)
	for _, c := range chunks {
		c.Release()
	}
	return data
}

// sampledLowCardinality reports whether at most half of a strided,
// deterministic sample of up to dictSampleSize non-null values are distinct.
func sampledLowCardinality[T any, K comparable](values []T, nulls []bool, key func(T) K) bool {
	step := max(1, len(values)/dictSampleSize)
	distinct := make(map[K]struct{}, min(len(values), dictSampleSize))
	sampled := 0
	for i := 0; i < len(values) && sampled < dictSampleSize; i += step {
		if nulls != nil && nulls[i] {
			continue
		}
		distinct[key(values[i])] = struct{}{}
		sampled++
	}
	return len(distinct)*2 <= sampled
}

// codecFor maps a compression option string to an Arrow parquet codec. An empty
// string selects Snappy; an unknown codec name is an error (not a silent
// fallback).
func codecFor(name string) (compress.Compression, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "snappy":
		return compress.Codecs.Snappy, nil
	case "uncompressed", "none":
		return compress.Codecs.Uncompressed, nil
	case "gzip":
		return compress.Codecs.Gzip, nil
	case "zstd":
		return compress.Codecs.Zstd, nil
	case "brotli":
		return compress.Codecs.Brotli, nil
	case "lz4", "lz4_raw":
		return compress.Codecs.Lz4Raw, nil
	default:
		return compress.Codecs.Uncompressed, fmt.Errorf("unsupported parquet compression %q", name)
	}
}

// Read decodes a parquet file written by Write into a DataFrame. It first tries
// the columnar (Arrow) path; files in the legacy single-column JSON-envelope
// format (or raw JSON) fall through to the backward-compatible decoder.
func Read(input ReadInput) (frame.DataFrame, error) {
	if df, ok, err := readColumnar(input); ok {
		return df, err
	}
	return readLegacy(input)
}

// readColumnar reads a standard columnar parquet file. It returns ok=false (to
// defer to the legacy reader) when the file is not columnar parquet: either it
// is not a parquet file at all, or it is the legacy single "payload" envelope.
//
// Only the requested columns are decoded, and columns decode in parallel:
// pqarrow runs one goroutine per column and joins them before returning.
func readColumnar(input ReadInput) (frame.DataFrame, bool, error) {
	f, err := os.Open(input.Path)
	if err != nil {
		return frame.DataFrame{}, false, nil
	}
	defer func() { _ = f.Close() }() // read-only file

	mem := memory.NewGoAllocator()
	pf, err := file.NewParquetReader(f, file.WithReadProps(aparquet.NewReaderProperties(mem)))
	if err != nil {
		return frame.DataFrame{}, false, nil
	}
	fr, err := pqarrow.NewFileReader(pf, pqarrow.ArrowReadProperties{Parallel: true}, mem)
	if err != nil {
		return frame.DataFrame{}, false, nil
	}
	schema, err := fr.Schema()
	if err != nil || isLegacyEnvelope(schema) {
		return frame.DataFrame{}, false, nil
	}

	// ReadParquet takes no context; reads are bounded by the file itself.
	ctx := context.Background()
	var tbl goarrow.Table
	if len(input.Columns) == 0 {
		tbl, err = fr.ReadTable(ctx)
	} else {
		leaves := projectedLeaves(fr.Manifest.Fields, input.Columns)
		if len(leaves) == 0 {
			df, err := frame.New(frame.NewInput{})
			return df, true, err
		}
		rowGroups := make([]int, pf.NumRowGroups())
		for i := range rowGroups {
			rowGroups[i] = i
		}
		tbl, err = fr.ReadRowGroups(ctx, leaves, rowGroups)
	}
	if err != nil {
		return frame.DataFrame{}, false, nil
	}
	defer tbl.Release()

	df, err := tableToFrame(tbl)
	return df, true, err
}

// projectedLeaves returns, in file order, the leaf column indices of every
// top-level field named in columns. A nested field contributes all of its
// leaves, because pqarrow reads a field only from the leaves it is given.
func projectedLeaves(fields []pqarrow.SchemaField, columns []string) []int {
	var leaves []int
	var collect func(field *pqarrow.SchemaField)
	collect = func(field *pqarrow.SchemaField) {
		if field.IsLeaf() {
			leaves = append(leaves, field.ColIndex)
			return
		}
		for i := range field.Children {
			collect(&field.Children[i])
		}
	}
	for i := range fields {
		if slices.Contains(columns, fields[i].Field.Name) {
			collect(&fields[i])
		}
	}
	return leaves
}

// isLegacyEnvelope reports whether the schema is the old single-column JSON
// envelope produced by the previous writer.
func isLegacyEnvelope(schema *goarrow.Schema) bool {
	return schema.NumFields() == 1 && schema.Field(0).Name == "payload"
}

// tableToFrame converts an Arrow table to a DataFrame via the native bridge.
func tableToFrame(tbl goarrow.Table) (frame.DataFrame, error) {
	if tbl.NumRows() == 0 {
		return frame.New(frame.NewInput{})
	}
	tr := array.NewTableReader(tbl, tbl.NumRows())
	defer tr.Release()
	if !tr.Next() {
		return frame.New(frame.NewInput{})
	}
	return iarrow.FromArrowRecord(tr.RecordBatch())
}

// readLegacy decodes files written by the previous JSON-envelope writer (a
// single-column parquet whose value is a JSON-encoded frame) and raw-JSON
// fallback files, preserving backward compatibility.
func readLegacy(input ReadInput) (frame.DataFrame, error) {
	rows, err := pqgo.ReadFile[parquetEnvelope](input.Path)
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
	out := make([]frame.SeriesInput, 0, len(payload.Columns))
	for _, c := range payload.Columns {
		if len(columns) > 0 && !slices.Contains(columns, c.Name) {
			continue
		}
		values, err := decodeValues(c.Values, c.Type)
		if err != nil {
			return frame.DataFrame{}, err
		}
		out = append(out, frame.SeriesInput{Name: c.Name, Values: values})
	}
	return frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: out})
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
