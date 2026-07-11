package frame

import (
	"runtime"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/chunk"
)

// uniqueKeys extracts the key column's values (as any) from a Unique result in
// row order, so tests can assert the kept set and its encounter order.
func uniqueKeyValues(t *testing.T, d DataFrame, key string) []any {
	t.Helper()
	col := d.cols[key]
	out := make([]any, d.height)
	for i := 0; i < d.height; i++ {
		out[i] = col.Value(i)
	}
	return out
}

// TestUniqueKeepFirstEncounterOrder checks the single-key Unique keeps the first
// occurrence of each key in encounter order (the maintain-order keep-first
// semantics the lean FirstRows path must preserve).
func TestUniqueKeepFirstEncounterOrder(t *testing.T) {
	df := mustFrame(t,
		SeriesInput{Name: "g", Values: []any{"b", "a", "b", "c", "a", "c"}},
		SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}},
	)
	got, err := df.Unique("g")
	if err != nil {
		t.Fatalf("unique: %v", err)
	}
	keys := uniqueKeyValues(t, got, "g")
	want := []any{"b", "a", "c"} // first-seen order
	if len(keys) != len(want) {
		t.Fatalf("keys=%v, want %v", keys, want)
	}
	for i, w := range want {
		if keys[i] != w {
			t.Fatalf("row %d key=%v, want %v (full: %v)", i, keys[i], w, keys)
		}
	}
	// The kept payload must be the first-seen row's payload, not a later one.
	vcol := got.cols["v"]
	if vcol.Value(0) != int64(1) || vcol.Value(1) != int64(2) || vcol.Value(2) != int64(4) {
		t.Fatalf("payload not keep-first: %v %v %v", vcol.Value(0), vcol.Value(1), vcol.Value(2))
	}
}

// TestUniqueNullKeysOneGroup checks all null-key rows collapse to a single kept
// row (the first null), matching prior GroupIDs-based semantics.
func TestUniqueNullKeysOneGroup(t *testing.T) {
	df := mustFrame(t,
		SeriesInput{Name: "g", Values: []any{int64(1), nil, int64(1), nil, int64(2)}},
	)
	got, err := df.Unique("g")
	if err != nil {
		t.Fatalf("unique: %v", err)
	}
	// Distinct keys in encounter order: 1, null, 2.
	if got.height != 3 {
		t.Fatalf("height=%d, want 3 (1,null,2)", got.height)
	}
	col := got.cols["g"]
	if col.Value(0) != int64(1) || !col.IsNull(1) || col.Value(2) != int64(2) {
		t.Fatalf("kept keys wrong: %v null@1=%v %v", col.Value(0), col.IsNull(1), col.Value(2))
	}
}

// buildUniqueFrame builds an n-row frame with a low-cardinality Int64 key "g"
// (gCard distinct), a mid-cardinality Int64 key "i" (iCard distinct), and a
// String tag "s" so multi-key composite discovery is exercised too.
func buildUniqueFrame(t *testing.T, n, gCard, iCard int) DataFrame {
	t.Helper()
	g := make([]any, n)
	i := make([]any, n)
	s := make([]any, n)
	tags := []string{"a", "b", "c", "d"}
	for r := 0; r < n; r++ {
		g[r] = int64(r % gCard)
		i[r] = int64(r % iCard)
		s[r] = tags[r%len(tags)]
	}
	return mustFrame(t,
		SeriesInput{Name: "g", Values: g},
		SeriesInput{Name: "i", Values: i},
		SeriesInput{Name: "s", Values: s},
	)
}

// keyCols resolves the *chunk.Column backing each named key.
func keyCols(d DataFrame, names ...string) []*chunk.Column {
	cols := make([]*chunk.Column, len(names))
	for j, name := range names {
		cols[j] = d.cols[name].Column()
	}
	return cols
}

// TestFirstRowsParallelMatchesSequential proves the sharded discovery path
// (d.firstRows above the threshold) returns exactly the sequential
// chunk.FirstRows keep-first indices — same rows, same encounter order — for
// low-card, mid-card, and multi-key inputs, under the race detector.
func TestFirstRowsParallelMatchesSequential(t *testing.T) {
	if runtime.GOMAXPROCS(0) < 2 {
		t.Skip("needs >1 worker to exercise the parallel path")
	}
	const n = parallelUniqueThreshold * 3 // safely above the threshold
	df := buildUniqueFrame(t, n, 5, 1000)
	cases := [][]string{{"g"}, {"i"}, {"s"}, {"g", "s"}, {"i", "s"}}
	for _, names := range cases {
		cols := keyCols(df, names...)
		want := chunk.FirstRows(cols, df.height) // sequential reference
		got := df.firstRows(cols)                // dispatches to parallel (n >= threshold)
		if len(got) != len(want) {
			t.Fatalf("%v: len parallel=%d, sequential=%d", names, len(got), len(want))
		}
		for k := range want {
			if got[k] != want[k] {
				t.Fatalf("%v: firstRows[%d] parallel=%d, sequential=%d", names, k, got[k], want[k])
			}
		}
	}
}

// TestFirstRowsSubThresholdStaysSequential proves a below-threshold frame keeps
// the sequential result (identical rows/order).
func TestFirstRowsSubThresholdStaysSequential(t *testing.T) {
	df := buildUniqueFrame(t, parallelUniqueThreshold-1, 7, 50)
	cols := keyCols(df, "g")
	want := chunk.FirstRows(cols, df.height)
	got := df.firstRows(cols)
	if len(got) != len(want) {
		t.Fatalf("len=%d, want %d", len(got), len(want))
	}
	for k := range want {
		if got[k] != want[k] {
			t.Fatalf("firstRows[%d]=%d, want %d", k, got[k], want[k])
		}
	}
}

// TestUniqueMultiColumnComposite checks multi-column Unique keeps first-seen
// composite keys.
func TestUniqueMultiColumnComposite(t *testing.T) {
	df := mustFrame(t,
		SeriesInput{Name: "a", Values: []any{"x", "x", "y", "x", "y"}},
		SeriesInput{Name: "b", Values: []any{int64(1), int64(2), int64(1), int64(1), int64(1)}},
	)
	got, err := df.Unique("a", "b")
	if err != nil {
		t.Fatalf("unique: %v", err)
	}
	// Composite keys first-seen: (x,1),(x,2),(y,1) — rows 0,1,2; row 3=(x,1) dup,
	// row 4=(y,1) dup.
	if got.height != 3 {
		t.Fatalf("height=%d, want 3", got.height)
	}
	acol, bcol := got.cols["a"], got.cols["b"]
	wantA := []any{"x", "x", "y"}
	wantB := []any{int64(1), int64(2), int64(1)}
	for i := range wantA {
		if acol.Value(i) != wantA[i] || bcol.Value(i) != wantB[i] {
			t.Fatalf("row %d=(%v,%v), want (%v,%v)", i, acol.Value(i), bcol.Value(i), wantA[i], wantB[i])
		}
	}
}
