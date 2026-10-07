package frame

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// lowCardinalityFrame has n rows: an Int64 key "k" and a String key "s" taking
// 7 values in a scrambled order, a copy of "k" with every 5th row null ("kn"),
// a String key "s300" with 300 values, a Datetime key "ts" with 1000 values, a
// Boolean key "b", and the input row number "seq".
func lowCardinalityFrame(t testing.TB, n int) DataFrame {
	t.Helper()
	k := make([]int64, n)
	s := make([]string, n)
	kn := make([]int64, n)
	nulls := make([]bool, n)
	s300 := make([]string, n)
	ts := make([]time.Time, n)
	b := make([]bool, n)
	seq := make([]int64, n)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range n {
		scrambled := i * 2654435761
		k[i] = int64(scrambled) % 7
		s[i] = fmt.Sprintf("v%d", k[i])
		kn[i] = k[i]
		nulls[i] = i%5 == 0
		s300[i] = fmt.Sprintf("zone-%d", scrambled%300)
		ts[i] = base.Add(time.Duration(scrambled%1000) * time.Hour)
		b[i] = scrambled%3 == 0
		seq[i] = int64(i)
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromInt64("k", k, nil),
		series.FromString("s", s, nil),
		series.FromInt64("kn", kn, nulls),
		series.FromString("s300", s300, nil),
		series.FromColumn("ts", chunk.NewTime(ts, nil)),
		series.FromBool("b", b, nil),
		series.FromInt64("seq", seq, nil),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return df
}

// checkStableSort reports rows of got that are out of order by key, or whose
// seq does not increase within a run of equal keys. Nulls must all sit at the
// end when nullsLast, else at the start.
func checkStableSort(t *testing.T, got DataFrame, key string, desc, nullsLast bool) {
	t.Helper()
	keys, _ := got.Series(key)
	seq := columnInt64s(t, got, "seq")
	outOfOrder, unstable, misplacedNull := 0, 0, 0
	for i := 1; i < got.Height(); i++ {
		prevNull, curNull := keys.IsNull(i-1), keys.IsNull(i)
		switch {
		case prevNull != curNull:
			if prevNull == nullsLast {
				misplacedNull++
			}
			continue
		case prevNull:
			// Both null: equal keys.
		default:
			c := compareAny(keys.Value(i-1), keys.Value(i))
			if desc {
				c = -c
			}
			if c > 0 {
				outOfOrder++
				continue
			}
			if c < 0 {
				continue
			}
		}
		if seq[i] < seq[i-1] {
			unstable++
		}
	}
	if outOfOrder+unstable+misplacedNull > 0 {
		t.Errorf("Sort(%s, desc=%v, nullsLast=%v) on %d rows: %d out of order, %d equal-key pairs out of input order, %d misplaced nulls",
			key, desc, nullsLast, got.Height(), outOfOrder, unstable, misplacedNull)
	}
}

func TestSortKeepsEqualKeysInInputOrder(t *testing.T) {
	t.Parallel()
	df := lowCardinalityFrame(t, 100_000)
	for _, key := range []string{"k", "s", "kn"} {
		for _, desc := range []bool{false, true} {
			for _, nullsLast := range []bool{false, true} {
				got, err := df.Sort(SortInput{By: []string{key}, Descending: []bool{desc}, NullsLast: nullsLast})
				if err != nil {
					t.Fatalf("Sort(%s): %v", key, err)
				}
				checkStableSort(t, got, key, desc, nullsLast)
			}
		}
	}
}

func BenchmarkSortLowCardinality(b *testing.B) {
	df := lowCardinalityFrame(b, 100_000)
	for _, by := range [][]string{{"k"}, {"s"}, {"kn"}, {"s300"}, {"ts"}, {"b"}, {"b", "ts", "s300"}} {
		for _, desc := range []bool{false, true} {
			in := SortInput{By: by, Descending: []bool{desc, !desc}}
			b.Run(fmt.Sprintf("%s/desc=%v", strings.Join(by, ","), desc), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := df.Sort(in); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func TestSortMaintainOrderDoesNotChangeResult(t *testing.T) {
	t.Parallel()
	df := lowCardinalityFrame(t, 10_000)
	for _, by := range [][]string{{"k"}, {"s"}, {"kn", "s"}} {
		for _, desc := range []bool{false, true} {
			plain, err := df.Sort(SortInput{By: by, Descending: []bool{desc}})
			if err != nil {
				t.Fatalf("Sort(%v): %v", by, err)
			}
			kept, err := df.Sort(SortInput{By: by, Descending: []bool{desc}, MaintainOrder: true})
			if err != nil {
				t.Fatalf("Sort(%v, MaintainOrder): %v", by, err)
			}
			if a, b := columnInt64s(t, plain, "seq"), columnInt64s(t, kept, "seq"); !slices.Equal(a, b) {
				t.Errorf("Sort(%v, desc=%v) row order differs with MaintainOrder", by, desc)
			}
		}
	}
}

func TestSortNullsLastIgnoresDirection(t *testing.T) {
	t.Parallel()
	df, err := New(NewInput{Series: []series.Series{
		series.FromInt64("k", []int64{2, 0, 3, 1}, []bool{false, true, false, false}),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	tests := []struct {
		desc, nullsLast bool
		want            []any
	}{
		{false, false, []any{nil, int64(1), int64(2), int64(3)}},
		{false, true, []any{int64(1), int64(2), int64(3), nil}},
		{true, false, []any{nil, int64(3), int64(2), int64(1)}},
		{true, true, []any{int64(3), int64(2), int64(1), nil}},
	}
	for _, tt := range tests {
		got, err := df.Sort(SortInput{By: []string{"k"}, Descending: []bool{tt.desc}, NullsLast: tt.nullsLast})
		if err != nil {
			t.Fatalf("Sort: %v", err)
		}
		s, _ := got.Series("k")
		vals := make([]any, s.Len())
		for i := range vals {
			vals[i] = s.Value(i)
		}
		if !slices.Equal(vals, tt.want) {
			t.Errorf("Sort([2 null 3 1], desc=%v, nullsLast=%v) = %v, want %v", tt.desc, tt.nullsLast, vals, tt.want)
		}
	}
}

func TestSortNullsLastOnDescendingSecondaryKey(t *testing.T) {
	t.Parallel()
	df, err := New(NewInput{Series: []series.Series{
		series.FromInt64("a", []int64{1, 1, 0}, nil),
		series.FromInt64("b", []int64{0, 5, 9}, []bool{true, false, false}),
		series.FromInt64("seq", []int64{0, 1, 2}, nil),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := df.Sort(SortInput{By: []string{"a", "b"}, Descending: []bool{false, true}, NullsLast: true})
	if err != nil {
		t.Fatalf("Sort(a, b): %v", err)
	}
	if seq := columnInt64s(t, got, "seq"); !slices.Equal(seq, []int64{2, 1, 0}) {
		t.Errorf("Sort(a asc, b desc, nullsLast) = rows %v, want [2 1 0]", seq)
	}
}

func TestSortNaNFollowsDirection(t *testing.T) {
	t.Parallel()
	nan := math.NaN()
	df, err := New(NewInput{Series: []series.Series{
		series.FromColumn("f", chunk.NewFloat64([]float64{2, nan, 1}, nil)),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, tt := range []struct {
		desc bool
		want []float64
	}{
		{false, []float64{1, 2, nan}},
		{true, []float64{nan, 2, 1}},
	} {
		got, err := df.Sort(SortInput{By: []string{"f"}, Descending: []bool{tt.desc}})
		if err != nil {
			t.Fatalf("Sort(f): %v", err)
		}
		s, _ := got.Series("f")
		vals, _ := s.Column().Float64s()
		if !slices.EqualFunc(vals, tt.want, func(a, b float64) bool { return a == b || math.IsNaN(a) && math.IsNaN(b) }) {
			t.Errorf("Sort([2 NaN 1], desc=%v) = %v, want %v", tt.desc, vals, tt.want)
		}
	}
}
