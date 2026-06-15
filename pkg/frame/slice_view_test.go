package frame

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/series"
)

// buildIntFrame builds a 2-column Int64 frame of height rows: a = i, b = i*10.
func buildIntFrame(t *testing.T, height int) DataFrame {
	t.Helper()
	a := make([]int64, height)
	b := make([]int64, height)
	for i := 0; i < height; i++ {
		a[i] = int64(i)
		b[i] = int64(i) * 10
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromInt64("a", a, nil),
		series.FromInt64("b", b, nil),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return df
}

func mustInt64(t *testing.T, s series.Series, row int) int64 {
	t.Helper()
	v := s.Value(row)
	iv, ok := v.(int64)
	if !ok {
		t.Fatalf("row %d: expected int64, got %T (%v)", row, v, v)
	}
	return iv
}

// TestSliceViewWindowsEqualMaterialized verifies that Limit/Tail/Slice return the
// same rows as the prior materializing implementation across edge cases.
func TestSliceViewWindowsEqualMaterialized(t *testing.T) {
	df := buildIntFrame(t, 100)

	// Limit (head) of 10.
	head := df.Limit(10)
	if head.Height() != 10 {
		t.Fatalf("Limit height = %d, want 10", head.Height())
	}
	if got := mustInt64(t, head.cols["a"], 0); got != 0 {
		t.Fatalf("Limit a[0] = %d, want 0", got)
	}
	if got := mustInt64(t, head.cols["b"], 9); got != 90 {
		t.Fatalf("Limit b[9] = %d, want 90", got)
	}

	// Tail of 5: rows 95..99.
	tail := df.Tail(5)
	if tail.Height() != 5 {
		t.Fatalf("Tail height = %d, want 5", tail.Height())
	}
	if got := mustInt64(t, tail.cols["a"], 0); got != 95 {
		t.Fatalf("Tail a[0] = %d, want 95", got)
	}
	if got := mustInt64(t, tail.cols["a"], 4); got != 99 {
		t.Fatalf("Tail a[4] = %d, want 99", got)
	}

	// Slice(offset=20, length=5): rows 20..24.
	sl := df.Slice(20, 5)
	if sl.Height() != 5 {
		t.Fatalf("Slice height = %d, want 5", sl.Height())
	}
	if got := mustInt64(t, sl.cols["a"], 0); got != 20 {
		t.Fatalf("Slice a[0] = %d, want 20", got)
	}
	if got := mustInt64(t, sl.cols["b"], 4); got != 240 {
		t.Fatalf("Slice b[4] = %d, want 240", got)
	}

	// Negative offset counts from the end: last 3 rows.
	neg := df.Slice(-3, 3)
	if neg.Height() != 3 {
		t.Fatalf("Slice(-3,3) height = %d, want 3", neg.Height())
	}
	if got := mustInt64(t, neg.cols["a"], 0); got != 97 {
		t.Fatalf("Slice(-3,3) a[0] = %d, want 97", got)
	}

	// Window clamped past the end.
	clamp := df.Slice(98, 10)
	if clamp.Height() != 2 {
		t.Fatalf("Slice(98,10) height = %d, want 2", clamp.Height())
	}

	// Schema/order preserved.
	if len(head.order) != 2 || head.order[0] != "a" || head.order[1] != "b" {
		t.Fatalf("order not preserved: %v", head.order)
	}
	if len(head.Schema()) != 2 {
		t.Fatalf("schema length = %d, want 2", len(head.Schema()))
	}
}

// TestSliceViewAllocationIsSizeIndependent verifies head(10) allocates the same
// regardless of parent frame size — i.e. it no longer re-materializes per row.
func TestSliceViewAllocationIsSizeIndependent(t *testing.T) {
	small := buildIntFrame(t, 1_000)
	large := buildIntFrame(t, 1_000_000)

	allocSmall := testing.AllocsPerRun(100, func() { _ = small.Limit(10) })
	allocLarge := testing.AllocsPerRun(100, func() { _ = large.Limit(10) })

	if allocSmall != allocLarge {
		t.Fatalf("head(10) allocs differ by frame size: 1K=%v, 1M=%v (want equal — size-independent)", allocSmall, allocLarge)
	}

	// And the head of a 1M frame must not allocate per-row (a full copy would be
	// ~1M int64 of book-keeping); a handful of allocs for the view + frame map.
	if allocLarge > 16 {
		t.Fatalf("head(10) on 1M allocated %v objects, want a small O(columns) constant", allocLarge)
	}
}

// TestSliceViewIsZeroCopy confirms the view shares the parent's backing storage
// (no per-row copy) by checking values read through the view match the parent.
func TestSliceViewIsZeroCopy(t *testing.T) {
	df := buildIntFrame(t, 1_000)
	view := df.Slice(500, 10)
	for i := 0; i < 10; i++ {
		if got, want := mustInt64(t, view.cols["a"], i), int64(500+i); got != want {
			t.Fatalf("view a[%d] = %d, want %d", i, got, want)
		}
	}
	// The shared-column COW flag must be set so any future in-place mutator clones.
	if !df.cols["a"].Column().IsShared() {
		t.Fatalf("parent column not marked shared after view; COW invariant broken")
	}
	if !view.cols["a"].Column().IsShared() {
		t.Fatalf("view column not marked shared; COW invariant broken")
	}
}

// TestUniqueParallelMatchesEncounterOrder verifies unique() through the shared
// parallel gather preserves first-seen encounter order and distinct rows.
func TestUniqueParallelMatchesEncounterOrder(t *testing.T) {
	// Keys repeat in a known pattern; first occurrences are rows 0,1,2 (keys 0,1,2).
	keys := []int64{0, 1, 2, 0, 1, 2, 0, 1, 2}
	tag := []int64{10, 11, 12, 13, 14, 15, 16, 17, 18}
	df, err := New(NewInput{Series: []series.Series{
		series.FromInt64("k", keys, nil),
		series.FromInt64("t", tag, nil),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	u, err := df.Unique("k")
	if err != nil {
		t.Fatalf("Unique: %v", err)
	}
	if u.Height() != 3 {
		t.Fatalf("Unique height = %d, want 3", u.Height())
	}
	// Encounter order: first rows for keys 0,1,2 carry tags 10,11,12.
	wantTag := []int64{10, 11, 12}
	for i, w := range wantTag {
		if got := mustInt64(t, u.cols["t"], i); got != w {
			t.Fatalf("Unique row %d tag = %d, want %d (encounter order broken)", i, got, w)
		}
	}
}
