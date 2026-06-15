package frame

import (
	"runtime"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/series"
)

// makeJoinLeft builds a left frame of `height` rows whose Int64 key `k` cycles
// through `distinct` values (so it matches a `distinct`-row right dimension). It
// carries a Float64 `v` (which collides with the right's `v`, exercising suffix
// renaming) and an Int64 `id`. With withNulls, a slice of key rows are null, so
// the null-key match path (null matches null) is exercised.
func makeJoinLeft(t *testing.T, height, distinct int, withNulls bool) DataFrame {
	t.Helper()
	k := make([]int64, height)
	v := make([]float64, height)
	id := make([]int64, height)
	var kn []bool
	if withNulls {
		kn = make([]bool, height)
	}
	for i := range height {
		k[i] = int64(i % distinct)
		v[i] = float64(i)
		id[i] = int64(i)
		if withNulls && i%37 == 0 {
			kn[i] = true
		}
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromInt64("k", k, kn),
		series.FromFloat64("v", v, nil),
		series.FromInt64("id", id, nil),
	}})
	if err != nil {
		t.Fatalf("makeJoinLeft New: %v", err)
	}
	return df
}

// makeJoinRight builds a right frame of `rows` rows with Int64 key `k` = j,
// a colliding Float64 `v`, and Int64 `rid`.
func makeJoinRight(t *testing.T, rows int, withNulls bool) DataFrame {
	t.Helper()
	k := make([]int64, rows)
	v := make([]float64, rows)
	rid := make([]int64, rows)
	var kn []bool
	if withNulls {
		kn = make([]bool, rows)
	}
	for j := range rows {
		k[j] = int64(j)
		v[j] = float64(j) * 100
		rid[j] = int64(j) * 7
		if withNulls && j%50 == 0 {
			kn[j] = true
		}
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromInt64("k", k, kn),
		series.FromFloat64("v", v, nil),
		series.FromInt64("rid", rid, nil),
	}})
	if err != nil {
		t.Fatalf("makeJoinRight New: %v", err)
	}
	return df
}

// makeCompositeFrame builds a frame whose join keys are a composite of an Int64
// and a String column, which cannot be packed into a uint64 and so forces the
// byte-encoded fallback probe path.
func makeCompositeFrame(t *testing.T, height, distinct int, vScale float64, vName string) DataFrame {
	t.Helper()
	letters := []string{"a", "b", "c", "d", "e"}
	k := make([]int64, height)
	g := make([]string, height)
	v := make([]float64, height)
	for i := range height {
		k[i] = int64(i % distinct)
		g[i] = letters[i%len(letters)]
		v[i] = float64(i) * vScale
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromInt64("k", k, nil),
		series.FromString("g", g, nil),
		series.FromFloat64(vName, v, nil),
	}})
	if err != nil {
		t.Fatalf("makeCompositeFrame New: %v", err)
	}
	return df
}

// joinSeqVsPar runs the same join with GOMAXPROCS forced to 1 (the sequential
// build/probe/gather path) and with GOMAXPROCS >= 4 (the parallel path), then
// returns both results for an equality check.
func joinSeqVsPar(t *testing.T, left DataFrame, in JoinInput) (seq, par DataFrame) {
	t.Helper()
	prev := runtime.GOMAXPROCS(1)
	s, err := join(left, in)
	if err != nil {
		runtime.GOMAXPROCS(prev)
		t.Fatalf("sequential join (%s): %v", in.How, err)
	}
	runtime.GOMAXPROCS(max(prev, 4))
	p, err := join(left, in)
	runtime.GOMAXPROCS(prev)
	if err != nil {
		t.Fatalf("parallel join (%s): %v", in.How, err)
	}
	return s, p
}

// assertFramesEqual fails if a and b differ in height, column order, or any
// cell value (including null-vs-non-null). Test data avoids NaN so == is exact.
func assertJoinFramesEqual(t *testing.T, a, b DataFrame, ctx string) {
	t.Helper()
	if a.Height() != b.Height() {
		t.Fatalf("%s: height %d != %d", ctx, a.Height(), b.Height())
	}
	if len(a.order) != len(b.order) {
		t.Fatalf("%s: column count %d != %d (%v vs %v)", ctx, len(a.order), len(b.order), a.order, b.order)
	}
	for ci, name := range a.order {
		if b.order[ci] != name {
			t.Fatalf("%s: column %d name %q != %q", ctx, ci, name, b.order[ci])
		}
		ca, cb := a.cols[name], b.cols[name]
		for i := range a.Height() {
			va, vb := ca.Value(i), cb.Value(i)
			if va != vb {
				t.Fatalf("%s: column %q row %d: %v != %v", ctx, name, i, va, vb)
			}
		}
	}
}

