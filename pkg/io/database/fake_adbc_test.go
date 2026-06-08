package database

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/apache/arrow-adbc/go/adbc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

// This file provides a dependency-free, in-memory fake adbc.Connection so the
// write_database / read_database engine can be exercised end-to-end in CI
// without CGO or an external driver. It stores ingested Arrow batches per table
// and replays them for queries, honoring the ingest modes.

type storedTable struct {
	schema  *arrow.Schema
	records []arrow.RecordBatch
}

type fakeDB struct {
	tables map[string]*storedTable
}

func newFakeDB() *fakeDB { return &fakeDB{tables: map[string]*storedTable{}} }

type fakeConn struct{ db *fakeDB }

func (c *fakeConn) NewStatement() (adbc.Statement, error) { return &fakeStmt{db: c.db}, nil }
func (c *fakeConn) Close() error                          { return nil }
func (c *fakeConn) Commit(context.Context) error          { return nil }
func (c *fakeConn) Rollback(context.Context) error        { return nil }

func (c *fakeConn) GetInfo(context.Context, []adbc.InfoCode) (array.RecordReader, error) {
	return nil, notImpl()
}
func (c *fakeConn) GetObjects(context.Context, adbc.ObjectDepth, *string, *string, *string, *string, []string) (array.RecordReader, error) {
	return nil, notImpl()
}
func (c *fakeConn) GetTableSchema(context.Context, *string, *string, string) (*arrow.Schema, error) {
	return nil, notImpl()
}
func (c *fakeConn) GetTableTypes(context.Context) (array.RecordReader, error) { return nil, notImpl() }
func (c *fakeConn) ReadPartition(context.Context, []byte) (array.RecordReader, error) {
	return nil, notImpl()
}

type fakeStmt struct {
	db *fakeDB

	ingestTable string
	ingestMode  string
	bound       []arrow.RecordBatch
	boundSchema *arrow.Schema

	query string
}

func (s *fakeStmt) Close() error { return nil }

func (s *fakeStmt) SetOption(key, val string) error {
	switch key {
	case adbc.OptionKeyIngestTargetTable:
		s.ingestTable = val
	case adbc.OptionKeyIngestMode:
		s.ingestMode = val
	}
	return nil
}

func (s *fakeStmt) BindStream(_ context.Context, stream array.RecordReader) error {
	s.boundSchema = stream.Schema()
	for stream.Next() {
		rec := stream.RecordBatch()
		rec.Retain()
		s.bound = append(s.bound, rec)
	}
	return stream.Err()
}

func (s *fakeStmt) ExecuteUpdate(context.Context) (int64, error) {
	if s.ingestTable == "" {
		return -1, adbc.Error{Code: adbc.StatusInvalidArgument, Msg: "no ingest target table"}
	}
	existing, ok := s.db.tables[s.ingestTable]
	switch s.ingestMode {
	case adbc.OptionValueIngestModeCreate, "":
		if ok {
			return -1, adbc.Error{Code: adbc.StatusAlreadyExists, Msg: "table " + s.ingestTable + " already exists"}
		}
		s.db.tables[s.ingestTable] = &storedTable{schema: s.boundSchema, records: s.bound}
	case adbc.OptionValueIngestModeCreateAppend:
		if !ok {
			s.db.tables[s.ingestTable] = &storedTable{schema: s.boundSchema, records: s.bound}
		} else {
			existing.records = append(existing.records, s.bound...)
		}
	case adbc.OptionValueIngestModeReplace:
		s.db.tables[s.ingestTable] = &storedTable{schema: s.boundSchema, records: s.bound}
	default:
		return -1, adbc.Error{Code: adbc.StatusInvalidArgument, Msg: "unknown mode " + s.ingestMode}
	}
	var n int64
	for _, r := range s.bound {
		n += r.NumRows()
	}
	return n, nil
}

var fromTableRe = regexp.MustCompile("(?i)\\bfrom\\s+[\"`]?([A-Za-z0-9_.]+)")

func (s *fakeStmt) SetSqlQuery(query string) error { s.query = query; return nil }

func (s *fakeStmt) ExecuteQuery(context.Context) (array.RecordReader, int64, error) {
	m := fromTableRe.FindStringSubmatch(s.query)
	if m == nil {
		return nil, -1, adbc.Error{Code: adbc.StatusInvalidArgument, Msg: "fake: no FROM table in query"}
	}
	tbl, ok := s.db.tables[m[1]]
	if !ok {
		return nil, -1, adbc.Error{Code: adbc.StatusNotFound, Msg: "fake: unknown table " + m[1]}
	}
	// A WHERE clause the fake cannot evaluate yields zero rows (schema preserved).
	recs := tbl.records
	if strings.Contains(strings.ToUpper(s.query), " WHERE ") {
		recs = nil
	}
	rr, err := array.NewRecordReader(tbl.schema, recs)
	if err != nil {
		return nil, -1, err
	}
	return rr, -1, nil
}

