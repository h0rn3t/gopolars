package polars

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestConcatRejectsMismatchedDtypes(t *testing.T) {
	t.Parallel()
	ints := mscFrame(t, mscCol("x", int64(1), int64(2)))
	floats := mscFrame(t, mscCol("x", 3.5, 4.5))
	_, err := ints.Concat(ConcatInput{Others: []DataFrame{floats}})
	if err == nil || !strings.Contains(err.Error(), `"x"`) {
		t.Errorf("Concat(int64 x, float64 x) error = %v, want a dtype mismatch naming x", err)
	}
}

func TestReadParquetDirectoryRejectsMismatchedDtypes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for name, d := range map[string]DataFrame{
		"a.parquet": mscFrame(t, mscCol("x", int64(1), int64(2))),
		"b.parquet": mscFrame(t, mscCol("x", 3.5, 4.5)),
	} {
		if err := d.WriteParquet(WriteParquetInput{Path: filepath.Join(dir, name)}); err != nil {
			t.Fatalf("WriteParquet(%s): %v", name, err)
		}
	}
	got, err := NewIO().ReadParquet(ReadParquetInput{Path: dir})
	if err == nil {
		t.Errorf("ReadParquet(dir of int64 and float64 x) = %v, want a dtype mismatch error", got.ToDicts())
	}
}
