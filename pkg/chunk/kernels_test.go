package chunk

import (
	"math"
	"testing"
)

func TestGatherNullFillAndOrder(t *testing.T) {
	src := NewInt64([]int64{10, 20, 30}, nil)
	out := src.Gather([]int{2, -1, 0, -1})
	if out.Len() != 4 {
		t.Fatalf("len = %d, want 4", out.Len())
	}
	if out.IsNull(0) || out.i64[0] != 30 {
		t.Errorf("row 0 = (%v,null=%v), want 30", out.i64[0], out.IsNull(0))
	}
	if !out.IsNull(1) {
		t.Errorf("row 1 should be null (sentinel -1)")
	}
	if out.IsNull(2) || out.i64[2] != 10 {
		t.Errorf("row 2 = %v, want 10", out.i64[2])
	}
	if !out.IsNull(3) {
		t.Errorf("row 3 should be null")
	}
	// Source must be unmodified.
	if src.Len() != 3 || src.i64[0] != 10 {
		t.Errorf("source mutated")
	}
}

func TestGatherPreservesSourceNulls(t *testing.T) {
	src := NewFloat64([]float64{1, 2, 3}, []bool{false, true, false})
	out := src.Gather([]int{1, 2})
	if !out.IsNull(0) {
		t.Errorf("row 0 should inherit source null at index 1")
	}
	if out.IsNull(1) || out.f64[1] != 3 {
		t.Errorf("row 1 = %v, want 3", out.f64[1])
	}
}

func TestNullCountCachedAndReset(t *testing.T) {
	c := NewFloat64([]float64{1, 2, 3, 4}, []bool{false, true, true, false})
	if got := c.NullCount(); got != 2 {
		t.Fatalf("NullCount = %d, want 2", got)
	}
	// Second call returns cached value (== fresh scan).
	if got := c.NullCount(); got != 2 {
		t.Fatalf("cached NullCount = %d, want 2", got)
	}
	// A sliced column resets the cache and recomputes correctly.
	sl := c.Slice([]int{0, 1})
	if got := sl.NullCount(); got != 1 {
		t.Fatalf("sliced NullCount = %d, want 1", got)
	}
	// Shift resets the cache.
	sh := c.Shift(1)
	if got := sh.NullCount(); got != 3 {
		t.Fatalf("shifted NullCount = %d, want 3 (1 new null + 2 carried)", got)
	}
}

func TestFillNullFloat64(t *testing.T) {
	c := NewFloat64([]float64{1, 0, 3}, []bool{false, true, false})
	out, ok := c.FillNullFloat64(99)
	if !ok {
		t.Fatal("expected float64 column")
	}
	want := []float64{1, 99, 3}
	for i, w := range want {
		if out.f64[i] != w || out.IsNull(i) {
			t.Errorf("row %d = (%v,null=%v), want %v non-null", i, out.f64[i], out.IsNull(i), w)
		}
	}
	if out.NullCount() != 0 {
		t.Errorf("filled column should have no nulls, got %d", out.NullCount())
	}
}

func TestFillNaNFloat64PreservesNulls(t *testing.T) {
	c := NewFloat64([]float64{1, math.NaN(), 0, math.NaN()}, []bool{false, false, true, false})
	out, ok := c.FillNaNFloat64(7)
	if !ok {
		t.Fatal("expected float64 column")
	}
	if out.f64[1] != 7 || out.f64[3] != 7 {
		t.Errorf("NaN values not filled: %v", out.f64)
	}
	if !out.IsNull(2) {
		t.Errorf("null entry must be preserved")
	}
	if out.f64[0] != 1 {
		t.Errorf("non-NaN value changed")
	}
}

func TestDropNaNFloat64(t *testing.T) {
	c := NewFloat64([]float64{1, math.NaN(), 0, 4}, []bool{false, false, true, false})
	out, ok := c.DropNaNFloat64()
	if !ok {
		t.Fatal("expected float64 column")
	}
	// NaN row dropped; null row (value 0) kept.
	if out.Len() != 3 {
		t.Fatalf("len = %d, want 3", out.Len())
	}
	if out.f64[0] != 1 || !out.IsNull(1) || out.f64[2] != 4 {
		t.Errorf("unexpected result: vals=%v nulls=%v", out.f64, out.nulls)
	}
}