func (s *fakeStmt) Prepare(context.Context) error                 { return nil }
func (s *fakeStmt) Bind(context.Context, arrow.RecordBatch) error { return notImpl() }
func (s *fakeStmt) SetSubstraitPlan([]byte) error                 { return notImpl() }
func (s *fakeStmt) GetParameterSchema() (*arrow.Schema, error)    { return nil, notImpl() }
func (s *fakeStmt) ExecutePartitions(context.Context) (*arrow.Schema, adbc.Partitions, int64, error) {
	return nil, adbc.Partitions{}, -1, notImpl()
}

func notImpl() error { return adbc.Error{Code: adbc.StatusNotImplemented} }

var (
	_ adbc.Connection = (*fakeConn)(nil)
	_ adbc.Statement  = (*fakeStmt)(nil)
)

// --- Tests against the fake -------------------------------------------------

func TestEngineWriteReadRoundTrip(t *testing.T) {
	conn := &fakeConn{db: newFakeDB()}
	ctx := context.Background()
	df := sampleFrame(t)

	n, err := Write(ctx, df, WriteInput{TableName: "events", IfTableExists: IfTableExistsReplace, Conn: conn, BatchSize: 2})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if n != 3 {
		t.Fatalf("rows affected = %d, want 3", n)
	}

	got, err := Read(ctx, ReadInput{Query: "SELECT id, score, name, ok, ts FROM events ORDER BY id", Conn: conn})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Height() != 3 || got.Width() != 5 {
		t.Fatalf("read shape = %dx%d, want 3x5", got.Height(), got.Width())
	}
	idCol, _ := got.GetColumn("id")
	nameCol, _ := got.GetColumn("name")
	for i := 0; i < 3; i++ {
		if idCol.Value(i) != int64(i+1) {
			t.Fatalf("id[%d] = %v, want %d", i, idCol.Value(i), i+1)
		}
	}
	if nameCol.Value(0) != "a" || nameCol.Value(2) != "c" {
		t.Fatalf("name round-trip mismatch: %v %v", nameCol.Value(0), nameCol.Value(2))
	}
}

func TestEngineIfTableExistsModes(t *testing.T) {
	conn := &fakeConn{db: newFakeDB()}
	ctx := context.Background()
	df := sampleFrame(t)
	const table = "modes"

	if _, err := Write(ctx, df, WriteInput{TableName: table, IfTableExists: IfTableExistsFail, Conn: conn}); err != nil {
		t.Fatalf("initial create: %v", err)
	}
	// fail on existing
	if _, err := Write(ctx, df, WriteInput{TableName: table, IfTableExists: IfTableExistsFail, Conn: conn}); err == nil {
		t.Fatalf("fail mode on existing table should error")
	}
	// append doubles the rows
	if _, err := Write(ctx, df, WriteInput{TableName: table, IfTableExists: IfTableExistsAppend, Conn: conn}); err != nil {
		t.Fatalf("append: %v", err)
	}
	appended, err := Read(ctx, ReadInput{Query: "SELECT id FROM " + table, Conn: conn})
	if err != nil {
		t.Fatalf("read after append: %v", err)
	}
	if appended.Height() != 6 {
		t.Fatalf("append height = %d, want 6", appended.Height())
	}
	// replace resets
	if _, err := Write(ctx, df, WriteInput{TableName: table, IfTableExists: IfTableExistsReplace, Conn: conn}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	replaced, err := Read(ctx, ReadInput{Query: "SELECT id FROM " + table, Conn: conn})
	if err != nil {
		t.Fatalf("read after replace: %v", err)
	}
	if replaced.Height() != 3 {
		t.Fatalf("replace height = %d, want 3", replaced.Height())
	}
}

func TestEngineEmptyResultTypedFrame(t *testing.T) {
	conn := &fakeConn{db: newFakeDB()}
	ctx := context.Background()

	// Ingest an explicitly empty (but typed) frame.
	empty := sampleFrame(t).Slice(0, 0)
	if _, err := Write(ctx, empty, WriteInput{TableName: "empty", IfTableExists: IfTableExistsReplace, Conn: conn}); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	got, err := Read(ctx, ReadInput{Query: "SELECT id, name FROM empty", Conn: conn})
	if err != nil {
		t.Fatalf("read empty: %v", err)
	}
	if got.Height() != 0 {
		t.Fatalf("empty height = %d, want 0", got.Height())
	}
	if got.Width() != 5 {
		t.Fatalf("empty width = %d, want 5 (schema preserved)", got.Width())
	}
}
