package polars

// Parity: Parquet round-trip (spec: Parquet round-trip parity).
// Mirrors write_parquet/read_parquet (lz4) in ../ms-calculations
// (generate_parquet.py, banking_rounding2.py:85).

import (
	"path/filepath"
	"testing"
	"time"
)

func mscParquetFrame(t *testing.T) DataFrame {
	t.Helper()
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return mscFrame(t,
		mscCol("z_id", int64(1), int64(2), int64(3)),
		mscCol("z_time", base, base.Add(time.Hour), base.Add(2*time.Hour)),
		mscCol("z_quantity", 1.25, 2.5, 3.75),
	)
}

// TestParityParquetRoundTrip pins generate_parquet.py — a write-then-read round-trip preserves the
// schema, height, and values of int/float/datetime columns.
func TestParityParquetRoundTrip(t *testing.T) {
	df := mscParquetFrame(t)
	path := filepath.Join(t.TempDir(), "balance.parquet")

	if err := df.WriteParquet(WriteParquetInput{Path: path}); err != nil {
		t.Fatalf("WriteParquet: %v", err)
	}
	back, err := NewIO().ReadParquet(ReadParquetInput{Path: path})
	if err != nil {
		t.Fatalf("ReadParquet: %v", err)
	}
	if back.Height() != df.Height() {
		t.Errorf("round-trip height = %d, want %d", back.Height(), df.Height())
	}
	eq, err := back.Equals(df)
	if err != nil {
		t.Fatalf("Equals: %v", err)
	}
	if !eq {
		t.Errorf("round-trip frame not equal to original\n got schema %v\nwant schema %v", back.Schema(), df.Schema())
	}
}

// TestParityParquetLZ4 pins the lz4 compression used by ms-calculations (compression="lz4").
func TestParityParquetLZ4(t *testing.T) {
	df := mscParquetFrame(t)
	path := filepath.Join(t.TempDir(), "balance_lz4.parquet")

	if err := df.WriteParquet(WriteParquetInput{Path: path, Compression: "lz4"}); err != nil {
		t.Fatalf("WriteParquet(compression=lz4): %v", err)
	}
	back, err := NewIO().ReadParquet(ReadParquetInput{Path: path})
	if err != nil {
		t.Fatalf("ReadParquet(lz4): %v", err)
	}
	eq, err := back.Equals(df)
	if err != nil {
		t.Fatalf("Equals: %v", err)
	}
	if !eq {
		t.Errorf("lz4 round-trip frame not equal to original")
	}
}
