package chunk

import (
	"testing"
)

// TestColumnNulls covers the Nulls accessor for a column with and without a
// null bitmap.
func TestColumnNulls(t *testing.T) {
	t.Parallel()

	withNulls := NewInt64([]int64{1, 2, 3}, []bool{false, true, false})
	n := withNulls.Nulls()
	if len(n) != 3 || !n[1] || n[0] {
		t.Fatalf("Nulls() = %v, want [false true false]", n)
	}

	// A nil null argument is normalized; no row reports as null.
	noNulls := NewInt64([]int64{1, 2}, nil)
	for i, isNull := range noNulls.Nulls() {
		if isNull {
			t.Fatalf("row %d unexpectedly null in a no-null column", i)
		}
	}
}

// TestConcatColumns covers ConcatColumns for the empty, int, and float cases,
// including null-bitmap concatenation.
func TestConcatColumns(t *testing.T) {
	t.Parallel()

	// Empty input -> a column of length 0.
	if got := ConcatColumns(nil); got.Len() != 0 {
		t.Fatalf("ConcatColumns(nil) len = %d, want 0", got.Len())
	}

	a := NewInt64([]int64{1, 2}, []bool{false, true})
	b := NewInt64([]int64{3, 4}, nil)
	merged := ConcatColumns([]*Column{a, b})
	if merged.Len() != 4 {
		t.Fatalf("merged len = %d, want 4", merged.Len())
	}
	vals, ok := merged.Int64s()
	if !ok || vals[0] != 1 || vals[2] != 3 || vals[3] != 4 {
		t.Fatalf("merged int values = %v ok=%v", vals, ok)
	}
	// Null from a survives, b contributes no nulls.
	if !merged.IsNull(1) || merged.IsNull(2) {
		t.Fatalf("merged nulls wrong: idx1=%v idx2=%v", merged.IsNull(1), merged.IsNull(2))
	}

	// Float path.
	fa := NewFloat64([]float64{1.5}, nil)
	fb := NewFloat64([]float64{2.5, 3.5}, nil)
	fmerged := ConcatColumns([]*Column{fa, fb})
	fvals, ok := fmerged.Float64s()
	if !ok || len(fvals) != 3 || fvals[2] != 3.5 {
		t.Fatalf("float merged = %v ok=%v", fvals, ok)
	}
}

// TestAppendRowKey covers the multi-column row-key encoding: equal rows produce
// equal keys, differing rows (and null vs non-null) produce different keys.
func TestAppendRowKey(t *testing.T) {
	t.Parallel()

	c1 := NewInt64([]int64{10, 10, 10}, []bool{false, false, true})
	c2 := NewString([]string{"a", "b", "a"}, nil)
	cols := []*Column{c1, c2}

	key := func(row int) string { return string(AppendRowKey(nil, cols, row)) }

	// Rows 0 and 1 differ on c2 -> different keys.
	if key(0) == key(1) {
		t.Fatal("rows with different string values produced equal keys")
	}
	// Row 2 has a null in c1 -> different key from row 0.
	if key(0) == key(2) {
		t.Fatal("null row produced same key as non-null row")
	}

	// Identical rows produce identical keys.
	d1 := NewInt64([]int64{5, 5}, nil)
	d2 := NewString([]string{"x", "x"}, nil)
	same := []*Column{d1, d2}
	if string(AppendRowKey(nil, same, 0)) != string(AppendRowKey(nil, same, 1)) {
		t.Fatal("identical rows produced different keys")
	}
}

// TestArgsortFloat64Parallel covers the parallel float argsort, including a size
// large enough to exercise the parallel path.
func TestArgsortFloat64Parallel(t *testing.T) {
	t.Parallel()

	// Small input.
	small := []float64{3, 1, 2}
	idx := ArgsortFloat64Parallel(small)
	if len(idx) != 3 || small[idx[0]] != 1 || small[idx[2]] != 3 {
		t.Fatalf("small argsort = %v", idx)
	}

	// Large input: descending values, result must be ascending order.
	const n = 5000
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = float64(n - i)
	}
	got := ArgsortFloat64Parallel(vals)
	if len(got) != n {
		t.Fatalf("argsort len = %d, want %d", len(got), n)
	}
	for i := 1; i < len(got); i++ {
		if vals[got[i-1]] > vals[got[i]] {
			t.Fatalf("not sorted at %d: %v > %v", i, vals[got[i-1]], vals[got[i]])
		}
	}
}

// TestGroupIDsSingle covers the single-column group-id assignment for int and
// float columns, including null grouping.
func TestGroupIDsSingle(t *testing.T) {
	t.Parallel()

	// int64 column: values 10,20,10,<null>,20 -> groups by distinct value + null.
	c := NewInt64([]int64{10, 20, 10, 0, 20}, []bool{false, false, false, true, false})
	ids, firstRow, ok := groupIDsSingle(c, c.Len())
	if !ok {
		t.Fatal("groupIDsSingle(int) returned ok=false")
	}
	// rows 0 and 2 share a group; rows 1 and 4 share a group; row 3 (null) is its own.
	if ids[0] != ids[2] {
		t.Fatalf("rows 0,2 should share a group: %v", ids)
	}
	if ids[1] != ids[4] {
		t.Fatalf("rows 1,4 should share a group: %v", ids)
	}
	if ids[0] == ids[1] || ids[0] == ids[3] {
		t.Fatalf("distinct values should differ in group id: %v", ids)
	}
	// 3 groups -> 3 firstRow entries.
	if len(firstRow) != 3 {
		t.Fatalf("firstRow = %v, want 3 groups", firstRow)
	}

	// float64 column.
	f := NewFloat64([]float64{1.5, 2.5, 1.5}, nil)
	fids, _, ok := groupIDsSingle(f, f.Len())
	if !ok || fids[0] != fids[2] || fids[0] == fids[1] {
		t.Fatalf("float group ids = %v ok=%v", fids, ok)
	}
}
