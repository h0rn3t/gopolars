package chunk

import (
	"fmt"
	"runtime"
	"testing"
)

// normalizePartition renumbers group ids in order of first appearance, so two id
// slices compare equal exactly when they describe the same partitioning of rows
// — which is the contract GroupIDsUnordered guarantees (same partitions, some
// unspecified numbering).
func normalizePartition(ids []int) []int {
	seen := make(map[int]int, 16)
	out := make([]int, len(ids))
	for i, id := range ids {
		g, hit := seen[id]
		if !hit {
			g = len(seen)
			seen[id] = g
		}
		out[i] = g
	}
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// groupTestCases builds the key columns the unordered builder must agree with
// GroupIDs on: low and high cardinality, with and without nulls, int64 and
// string, and sizes below and above the sharding threshold.
func groupTestCases() []struct {
	name string
	col  *Column
	n    int
} {
	var cases []struct {
		name string
		col  *Column
		n    int
	}
	add := func(name string, col *Column, n int) {
		cases = append(cases, struct {
			name string
			col  *Column
			n    int
		}{name, col, n})
	}

	for _, n := range []int{1, 7, 1000, 40000} {
		for _, cardinality := range []int{1, 5, 997} {
			i64 := make([]int64, n)
			str := make([]string, n)
			for i := 0; i < n; i++ {
				i64[i] = int64(i % cardinality)
				str[i] = fmt.Sprintf("g%d", i%cardinality)
			}
			add(fmt.Sprintf("int64/n=%d/card=%d", n, cardinality), NewInt64(i64, nil), n)
			add(fmt.Sprintf("string/n=%d/card=%d", n, cardinality), NewString(str, nil), n)

			// Same keys, but every 7th row null: nulls must collapse into one group.
			nulls := make([]bool, n)
			for i := 0; i < n; i++ {
				nulls[i] = i%7 == 0
			}
			i64n := make([]int64, n)
			strn := make([]string, n)
			copy(i64n, i64)
			copy(strn, str)
			add(fmt.Sprintf("int64+nulls/n=%d/card=%d", n, cardinality), NewInt64(i64n, nulls), n)
			add(fmt.Sprintf("string+nulls/n=%d/card=%d", n, cardinality), NewString(strn, nulls), n)
		}

		// Fully unique keys: the high-cardinality guard must send this to the
		// sequential build, and the result must still be correct.
		unique := make([]int64, n)
		for i := 0; i < n; i++ {
			unique[i] = int64(i)
		}
		add(fmt.Sprintf("int64/unique/n=%d", n), NewInt64(unique, nil), n)
	}
	return cases
}

// TestGroupIDsUnorderedMatchesGroupIDs checks the sharded builder produces the
// same partitioning and the same group count as the sequential GroupIDs, for
// every dtype, cardinality, null pattern and size it accepts.
func TestGroupIDsUnorderedMatchesGroupIDs(t *testing.T) {
	for _, tc := range groupTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			wantIDs, wantFirst := GroupIDs([]*Column{tc.col}, tc.n)
			gotIDs, gotN := GroupIDsUnordered([]*Column{tc.col}, tc.n)

			if gotN != len(wantFirst) {
				t.Fatalf("ngroups=%d, want %d", gotN, len(wantFirst))
			}
			if !equalInts(normalizePartition(gotIDs), normalizePartition(wantIDs)) {
				t.Fatalf("partitioning differs from GroupIDs\n got=%v\nwant=%v", gotIDs, wantIDs)
			}
			for _, id := range gotIDs {
				if id < 0 || id >= gotN {
					t.Fatalf("group id %d out of range [0,%d)", id, gotN)
				}
			}
		})
	}
}

// TestGroupIDsUnorderedIndependentOfWorkerCount checks the partitioning does not
// depend on how the rows were sharded.
func TestGroupIDsUnorderedIndependentOfWorkerCount(t *testing.T) {
	const n = 40000
	vals := make([]int64, n)
	nulls := make([]bool, n)
	for i := 0; i < n; i++ {
		vals[i] = int64(i % 11)
		nulls[i] = i%13 == 0
	}
	col := NewInt64(vals, nulls)

	original := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(original)

	var reference []int
	var referenceN int
	for _, procs := range []int{1, 2, 3, 8} {
		runtime.GOMAXPROCS(procs)
		ids, ngroups := GroupIDsUnordered([]*Column{col}, n)
		normalized := normalizePartition(ids)
		if reference == nil {
			reference, referenceN = normalized, ngroups
			continue
		}
		if ngroups != referenceN {
			t.Fatalf("GOMAXPROCS=%d: ngroups=%d, want %d", procs, ngroups, referenceN)
		}
		if !equalInts(normalized, reference) {
			t.Fatalf("GOMAXPROCS=%d: partitioning differs from the single-worker result", procs)
		}
	}
}

// TestGroupIDsUnorderedFallsBackForCompositeAndFloatKeys checks the dtypes and
// arities the sharded path does not handle still return a correct result via the
// sequential builder — including that float keys keep 0.0 and -0.0 distinct.
func TestGroupIDsUnorderedFallsBackForCompositeAndFloatKeys(t *testing.T) {
	const n = 20000

	f := make([]float64, n)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			f[i] = 0.0
		} else {
			f[i] = negZero()
		}
	}
	floatCol := NewFloat64(f, nil)
	wantIDs, wantFirst := GroupIDs([]*Column{floatCol}, n)
	gotIDs, gotN := GroupIDsUnordered([]*Column{floatCol}, n)
	if gotN != len(wantFirst) {
		t.Fatalf("float ngroups=%d, want %d", gotN, len(wantFirst))
	}
	if !equalInts(normalizePartition(gotIDs), normalizePartition(wantIDs)) {
		t.Fatalf("float partitioning differs from GroupIDs")
	}

	a := make([]int64, n)
	b := make([]string, n)
	for i := 0; i < n; i++ {
		a[i] = int64(i % 3)
		b[i] = fmt.Sprintf("k%d", i%4)
	}
	composite := []*Column{NewInt64(a, nil), NewString(b, nil)}
	wantIDs, wantFirst = GroupIDs(composite, n)
	gotIDs, gotN = GroupIDsUnordered(composite, n)
	if gotN != len(wantFirst) {
		t.Fatalf("composite ngroups=%d, want %d", gotN, len(wantFirst))
	}
	if !equalInts(normalizePartition(gotIDs), normalizePartition(wantIDs)) {
		t.Fatalf("composite partitioning differs from GroupIDs")
	}
}

// negZero returns -0.0 without the compiler folding it to +0.0.
func negZero() float64 {
	z := 0.0
	return -z
}
