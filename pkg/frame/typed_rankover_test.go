package frame

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

func TestRankTypedParity(t *testing.T) {
	mk := func() DataFrame {
		v := series.FromFloat64("v", []float64{3, 1, 2, 1, 5}, nil)
		df, _ := New(NewInput{Series: []series.Series{v}})
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
	tc := typed.cols[typed.order[0]]
	rc := rowwise.cols[rowwise.order[0]]
	for i := 0; i < typed.height; i++ {
		if tc.Value(i).(int64) != rc.Value(i).(int64) {
			t.Errorf("rank[%d]: typed %v rowwise %v", i, tc.Value(i), rc.Value(i))
		}
	}
}

func TestRankIsOrdinalPermutation(t *testing.T) {
	v := series.FromFloat64("v", []float64{30, 10, 20}, nil)
	df, _ := New(NewInput{Series: []series.Series{v}})
	got, err := df.Select(expr.Col("v").Rank())
	if err != nil {
		t.Fatal(err)
	}
	c := got.cols[got.order[0]]
	// values 10,20,30 -> ranks 1,2,3 at original positions [30=3,10=1,20=2].
	want := []int64{3, 1, 2}
	for i, w := range want {
		if c.Value(i).(int64) != w {
			t.Errorf("rank[%d] = %v, want %v", i, c.Value(i), w)
		}
	}
}

func TestOverCumSumTypedParity(t *testing.T) {
	mk := func() DataFrame {
		g := series.FromString("g", []string{"a", "b", "a", "b", "a"}, nil)
		v := series.FromFloat64("v", []float64{1, 2, 3, 4, 5}, nil)
		df, _ := New(NewInput{Series: []series.Series{g, v}})
		return df
	}
	typed, err := mk().Select(expr.Col("v").CumSum().Over("g"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
	rowwise, err := mk().Select(expr.Col("v").CumSum().Over("g"))
	if err != nil {
		t.Fatal(err)
	}
	tc := typed.cols[typed.order[0]]
	rc := rowwise.cols[rowwise.order[0]]
	for i := 0; i < typed.height; i++ {
		if tc.Value(i).(float64) != rc.Value(i).(float64) {
			t.Errorf("over[%d]: typed %v rowwise %v", i, tc.Value(i), rc.Value(i))
		}
	}
}

func buildRankBenchFrame() DataFrame {
	n := 1_000_000
	v := make([]float64, n)
	g := make([]string, n)
	for i := range v {
		v[i] = float64((i * 2654435761) % 1000)
		g[i] = []string{"a", "b", "c", "d"}[i%4]
	}
	df, _ := New(NewInput{Series: []series.Series{
		series.FromString("g", g, nil),
		series.FromFloat64("v", v, nil),
	}})
	return df
}

func BenchmarkRank(b *testing.B) {
	df := buildRankBenchFrame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Select(expr.Col("v").Rank()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOverCumSum(b *testing.B) {
	df := buildRankBenchFrame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Select(expr.Col("v").CumSum().Over("g")); err != nil {
			b.Fatal(err)
		}
	}
}
