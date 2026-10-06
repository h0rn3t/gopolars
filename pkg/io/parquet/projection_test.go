package parquet

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow/memory"
	aparquet "github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	pqgo "github.com/parquet-go/parquet-go"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// projectionFixture writes a multi-row-group file with nulls in every column
// and a nested List column, and returns its path.
func projectionFixture(t *testing.T) string {
	t.Helper()
	const n = 1000
	id := make([]any, n)
	v := make([]any, n)
	s := make([]any, n)
	ts := make([]any, n)
	l := make([]any, n)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range n {
		if i%7 == 0 {
			continue // null row in every column
		}
		id[i] = int64(i)
		v[i] = float64(i) / 4
		s[i] = []string{"alpha", "beta", "gamma"}[i%3]
		ts[i] = base.Add(time.Duration(i) * time.Second)
		l[i] = []any{int64(i), int64(i + 1)}
	}
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: id, DType: dtypes.Int64},
		{Name: "v", Values: v, DType: dtypes.Float64},
		{Name: "s", Values: s, DType: dtypes.String},
		{Name: "ts", Values: ts, DType: dtypes.Datetime},
		{Name: "l", Values: l, DType: dtypes.List},
	}})
	if err != nil {
		t.Fatalf("FromAnyColumns: %v", err)
	}
	path := filepath.Join(t.TempDir(), "projection.parquet")
	if err := Write(df, WriteInput{Path: path, RowGroupSize: 128}); err != nil {
		t.Fatalf("Write(%q): %v", path, err)
	}
	return path
}

// sequentialRead is the reference reader: decode the whole file without
// parallelism, then project after conversion.
func sequentialRead(t *testing.T, path string, columns []string) frame.DataFrame {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}
	defer func() { _ = f.Close() }() // read-only file
	mem := memory.NewGoAllocator()
	tbl, err := pqarrow.ReadTable(context.Background(), f, aparquet.NewReaderProperties(mem), pqarrow.ArrowReadProperties{}, mem)
	if err != nil {
		t.Fatalf("pqarrow.ReadTable(%q): %v", path, err)
	}
	defer tbl.Release()
	df, err := tableToFrame(tbl)
	if err != nil {
		t.Fatalf("tableToFrame: %v", err)
	}
	if len(columns) == 0 {
		return df
	}
	var selected []series.Series
	for _, name := range df.Columns() {
		if slices.Contains(columns, name) {
			s, _ := df.Series(name)
			selected = append(selected, s)
		}
	}
	projected, err := frame.New(frame.NewInput{Series: selected})
	if err != nil {
		t.Fatalf("frame.New(%v): %v", columns, err)
	}
	return projected
}

func assertSameFrame(t *testing.T, label string, got, want frame.DataFrame) {
	t.Helper()
	if !slices.Equal(got.Columns(), want.Columns()) {
		t.Fatalf("%s columns = %v, want %v", label, got.Columns(), want.Columns())
	}
	if !slices.Equal(got.Dtypes(), want.Dtypes()) {
		t.Errorf("%s dtypes = %v, want %v", label, got.Dtypes(), want.Dtypes())
	}
	if eq, err := got.Equals(want); err != nil || !eq {
		t.Errorf("%s Equals(reference) = %v, %v; want true, nil", label, eq, err)
	}
}

func TestReadProjectionMatchesReference(t *testing.T) {
	t.Parallel()
	path := projectionFixture(t)

	tests := []struct {
		name        string
		columns     []string
		wantColumns []string
	}{
		{"all_columns_nil", nil, []string{"id", "v", "s", "ts", "l"}},
		{"subset", []string{"id", "v"}, []string{"id", "v"}},
		{"file_order_wins", []string{"ts", "id"}, []string{"id", "ts"}},
		{"unknown_name_ignored", []string{"id", "missing"}, []string{"id"}},
		{"nested_list", []string{"l", "s"}, []string{"s", "l"}},
		{"no_match", []string{"missing"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Read(ReadInput{Path: path, Columns: tt.columns})
			if err != nil {
				t.Fatalf("Read(%v): %v", tt.columns, err)
			}
			if !slices.Equal(got.Columns(), tt.wantColumns) {
				t.Fatalf("Read(%v) columns = %v, want %v", tt.columns, got.Columns(), tt.wantColumns)
			}
			if len(tt.wantColumns) == 0 {
				return
			}
			assertSameFrame(t, "Read("+tt.name+")", got, sequentialRead(t, path, tt.columns))
		})
	}
}

func TestReadLegacyEnvelopeWithProjection(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "envelope.parquet")
	payload := `{"columns":[{"name":"a","type":"int64","values":[1,2]},{"name":"b","type":"string","values":["x","y"]}]}`
	if err := pqgo.WriteFile(path, []parquetEnvelope{{Payload: payload}}); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
	df, err := Read(ReadInput{Path: path, Columns: []string{"b"}})
	if err != nil {
		t.Fatalf("Read(envelope, [b]): %v", err)
	}
	if !slices.Equal(df.Columns(), []string{"b"}) {
		t.Fatalf("Read(envelope, [b]) columns = %v, want [b]", df.Columns())
	}
	s, _ := df.Series("b")
	if got := []any{s.Value(0), s.Value(1)}; !slices.Equal(got, []any{"x", "y"}) {
		t.Errorf("Read(envelope, [b]) values = %v, want [x y]", got)
	}
}