// TestJoinParallelMatchesSequential verifies the parallel build/probe/gather
// path produces byte-identical results to the sequential path for every join
// mode, across a single-int-key (packed) frame with a small right dim, a large
// right dim (parallel build), a null-key frame, and a composite-key (byte
// fallback) frame. Run under -race to confirm the concurrency is data-race-free.
func TestJoinParallelMatchesSequential(t *testing.T) {
	const large = 1 << 16 // 65536: above parallelJoinThreshold (32768)
	modes := []JoinType{
		JoinTypeInner, JoinTypeLeft, JoinTypeRight,
		JoinTypeFull, JoinTypeSemi, JoinTypeAnti,
	}

	t.Run("single_int_key_small_right", func(t *testing.T) {
		left := makeJoinLeft(t, large, 1000, false)
		right := makeJoinRight(t, 1000, false)
		for _, how := range modes {
			seq, par := joinSeqVsPar(t, left, JoinInput{
				Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: how,
			})
			assertJoinFramesEqual(t, seq, par, "single_int_key/"+string(how))
		}
	})

	t.Run("single_int_key_large_right_parallel_build", func(t *testing.T) {
		// distinct == large so the right dim is itself above the threshold and the
		// probe table is built sharded across workers.
		left := makeJoinLeft(t, large, large, false)
		right := makeJoinRight(t, large, false)
		for _, how := range modes {
			seq, par := joinSeqVsPar(t, left, JoinInput{
				Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: how,
			})
			assertJoinFramesEqual(t, seq, par, "large_right/"+string(how))
		}
	})

	t.Run("null_keys_match", func(t *testing.T) {
		left := makeJoinLeft(t, large, 1000, true)
		right := makeJoinRight(t, 1000, true)
		for _, how := range modes {
			seq, par := joinSeqVsPar(t, left, JoinInput{
				Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: how,
			})
			assertJoinFramesEqual(t, seq, par, "null_keys/"+string(how))
		}
	})

	t.Run("composite_key_byte_fallback", func(t *testing.T) {
		left := makeCompositeFrame(t, large, 1000, 1, "v")
		right := makeCompositeFrame(t, 1000, 1000, 100, "v")
		for _, how := range modes {
			seq, par := joinSeqVsPar(t, left, JoinInput{
				Other: right, LeftOn: []string{"k", "g"}, RightOn: []string{"k", "g"}, How: how,
			})
			assertJoinFramesEqual(t, seq, par, "composite/"+string(how))
		}
	})

	t.Run("cross", func(t *testing.T) {
		// Cross product is O(n*m); keep both sides small.
		left := makeJoinLeft(t, 200, 50, false)
		right := makeJoinRight(t, 80, false)
		seq, par := joinSeqVsPar(t, left, JoinInput{Other: right, How: JoinTypeCross})
		assertJoinFramesEqual(t, seq, par, "cross")
	})
}

// TestJoinAllocsDoNotScaleWithRows asserts the per-join allocation count is
// independent of the left-row count: a 1,000,000-row join against a fixed 1,000-
// row dimension allocates essentially the same number of objects as a 100,000-
// row join. The prior map[string][]int build allocated a key string per row, so
// its alloc count grew with rows; the packed-key build does not.
func TestJoinAllocsDoNotScaleWithRows(t *testing.T) {
	right := makeJoinRight(t, 1000, false)
	in := func(left DataFrame) (DataFrame, JoinInput) {
		return left, JoinInput{Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: JoinTypeInner}
	}

	mid := makeJoinLeft(t, 100_000, 1000, false)
	big := makeJoinLeft(t, 1_000_000, 1000, false)

	allocMid := testing.AllocsPerRun(3, func() {
		l, q := in(mid)
		if _, err := join(l, q); err != nil {
			t.Fatalf("join mid: %v", err)
		}
	})
	allocBig := testing.AllocsPerRun(3, func() {
		l, q := in(big)
		if _, err := join(l, q); err != nil {
			t.Fatalf("join big: %v", err)
		}
	})

	// The allocation count must not scale with the left/probe row count: both
	// joins hit the same 1,000-distinct-key right dimension, so their only
	// row-count-dependent allocations are O(1) (the packed key slice and the
	// int32 pair buffers — a handful of slices regardless of row count). A
	// per-row key string would make the 1M join allocate ~10x the 100K join and
	// fail this bound. (The ~O(distinct right keys) posting-list slices are
	// constant across the two sizes and are not per-row.)
	if allocBig > allocMid+16 {
		t.Fatalf("join allocs scale with rows: 100K=%v, 1M=%v (want ~equal — no per-row key allocation)", allocMid, allocBig)
	}
}
