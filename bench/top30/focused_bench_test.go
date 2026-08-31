package top30

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
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

// The rolling benchmarks below measure the SAME entry point BenchmarkTop30's
// Expr/rolling_* cases take — the public polars facade over makeDataFrame's "v"
// column, window 100 at the 1M reference scale — so their numbers are directly
// comparable to the parity budget. The internal pkg/frame path measures ~1 ms
// faster on the same commit, so it must not be used to judge a rolling budget.
//
// rolling_sum and rolling_mean run a Neumaier-compensated running accumulator;
// rolling_min runs a monotonic deque over the same windowing path and is the
// control: a change that moves the accumulator's constant factor must leave it
// untouched.

const rollingWindow = 100

func BenchmarkRollingMeanFacade(b *testing.B) {
	df := benchFrame(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := df.Select(polars.Col("v").RollingMean(rollingWindow)); err != nil {
			b.Fatalf("rolling_mean: %v", err)
		}
	}
}

func BenchmarkRollingSumFacade(b *testing.B) {
	df := benchFrame(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := df.Select(polars.Col("v").RollingSum(rollingWindow)); err != nil {
			b.Fatalf("rolling_sum: %v", err)
		}
	}
}

// BenchmarkRollingMinFacade is the deque control for the two above.
func BenchmarkRollingMinFacade(b *testing.B) {
	df := benchFrame(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := df.Select(polars.Col("v").RollingMin(rollingWindow)); err != nil {
			b.Fatalf("rolling_min: %v", err)
		}
	}
}

// wideKeyFrame builds an n-row frame with a unique Int64 key `k` = 0..n-1 and a
// Float64 payload. Unlike benchFrame's 1000-distinct "i", the key is fully
// distinct, so a self-join with another wideKeyFrame has BOTH sides above the
// parallel threshold — the large-right case that exercises the sharded build,
// which benchFrame's tiny right dimension never reaches.
func wideKeyFrame(b *testing.B, n int) polars.DataFrame {
	b.Helper()
	k := make([]any, n)
	v := make([]any, n)
	for i := range n {
		k[i] = int64(i)
		v[i] = float64(i) * 0.5
	}
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "k", Values: k},
			{Name: "w", Values: v},
		},
	})
	if err != nil {
		b.Fatalf("wideKeyFrame: %v", err)
	}
	return df
}

// BenchmarkJoinLargeRight measures an inner join where BOTH sides are 1M rows
// with fully distinct keys, so the build side exceeds the threshold and the
// sharded parallel build runs — the path the small-right BenchmarkJoin1M never
// reaches. Used to decide whether the parallel build pays for its complexity.
func BenchmarkJoinLargeRight(b *testing.B) {
	left := wideKeyFrame(b, focusedRows)
	right := wideKeyFrame(b, focusedRows)
	in := polars.JoinInput{Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: polars.JoinTypeInner}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := left.Join(in); err != nil {
			b.Fatalf("join: %v", err)
		}
	}
}
