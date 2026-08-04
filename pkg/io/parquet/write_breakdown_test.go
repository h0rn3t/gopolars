package parquet

import (
	"os"
	"path/filepath"
	"testing"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	aparquet "github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

// benchWriteFrame mirrors the bench/top30 dataset shape (string key, two floats,
// one int) so the breakdown below is comparable to the cross-language numbers.
func benchWriteFrame(tb testing.TB, n int) frame.DataFrame {
	tb.Helper()
	g := make([]any, n)
	v := make([]any, n)
	nn := make([]any, n)
	i := make([]any, n)
	groups := []string{"a", "b", "c", "d", "e"}
	for k := 0; k < n; k++ {
		g[k] = groups[k%len(groups)]
		v[k] = float64(k%997) - 500
		if k%10 == 0 {
			nn[k] = nil
		} else {
			nn[k] = float64(k%89) - 44
		}
		i[k] = int64(k % 1000)
	}
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: g, DType: dtypes.String},
		{Name: "v", Values: v, DType: dtypes.Float64},
		{Name: "n", Values: nn, DType: dtypes.Float64},
		{Name: "i", Values: i, DType: dtypes.Int64},
	}})
	if err != nil {
		tb.Fatalf("build frame: %v", err)
	}
	return df
}

// BenchmarkWriteParquetBreakdown splits WriteParquet's cost into the part
// gopolars owns (converting the frame to Arrow) and the part the third-party
// encoder owns (pqarrow.WriteTable), which is what the phase C go/no-go for
// parquet turns on. Run with:
//
//	go test -run '^$' -bench BenchmarkWriteParquetBreakdown -benchtime=5x ./pkg/io/parquet/
func BenchmarkWriteParquetBreakdown(b *testing.B) {
	const n = 1_000_000
	df := benchWriteFrame(b, n)

	b.Run("1_ToArrowRecord", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			rec, err := iarrow.ToArrowRecord(df)
			if err != nil {
				b.Fatalf("ToArrowRecord: %v", err)
			}
			rec.Release()
		}
	})

	b.Run("2_pqarrowWriteTable", func(b *testing.B) {
		rec, err := iarrow.ToArrowRecord(df)
		if err != nil {
			b.Fatalf("ToArrowRecord: %v", err)
		}
		defer rec.Release()
		tbl := array.NewTableFromRecords(rec.Schema(), []goarrow.RecordBatch{rec})
		defer tbl.Release()
		codec, err := codecFor("")
		if err != nil {
			b.Fatalf("codecFor: %v", err)
		}
		props := aparquet.NewWriterProperties(
			aparquet.WithCompression(codec),
			aparquet.WithMaxRowGroupLength(defaultRowGroupRows),
		)
		dir := b.TempDir()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f, err := os.Create(filepath.Join(dir, "out.parquet"))
			if err != nil {
				b.Fatalf("create: %v", err)
			}
			if err := pqarrow.WriteTable(tbl, f, defaultRowGroupRows, props, pqarrow.DefaultWriterProps()); err != nil {
				b.Fatalf("WriteTable: %v", err)
			}
			_ = f.Close()
		}
	})

	b.Run("3_WriteEndToEnd", func(b *testing.B) {
		dir := b.TempDir()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := Write(df, WriteInput{Path: filepath.Join(dir, "e2e.parquet")}); err != nil {
				b.Fatalf("Write: %v", err)
			}
		}
	})
}

// BenchmarkWriteParquetCodecs shows what the compression choice costs, so the
// cross-language comparison can be checked for a like-for-like codec (gopolars
// defaults to Snappy; Polars' write_parquet defaults to zstd).
func BenchmarkWriteParquetCodecs(b *testing.B) {
	const n = 1_000_000
	df := benchWriteFrame(b, n)
	for _, codec := range []string{"snappy", "uncompressed", "zstd", "lz4"} {
		b.Run(codec, func(b *testing.B) {
			dir := b.TempDir()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := Write(df, WriteInput{Path: filepath.Join(dir, "c.parquet"), Compression: codec}); err != nil {
					b.Fatalf("Write(%s): %v", codec, err)
				}
			}
		})
	}
}
