//go:build !duckdb || !duckdb_arrow

package polars

import (
	"context"
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestSQLStubReturnsBuildTagError verifies the default (non-DuckDB) build keeps
// the SQL API present but returns a clear build-tag error.
func TestSQLStubReturnsBuildTagError(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	if _, err := d.SQL(context.Background(), "SELECT * FROM self"); err == nil || !strings.Contains(err.Error(), "duckdb") {
		t.Fatalf("want duckdb build-tag error, got %v", err)
	}
	// Registration is pure bookkeeping and still works; only execution errors.
	ctx := NewSQLContext()
	if err := ctx.Register("t", d); err != nil {
		t.Fatalf("register should work in stub build: %v", err)
	}
	if _, err := ctx.Execute(context.Background(), "SELECT * FROM t"); err == nil {
		t.Fatalf("want build-tag error from Execute in stub build")
	}
}
