//go:build duckdb && duckdb_arrow

package polars

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

func idFrame(t *testing.T, ids ...int64) DataFrame {
	t.Helper()
	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = id
	}
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{{Name: "id", Values: values}}})
	if err != nil {
		t.Fatalf("NewDataFrame(%v): %v", ids, err)
	}
	return d
}

func idsOf(t *testing.T, d DataFrame) []int64 {
	t.Helper()
	col, err := d.GetColumn("id")
	if err != nil {
		t.Fatalf("GetColumn(%q): %v", "id", err)
	}
	ids := make([]int64, col.Len())
	for i := range ids {
		ids[i], _ = col.Value(i).(int64)
	}
	return ids
}

// TestDuckDBQuotedTableNames registers frames under names that need quoting
// and reads each back by its quoted name.
func TestDuckDBQuotedTableNames(t *testing.T) {
	cases := []struct {
		name  string
		query string
	}{
		{name: `my"tbl`, query: `SELECT * FROM "my""tbl" ORDER BY id`},
		{name: "my tbl", query: `SELECT * FROM "my tbl" ORDER BY id`},
		{name: "select", query: `SELECT * FROM "select" ORDER BY id`},
		{
			name:  `a" AS SELECT 'injected' AS secret --`,
			query: `SELECT * FROM "a"" AS SELECT 'injected' AS secret --" ORDER BY id`,
		},
	}
	for _, tc := range cases {
		sc := NewSQLContext()
		if err := sc.Register(tc.name, idFrame(t, 1, 2)); err != nil {
			t.Fatalf("Register(%q): %v", tc.name, err)
		}
		lf, err := sc.Execute(t.Context(), tc.query)
		if err != nil {
			t.Errorf("Execute(%q) with table %q error = %v, want nil", tc.query, tc.name, err)
			continue
		}
		if got, want := idsOf(t, collect(t, lf, nil)), []int64{1, 2}; !slices.Equal(got, want) {
			t.Errorf("Execute(%q) with table %q ids = %v, want %v", tc.query, tc.name, got, want)
		}
	}
}

// TestDuckDBInjectedNameIsOnlyAName checks that a name cannot end the
// identifier and create a table of its own.
func TestDuckDBInjectedNameIsOnlyAName(t *testing.T) {
	sc := NewSQLContext()
	name := `a" AS SELECT 'injected' AS secret --`
	if err := sc.Register(name, idFrame(t, 1)); err != nil {
		t.Fatalf("Register(%q): %v", name, err)
	}
	if lf, err := sc.Execute(t.Context(), "SELECT * FROM a"); err == nil {
		t.Errorf("Execute(%q) with table %q = %v, want an error: no table a exists", "SELECT * FROM a", name, collect(t, lf, nil).Rows())
	}
}

// TestDuckDBTableNameCannotRunStatements checks that a table name cannot add
// a statement that writes a file.
func TestDuckDBTableNameCannotRunStatements(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pwn.csv")
	table := `x" AS SELECT 1 AS c; COPY (SELECT 'owned' AS v) TO '` + path + `'; --`
	if _, err := idFrame(t, 1).Lazy().SQL(t.Context(), "SELECT 1 AS one", table); err != nil {
		t.Errorf("SQL(%q, table %q) error = %v, want nil", "SELECT 1 AS one", table, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}
	if len(entries) != 0 {
		t.Errorf("SQL(%q, table %q) created %v in %q, want no file", "SELECT 1 AS one", table, entries, dir)
	}
}

// TestDuckDBDMLOnQuotedTableName checks that the frame returned for a DML
// statement is read from its target table, also when another registered
// name is a prefix of it.
func TestDuckDBDMLOnQuotedTableName(t *testing.T) {
	sc := NewSQLContext()
	for name, d := range map[string]DataFrame{"my": idFrame(t, 10, 20), "my tbl": idFrame(t, 1, 2, 3)} {
		if err := sc.Register(name, d); err != nil {
			t.Fatalf("Register(%q): %v", name, err)
		}
	}
	query := `DELETE FROM "my tbl" WHERE id = 1`
	lf, err := sc.Execute(t.Context(), query)
	if err != nil {
		t.Fatalf("Execute(%q) error = %v, want nil", query, err)
	}
	got := idsOf(t, collect(t, lf, nil))
	slices.Sort(got)
	if want := []int64{2, 3}; !slices.Equal(got, want) {
		t.Errorf("Execute(%q) ids = %v, want %v", query, got, want)
	}
}

func TestDMLTarget(t *testing.T) {
	cases := []struct {
		query  string
		want   string
		wantOK bool
	}{
		{query: "DELETE FROM tbl WHERE id = 1", want: `"tbl"`, wantOK: true},
		{query: `  delete from "my tbl" where id = 1`, want: `"my tbl"`, wantOK: true},
		{query: `UPDATE "my""tbl" SET v = 1`, want: `"my""tbl"`, wantOK: true},
		{query: `INSERT INTO main."my tbl" VALUES (1)`, want: `"main"."my tbl"`, wantOK: true},
		{query: "TRUNCATE TABLE таблиця", want: `"таблиця"`, wantOK: true},
		{query: "TRUNCATE self", want: `"self"`, wantOK: true},
		{query: "UPDATE t_1$x SET v = 1", want: `"t_1$x"`, wantOK: true},
		{query: "SELECT * FROM tbl"},
		{query: "DELETE FROM tbl RETURNING *"},
		{query: `DELETE FROM "unterminated`},
		{query: "DELETE FROM 1tbl"},
	}
	for _, tc := range cases {
		got, ok := dmlTarget(tc.query)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("dmlTarget(%q) = %q, %t, want %q, %t", tc.query, got, ok, tc.want, tc.wantOK)
		}
	}
}
