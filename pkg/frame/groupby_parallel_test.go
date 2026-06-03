package frame

import (
	"math"
	"math/rand"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// valEqual compares two boxed aggregate/key values. NaN==NaN is treated as equal
// (so NaN keys compare). Finite floats are compared within a small relative
// tolerance: the parallel path reduces shard-partial sums in a different order
// than the sequential element-order fold, and floating-point addition is not
// associative, so float sum/mean match only up to a few ULPs. Integer sums and
// min/max are associative and compare exactly.
func valEqual(a, b any) bool {
	switch av := a.(type) {
	case float64:
		bv, ok := b.(float64)
		if !ok {
			return false
		}
		if math.IsNaN(av) || math.IsNaN(bv) {
			return math.IsNaN(av) && math.IsNaN(bv)
		}
		diff := math.Abs(av - bv)
		scale := math.Max(1, math.Max(math.Abs(av), math.Abs(bv)))
		return diff <= 1e-9*scale
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	default:
		return a == b
	}
}

// assertFramesEqual checks two frames have the same columns, the same row order,
// and the same values (including null placement) — i.e. identical results, not
// merely equal as sets. This validates the deterministic group ordering too.
func assertGroupFramesEqual(t *testing.T, label string, got, want DataFrame) {
	t.Helper()
	if got.height != want.height {
		t.Fatalf("%s: height %d vs %d", label, got.height, want.height)
	}
	if len(got.order) != len(want.order) {
		t.Fatalf("%s: columns %v vs %v", label, got.order, want.order)
	}
	for ci, name := range want.order {
		if got.order[ci] != name {
			t.Fatalf("%s: col %d = %q, want %q", label, ci, got.order[ci], name)
		}
		gc := got.cols[name]
		wc := want.cols[name]
		for r := 0; r < want.height; r++ {
			gn, wn := gc.IsNull(r), wc.IsNull(r)
			if gn != wn {
				t.Fatalf("%s: %s[%d] null %v vs %v", label, name, r, gn, wn)
			}
			if wn {
				continue
			}
			if !valEqual(gc.Value(r), wc.Value(r)) {
				t.Fatalf("%s: %s[%d] = %v, want %v", label, name, r, gc.Value(r), wc.Value(r))
			}
		}
	}
}

// buildParallelGroupFrame builds a >threshold frame with varied keys and values:
// a nullable string key, an int key, a float key with NaN, and float/int value
// columns carrying nulls and NaN so sum/mean/min/max null handling is exercised.
func buildParallelGroupFrame(n int) DataFrame {
	r := rand.New(rand.NewSource(20260602))
	gs := make([]string, n)
	gsNull := make([]bool, n)
	gi := make([]int64, n)
	gf := make([]float64, n)
	v := make([]float64, n)
	vNull := make([]bool, n)
	iv := make([]int64, n)
	ivNull := make([]bool, n)
	for i := 0; i < n; i++ {
		gs[i] = "k" + string(rune('A'+r.Intn(37)))
		gsNull[i] = i%53 == 0 // scattered null keys -> one null group
		gi[i] = int64(r.Intn(60))
		switch r.Intn(20) {
		case 0:
			gf[i] = math.NaN() // NaN keys must canonicalize to one group
		default:
			gf[i] = float64(r.Intn(50))
		}
		switch r.Intn(15) {
		case 0:
			v[i] = math.NaN() // NaN values are skipped by sum/mean/min/max
		default:
			v[i] = r.NormFloat64() * 100
		}
		vNull[i] = i%41 == 0
		iv[i] = int64(r.Intn(1000) - 500)
		ivNull[i] = i%47 == 0
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromString("gs", gs, gsNull),
		series.FromInt64("gi", gi, nil),
		series.FromFloat64("gf", gf, nil),
		series.FromFloat64("v", v, vNull),
		series.FromInt64("iv", iv, ivNull),
	}})
	if err != nil {
		panic(err)
	}
	return df
}

