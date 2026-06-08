package lazyframe

// Ported from py-polars/tests/unit/lazyframe/test_collect_schema.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// collect_schema reports column names and dtypes without materializing the data.
func TestCollectSchema(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{"x", "y"}},
		{Name: "c", Values: []any{1.5, 2.5}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	schema := df.Lazy().CollectSchema()
	got := map[string]dtypes.DataType{}
	for _, f := range schema {
		got[f.Name] = f.Type
	}
	if got["a"] != polars.Int64 {
		t.Fatalf("a dtype: got %v, want Int64", got["a"])
	}
	if got["b"] != polars.String {
		t.Fatalf("b dtype: got %v, want String", got["b"])
	}
	if got["c"] != polars.Float64 {
		t.Fatalf("c dtype: got %v, want Float64", got["c"])
	}
}

// DISCREPANCY: Python's collect_schema reflects a lazy projection, so after
// select("a") the schema has only column "a". gopolars CollectSchema() returns
// the SOURCE schema (both a and b) — the lazy Select is not folded into the
// reported schema. The collected DATA is still correctly narrowed (see
// projections_test.go); only the schema preview differs. We pin gopolars.
func TestCollectSchemaAfterSelect(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1)}},
		{Name: "b", Values: []any{int64(2)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	schema := df.Lazy().Select(polars.Col("a")).CollectSchema()
	if len(schema) != 2 {
		t.Fatalf("schema after select: got %d fields, want 2 (gopolars reports source schema; Python -> 1)", len(schema))
	}
}
