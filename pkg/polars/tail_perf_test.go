package polars

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// buildTailFrame builds an n-row frame with an int64 key "a", a float "b", and a
// string label "c" — a mix of dtypes so the zero-copy view path covers pointer
// (string) and scalar columns alike.
func buildTailFrame(t testing.TB, n int) DataFrame {
	t.Helper()
	a := make([]any, n)
	b := make([]any, n)
	c := make([]any, n)
	for i := 0; i < n; i++ {
		a[i] = int64(i)
		b[i] = float64(i) * 0.5
		c[i] = "row"
	}
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: a},
		{Name: "b", Values: b},
		{Name: "c", Values: c},
	}})
	if err != nil {
		t.Fatalf("new dataframe: %v", err)
	}
	return df
}

// TestTailEqualsFrameTail proves the facade Tail returns the same rows/schema as
// a materializing reference (the last k rows in order).
func TestTailEqualsFrameTail(t *testing.T) {
	const n = 1000
	df := buildTailFrame(t, n)
	for _, k := range []int{0, 1, 100, n, n + 5} {
		got := df.Tail(k)
		wantH := k
		if wantH > n {
			wantH = n
		}
		if wantH < 0 {
			wantH = 0
		}
		if got.Height() != wantH {
			t.Fatalf("k=%d: height=%d, want %d", k, got.Height(), wantH)
		}
		start := n - wantH
		acol, err := got.GetColumn("a")
		if err != nil {
			t.Fatalf("k=%d GetColumn(a): %v", k, err)
		}
		for i := 0; i < wantH; i++ {
			if v, ok := acol.Value(i).(int64); !ok || v != int64(start+i) {
				t.Fatalf("k=%d row %d: a=%v, want %d", k, i, acol.Value(i), start+i)
			}
		}
	}
}

// TestTailAllocClassMatchesHead proves Tail no longer materializes an O(k) index
// buffer or per-column gather: its per-call allocation is within a small constant
// of Head for the same window (both are O(columns) zero-copy views).
func TestTailAllocClassMatchesHead(t *testing.T) {
	for _, n := range []int{1_000, 1_000_000} {
		df := buildTailFrame(t, n)
		k := 100
		headAllocs := testing.AllocsPerRun(50, func() { _ = df.Head(k) })
		tailAllocs := testing.AllocsPerRun(50, func() { _ = df.Tail(k) })
		if tailAllocs > headAllocs+2 {
			t.Fatalf("n=%d: Tail allocated %.0f times vs Head %.0f (want within +2; Tail must be a zero-copy view, not a gather)", n, tailAllocs, headAllocs)
		}
	}
}

func BenchmarkTail1K(b *testing.B) {
	df := buildTailFrame(b, 1_000)
	b.ReportAllocs()
	for b.Loop() {
		_ = df.Tail(100)
	}
}

func BenchmarkTail1M(b *testing.B) {
	df := buildTailFrame(b, 1_000_000)
	b.ReportAllocs()
	for b.Loop() {
		_ = df.Tail(100)
	}
}
