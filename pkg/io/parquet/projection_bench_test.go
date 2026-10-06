package parquet

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// prodShapeFrame mirrors the column mix a production consumer writes and reads:
// unique Int64/Float64/Datetime columns plus a low-cardinality String column.
func prodShapeFrame(tb testing.TB, n int) frame.DataFrame {
	tb.Helper()
	id := make([]any, n)
	v := make([]any, n)
	s := make([]any, n)
	ts := make([]any, n)
	labels := []string{"alpha", "beta", "gamma"}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range n {
		id[i] = int64(i)
		v[i] = float64(i) * 1.5
		s[i] = labels[i%len(labels)]
		ts[i] = base.Add(time.Duration(i) * time.Minute)
	}
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: id, DType: dtypes.Int64},
		{Name: "v", Values: v, DType: dtypes.Float64},
		{Name: "s", Values: s, DType: dtypes.String},
		{Name: "ts", Values: ts, DType: dtypes.Datetime},
	}})
	if err != nil {
		tb.Fatalf("build frame: %v", err)
	}
	return df
}

const prodShapeRows = 200_000

func BenchmarkReadParquetProjection(b *testing.B) {
	path := filepath.Join(b.TempDir(), "prod.parquet")
	if err := Write(prodShapeFrame(b, prodShapeRows), WriteInput{Path: path}); err != nil {
		b.Fatalf("Write: %v", err)
	}
	for _, bc := range []struct {
		name    string
		columns []string
	}{
		{"all", nil},
		{"two_of_four", []string{"id", "v"}},
	} {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := Read(ReadInput{Path: path, Columns: bc.columns}); err != nil {
					b.Fatalf("Read(%v): %v", bc.columns, err)
				}
			}
		})
	}
}

func BenchmarkWriteParquetProdShape(b *testing.B) {
	df := prodShapeFrame(b, prodShapeRows)
	path := filepath.Join(b.TempDir(), "prod.parquet")
	b.ReportAllocs()
	for b.Loop() {
		if err := Write(df, WriteInput{Path: path}); err != nil {
			b.Fatalf("Write: %v", err)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		b.Fatalf("stat %q: %v", path, err)
	}
	b.ReportMetric(float64(info.Size()), "file-bytes")
}
