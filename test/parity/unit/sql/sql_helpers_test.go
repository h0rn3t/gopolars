package sql

// Shared helpers for the ported SQL parity tests (py-polars/tests/unit/sql, py-1.28.1).

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// baseDF is the standard fixture: a (int), b (string), v (float).
func baseDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		{Name: "b", Values: []any{"x", "y", "x", "y"}},
		{Name: "v", Values: []any{10.0, 20.0, 30.0, 40.0}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// runSQL registers df as table "t" and collects the query result.
func runSQL(t *testing.T, df polars.DataFrame, query string) polars.DataFrame {
	t.Helper()
	ctx := polars.NewSQLContext()
	if err := ctx.Register("t", df); err != nil {
		t.Fatalf("register: %v", err)
	}
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

// execSQLErr returns the error from executing+collecting a query (or nil).
func execSQLErr(t *testing.T, df polars.DataFrame, query string) error {
	t.Helper()
	ctx := polars.NewSQLContext()
	if err := ctx.Register("t", df); err != nil {
		t.Fatalf("register: %v", err)
	}
	lf, err := ctx.Execute(context.Background(), query)
	if err != nil {
		return err
	}
	_, err = lf.Collect(context.Background())
	return err
}
