package frame

import (
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/series"
)

// stableArgsortFloat is the reference stable ascending permutation the
// parallel-merge radix must reproduce exactly.
func stableArgsortFloat(v []float64) []int {
	idx := make([]int, len(v))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return v[idx[a]] < v[idx[b]] })
	return idx
}

// TestParallelSortMatchesStableReference checks that a single-key sort above the
// parallel-merge threshold produces a permutation identical to a stable
// reference, on duplicate-heavy data (stability must hold across run boundaries).
func TestParallelSortMatchesStableReference(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	const n = 100000 // above parallelMergeThreshold so the parallel-merge path runs
	v := make([]float64, n)
	id := make([]int64, n)
	for i := range v {
		v[i] = float64(r.Intn(200)) // heavy duplication -> ties span many ranges
		id[i] = int64(i)
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromFloat64("v", v, nil),
		series.FromInt64("id", id, nil),
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := df.Sort(SortInput{By: []string{"v"}})
	if err != nil {
		t.Fatal(err)
	}
	// The "id" column in sorted order is the permutation the sort produced.
	ref := stableArgsortFloat(v)
	idc := got.cols["id"]
	for i := 0; i < n; i++ {
		if idc.Value(i).(int64) != int64(ref[i]) {
			t.Fatalf("permutation mismatch at %d: got id=%v, want %d", i, idc.Value(i), ref[i])
		}
	}
}

// TestParallelSortStabilityAcrossRuns checks equal-key rows keep ascending input
// order even when duplicates straddle the parallel range boundaries.
func TestParallelSortStabilityAcrossRuns(t *testing.T) {
	const n = 100000
	v := make([]float64, n)
	id := make([]int64, n)
	for i := range v {
		v[i] = float64(i % 8) // only 8 distinct values -> each spans every range
		id[i] = int64(i)
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromFloat64("v", v, nil),
		series.FromInt64("id", id, nil),
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := df.Sort(SortInput{By: []string{"v"}})
	if err != nil {
		t.Fatal(err)
	}
	vc := got.cols["v"]
	idc := got.cols["id"]
	for i := 1; i < n; i++ {
		if vc.Value(i-1).(float64) > vc.Value(i).(float64) {
			t.Fatalf("not sorted at %d", i)
		}
		// Within an equal-value run, ids must be ascending (stable).
		if vc.Value(i-1).(float64) == vc.Value(i).(float64) &&
			idc.Value(i-1).(int64) >= idc.Value(i).(int64) {
			t.Fatalf("stability broken at %d: ids %v then %v", i, idc.Value(i-1), idc.Value(i))
		}
	}
}

// TestParallelSortDescending checks descending order on a large single-key sort.
func TestParallelSortDescending(t *testing.T) {
	const n = 100000
	v := make([]float64, n)
	for i := range v {
		v[i] = float64((i*2654435761)%n) - float64(n/2)
	}
	df, err := New(NewInput{Series: []series.Series{series.FromFloat64("v", v, nil)}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := df.Sort(SortInput{By: []string{"v"}, Descending: []bool{true}})
	if err != nil {
		t.Fatal(err)
	}
	c := got.cols["v"]
	for i := 1; i < n; i++ {
		if c.Value(i-1).(float64) < c.Value(i).(float64) {
			t.Fatalf("descending order broken at %d", i)
		}
	}
}

// TestParallelSortNaNAndNullFallback checks that NaN-containing and nullable
// numeric columns decline the radix path and still produce the existing
// comparator order (NaN sorts last; nulls placed per NullsLast), at a length
// above the parallel threshold.
func TestParallelSortNaNAndNullFallback(t *testing.T) {
	const n = 100000
	t.Run("nan_last", func(t *testing.T) {
		v := make([]float64, n)
		nanCount := 0
		for i := range v {
			if i%1000 == 0 {
				v[i] = math.NaN()
				nanCount++
			} else {
				v[i] = float64(r0(i))
			}
		}
		df, _ := New(NewInput{Series: []series.Series{series.FromFloat64("v", v, nil)}})
		got, err := df.Sort(SortInput{By: []string{"v"}})
		if err != nil {
			t.Fatal(err)
		}
		c := got.cols["v"]
		// Non-NaN prefix ascending; all NaN at the tail.
		for i := 1; i < n-nanCount; i++ {
			if c.Value(i-1).(float64) > c.Value(i).(float64) {
				t.Fatalf("non-NaN prefix not sorted at %d", i)
			}
		}
		for i := n - nanCount; i < n; i++ {
			if !math.IsNaN(c.Value(i).(float64)) {
				t.Fatalf("expected NaN at tail index %d, got %v", i, c.Value(i))
			}
		}
	})
	t.Run("nulls_last", func(t *testing.T) {
		v := make([]float64, n)
		nulls := make([]bool, n)
		nullCount := 0
		for i := range v {
			v[i] = float64(r0(i))
			if i%1000 == 0 {
				nulls[i] = true
				nullCount++
			}
		}
		df, _ := New(NewInput{Series: []series.Series{series.FromFloat64("v", v, nulls)}})
		got, err := df.Sort(SortInput{By: []string{"v"}, NullsLast: true})
		if err != nil {
			t.Fatal(err)
		}
		c := got.cols["v"]
		for i := n - nullCount; i < n; i++ {
			if !c.IsNull(i) {
				t.Fatalf("expected null at tail index %d", i)
			}
		}
	})
}

func r0(i int) int { return (i * 7919) % 500 }

// stableMultiKeyRef is the reference stable lexicographic permutation for a
// (lead, sec) sort with sec ascending.
func stableMultiKeyRef(lead []float64, sec []int64, leadDesc bool) []int {
	idx := make([]int, len(lead))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		la, lb := lead[idx[a]], lead[idx[b]]
		if la != lb {
			if leadDesc {
				return la > lb
			}
			return la < lb
		}
		return sec[idx[a]] < sec[idx[b]]
	})
	return idx
}

// TestParallelSortMultiKey checks the multi-key radix path: order by the leading
// numeric key, ties resolved by the secondary key. Ascending-leading matches the
// stable reference exactly; descending-leading is checked for ordering
// correctness (leading desc, secondary asc within ties).
func TestParallelSortMultiKey(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	const n = 100000
	lead := make([]float64, n)
	sec := make([]int64, n)
	id := make([]int64, n)
	for i := range lead {
		lead[i] = float64(r.Intn(100)) // duplicate-heavy leading key
		sec[i] = int64(r.Intn(1000))
		id[i] = int64(i)
	}
	mk := func() DataFrame {
		df, err := New(NewInput{Series: []series.Series{
			series.FromFloat64("lead", append([]float64(nil), lead...), nil),
			series.FromInt64("sec", append([]int64(nil), sec...), nil),
			series.FromInt64("id", append([]int64(nil), id...), nil),
		}})
		if err != nil {
			t.Fatal(err)
		}
		return df
	}

	t.Run("ascending_matches_stable_ref", func(t *testing.T) {
		got, err := mk().Sort(SortInput{By: []string{"lead", "sec"}})
		if err != nil {
			t.Fatal(err)
		}
		ref := stableMultiKeyRef(lead, sec, false)
		idc := got.cols["id"]
		for i := 0; i < n; i++ {
			if idc.Value(i).(int64) != int64(ref[i]) {
				t.Fatalf("multi-key permutation mismatch at %d: got %v want %d", i, idc.Value(i), ref[i])
			}
		}
	})

	t.Run("descending_lead_ordering", func(t *testing.T) {
		got, err := mk().Sort(SortInput{By: []string{"lead", "sec"}, Descending: []bool{true, false}})
		if err != nil {
			t.Fatal(err)
		}
		lc := got.cols["lead"]
		sc := got.cols["sec"]
		for i := 1; i < n; i++ {
			l0, l1 := lc.Value(i-1).(float64), lc.Value(i).(float64)
			if l0 < l1 {
				t.Fatalf("leading key not descending at %d: %v then %v", i, l0, l1)
			}
			if l0 == l1 && sc.Value(i-1).(int64) > sc.Value(i).(int64) {
				t.Fatalf("secondary not ascending within tie at %d", i)
			}
		}
	})
}

// BenchmarkSortMultiKey measures the multi-key radix path (leading numeric key +
// secondary tie-break) at 1M rows.
func BenchmarkSortMultiKey(b *testing.B) {
	const n = 1_000_000
	lead := make([]float64, n)
	sec := make([]int64, n)
	for i := range lead {
		lead[i] = float64((i * 2654435761) % 5000)
		sec[i] = int64((i * 40503) % 1000)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		df, _ := New(NewInput{Series: []series.Series{
			series.FromFloat64("lead", append([]float64(nil), lead...), nil),
			series.FromInt64("sec", append([]int64(nil), sec...), nil),
		}})
		b.StartTimer()
		if _, err := df.Sort(SortInput{By: []string{"lead", "sec"}}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSortFloat64Small confirms small sorts (below the parallel-merge
// threshold) stay on the sequential radix and do not regress.
func BenchmarkSortFloat64Small(b *testing.B) {
	const n = 1000
	base := make([]float64, n)
	for i := range base {
		base[i] = float64((i * 2654435761) % n)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		df, _ := New(NewInput{Series: []series.Series{series.FromFloat64("v", append([]float64(nil), base...), nil)}})
		b.StartTimer()
		if _, err := df.Sort(SortInput{By: []string{"v"}}); err != nil {
			b.Fatal(err)
		}
	}
}
