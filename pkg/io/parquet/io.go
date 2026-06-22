package parquet

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	aparquet "github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/compress"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	pqgo "github.com/parquet-go/parquet-go"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
	"github.com/h0rn3t/gopolars/pkg/series"
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

	tbl := array.NewTableFromRecords(rec.Schema(), []goarrow.RecordBatch{rec})
	defer tbl.Release()

	codec, err := codecFor(input.Compression)
	if err != nil {
		return err
	}
	rowGroup := int64(input.RowGroupSize)
	if rowGroup <= 0 {
		rowGroup = defaultRowGroupRows
	}
	props := aparquet.NewWriterProperties(
		aparquet.WithCompression(codec),
		aparquet.WithMaxRowGroupLength(rowGroup),
	)

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
func readColumnar(input ReadInput) (frame.DataFrame, bool, error) {
	f, err := os.Open(input.Path)
	if err != nil {
		return frame.DataFrame{}, false, nil
	}
	defer func() { _ = f.Close() }()

	mem := memory.NewGoAllocator()
	tbl, err := pqarrow.ReadTable(context.Background(), f, aparquet.NewReaderProperties(mem), pqarrow.ArrowReadProperties{}, mem)
	if err != nil {
		return frame.DataFrame{}, false, nil
	}
	defer tbl.Release()

	if isLegacyEnvelope(tbl.Schema()) {
		return frame.DataFrame{}, false, nil
	}

	df, err := tableToFrame(tbl, input.Columns)
	return df, true, err
}

// isLegacyEnvelope reports whether the schema is the old single-column JSON
// envelope produced by the previous writer.
func isLegacyEnvelope(schema *goarrow.Schema) bool {
	return schema.NumFields() == 1 && schema.Field(0).Name == "payload"
}

// tableToFrame converts an Arrow table to a DataFrame via the native bridge and
// applies an optional column projection.
func tableToFrame(tbl goarrow.Table, columns []string) (frame.DataFrame, error) {
	if tbl.NumRows() == 0 {
		return frame.New(frame.NewInput{})
	}
	tr := array.NewTableReader(tbl, tbl.NumRows())
	defer tr.Release()
	if !tr.Next() {
		return frame.New(frame.NewInput{})
	}
	df, err := iarrow.FromArrowRecord(tr.RecordBatch())
	if err != nil {
		return frame.DataFrame{}, err
	}
	return projectFrame(df, columns)
}

// projectFrame returns df restricted to the named columns (preserving the
// frame's column order). An empty selection returns df unchanged.
func projectFrame(df frame.DataFrame, columns []string) (frame.DataFrame, error) {
	if len(columns) == 0 {
		return df, nil
	}
	want := make(map[string]struct{}, len(columns))
	for _, c := range columns {
		want[c] = struct{}{}
	}
	sel := make([]series.Series, 0, len(columns))
	for _, name := range df.Columns() {
		if _, ok := want[name]; !ok {
			continue
		}
		s, _ := df.Series(name)
		sel = append(sel, s)
	}
	return frame.New(frame.NewInput{Series: sel})
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
