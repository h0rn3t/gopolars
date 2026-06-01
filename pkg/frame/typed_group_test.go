package frame

import (
	"math"
	"sort"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// collectGroupSums returns a map of group-key -> aggregated sum so results can
// be compared independent of group ordering.
func collectGroupSums(t *testing.T, df DataFrame, keyCol, sumCol string) map[string]float64 {
	t.Helper()
	out := map[string]float64{}
	k := df.cols[keyCol]
	v := df.cols[sumCol]
	for i := 0; i < df.height; i++ {
		key := ""
		if !k.IsNull(i) {
			key = k.Value(i).(string)
		} else {
			key = "<null>"
		}
		switch t := v.Value(i).(type) {
		case float64:
			out[key] = t
		case int64:
			out[key] = float64(t)
		case nil:
			out[key] = math.NaN()
		}
	}
	return out
}

func TestGroupByTypedMatchesExpected(t *testing.T) {
	g := series.FromString("g", []string{"a", "b", "a", "b", "a"}, nil)
	v := series.FromFloat64("v", []float64{1, 10, 2, 20, 3}, nil)
	df, _ := New(NewInput{Series: []series.Series{g, v}})

	got, err := df.GroupBy("g").Agg(expr.Sum(expr.Col("v")))
	if err != nil {
		t.Fatalf("group_by: %v", err)
	}
	if got.height != 2 {
		t.Fatalf("groups = %d, want 2", got.height)
	}
	sums := collectGroupSums(t, got, "g", "sum_v")
	if sums["a"] != 6 {
		t.Errorf("sum(a) = %v, want 6", sums["a"])
	}
	if sums["b"] != 30 {
		t.Errorf("sum(b) = %v, want 30", sums["b"])
	}
}

func TestGroupByTypedNullKey(t *testing.T) {
	g := series.FromString("g", []string{"a", "", "a", ""}, []bool{false, true, false, true})
	v := series.FromInt64("v", []int64{1, 5, 2, 7}, nil)
	df, _ := New(NewInput{Series: []series.Series{g, v}})

	got, err := df.GroupBy("g").Agg(expr.Sum(expr.Col("v")))
	if err != nil {
		t.Fatalf("group_by: %v", err)
	}
	// Two groups: "a" -> 3, null -> 12.
	if got.height != 2 {
		t.Fatalf("groups = %d, want 2", got.height)
	}
	sums := collectGroupSums(t, got, "g", "sum_v")
	if sums["a"] != 3 {
		t.Errorf("sum(a) = %v, want 3", sums["a"])
	}
	if sums["<null>"] != 12 {
		t.Errorf("sum(null) = %v, want 12", sums["<null>"])
	}
}

func TestGroupByTypedMultiKey(t *testing.T) {
	a := series.FromString("a", []string{"x", "x", "y", "x"}, nil)
	b := series.FromInt64("b", []int64{1, 2, 1, 1}, nil)
	v := series.FromFloat64("v", []float64{10, 20, 30, 5}, nil)
	df, _ := New(NewInput{Series: []series.Series{a, b, v}})

	got, err := df.GroupBy("a", "b").Agg(expr.Sum(expr.Col("v")))
	if err != nil {
		t.Fatalf("group_by: %v", err)
	}
	// (x,1)->15, (x,2)->20, (y,1)->30.
	if got.height != 3 {
		t.Fatalf("groups = %d, want 3", got.height)
	}
	total := 0.0
	for i := 0; i < got.height; i++ {
		total += got.cols["sum_v"].Value(i).(float64)
	}
	if total != 65 {
		t.Errorf("total of group sums = %v, want 65", total)
	}
}

func TestUniqueTypedSingleColumn(t *testing.T) {
	g := series.FromString("g", []string{"a", "b", "a", "c", "b"}, nil)
	v := series.FromInt64("v", []int64{1, 2, 3, 4, 5}, nil)
	df, _ := New(NewInput{Series: []series.Series{g, v}})

	got, err := df.Unique("g")
	if err != nil {
		t.Fatalf("unique: %v", err)
	}
	if got.height != 3 {
		t.Fatalf("unique rows = %d, want 3", got.height)
	}
	// First-occurrence order preserved: a(1), b(2), c(4).
	var keys []string
	for i := 0; i < got.height; i++ {
		keys = append(keys, got.cols["g"].Value(i).(string))
	}
	sort.Strings(keys)
	want := []string{"a", "b", "c"}
	for i := range want {
		if keys[i] != want[i] {
			t.Errorf("unique keys = %v, want %v", keys, want)
			break
		}
	}
}

func TestNUniqueTyped(t *testing.T) {
	g := series.FromFloat64("g", []float64{1.5, 2.5, 1.5, math.NaN(), math.NaN(), 2.5}, nil)
	df, _ := New(NewInput{Series: []series.Series{g}})
	n, err := df.NUnique("g")
	if err != nil {
		t.Fatalf("nunique: %v", err)
	}
	// distinct: 1.5, 2.5, NaN -> 3
	if n != 3 {
		t.Errorf("nunique = %d, want 3", n)
	}
}

// TestGroupByTypedVsRowWise checks the typed path matches the row-wise fallback
// (GOPOLARS_TYPED_STORAGE=0) on the same data.
func TestGroupByTypedVsRowWise(t *testing.T) {
	mk := func() DataFrame {
		g := series.FromString("g", []string{"a", "b", "a", "b", "c", "a"}, nil)
		v := series.FromFloat64("v", []float64{1, 2, 3, 4, 5, 6}, nil)
		df, _ := New(NewInput{Series: []series.Series{g, v}})
		return df
	}
	typed, err := mk().GroupBy("g").Agg(expr.Sum(expr.Col("v")))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
	rowwise, err := mk().GroupBy("g").Agg(expr.Sum(expr.Col("v")))
	if err != nil {
		t.Fatal(err)
	}
	ts := collectGroupSums(t, typed, "g", "sum_v")
	rs := collectGroupSums(t, rowwise, "g", "sum_v")
	if len(ts) != len(rs) {
		t.Fatalf("group count mismatch: typed %d rowwise %d", len(ts), len(rs))
	}
	for k, v := range ts {
		if rs[k] != v {
			t.Errorf("group %q: typed %v rowwise %v", k, v, rs[k])
		}
	}
}

func buildGroupBenchFrame(b *testing.B, n, groups int) DataFrame {
	b.Helper()
	g := make([]string, n)
	v := make([]float64, n)
	labels := make([]string, groups)
	for i := range labels {
		labels[i] = "grp" + string(rune('A'+i%26)) + string(rune('0'+i/26))
	}
	for i := 0; i < n; i++ {
		g[i] = labels[i%groups]
		v[i] = float64(i)
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromString("g", g, nil),
		series.FromFloat64("v", v, nil),
	}})
	if err != nil {
		b.Fatal(err)
	}
	return df
}

// BenchmarkGroupBySum: allocations should scale with group count, not rows.
func BenchmarkGroupBySum(b *testing.B) {
	df := buildGroupBenchFrame(b, 1_000_000, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("g").Agg(expr.Sum(expr.Col("v"))); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnique(b *testing.B) {
	df := buildGroupBenchFrame(b, 1_000_000, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Unique("g"); err != nil {
			b.Fatal(err)
		}
	}
}
