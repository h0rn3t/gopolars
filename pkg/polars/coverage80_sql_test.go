//go:build !duckdb || !duckdb_arrow

package polars

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// covSQLFrame builds a small frame for SQL surface coverage.
func covSQLFrame(t *testing.T) DataFrame {
	t.Helper()
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2)}},
		{Name: "v", Values: []any{10.0, 20.0}},
	}})
	if err != nil {
		t.Fatalf("covSQLFrame: %v", err)
	}
	return d
}

// TestSQLSurfaceStubErrors exercises the SQL entry points that all route
// through execSQL (which errors in the default non-duckdb build). We assert the
// bookkeeping paths succeed and that execution returns an error.
func TestSQLSurfaceStubErrors(t *testing.T) {
	ctx := context.Background()
	d := covSQLFrame(t)

	// df.Sql lowercase alias.
	if _, err := d.Sql(ctx, "SELECT * FROM self"); err == nil {
		t.Fatalf("df.Sql want error in stub build")
	}

	// lf.SQL collects then errors at execution.
	if _, err := d.Lazy().SQL(ctx, "SELECT * FROM t", "t"); err == nil {
		t.Fatalf("lf.SQL want error in stub build")
	}

	// ioFacade.SQL with no source table.
	if _, err := NewIO().SQL(ctx, "SELECT 1 AS x"); err == nil {
		t.Fatalf("io.SQL want error in stub build")
	}
}

// TestSQLContextRegistrationSurface covers RegisterMany/RegisterGlobals/
// Unregister/Tables/ExecuteGlobal bookkeeping (no execution dependence).
func TestSQLContextRegistrationSurface(t *testing.T) {
	ctx := context.Background()
	d := covSQLFrame(t)

	sc := NewSQLContext()
	if err := sc.RegisterMany(map[string]DataFrame{"a": d, "b": d}); err != nil {
		t.Fatalf("RegisterMany: %v", err)
	}
	tables := sc.Tables()
	if len(tables) != 2 || tables[0] != "a" || tables[1] != "b" {
		t.Fatalf("Tables = %v, want sorted [a b]", tables)
	}

	// RegisterGlobals is an alias of RegisterMany.
	if err := sc.RegisterGlobals(map[string]DataFrame{"c": d}); err != nil {
		t.Fatalf("RegisterGlobals: %v", err)
	}
	if got := len(sc.Tables()); got != 3 {
		t.Fatalf("after RegisterGlobals Tables len = %d, want 3", got)
	}

	sc.Unregister("b")
	if got := len(sc.Tables()); got != 2 {
		t.Fatalf("after Unregister Tables len = %d, want 2", got)
	}

	// ExecuteGlobal routes through Execute -> execSQL (errors in stub build).
	if _, err := sc.ExecuteGlobal(ctx, "SELECT * FROM a"); err == nil {
		t.Fatalf("ExecuteGlobal want error in stub build")
	}
}

// TestSQLContextRegisterErrors covers the error branches of Register.
func TestSQLContextRegisterErrors(t *testing.T) {
	sc := NewSQLContext()
	if err := sc.Register("", covSQLFrame(t)); err == nil {
		t.Fatalf("Register with empty name want error")
	}
	// RegisterMany propagates Register errors.
	if err := sc.RegisterMany(map[string]DataFrame{"": covSQLFrame(t)}); err == nil {
		t.Fatalf("RegisterMany with empty name want error")
	}
}
