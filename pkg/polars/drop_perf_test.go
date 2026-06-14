package polars

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// buildDropFrame builds an n-row frame with an int64 key column "a" (null on
// every nullEvery-th row when nullEvery > 0), a null-free float "b", and a
// string label "c". nullEvery == 0 leaves "a" fully populated.
func buildDropFrame(t testing.TB, n, nullEvery int) DataFrame {
	t.Helper()
	a := make([]any, n)
	b := make([]any, n)
	c := make([]any, n)
	for i := 0; i < n; i++ {
		b[i] = float64(i)
		c[i] = "row"
		if nullEvery > 0 && i%nullEvery == 0 {
			a[i] = nil
		} else {
			a[i] = int64(i)
		}
	}
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: a, DType: dtypes.Int64}, // pinned so an all-null "a" still types
		{Name: "b", Values: b},
		{Name: "c", Values: c},
	}})
	if err != nil {
		t.Fatalf("new dataframe: %v", err)
	}
	return df
}

// TestDropNullsSelectivityEquality checks drop_nulls across the empty / half /
// full selectivity profiles: the surviving rows (and their values) must match a
// straightforward reference computed from where "a" is non-null.
func TestDropNullsSelectivityEquality(t *testing.T) {
	const n = 1000
	for _, nullEvery := range []int{0, 2, 1} { // none dropped, ~half, all
		df := buildDropFrame(t, n, nullEvery)
		got := df.DropNulls()

		var wantB []float64
		for i := 0; i < n; i++ {
			if nullEvery == 0 || i%nullEvery != 0 {
				wantB = append(wantB, float64(i))
			}
		}
		if got.Height() != len(wantB) {
			t.Fatalf("nullEvery=%d: height=%d, want %d", nullEvery, got.Height(), len(wantB))
		}
		bcol, err := got.GetColumn("b")
		if err != nil {
			t.Fatalf("GetColumn(b): %v", err)
		}
		for i, want := range wantB {
			if v, ok := bcol.Value(i).(float64); !ok || v != want {
				t.Fatalf("nullEvery=%d row %d: b=%v, want %v", nullEvery, i, bcol.Value(i), want)
			}
		}
	}
}

// TestDropNaNsEquality checks drop_nans removes exactly the non-null NaN rows of
// a float column and leaves null rows in place (matching the prior semantics).
func TestDropNaNsEquality(t *testing.T) {
	vals := []any{1.0, math.NaN(), 3.0, nil, math.NaN(), 6.0}
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "x", Values: vals},
	}})
	if err != nil {
		t.Fatalf("new dataframe: %v", err)
	}
	got := df.DropNaNs()
	// NaN rows (1 and 4) are removed; the null row (3) is retained.
	want := []any{1.0, 3.0, nil, 6.0}
	if got.Height() != len(want) {
		t.Fatalf("height=%d, want %d", got.Height(), len(want))
	}
	xcol, err := got.GetColumn("x")
	if err != nil {
		t.Fatalf("GetColumn(x): %v", err)
	}
	for i, w := range want {
		gv := xcol.Value(i)
		if w == nil {
			if gv != nil {
				t.Fatalf("row %d: x=%v, want null", i, gv)
			}
			continue
		}
		if v, ok := gv.(float64); !ok || v != w.(float64) {
			t.Fatalf("row %d: x=%v, want %v", i, gv, w)
		}
	}
}

// TestDropNullsNoNullsShares proves the no-op share path: drop_nulls on a
// null-free frame returns the existing columns without gathering a copy. The
// only allocation is the public facade wrapper (one alloc); a gather of all
// 4096 rows across three columns would allocate many more.
func TestDropNullsNoNullsShares(t *testing.T) {
	df := buildDropFrame(t, 4096, 0)
	allocs := testing.AllocsPerRun(20, func() { _ = df.DropNulls() })
	if allocs > 1 {
		t.Fatalf("drop_nulls on a null-free frame allocated %.0f times, want <=1 (share path, not a full gather)", allocs)
	}
}

func BenchmarkDropNulls1M(b *testing.B) {
	df := buildDropFrame(b, 1_000_000, 10) // 10% null
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = df.DropNulls()
	}
}
