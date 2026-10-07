package parquet

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestReadParquetNarrowNumericTypes reads a file written by another Arrow
// writer with int32 and float32 columns, which gopolars imports as Int64 and
// Float64.
func TestReadParquetNarrowNumericTypes(t *testing.T) {
	mem := memory.NewGoAllocator()
	ib := array.NewInt32Builder(mem)
	defer ib.Release()
	ib.AppendValues([]int32{7, 0, 9}, []bool{true, false, true})
	fb := array.NewFloat32Builder(mem)
	defer fb.Release()
	fb.AppendValues([]float32{1.5, 0, -2.25}, []bool{true, false, true})
	i32, f32 := ib.NewArray(), fb.NewArray()
	defer i32.Release()
	defer f32.Release()
	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: "i32", Type: i32.DataType(), Nullable: true},
		{Name: "f32", Type: f32.DataType(), Nullable: true},
	}, nil)
	rec := array.NewRecordBatch(schema, []goarrow.Array{i32, f32}, 3)
	defer rec.Release()
	tbl := array.NewTableFromRecords(schema, []goarrow.RecordBatch{rec})
	defer tbl.Release()

	path := filepath.Join(t.TempDir(), "narrow.parquet")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %q: %v", path, err)
	}
	// WriteTable closes f once the footer is written.
	if err := pqarrow.WriteTable(tbl, f, 1024, nil, pqarrow.DefaultWriterProps()); err != nil {
		t.Fatalf("pqarrow.WriteTable(%q): %v", path, err)
	}

	df, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("Read(%q) error = %v, want nil", path, err)
	}
	cases := []struct {
		name      string
		wantDtype dtypes.DataType
		want      []any
	}{
		{name: "i32", wantDtype: dtypes.Int64, want: []any{int64(7), nil, int64(9)}},
		{name: "f32", wantDtype: dtypes.Float64, want: []any{1.5, nil, -2.25}},
	}
	for _, tc := range cases {
		s, ok := df.Series(tc.name)
		if !ok {
			t.Errorf("Read(%q) columns = %v, want a column %q", path, df.Columns(), tc.name)
			continue
		}
		if got := s.DataType(); got != tc.wantDtype {
			t.Errorf("Read(%q) column %q dtype = %s, want %s", path, tc.name, got, tc.wantDtype)
		}
		got := make([]any, s.Len())
		for i := range got {
			got[i] = s.Value(i)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Read(%q) column %q = %#v, want %#v", path, tc.name, got, tc.want)
		}
	}
}
