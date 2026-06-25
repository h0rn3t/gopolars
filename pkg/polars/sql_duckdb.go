//go:build duckdb && duckdb_arrow

package polars

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	duckdb "github.com/marcboeker/go-duckdb/v2"

	"github.com/h0rn3t/gopolars/pkg/frame"
	parrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

// execSQL runs a query against the given in-memory frames using an embedded
// DuckDB engine. Each frame is registered as an Arrow view and materialized into
// a real DuckDB table (querying the raw Arrow view ignores WHERE/aggregates),
// then the result is read back as Arrow and converted to a DataFrame.
func execSQL(ctx context.Context, query string, tables map[string]frame.DataFrame) (frame.DataFrame, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return frame.DataFrame{}, fmt.Errorf("open duckdb: %w", err)
	}
	defer func() { _ = db.Close() }()
	conn, err := db.Conn(ctx)
	if err != nil {
		return frame.DataFrame{}, fmt.Errorf("duckdb conn: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if err := registerUDFs(conn); err != nil {
		return frame.DataFrame{}, err
	}

	var result frame.DataFrame
	rawErr := conn.Raw(func(dc any) error {
		ar, err := duckdb.NewArrowFromConn(dc.(driver.Conn))
		if err != nil {
			return err
		}
		// Stabilize TIMESTAMPTZ rendering against the session timezone.
		if err := execStmt(ctx, ar, "SET TimeZone='UTC'"); err != nil {
			return err
		}
		for name, f := range tables {
			if err := registerTable(ctx, ar, name, f); err != nil {
				return fmt.Errorf("register table %q: %w", name, err)
			}
		}
		// polars returns the resulting table for a DML statement (DELETE/TRUNCATE/
		// UPDATE/INSERT); DuckDB returns a row count. Since each frame is
		// materialized into a real, mutable table, run the DML and then read the
		// affected table back so the result matches polars. (RETURNING-bearing
		// statements keep DuckDB's native semantics.)
		runQuery := query
		if target, ok := dmlTarget(query); ok {
			if err := execStmt(ctx, ar, query); err != nil {
				return err
			}
			runQuery = fmt.Sprintf(`SELECT * FROM "%s"`, target)
		}
		res, err := ar.QueryContext(ctx, runQuery)
		if err != nil {
			return err
		}
		defer res.Release()
		out, err := readResult(res)
		if err != nil {
			return err
		}
		result = out
		return nil
	})
	if rawErr != nil {
		return frame.DataFrame{}, rawErr
	}
	return result, nil
}

// dmlReTarget extracts the target table of a leading DML statement
// (DELETE FROM / UPDATE / INSERT INTO / TRUNCATE [TABLE]).
var dmlReTarget = regexp.MustCompile(`(?is)^\s*(?:DELETE\s+FROM|UPDATE|INSERT\s+INTO|TRUNCATE(?:\s+TABLE)?)\s+"?([A-Za-z_][A-Za-z0-9_]*)"?`)

