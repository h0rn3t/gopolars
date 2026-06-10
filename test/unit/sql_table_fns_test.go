package unit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	iipc "github.com/h0rn3t/gopolars/pkg/io/ipc"
	iparquet "github.com/h0rn3t/gopolars/pkg/io/parquet"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// tableFnFixture writes the shared dataset (id, val) in every supported format
// under a temp dir and returns the path for the requested format.
func tableFnFixture(t *testing.T, format string) string {
	t.Helper()
	dir := t.TempDir()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "val", Values: []any{"x", "y", "z"}},
		},
	})
	if err != nil {
		t.Fatalf("fixture df: %v", err)
	}
	internal := mustInternalFrame(t, df)
	switch format {
	case "csv":
		path := filepath.Join(dir, "data.csv")
		if err := os.WriteFile(path, []byte("id,val\n1,x\n2,y\n3,z\n"), 0o644); err != nil {
			t.Fatalf("write csv: %v", err)
		}
		return path
	case "ndjson":
		path := filepath.Join(dir, "data.ndjson")
		content := `{"id": 1, "val": "x"}` + "\n" + `{"id": 2, "val": "y"}` + "\n" + `{"id": 3, "val": "z"}` + "\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write ndjson: %v", err)
		}
		return path
	case "parquet":
		path := filepath.Join(dir, "data.parquet")
		if err := iparquet.Write(internal, iparquet.WriteInput{Path: path}); err != nil {
			t.Fatalf("write parquet: %v", err)
		}
		return path
	case "ipc":
		path := filepath.Join(dir, "data.arrow")
		if err := iipc.Write(internal, iipc.WriteInput{Path: path}); err != nil {
			t.Fatalf("write ipc: %v", err)
		}
		return path
	}
	t.Fatalf("unknown format %s", format)
	return ""
}

// mustInternalFrame extracts the internal frame.DataFrame via the Arrow-free
// round trip available on the public API.
func mustInternalFrame(t *testing.T, df polars.DataFrame) frame.DataFrame {
	t.Helper()
	type unwrapper interface{ Unwrap() frame.DataFrame }
	if u, ok := df.(unwrapper); ok {
		return u.Unwrap()
	}
	// Fall back to rebuilding from rows.
	rows := df.ToDicts()
	cols := df.Columns()
	inputs := make([]frame.SeriesInput, len(cols))
	for i, c := range cols {
		values := make([]any, len(rows))
		for j, r := range rows {
			values[j] = r[c]
		}
		inputs[i] = frame.SeriesInput{Name: c, Values: values}
	}
	out, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: inputs})
	if err != nil {
		t.Fatalf("rebuild frame: %v", err)
	}
	return out
}

func runTableFnQuery(t *testing.T, query string) polars.DataFrame {
	t.Helper()
	ctx := polars.NewSQLContext()
	lf, err := ctx.Execute(context.Background(), query)
	if err != nil {
		t.Fatalf("execute %q: %v", query, err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect %q: %v", query, err)
	}
	return out
}

func TestSQLReadCSVTableFunction(t *testing.T) {
	path := tableFnFixture(t, "csv")
	out := runTableFnQuery(t, "SELECT * FROM read_csv('"+path+"') ORDER BY id")
	if out.Height() != 3 {
		t.Fatalf("height: got %d, want 3", out.Height())
	}
	val, _ := out.Series("val")
	if val.Value(0) != "x" || val.Value(2) != "z" {
		t.Fatalf("rows: got %v", out.ToDicts())
	}
}

func TestSQLReadCSVMissingFile(t *testing.T) {
	ctx := polars.NewSQLContext()
	_, err := ctx.Execute(context.Background(), "SELECT * FROM read_csv('missing_nowhere.csv')")
	if err == nil || !strings.Contains(err.Error(), "missing_nowhere.csv") {
		t.Fatalf("err = %v, want error naming the path", err)
	}
}

func TestSQLReadParquetTableFunction(t *testing.T) {
	path := tableFnFixture(t, "parquet")
	out := runTableFnQuery(t, "SELECT id, val FROM read_parquet('"+path+"') WHERE id >= 2 ORDER BY id")
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2", out.Height())
	}
}

func TestSQLReadJSONTableFunction(t *testing.T) {
	path := tableFnFixture(t, "ndjson")
	out := runTableFnQuery(t, "SELECT * FROM read_json('"+path+"') ORDER BY id")
	if out.Height() != 3 {
		t.Fatalf("height: got %d, want 3", out.Height())
	}
}

func TestSQLReadIPCTableFunction(t *testing.T) {
	path := tableFnFixture(t, "ipc")
	out := runTableFnQuery(t, "SELECT * FROM read_ipc('"+path+"') ORDER BY id")
	if out.Height() != 3 {
		t.Fatalf("height: got %d, want 3", out.Height())
	}
}

func TestSQLTableFunctionAliasAndJoin(t *testing.T) {
	path := tableFnFixture(t, "csv")
	people, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "who", Values: []any{"ann", "bob"}},
		},
	})
	if err != nil {
		t.Fatalf("people df: %v", err)
	}
	ctx := polars.NewSQLContext()
	if err := ctx.Register("people", people); err != nil {
		t.Fatalf("register: %v", err)
	}
	lf, err := ctx.Execute(context.Background(),
		"SELECT t.val, u.who FROM read_csv('"+path+"') AS t INNER JOIN people AS u ON t.id = u.id ORDER BY t.val")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("join height: got %d, want 2", out.Height())
	}
	who, _ := out.Series("who")
	if who.Value(0) != "ann" || who.Value(1) != "bob" {
		t.Fatalf("join rows: got %v", out.ToDicts())
	}
}

func TestSQLTableFunctionInCTE(t *testing.T) {
	path := tableFnFixture(t, "parquet")
	out := runTableFnQuery(t,
		"WITH src AS (SELECT id FROM read_parquet('"+path+"')) SELECT COUNT(*) AS n FROM src")
	n, _ := out.Series("n")
	if n.Len() != 1 || n.Value(0) != int64(3) {
		t.Fatalf("cte count: got %v", out.ToDicts())
	}
}

func TestSQLTableFunctionErrors(t *testing.T) {
	ctx := polars.NewSQLContext()
	if _, err := ctx.Execute(context.Background(), "SELECT * FROM read_xlsx('x.xlsx')"); err == nil ||
		!strings.Contains(err.Error(), "read_xlsx") {
		t.Fatalf("unknown fn: err = %v, want error naming read_xlsx", err)
	}
	if _, err := ctx.Execute(context.Background(), "SELECT * FROM read_csv(col)"); err == nil ||
		!strings.Contains(err.Error(), "string literal") {
		t.Fatalf("non-literal arg: err = %v, want string literal error", err)
	}
	if _, err := ctx.Execute(context.Background(), "SELECT * FROM read_csv('a.csv', 'b.csv')"); err == nil ||
		!strings.Contains(err.Error(), "exactly one argument") {
		t.Fatalf("arity: err = %v, want exactly-one-argument error", err)
	}
}
