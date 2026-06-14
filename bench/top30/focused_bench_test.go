package top30

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// These focused, Go-only (no Python) benchmarks pin the path each lagging
// DataFrame op actually takes at the 1M reference scale, with -benchmem, so a
// regression in the parallel join / row-materialization paths is caught without
// the full cross-language harness. The dataset matches bench/top30: column "n"
// carries ~10% nulls (the sparse drop_nulls case), "v" is ~50% positive (the
// filter case), and "i" has 1000 distinct Int64 keys (the join/unique case).

const focusedRows = 1_000_000

func benchFrame(b *testing.B) polars.DataFrame {
	b.Helper()
	df, err := makeDataFrame(focusedRows)
	if err != nil {
		b.Fatalf("make dataframe: %v", err)
	}
	return df
}

// BenchmarkDropNullsSparse measures drop_nulls on the ~10%-null "n" column: most
// rows survive, so the no-null fast path does NOT apply and every column is
// gathered — the slow case the parallel gather targets.
func BenchmarkDropNullsSparse(b *testing.B) {
	df := benchFrame(b)
	b.ReportAllocs()
	for b.Loop() {
		_ = df.DropNulls("n")
	}
}

// BenchmarkFilterHalf measures filter at ~50% selectivity (v > 0).
func BenchmarkFilterHalf(b *testing.B) {
	df := benchFrame(b)
	pred := polars.Col("v").Gt(polars.Lit(float64(0)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := df.Filter(pred); err != nil {
			b.Fatalf("filter: %v", err)
		}
	}
}

// BenchmarkUnique1M measures unique over the 1000-distinct-key "i" column.
func BenchmarkUnique1M(b *testing.B) {
	df := benchFrame(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := df.Unique("i"); err != nil {
			b.Fatalf("unique: %v", err)
		}
	}
}

// BenchmarkJoin1M measures the headline equi-join: 1M rows joined to the
// deduplicated 1000-row "i" dimension on a single Int64 key (the packed,
// parallel path).
func BenchmarkJoin1M(b *testing.B) {
	df := benchFrame(b)
	dim, err := df.Unique("i")
	if err != nil {
		b.Fatalf("unique dim: %v", err)
	}
	in := polars.JoinInput{Other: dim, LeftOn: []string{"i"}, RightOn: []string{"i"}, How: polars.JoinTypeInner}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := df.Join(in); err != nil {
			b.Fatalf("join: %v", err)
		}
	}
}
