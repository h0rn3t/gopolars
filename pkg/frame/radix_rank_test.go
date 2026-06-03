package frame

import (
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// referenceOrdinalRankFloat reproduces the original comparison-sort rank: a
// stable ascending sort whose inverse permutation is the 1-based ordinal rank
// (equal values keep input order). The radix path must match this exactly.
func referenceOrdinalRankFloat(vals []float64) []int64 {
	idx := make([]int, len(vals))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return vals[idx[a]] < vals[idx[b]] })
	out := make([]int64, len(vals))
	for r, i := range idx {
		out[i] = int64(r + 1)
	}
	return out
}

func referenceOrdinalRankInt(vals []int64) []int64 {
	idx := make([]int, len(vals))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return vals[idx[a]] < vals[idx[b]] })
	out := make([]int64, len(vals))
	for r, i := range idx {
		out[i] = int64(r + 1)
	}
	return out
}

// TestRankRadixMatchesStableReference checks the O(n) radix rank produces the
// exact ordinal vector of the comparison-sort path on duplicate-heavy and
// mixed-sign data, for both Float64 and Int64, at a length above the radix gate.
func TestRankRadixMatchesStableReference(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	const n = 1000 // >= radixSortThreshold, so the radix path is exercised
	f := make([]float64, n)
	iv := make([]int64, n)
	for i := range f {
		f[i] = float64(r.Intn(50) - 25) // many duplicates, mixed sign
		iv[i] = int64(r.Intn(50) - 25)
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromFloat64("f", f, nil),
		series.FromInt64("iv", iv, nil),
	}})
	if err != nil {
		t.Fatal(err)
	}

	gotF, err := df.Select(expr.Col("f").Rank())
	if err != nil {
		t.Fatal(err)
	}
	gotI, err := df.Select(expr.Col("iv").Rank())
	if err != nil {
		t.Fatal(err)
	}
	wantF := referenceOrdinalRankFloat(f)
	wantI := referenceOrdinalRankInt(iv)
	fc := gotF.cols[gotF.order[0]]
	ic := gotI.cols[gotI.order[0]]
	for i := 0; i < n; i++ {
		if fc.Value(i).(int64) != wantF[i] {
			t.Fatalf("float rank[%d] = %v, want %d", i, fc.Value(i), wantF[i])
		}
		if ic.Value(i).(int64) != wantI[i] {
			t.Fatalf("int rank[%d] = %v, want %d", i, ic.Value(i), wantI[i])
		}
	}
}

// TestRankRadixEqualValuesKeepInputOrder asserts the radix rank is stable: for
// equal values at i < j, position i receives the smaller rank.
func TestRankRadixEqualValuesKeepInputOrder(t *testing.T) {
	const n = 512 // above the radix gate
	v := make([]float64, n)
	for i := range v {
		v[i] = float64(i % 4) // heavy duplication across the four values
	}
	df, err := New(NewInput{Series: []series.Series{series.FromFloat64("v", v, nil)}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := df.Select(expr.Col("v").Rank())
	if err != nil {
		t.Fatal(err)
	}
	c := got.cols[got.order[0]]
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if v[i] == v[j] && c.Value(i).(int64) >= c.Value(j).(int64) {
				t.Fatalf("equal values at %d,%d out of input order: rank %v vs %v",
					i, j, c.Value(i), c.Value(j))
			}
		}
	}
}

// TestRankFallbackMatchesRowwise checks that inputs outside the radix fast path
// (NaN floats, nullable columns, non-numeric dtypes) fall back correctly and
// still match the row-wise reference path, all above the radix length gate.
func TestRankFallbackMatchesRowwise(t *testing.T) {
	const n = 600
	cases := []struct {
		name string
		mk   func() series.Series
	}{
		{"nan_float", func() series.Series {
			v := make([]float64, n)
			for i := range v {
				if i%37 == 0 {
					v[i] = math.NaN()
				} else {
					v[i] = float64((i * 13) % 50)
				}
			}
			return series.FromFloat64("v", v, nil)
		}},
		{"nullable_float", func() series.Series {
			v := make([]float64, n)
			nulls := make([]bool, n)
			for i := range v {
				v[i] = float64((i * 7) % 40)
				nulls[i] = i%29 == 0
			}
			return series.FromFloat64("v", v, nulls)
		}},
		{"nullable_int", func() series.Series {
			v := make([]int64, n)
			nulls := make([]bool, n)
			for i := range v {
				v[i] = int64((i * 5) % 40)
				nulls[i] = i%31 == 0
			}
			return series.FromInt64("v", v, nulls)
		}},
		{"string", func() series.Series {
			v := make([]string, n)
			for i := range v {
				v[i] = string(rune('a' + (i % 7)))
			}
			return series.FromString("v", v, nil)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mk := func() DataFrame {
				df, err := New(NewInput{Series: []series.Series{tc.mk()}})
				if err != nil {
					t.Fatal(err)
				}
				return df
			}
			typed, err := mk().Select(expr.Col("v").Rank())
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
			rowwise, err := mk().Select(expr.Col("v").Rank())
			if err != nil {
				t.Fatal(err)
			}
			tcol := typed.cols[typed.order[0]]
			rcol := rowwise.cols[rowwise.order[0]]
			for i := 0; i < n; i++ {
				if tcol.Value(i).(int64) != rcol.Value(i).(int64) {
					t.Fatalf("rank[%d]: typed %v rowwise %v", i, tcol.Value(i), rcol.Value(i))
				}
			}
		})
	}
}

// BenchmarkOverRank exercises the per-partition radix rank path: buildRankBenchFrame
// has four groups over 1M rows, so each partition is ~250K rows (above the gate).
func BenchmarkOverRank(b *testing.B) {
	df := buildRankBenchFrame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Select(expr.Col("v").Rank().Over("g")); err != nil {
			b.Fatal(err)
		}
	}
}

// TestOverRankRadixParity checks rank-within-over: large partitions take the
// per-partition radix path and must match the row-wise fallback exactly.
func TestOverRankRadixParity(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	const n = 2000 // two groups -> ~1000-row partitions, above the radix gate
	g := make([]string, n)
	v := make([]float64, n)
	iv := make([]int64, n)
	for i := range v {
		g[i] = []string{"a", "b"}[i%2]
		v[i] = float64(r.Intn(30)) // duplicate-heavy within each partition
		iv[i] = int64(r.Intn(30))
	}
	mk := func() DataFrame {
		df, err := New(NewInput{Series: []series.Series{
			series.FromString("g", append([]string(nil), g...), nil),
			series.FromFloat64("v", append([]float64(nil), v...), nil),
			series.FromInt64("iv", append([]int64(nil), iv...), nil),
		}})
		if err != nil {
			t.Fatal(err)
		}
		return df
	}
	for _, col := range []string{"v", "iv"} {
		typed, err := mk().Select(expr.Col(col).Rank().Over("g"))
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
		rowwise, err := mk().Select(expr.Col(col).Rank().Over("g"))
		if err != nil {
			t.Fatal(err)
		}
		tc := typed.cols[typed.order[0]]
		rc := rowwise.cols[rowwise.order[0]]
		for i := 0; i < n; i++ {
			if tc.Value(i).(int64) != rc.Value(i).(int64) {
				t.Fatalf("over-rank %s[%d]: typed %v rowwise %v", col, i, tc.Value(i), rc.Value(i))
			}
		}
		t.Setenv("GOPOLARS_TYPED_STORAGE", "1")
	}
}
