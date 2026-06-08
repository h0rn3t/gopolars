package toplevel

// Ported from py-polars/tests/unit/test_schema.py (py-1.28.1, feasible subset).
//
// gopolars has no standalone pl.Schema object, nor the Int8/UInt32/Duration/Object
// dtypes most of that file parametrizes over. The portable intent — a DataFrame's
// schema reports ordered (name, dtype) fields, distinguishes by dtype, and is
// preserved/updated by select/rename/with_columns — is asserted here against
// DataFrame.Schema()/CollectSchema().

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func schemaDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "foo", Values: []any{int64(1), int64(2)}},
		{Name: "bar", Values: []any{1.0, 2.0}},
		{Name: "baz", Values: []any{"a", "b"}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// test_schema: ordered names + dtypes, and lookup by name.
func TestSchemaNamesAndDtypes(t *testing.T) {
	t.Parallel()
	s := schemaDF(t).Schema()
	if len(s) != 3 {
		t.Fatalf("schema len: got %d, want 3", len(s))
	}
	wantNames := []string{"foo", "bar", "baz"}
	wantTypes := []dtypes.DataType{polars.Int64, polars.Float64, polars.String}
	for i, f := range s {
		if f.Name != wantNames[i] {
			t.Fatalf("name[%d]: got %q, want %q", i, f.Name, wantNames[i])
		}
		if f.Type != wantTypes[i] {
			t.Fatalf("type[%d]: got %v, want %v", i, f.Type, wantTypes[i])
		}
	}
	if s.IndexOf("bar") != 1 {
		t.Fatalf("IndexOf(bar): got %d, want 1", s.IndexOf("bar"))
	}
	if s.IndexOf("missing") != -1 {
		t.Fatalf("IndexOf(missing): got %d, want -1", s.IndexOf("missing"))
	}
}

// test_schema_equality: same names but a differing dtype => schemas differ.
func TestSchemaEqualityByDtype(t *testing.T) {
	t.Parallel()
	a, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "foo", Values: []any{int64(1)}},
		{Name: "bar", Values: []any{1.0}},
	}})
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "foo", Values: []any{int64(1)}},
		{Name: "bar", Values: []any{"x"}},
	}})
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	sa, sb := a.Schema(), b.Schema()
	if sa[1].Type == sb[1].Type {
		t.Fatalf("bar dtype should differ: %v vs %v", sa[1].Type, sb[1].Type)
	}
	if sa[0].Type != sb[0].Type {
		t.Fatalf("foo dtype should match: %v vs %v", sa[0].Type, sb[0].Type)
	}
}

// Schema is projected by Select, and CollectSchema agrees with Schema.
func TestSchemaProjectedBySelect(t *testing.T) {
	t.Parallel()
	out, err := schemaDF(t).Select(polars.Col("baz"), polars.Col("foo"))
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	s := out.Schema()
	if len(s) != 2 || s[0].Name != "baz" || s[1].Name != "foo" {
		t.Fatalf("projected schema: got %+v", s)
	}
	if s[0].Type != polars.String || s[1].Type != polars.Int64 {
		t.Fatalf("projected types: got %v,%v", s[0].Type, s[1].Type)
	}
	cs := out.CollectSchema()
	if len(cs) != len(s) || cs[0].Name != s[0].Name || cs[0].Type != s[0].Type {
		t.Fatalf("CollectSchema disagrees with Schema: %+v vs %+v", cs, s)
	}
}

// Rename changes the name but preserves the dtype in the schema.
func TestSchemaRenamePreservesType(t *testing.T) {
	t.Parallel()
	renamed, err := schemaDF(t).Rename(map[string]string{"foo": "qux"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	s := renamed.Schema()
	if s[0].Name != "qux" {
		t.Fatalf("renamed name: got %q, want qux", s[0].Name)
	}
	if s[0].Type != polars.Int64 {
		t.Fatalf("renamed type: got %v, want Int64", s[0].Type)
	}
}

// with_columns extends the schema with the derived column's dtype.
func TestSchemaWithColumnsExtends(t *testing.T) {
	t.Parallel()
	out, err := schemaDF(t).WithColumns(polars.Col("foo").Add(polars.Lit(int64(1))).Alias("foo2"))
	if err != nil {
		t.Fatalf("with_columns: %v", err)
	}
	s := out.Schema()
	if len(s) != 4 {
		t.Fatalf("schema len: got %d, want 4", len(s))
	}
	if s[3].Name != "foo2" || s[3].Type != polars.Int64 {
		t.Fatalf("added field: got %+v, want {foo2 Int64}", s[3])
	}
}
