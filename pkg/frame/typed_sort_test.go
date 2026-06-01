package frame

import (
	"math/rand"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/series"
)

// TestSortRadixMatchesComparator checks the radix fast path yields the same
// sorted key order and a row permutation matching the comparator sort.
func TestSortRadixMatchesComparator(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	n := 5000
	v := make([]float64, n)
	id := make([]int64, n)
	for i := range v {
		v[i] = r.Float64()*1000 - 500
		id[i] = int64(i)
	}
	mk := func() DataFrame {
		df, _ := New(NewInput{Series: []series.Series{
			series.FromFloat64("v", append([]float64(nil), v...), nil),
			series.FromInt64("id", append([]int64(nil), id...), nil),
		}})
		return df
	}
	got, err := mk().Sort(SortInput{By: []string{"v"}})
	if err != nil {
		t.Fatal(err)
	}
	// Sorted ascending on v.
	col := got.cols["v"]
	for i := 1; i < got.height; i++ {
		if col.Value(i-1).(float64) > col.Value(i).(float64) {
			t.Fatalf("not sorted at %d", i)
		}
	}
	// Row integrity: each (v,id) pair preserved (id maps to original v).
	idc := got.cols["id"]
	for i := 0; i < got.height; i++ {
		origID := idc.Value(i).(int64)
		if col.Value(i).(float64) != v[origID] {
			t.Fatalf("row corrupted at %d: v=%v but id=%d had v=%v", i, col.Value(i), origID, v[origID])
		}
	}
}

func TestSortDescendingRadix(t *testing.T) {
	df, _ := New(NewInput{Series: []series.Series{
		series.FromFloat64("v", makeSeq(1000), nil),
	}})
	got, err := df.Sort(SortInput{By: []string{"v"}, Descending: []bool{true}})
	if err != nil {
		t.Fatal(err)
	}
	col := got.cols["v"]
	for i := 1; i < got.height; i++ {
		if col.Value(i-1).(float64) < col.Value(i).(float64) {
			t.Fatalf("descending sort broken at %d", i)
		}
	}
}

func makeSeq(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = float64((i*7919)%n) - float64(n/2)
	}
	return out
}

func BenchmarkSortFloat64(b *testing.B) {
	n := 1_000_000
	base := make([]float64, n)
	for i := range base {
		base[i] = float64((i*2654435761)%n) - float64(n/2)
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
