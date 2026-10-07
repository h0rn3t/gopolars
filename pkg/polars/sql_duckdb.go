//go:build duckdb && duckdb_arrow

package polars

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	duckdb "github.com/marcboeker/go-duckdb/v2"

	"github.com/h0rn3t/gopolars/pkg/frame"
	parrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
	idatabase "github.com/h0rn3t/gopolars/pkg/io/database"
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
			runQuery = "SELECT * FROM " + target
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

// dmlPrefix matches a leading DML statement up to its target table
// (DELETE FROM / UPDATE / INSERT INTO / TRUNCATE [TABLE]).
var dmlPrefix = regexp.MustCompile(`(?is)^\s*(?:DELETE\s+FROM|UPDATE|INSERT\s+INTO|TRUNCATE(?:\s+TABLE)?)\s+`)

// dmlTarget reports whether query is a (non-RETURNING) DML statement and, if so,
// the table it mutates as a quoted, possibly schema-qualified identifier.
// RETURNING statements keep DuckDB's native result.
func dmlTarget(query string) (string, bool) {
	if strings.Contains(strings.ToUpper(query), "RETURNING") {
		return "", false
	}
	loc := dmlPrefix.FindStringIndex(query)
	if loc == nil {
		return "", false
	}
	rest := query[loc[1]:]
	var parts []string
	for {
		part, n := leadingIdentifier(rest)
		if n == 0 {
			return "", false
		}
		parts = append(parts, idatabase.QuoteIdentifier(part))
		after, dotted := strings.CutPrefix(rest[n:], ".")
		if !dotted {
			return strings.Join(parts, "."), true
		}
		rest = after
	}
}

// leadingIdentifier reads the identifier at the start of s and returns its
// name and length in bytes, or a zero length when s does not start with one.
// The identifier is bare, or double-quoted with "" for a literal quote.
func leadingIdentifier(s string) (string, int) {
	if !strings.HasPrefix(s, `"`) {
		n := len(s)
		for i, r := range s {
			identChar := r == '_' || unicode.IsLetter(r) || i > 0 && (r == '$' || unicode.IsDigit(r))
			if !identChar {
				n = i
				break
			}
		}
		return s[:n], n
	}
	var name strings.Builder
	for i := 1; i < len(s); i++ {
		switch {
		case s[i] != '"':
			name.WriteByte(s[i])
		case i+1 < len(s) && s[i+1] == '"':
			name.WriteByte('"')
			i++
		default:
			return name.String(), i + 1
		}
	}
	return "", 0 // unterminated
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
	return execStmt(ctx, ar, fmt.Sprintf("CREATE TABLE %s AS SELECT * FROM %s",
		idatabase.QuoteIdentifier(name), idatabase.QuoteIdentifier(view)))
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

// isHugeint reports whether a is how DuckDB surfaces HUGEINT and integer
// aggregates (SUM/COUNT...): Decimal128 with precision 38 and scale 0. An
// explicit DECIMAL/NUMERIC cast keeps its own precision (< 38) and imports as a
// gopolars Decimal column. Every other type is left to parrow.FromArrowRecord,
// which widens narrow numbers and rejects values it cannot represent.
func isHugeint(a goarrow.Array) bool {
	dec, ok := a.(*array.Decimal128)
	if !ok {
		return false
	}
	dt := dec.DataType().(*goarrow.Decimal128Type)
	return dt.Precision == 38 && dt.Scale == 0
}

// normalizeRecord converts HUGEINT columns to Int64 (or Float64 when a value
// does not fit) and passes every other column through. Returns a record the
// caller must Release.
func normalizeRecord(rec goarrow.RecordBatch) (goarrow.RecordBatch, error) {
	needs := slices.ContainsFunc(rec.Columns(), isHugeint)
	if !needs {
		rec.Retain()
		return rec, nil
	}
	mem := memory.DefaultAllocator
	schema := rec.Schema()
	fields := make([]goarrow.Field, rec.NumCols())
	cols := make([]goarrow.Array, rec.NumCols())
	for i := range int(rec.NumCols()) {
		field := schema.Field(i)
		col := rec.Column(i)
		if isHugeint(col) {
			cols[i], field.Type = decimalToNumeric(mem, col.(*array.Decimal128), 0)
		} else {
			col.Retain()
			cols[i] = col
		}
		fields[i] = field
	}
	out := array.NewRecordBatch(goarrow.NewSchema(fields, nil), cols, rec.NumRows())
	for _, c := range cols {
		c.Release()
	}
	return out, nil
}

// decimalToNumeric converts a Decimal128 array to Int64 (scale 0, all values fit)
// or Float64 otherwise.
func decimalToNumeric(mem memory.Allocator, dec *array.Decimal128, scale int32) (goarrow.Array, goarrow.DataType) {
	n := dec.Len()
	if scale == 0 {
		ib := array.NewInt64Builder(mem)
		fits := true
		for i := range n {
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
	for i := range n {
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
