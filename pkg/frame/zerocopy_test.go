package frame

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

func makeZeroCopyFrame(t *testing.T) DataFrame {
	t.Helper()
	g := series.FromString("g", []string{"a", "b", "a", "c"}, nil)
	v := series.FromFloat64("v", []float64{1, 2, 3, 4}, nil)
	df, err := New(NewInput{Series: []series.Series{g, v}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	return df
}

func TestSelectSharesColumnPointer(t *testing.T) {
	df := makeZeroCopyFrame(t)
	src := df.cols["v"].Column()

	out, err := df.Select(expr.Col("v"))
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got := out.cols["v"].Column(); got != src {
		t.Errorf("Select did not share the source column pointer (got %p want %p)", got, src)
	}
	if !src.IsShared() {
		t.Errorf("projected column should be marked shared")
	}
}

func TestRenameSharesColumnPointer(t *testing.T) {
	df := makeZeroCopyFrame(t)
	src := df.cols["v"]
	renamed := src.Rename("v2")
	if renamed.Column() != src.Column() {
		t.Errorf("Rename should share the underlying column pointer")
	}
	if renamed.Name() != "v2" {
		t.Errorf("Rename name = %q, want v2", renamed.Name())
	}
}

func TestWithColumnsReusesUnchangedColumns(t *testing.T) {
	df := makeZeroCopyFrame(t)
	gPtr := df.cols["g"].Column()
	vPtr := df.cols["v"].Column()

	// A computed (not plain-projection) column is the only thing that should be
	// freshly allocated.
	out, err := df.WithColumns(expr.Col("v").Mul(expr.Lit(float64(2))).Alias("v3"))
	if err != nil {
		t.Fatalf("with_columns: %v", err)
	}
	// The two pre-existing columns must be reused by pointer.
	if out.cols["g"].Column() != gPtr {
		t.Errorf("WithColumns copied unchanged column g")
	}
	if out.cols["v"].Column() != vPtr {
		t.Errorf("WithColumns copied unchanged column v")
	}
	// The new computed column must exist and not alias an existing buffer.
	newCol := out.cols["v3"].Column()
	if newCol == nil || newCol == vPtr || newCol == gPtr {
		t.Errorf("new computed column v3 should be freshly allocated")
	}
}

// TestCopyOnWriteGuard verifies the documented mutation contract: a mutator that
// routes a write through CloneIfShared never alters a frame that shares the
// buffer.
func TestCopyOnWriteGuard(t *testing.T) {
	df := makeZeroCopyFrame(t)
	// Share v across a derived frame.
	out, err := df.WithColumns(expr.Col("v").Alias("v2"))
	if err != nil {
		t.Fatalf("with_columns: %v", err)
	}
	shared := out.cols["v"].Column()
	if !shared.IsShared() {
		t.Fatalf("column should be shared after projection")
	}
	// A hypothetical in-place mutator obtains a private buffer first.
	priv := shared.CloneIfShared()
	if priv == shared {
		t.Fatalf("CloneIfShared must return a distinct buffer for a shared column")
	}
	privF, _ := priv.Float64s()
	privF[0] = 999 // mutate the private copy
	// The original (and df) must be untouched.
	srcF, _ := df.cols["v"].Column().Float64s()
	if srcF[0] != 1 {
		t.Errorf("mutation leaked into source frame: got %v want 1", srcF[0])
	}
}

// TestNoExportedPathMutatesSharedColumn audits that running a battery of
// exported operations on a frame whose columns are shared never changes the
// source column buffers in place.
func TestNoExportedPathMutatesSharedColumn(t *testing.T) {
	df := makeZeroCopyFrame(t)
	// Force the v column to be shared by deriving a frame from df.
	if _, err := df.WithColumns(expr.Col("v").Alias("v_copy")); err != nil {
		t.Fatalf("with_columns: %v", err)
	}
	vCol := df.cols["v"].Column()
	if !vCol.IsShared() {
		t.Fatalf("v should be shared")
	}
	snapshot := func() []float64 {
		f, _ := vCol.Float64s()
		return append([]float64(nil), f...)
	}
	before := snapshot()

	// Run a battery of exported derivations.
	if _, err := df.Select(expr.Col("v"), expr.Col("g")); err != nil {
		t.Fatalf("select: %v", err)
	}
	if _, err := df.WithColumns(expr.Col("v").Mul(expr.Lit(float64(2))).Alias("v2")); err != nil {
		t.Fatalf("with_columns mul: %v", err)
	}
	if _, err := df.Filter(expr.Col("v").Gt(expr.Lit(float64(1)))); err != nil {
		t.Fatalf("filter: %v", err)
	}
	if _, err := df.Sort(SortInput{By: []string{"v"}}); err != nil {
		t.Fatalf("sort: %v", err)
	}
	_ = df.Limit(2)
	_ = df.Tail(2)
	if _, err := df.Unique("g"); err != nil {
		t.Fatalf("unique: %v", err)
	}
	if _, err := df.GroupBy("g").Agg(expr.Sum(expr.Col("v"))); err != nil {
		t.Fatalf("group_by: %v", err)
	}

	after := snapshot()
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("shared column v mutated in place at %d: %v -> %v", i, before[i], after[i])
		}
	}
}

func buildProjectionBenchFrame(b *testing.B, n int) DataFrame {
	b.Helper()
	v := make([]float64, n)
	w := make([]float64, n)
	g := make([]string, n)
	for i := 0; i < n; i++ {
		v[i] = float64(i)
		w[i] = float64(i) * 2
		g[i] = []string{"a", "b", "c"}[i%3]
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromString("g", g, nil),
		series.FromFloat64("v", v, nil),
		series.FromFloat64("w", w, nil),
	}})
	if err != nil {
		b.Fatalf("frame: %v", err)
	}
	return df
}

// BenchmarkSelectProjection should allocate O(columns), independent of row count.
func BenchmarkSelectProjection(b *testing.B) {
	df := buildProjectionBenchFrame(b, 1_000_000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Select(expr.Col("g"), expr.Col("v")); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWithColumnsAlias projects an existing column under a new name; it
// must not copy any row-proportional buffer.
func BenchmarkWithColumnsAlias(b *testing.B) {
	df := buildProjectionBenchFrame(b, 1_000_000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.WithColumns(expr.Col("v").Alias("v2")); err != nil {
			b.Fatal(err)
		}
	}
}