// dmlTarget reports whether query is a (non-RETURNING) DML statement and, if so,
// the table it mutates. RETURNING statements keep DuckDB's native result.
func dmlTarget(query string) (string, bool) {
	if strings.Contains(strings.ToUpper(query), "RETURNING") {
		return "", false
	}
	m := dmlReTarget.FindStringSubmatch(query)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// execStmt runs a statement that produces no rows and releases its reader.
func execStmt(ctx context.Context, ar *duckdb.Arrow, stmt string) error {
	r, err := ar.QueryContext(ctx, stmt)
	if err != nil {
		return err
	}
	r.Release()
	return nil
}

// registerTable registers a frame as an Arrow view and materializes it into a
// real DuckDB table named `name`.
func registerTable(ctx context.Context, ar *duckdb.Arrow, name string, f frame.DataFrame) error {
	rec, err := parrow.ToArrowRecord(f)
	if err != nil {
		return err
	}
	defer rec.Release()
	reader, err := array.NewRecordReader(rec.Schema(), []goarrow.RecordBatch{rec})
	if err != nil {
		return err
	}
	defer reader.Release()
	view := "__gopolars_view_" + name
	release, err := ar.RegisterView(reader, view)
	if err != nil {
		return err
	}
	defer release()
	return execStmt(ctx, ar, fmt.Sprintf(`CREATE TABLE "%s" AS SELECT * FROM "%s"`, name, view))
}

// readResult drains an Arrow result reader into a single DataFrame.
func readResult(res array.RecordReader) (frame.DataFrame, error) {
	var frames []frame.DataFrame
	for res.Next() {
		norm, err := normalizeRecord(res.RecordBatch())
		if err != nil {
			return frame.DataFrame{}, err
		}
		f, err := parrow.FromArrowRecord(norm)
		norm.Release()
		if err != nil {
			return frame.DataFrame{}, err
		}
		frames = append(frames, f)
	}
	if err := res.Err(); err != nil {
		return frame.DataFrame{}, err
	}
	switch len(frames) {
	case 0:
		// Empty result: build a 0-row frame carrying the result schema.
		rb := array.NewRecordBuilder(memory.DefaultAllocator, res.Schema())
		empty := rb.NewRecordBatch()
		norm, err := normalizeRecord(empty)
		empty.Release()
		rb.Release()
		if err != nil {
			return frame.DataFrame{}, err
		}
		f, err := parrow.FromArrowRecord(norm)
		norm.Release()
		return f, err
	case 1:
		return frames[0], nil
	default:
		return frame.ConcatVertical(frames[0], frames[1:]...)
	}
}

// nativeArrow reports whether the gopolars Arrow converter already represents
// the array's type directly.
// List/Struct are passed through untouched: the gopolars Arrow bridge converts
// them (and normalizes any narrow numeric leaves) into boxed List/Struct columns.
func nativeArrow(a goarrow.Array) bool {
	switch a.(type) {
	case *array.Float64, *array.Int64, *array.Boolean, *array.String, *array.LargeString, *array.Timestamp:
		return true
	case *array.Date32, *array.Date64, *array.Time32, *array.Time64, *array.Binary, *array.LargeBinary:
		return true
	case *array.MonthDayNanoInterval:
		return true
	case *array.List, *array.LargeList, *array.FixedSizeList, *array.Struct:
		return true
	case *array.Decimal128:
		// Preserve an explicit DECIMAL/NUMERIC cast as a gopolars Decimal column, but
		// widen HUGEINT and integer aggregates (SUM/COUNT...) to int64/float64. DuckDB
		// surfaces those as Decimal128(precision=38, scale=0), which is byte-identical
		// to an explicit ::numeric(p,0) only when p==38; a user precision is < 38.
		dt := a.DataType().(*goarrow.Decimal128Type)
		return dt.Precision != 38 || dt.Scale != 0
	}
	return false
}

// normalizeRecord converts DuckDB result columns whose Arrow type gopolars cannot
// represent natively (narrow ints, unsigned ints, float32, Decimal128/HUGEINT)
// into Int64/Float64. Truly unsupported types (e.g. list/struct) return an error.
// Returns a record the caller must Release.
func normalizeRecord(rec goarrow.RecordBatch) (goarrow.RecordBatch, error) {
	needs := false
	for i := 0; i < int(rec.NumCols()); i++ {
		if !nativeArrow(rec.Column(i)) {
			needs = true
			break
		}
	}
	if !needs {
		rec.Retain()
		return rec, nil
	}
	mem := memory.DefaultAllocator
	schema := rec.Schema()
	fields := make([]goarrow.Field, rec.NumCols())
	cols := make([]goarrow.Array, rec.NumCols())
	for i := 0; i < int(rec.NumCols()); i++ {
		field := schema.Field(i)
		col := rec.Column(i)
		if nativeArrow(col) {
			col.Retain()
			cols[i] = col
		} else {
			arr, typ, err := convertArray(mem, col)
			if err != nil {
				for j := 0; j < i; j++ {
					cols[j].Release()
				}
				return nil, fmt.Errorf("column %q: %w", field.Name, err)
			}
			cols[i] = arr
			field.Type = typ
		}
		fields[i] = field
	}
	out := array.NewRecordBatch(goarrow.NewSchema(fields, nil), cols, rec.NumRows())
	for _, c := range cols {
		c.Release()
	}
	return out, nil
}

// convertArray widens a non-native numeric array to Int64 or Float64.
func convertArray(mem memory.Allocator, a goarrow.Array) (goarrow.Array, goarrow.DataType, error) {
	switch arr := a.(type) {
	case *array.Int8:
		return widenInt(mem, a.Len(), arr.IsNull, func(i int) int64 { return int64(arr.Value(i)) }), goarrow.PrimitiveTypes.Int64, nil
	case *array.Int16:
		return widenInt(mem, a.Len(), arr.IsNull, func(i int) int64 { return int64(arr.Value(i)) }), goarrow.PrimitiveTypes.Int64, nil
	case *array.Int32:
		return widenInt(mem, a.Len(), arr.IsNull, func(i int) int64 { return int64(arr.Value(i)) }), goarrow.PrimitiveTypes.Int64, nil
	case *array.Uint8:
		return widenInt(mem, a.Len(), arr.IsNull, func(i int) int64 { return int64(arr.Value(i)) }), goarrow.PrimitiveTypes.Int64, nil
	case *array.Uint16:
		return widenInt(mem, a.Len(), arr.IsNull, func(i int) int64 { return int64(arr.Value(i)) }), goarrow.PrimitiveTypes.Int64, nil
	case *array.Uint32:
		return widenInt(mem, a.Len(), arr.IsNull, func(i int) int64 { return int64(arr.Value(i)) }), goarrow.PrimitiveTypes.Int64, nil
	case *array.Uint64:
		return widenInt(mem, a.Len(), arr.IsNull, func(i int) int64 { return int64(arr.Value(i)) }), goarrow.PrimitiveTypes.Int64, nil
	case *array.Float32:
		fb := array.NewFloat64Builder(mem)
		for i := 0; i < a.Len(); i++ {
			if arr.IsNull(i) {
				fb.AppendNull()
			} else {
				fb.Append(float64(arr.Value(i)))
			}
		}
		out := fb.NewArray()
		fb.Release()
		return out, goarrow.PrimitiveTypes.Float64, nil
	case *array.Decimal128:
		dt := arr.DataType().(*goarrow.Decimal128Type)
		out, typ := decimalToNumeric(mem, arr, dt.Scale)
		return out, typ, nil
	}
	return nil, nil, fmt.Errorf("unsupported arrow type %s", a.DataType())
}

// widenInt builds an Int64 array from a narrower integer source.
func widenInt(mem memory.Allocator, n int, isNull func(int) bool, val func(int) int64) goarrow.Array {
	b := array.NewInt64Builder(mem)
	for i := 0; i < n; i++ {
		if isNull(i) {
			b.AppendNull()
		} else {
			b.Append(val(i))
		}
	}
	out := b.NewArray()
	b.Release()
	return out
}

// decimalToNumeric converts a Decimal128 array to Int64 (scale 0, all values fit)
// or Float64 otherwise.
func decimalToNumeric(mem memory.Allocator, dec *array.Decimal128, scale int32) (goarrow.Array, goarrow.DataType) {
	n := dec.Len()
	if scale == 0 {
		ib := array.NewInt64Builder(mem)
		fits := true
		for i := 0; i < n; i++ {
			if dec.IsNull(i) {
				ib.AppendNull()
				continue
			}
			bi := dec.Value(i).BigInt()
			if !bi.IsInt64() {
				fits = false
				break
			}
			ib.Append(bi.Int64())
		}
		if fits {
			arr := ib.NewArray()
			ib.Release()
			return arr, goarrow.PrimitiveTypes.Int64
		}
		ib.Release()
	}
	fb := array.NewFloat64Builder(mem)
	for i := 0; i < n; i++ {
		if dec.IsNull(i) {
			fb.AppendNull()
			continue
		}
		fb.Append(dec.Value(i).ToFloat64(scale))
	}
	arr := fb.NewArray()
	fb.Release()
	return arr, goarrow.PrimitiveTypes.Float64
}
