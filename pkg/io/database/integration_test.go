//go:build cgo

package database

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/apache/arrow-adbc/go/adbc"
	"github.com/apache/arrow-adbc/go/adbc/drivermgr"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// locateSQLiteDriver finds the adbc-driver-sqlite shared library bundled in the
// pip package, so the integration tests run automatically wherever
// `pip install adbc-driver-sqlite` has been done.
const locateSQLiteDriver = `
import os, sys
try:
    import adbc_driver_sqlite as m
except Exception:
    sys.exit(1)
d = os.path.dirname(m.__file__)
for name in ("libadbc_driver_sqlite.so", "libadbc_driver_sqlite.dylib", "adbc_driver_sqlite.dll"):
    p = os.path.join(d, name)
    if os.path.exists(p):
        print(p); break
`

// resolveTestDriver returns the ADBC driver shared library and database URI for
// the integration tests: GOPOLARS_TEST_ADBC_DRIVER (+ _URI) if set, otherwise
// the auto-discovered SQLite driver with an in-memory database. Skips the test
// when neither is available.
func resolveTestDriver(t *testing.T) (driver, uri string) {
	t.Helper()
	if d := os.Getenv("GOPOLARS_TEST_ADBC_DRIVER"); d != "" {
		return d, os.Getenv("GOPOLARS_TEST_ADBC_URI")
	}
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("no GOPOLARS_TEST_ADBC_DRIVER and no python3 to locate adbc-driver-sqlite")
	}
	out, err := exec.Command(py, "-c", locateSQLiteDriver).Output()
	if err != nil {
		t.Skip("adbc-driver-sqlite not installed (pip install adbc-driver-sqlite) and GOPOLARS_TEST_ADBC_DRIVER unset")
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		t.Skip("could not locate the SQLite ADBC driver shared library")
	}
	return path, ":memory:"
}

// adbcTestConn opens an ADBC connection for the integration tests against a real
// driver (the SQLite driver by default), or skips when none is available.
func adbcTestConn(t *testing.T) adbc.Connection {
	t.Helper()
	driver, uri := resolveTestDriver(t)
	opts := map[string]string{"driver": driver}
	if uri != "" {
		opts["uri"] = uri
	}
	var drv drivermgr.Driver
	db, err := drv.NewDatabaseWithContext(context.Background(), opts)
	if err != nil {
		t.Skipf("open ADBC database (driver %q): %v", driver, err)
	}
	conn, err := db.Open(context.Background())
	if err != nil {
		_ = db.Close()
		t.Skipf("open ADBC connection (driver %q): %v", driver, err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
		_ = db.Close()
	})
	return conn
}

func TestWriteReadRoundTripIntegration(t *testing.T) {
	conn := adbcTestConn(t)
	ctx := context.Background()
	df := sampleFrame(t)

	n, err := Write(ctx, df, WriteInput{TableName: "gp_roundtrip", IfTableExists: IfTableExistsReplace, Conn: conn})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if n != 3 && n != -1 {
		t.Fatalf("rows affected = %d, want 3 or -1", n)
	}

	got, err := ReadFromConn(ctx, conn, "SELECT id, score, name, ok, ts FROM gp_roundtrip ORDER BY id")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Height() != 3 {
		t.Fatalf("read height = %d, want 3", got.Height())
	}
	if got.Width() != 5 {
		t.Fatalf("read width = %d, want 5", got.Width())
	}
	// Verify every supported dtype round-trips through the real database. The
	// name (string) check in particular exercises the C-imported-buffer clone in
	// pkg/io/arrow — without it these values would dangle into freed memory.
	//
	// id/score/name are preserved by every engine. Boolean and Datetime depend on
	// the backend's type system: SQLite (weak typing) returns the bool column as
	// INTEGER 1/0 and the timestamp as ISO TEXT, whereas a strongly-typed engine
	// keeps bool/timestamp. The checks below assert value equivalence tolerantly
	// so the test holds for either.
	idCol, _ := got.GetColumn("id")
	scoreCol, _ := got.GetColumn("score")
	nameCol, _ := got.GetColumn("name")
	okCol, _ := got.GetColumn("ok")
	tsCol, _ := got.GetColumn("ts")

	wantName := []string{"a", "b", "c"}
	wantScore := []float64{1.5, 2.5, 3.5}
	wantOK := []bool{true, false, true}
	for i := 0; i < 3; i++ {
		if idCol.Value(i) != int64(i+1) {
			t.Fatalf("id[%d] = %v, want %d", i, idCol.Value(i), i+1)
		}
		if scoreCol.Value(i) != wantScore[i] {
			t.Fatalf("score[%d] = %v, want %v", i, scoreCol.Value(i), wantScore[i])
		}
		if nameCol.Value(i) != wantName[i] {
			t.Fatalf("name[%d] = %v, want %q", i, nameCol.Value(i), wantName[i])
		}
		if truthy(okCol.Value(i)) != wantOK[i] {
			t.Fatalf("ok[%d] = %v (%T), want %v", i, okCol.Value(i), okCol.Value(i), wantOK[i])
		}
		if !timestampMatches(tsCol.Value(i), int64(i+1)) {
			t.Fatalf("ts[%d] = %v (%T) does not match unix %d", i, tsCol.Value(i), tsCol.Value(i), i+1)
		}
	}
}

