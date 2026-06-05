package chunk

import (
	"math/rand"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestNewBoxed pins the boxed-column constructor used for dtypes without a typed
// backing slice (Decimal/List/Struct).
func TestNewBoxed(t *testing.T) {
	row := map[string]any{"x": int64(1)}
	c := NewBoxed(dtypes.Struct, []any{row, nil, "raw"}, []bool{false, true, false})
	if c.Len() != 3 {
		t.Fatalf("Len = %d, want 3", c.Len())
	}
	if c.DataType() != dtypes.Struct {
		t.Errorf("DataType = %s, want struct", c.DataType())
	}
	if !c.IsNull(1) {
		t.Errorf("IsNull(1) = false, want true")
	}
	if got := c.ValueAt(0); got == nil {
		t.Errorf("ValueAt(0) = nil, want the struct row")
	}
	if got := c.ValueAt(1); got != nil {
		t.Errorf("ValueAt(1) = %v, want nil (null)", got)
	}
	if got := c.ValueAt(2); got != "raw" {
		t.Errorf("ValueAt(2) = %v, want raw", got)
	}
}

// TestStringifyBoxed pins the boxed-key stringifier used for composite group keys.
func TestStringifyBoxed(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{int64(7), "7"},
		{"abc", "abc"},
		{[]any{1, 2}, "[1 2]"},
	}
	for _, tc := range cases {
		if got := stringifyBoxed(tc.in); got != tc.want {
			t.Errorf("stringifyBoxed(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestArgsortInt64ParallelMatchesSequential verifies the parallel argsort yields
// the same stable ascending permutation as the sequential path. The input is
// large enough (> parallelMergeThreshold) to exercise the parallel-merge branch.
func TestArgsortInt64ParallelMatchesSequential(t *testing.T) {
	const n = (1 << 16) + 5000 // above parallelMergeThreshold
	rng := rand.New(rand.NewSource(42))
	vals := make([]int64, n)
	for i := range vals {
		vals[i] = int64(rng.Intn(1000)) // many duplicates to test stability
	}

	got := ArgsortInt64Parallel(vals)
	want := ArgsortInt64(vals)

	if len(got) != n {
		t.Fatalf("len = %d, want %d", len(got), n)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("permutation mismatch at %d: got %d, want %d", i, got[i], want[i])
		}
	}
	// Sanity: the permutation actually sorts the values ascending.
	for i := 1; i < n; i++ {
		if vals[got[i-1]] > vals[got[i]] {
			t.Fatalf("not sorted at %d: %d > %d", i, vals[got[i-1]], vals[got[i]])
		}
	}
}