// TestGroupByParallelParity checks the parallel path (typed storage on) produces
// byte-identical results to the sequential typed path (GOPOLARS_TYPED_STORAGE=0)
// across single/multi/null/float keys and every associative aggregate.
func TestGroupByParallelParity(t *testing.T) {
	const n = 50000 // above parallelGroupByThreshold so the parallel path runs
	df := buildParallelGroupFrame(n)

	cases := []struct {
		name  string
		keys  []string
		exprs []expr.Expr
	}{
		{"string_key_float_aggs", []string{"gs"}, []expr.Expr{
			expr.Sum(expr.Col("v")), expr.Count(), expr.Mean(expr.Col("v")),
			expr.Min(expr.Col("v")), expr.Max(expr.Col("v")),
		}},
		{"int_key_int_aggs", []string{"gi"}, []expr.Expr{
			expr.Sum(expr.Col("iv")), expr.Count(), expr.Mean(expr.Col("iv")),
			expr.Min(expr.Col("iv")), expr.Max(expr.Col("iv")),
		}},
		{"float_key_nan", []string{"gf"}, []expr.Expr{
			expr.Sum(expr.Col("v")), expr.Count(),
		}},
		{"multi_key", []string{"gs", "gi"}, []expr.Expr{
			expr.Sum(expr.Col("v")), expr.Count(), expr.Min(expr.Col("iv")),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GOPOLARS_TYPED_STORAGE", "1") // parallel path
			par, err := df.GroupBy(tc.keys...).Agg(tc.exprs...)
			if err != nil {
				t.Fatalf("parallel: %v", err)
			}
			t.Setenv("GOPOLARS_TYPED_STORAGE", "0") // sequential typed path
			seq, err := df.GroupBy(tc.keys...).Agg(tc.exprs...)
			if err != nil {
				t.Fatalf("sequential: %v", err)
			}
			assertGroupFramesEqual(t, tc.name, par, seq)
		})
	}
}

// TestGroupByParallelNonAssociativeFallback checks that an n_unique aggregate
// (not an associative running reduce) declines the parallel path and still
// returns correct results matching the sequential path on a >threshold frame.
func TestGroupByParallelNonAssociativeFallback(t *testing.T) {
	const n = 50000
	df := buildParallelGroupFrame(n)

	t.Setenv("GOPOLARS_TYPED_STORAGE", "1")
	got, err := df.GroupBy("gs").Agg(expr.NUnique(expr.Col("iv")))
	if err != nil {
		t.Fatalf("typed: %v", err)
	}
	t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
	want, err := df.GroupBy("gs").Agg(expr.NUnique(expr.Col("iv")))
	if err != nil {
		t.Fatalf("sequential: %v", err)
	}
	assertGroupFramesEqual(t, "n_unique_fallback", got, want)
}

// TestGroupByParallelDeterministicOrder asserts the parallel path emits groups in
// ascending first-seen row order (matching the sequential encounter order) across
// repeated runs.
func TestGroupByParallelDeterministicOrder(t *testing.T) {
	const n = 40000
	df := buildParallelGroupFrame(n)
	first, err := df.GroupBy("gs").Agg(expr.Sum(expr.Col("v")))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		again, err := df.GroupBy("gs").Agg(expr.Sum(expr.Col("v")))
		if err != nil {
			t.Fatal(err)
		}
		assertGroupFramesEqual(t, "repeat", again, first)
	}
}

// TestGroupByParallelRollback confirms the documented rollback knob: above the
// parallelism threshold the parallel path runs by default, and
// GOPOLARS_TYPED_STORAGE=0 cleanly reverts to the sequential typed path — both
// producing the correct aggregate.
func TestGroupByParallelRollback(t *testing.T) {
	const n = 50000 // above parallelGroupByThreshold
	g := make([]string, n)
	v := make([]float64, n)
	for i := range g {
		g[i] = []string{"a", "b", "c"}[i%3]
		v[i] = float64(i % 100)
	}
	want := map[string]float64{}
	for i := range g {
		want[g[i]] += v[i]
	}
	mk := func() DataFrame {
		df, err := New(NewInput{Series: []series.Series{
			series.FromString("g", g, nil),
			series.FromFloat64("v", v, nil),
		}})
		if err != nil {
			t.Fatal(err)
		}
		return df
	}
	check := func(t *testing.T) {
		got, err := mk().GroupBy("g").Agg(expr.Sum(expr.Col("v")))
		if err != nil {
			t.Fatal(err)
		}
		sums := collectGroupSums(t, got, "g", "sum_v")
		for k, w := range want {
			if math.Abs(sums[k]-w) > 1e-6*math.Max(1, math.Abs(w)) {
				t.Fatalf("group %q sum=%v want %v", k, sums[k], w)
			}
		}
	}
	t.Run("parallel_default", func(t *testing.T) {
		t.Setenv("GOPOLARS_TYPED_STORAGE", "1")
		check(t)
	})
	t.Run("sequential_rollback", func(t *testing.T) {
		t.Setenv("GOPOLARS_TYPED_STORAGE", "0") // forces the sequential typed path
		check(t)
	})
}

// BenchmarkGroupByParallel mirrors BenchmarkGroupBySum but at a size guaranteed
// to take the parallel path, for the phase-3 speedup measurement.
func BenchmarkGroupByParallel(b *testing.B) {
	df := buildGroupBenchFrame(b, 1_000_000, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.GroupBy("g").Agg(expr.Sum(expr.Col("v"))); err != nil {
			b.Fatal(err)
		}
	}
}