// truthy normalizes a round-tripped boolean: native bool, or SQLite's INTEGER
// 1/0, or a textual "true"/"1".
func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case int64:
		return x != 0
	case string:
		return x == "true" || x == "1"
	default:
		return false
	}
}

// timestampMatches reports whether a round-tripped timestamp value represents
// the given Unix-second instant, accepting a native time.Time, an integer at
// any sub-second scale, or an ISO text rendering (SQLite).
func timestampMatches(v any, wantUnix int64) bool {
	switch x := v.(type) {
	case time.Time:
		return x.Unix() == wantUnix
	case int64:
		for _, scale := range []int64{1, 1e3, 1e6, 1e9} {
			if x == wantUnix*scale {
				return true
			}
		}
		return false
	case string:
		return strings.Contains(x, time.Unix(wantUnix, 0).UTC().Format("15:04:05"))
	default:
		return false
	}
}

func TestIfTableExistsModesIntegration(t *testing.T) {
	conn := adbcTestConn(t)
	ctx := context.Background()
	df := sampleFrame(t)
	const table = "gp_modes"

	// Start from a known clean state.
	if _, err := Write(ctx, df, WriteInput{TableName: table, IfTableExists: IfTableExistsReplace, Conn: conn}); err != nil {
		t.Fatalf("seed replace: %v", err)
	}

	// fail mode must error on an existing table.
	if _, err := Write(ctx, df, WriteInput{TableName: table, IfTableExists: IfTableExistsFail, Conn: conn}); err == nil {
		t.Fatalf("fail mode on existing table should error")
	}

	// append mode adds rows.
	if _, err := Write(ctx, df, WriteInput{TableName: table, IfTableExists: IfTableExistsAppend, Conn: conn}); err != nil {
		t.Fatalf("append: %v", err)
	}
	appended, err := ReadFromConn(ctx, conn, "SELECT id FROM "+table)
	if err != nil {
		t.Fatalf("read after append: %v", err)
	}
	if appended.Height() != 6 {
		t.Fatalf("append height = %d, want 6", appended.Height())
	}

	// replace mode resets to the frame's rows.
	if _, err := Write(ctx, df, WriteInput{TableName: table, IfTableExists: IfTableExistsReplace, Conn: conn}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	replaced, err := ReadFromConn(ctx, conn, "SELECT id FROM "+table)
	if err != nil {
		t.Fatalf("read after replace: %v", err)
	}
	if replaced.Height() != 3 {
		t.Fatalf("replace height = %d, want 3", replaced.Height())
	}
}

func TestReadEmptyResultIntegration(t *testing.T) {
	conn := adbcTestConn(t)
	ctx := context.Background()
	df := sampleFrame(t)
	if _, err := Write(ctx, df, WriteInput{TableName: "gp_empty", IfTableExists: IfTableExistsReplace, Conn: conn}); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadFromConn(ctx, conn, "SELECT id, name FROM gp_empty WHERE id < 0")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Height() != 0 {
		t.Fatalf("empty read height = %d, want 0", got.Height())
	}
	if got.Width() != 2 {
		t.Fatalf("empty read should preserve 2 columns, got %d", got.Width())
	}
}

// ReadFromConn is a tiny helper used by the integration tests.
func ReadFromConn(ctx context.Context, conn adbc.Connection, query string) (frame.DataFrame, error) {
	return Read(ctx, ReadInput{Query: query, Conn: conn})
}