func TestGroupIDsSingleInt64(t *testing.T) {
	c := NewInt64([]int64{5, 7, 5, 7, 5}, nil)
	ids, first := GroupIDs([]*Column{c}, c.Len())
	if len(first) != 2 {
		t.Fatalf("groups = %d, want 2", len(first))
	}
	// rows 0,2,4 share a group; 1,3 share a group.
	if ids[0] != ids[2] || ids[0] != ids[4] {
		t.Errorf("value 5 rows not grouped: %v", ids)
	}
	if ids[1] != ids[3] || ids[1] == ids[0] {
		t.Errorf("value 7 rows not grouped distinctly: %v", ids)
	}
	if first[ids[0]] != 0 || first[ids[1]] != 1 {
		t.Errorf("firstRow wrong: %v", first)
	}
}

func TestGroupIDsNullsAndNaN(t *testing.T) {
	c := NewFloat64([]float64{math.NaN(), 0, math.NaN(), 0}, []bool{false, true, false, true})
	ids, first := GroupIDs([]*Column{c}, c.Len())
	// Two groups: all-NaN (rows 0,2) and all-null (rows 1,3).
	if len(first) != 2 {
		t.Fatalf("groups = %d, want 2", len(first))
	}
	if ids[0] != ids[2] {
		t.Errorf("NaN rows must group together: %v", ids)
	}
	if ids[1] != ids[3] {
		t.Errorf("null rows must group together: %v", ids)
	}
	if ids[0] == ids[1] {
		t.Errorf("NaN and null must be distinct groups")
	}
}

func TestGroupIDsMultiKey(t *testing.T) {
	a := NewString([]string{"x", "x", "y", "x"}, nil)
	b := NewInt64([]int64{1, 2, 1, 1}, nil)
	ids, first := GroupIDs([]*Column{a, b}, 4)
	// (x,1) rows 0 and 3 group; (x,2) and (y,1) are distinct.
	if len(first) != 3 {
		t.Fatalf("groups = %d, want 3", len(first))
	}
	if ids[0] != ids[3] {
		t.Errorf("(x,1) rows must group: %v", ids)
	}
	if ids[1] == ids[0] || ids[2] == ids[0] || ids[1] == ids[2] {
		t.Errorf("distinct composite keys collided: %v", ids)
	}
}

// TestFirstRowsMatchesGroupIDs pins FirstRows to the firstRow slice GroupIDs
// already produces, across the typed single-key fast paths, the null/NaN cases,
// and the composite multi-key encoder — so the lean Unique path stays byte-for-
// byte identical to the prior GroupIDs-based one.
func TestFirstRowsMatchesGroupIDs(t *testing.T) {
	cases := []struct {
		name string
		cols []*Column
	}{
		{"int64", []*Column{NewInt64([]int64{5, 7, 5, 7, 5}, nil)}},
		{"int64-null", []*Column{NewInt64([]int64{5, 0, 5, 0, 7}, []bool{false, true, false, true, false})}},
		{"float-nan-null", []*Column{NewFloat64([]float64{math.NaN(), 0, math.NaN(), 0}, []bool{false, true, false, true})}},
		{"string", []*Column{NewString([]string{"a", "b", "a", "c", "b"}, nil)}},
		{"bool", []*Column{NewBool([]bool{true, false, true, false}, nil)}},
		{"multi-key", []*Column{NewString([]string{"x", "x", "y", "x"}, nil), NewInt64([]int64{1, 2, 1, 1}, nil)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := tc.cols[0].Len()
			_, want := GroupIDs(tc.cols, n)
			got := FirstRows(tc.cols, n)
			if len(got) != len(want) {
				t.Fatalf("len(FirstRows)=%d, want %d (%v vs %v)", len(got), len(want), got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("FirstRows[%d]=%d, want %d (%v vs %v)", i, got[i], want[i], got, want)
				}
			}
		})
	}
}

func TestMarkSharedCloneIfShared(t *testing.T) {
	c := NewInt64([]int64{1, 2, 3}, nil)
	if c.IsShared() {
		t.Fatal("new column should not be shared")
	}
	if got := c.CloneIfShared(); got != c {
		t.Errorf("unshared CloneIfShared should return receiver")
	}
	c.MarkShared()
	if !c.IsShared() {
		t.Fatal("MarkShared not recorded")
	}
	cl := c.CloneIfShared()
	if cl == c {
		t.Errorf("shared CloneIfShared should return a private clone")
	}
	if cl.IsShared() {
		t.Errorf("clone should not inherit shared flag")
	}
}
