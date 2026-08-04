package polars

import (
	"reflect"
	"testing"
)

// allocsScaleWithHeight reports the allocation counts of fn for a small and a
// large frame of the same schema. An operation whose cost is O(columns) must
// allocate the same amount for both; one that materializes rows will not.
func allocsScaleWithHeight(t *testing.T, fn func(DataFrame)) (small, large float64) {
	t.Helper()
	smallDF := buildDropFrame(t, 1_000, 0)
	largeDF := buildDropFrame(t, 200_000, 0)
	small = testing.AllocsPerRun(20, func() { fn(smallDF) })
	large = testing.AllocsPerRun(20, func() { fn(largeDF) })
	return small, large
}

// TestRowDoesNotMaterializeFrame pins the O(columns) contract for Row: reading
// one row of a 200x taller frame must not cost 200x the allocations. The prior
// implementation went through ToDicts, allocating a map per row of the whole
// frame.
func TestRowDoesNotMaterializeFrame(t *testing.T) {
	small, large := allocsScaleWithHeight(t, func(df DataFrame) {
		if _, err := df.Row(0); err != nil {
			t.Fatalf("Row(0): %v", err)
		}
	})
	if large > small {
		t.Fatalf("Row allocated %.0f times on a 200k-row frame vs %.0f on a 1k-row frame; must not scale with height", large, small)
	}
}

// TestRowMatchesFullMaterialization checks the fast path returns exactly what
// materializing every row would return, for a frame that contains nulls.
func TestRowMatchesFullMaterialization(t *testing.T) {
	df := buildDropFrame(t, 64, 3) // every 3rd "a" is null
	want := df.ToDicts()
	for i := range want {
		got, err := df.Row(i)
		if err != nil {
			t.Fatalf("Row(%d): %v", i, err)
		}
		if !reflect.DeepEqual(got, want[i]) {
			t.Fatalf("row %d: got %v, want %v", i, got, want[i])
		}
	}
}

// TestRowRejectsOutOfRange checks the bounds guard still holds on both sides.
func TestRowRejectsOutOfRange(t *testing.T) {
	df := buildDropFrame(t, 8, 0)
	for _, idx := range []int{-1, 8, 99} {
		if _, err := df.Row(idx); err == nil {
			t.Fatalf("Row(%d): expected an error, got nil", idx)
		}
	}
}

// TestCloneDoesNotCopyBuffers pins the shallow-clone contract: Clone shares the
// source's column buffers, so its cost is O(columns) and independent of height.
func TestCloneDoesNotCopyBuffers(t *testing.T) {
	small, large := allocsScaleWithHeight(t, func(df DataFrame) { _ = df.Clone() })
	if large > small {
		t.Fatalf("Clone allocated %.0f times on a 200k-row frame vs %.0f on a 1k-row frame; must not deep-copy buffers", large, small)
	}
}

// TestCloneReadsIdenticalValues checks a clone and its source observe the same
// cells, including null positions, now that they share buffers.
func TestCloneReadsIdenticalValues(t *testing.T) {
	df := buildDropFrame(t, 128, 4)
	clone := df.Clone()
	if clone.Height() != df.Height() {
		t.Fatalf("clone height=%d, want %d", clone.Height(), df.Height())
	}
	if !reflect.DeepEqual(clone.Columns(), df.Columns()) {
		t.Fatalf("clone columns=%v, want %v", clone.Columns(), df.Columns())
	}
	for _, name := range df.Columns() {
		src, err := df.GetColumn(name)
		if err != nil {
			t.Fatalf("GetColumn(%s) on source: %v", name, err)
		}
		dst, err := clone.GetColumn(name)
		if err != nil {
			t.Fatalf("GetColumn(%s) on clone: %v", name, err)
		}
		for i := 0; i < df.Height(); i++ {
			if !reflect.DeepEqual(src.Value(i), dst.Value(i)) {
				t.Fatalf("column %s row %d: clone=%v, source=%v", name, i, dst.Value(i), src.Value(i))
			}
		}
	}
}

// TestCloneIsStructurallyIndependent checks that deriving a differently-shaped
// frame from a clone leaves the source's schema untouched — the guarantee that
// survives the switch from deep to shallow copying.
func TestCloneIsStructurallyIndependent(t *testing.T) {
	df := buildDropFrame(t, 64, 0)
	clone := df.Clone()

	dropped, err := clone.Drop("c")
	if err != nil {
		t.Fatalf("Drop on clone: %v", err)
	}
	renamed, err := clone.Rename(map[string]string{"a": "a2"})
	if err != nil {
		t.Fatalf("Rename on clone: %v", err)
	}

	if got := df.Columns(); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("source columns changed to %v after deriving from its clone", got)
	}
	if got := dropped.Columns(); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("dropped columns=%v, want [a b]", got)
	}
	if got := renamed.Columns(); !reflect.DeepEqual(got, []string{"a2", "b", "c"}) {
		t.Fatalf("renamed columns=%v, want [a2 b c]", got)
	}
}

// TestRenameDoesNotCopyBuffers pins Rename as a metadata-only operation.
func TestRenameDoesNotCopyBuffers(t *testing.T) {
	small, large := allocsScaleWithHeight(t, func(df DataFrame) {
		if _, err := df.Rename(map[string]string{"b": "b2"}); err != nil {
			t.Fatalf("Rename: %v", err)
		}
	})
	if large > small {
		t.Fatalf("Rename allocated %.0f times on a 200k-row frame vs %.0f on a 1k-row frame; must not copy buffers", large, small)
	}
}

// TestDropDoesNotCopyRetainedBuffers pins Drop as a metadata-only operation on
// the columns it keeps.
func TestDropDoesNotCopyRetainedBuffers(t *testing.T) {
	small, large := allocsScaleWithHeight(t, func(df DataFrame) {
		if _, err := df.Drop("c"); err != nil {
			t.Fatalf("Drop: %v", err)
		}
	})
	if large > small {
		t.Fatalf("Drop allocated %.0f times on a 200k-row frame vs %.0f on a 1k-row frame; must not copy retained buffers", large, small)
	}
}

// TestRenameAndDropPreserveValues checks the shared-buffer paths still return
// the right names, order and cells.
func TestRenameAndDropPreserveValues(t *testing.T) {
	df := buildDropFrame(t, 32, 3)

	renamed, err := df.Rename(map[string]string{"b": "b2"})
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if got := renamed.Columns(); !reflect.DeepEqual(got, []string{"a", "b2", "c"}) {
		t.Fatalf("renamed columns=%v, want [a b2 c]", got)
	}
	src, err := df.GetColumn("b")
	if err != nil {
		t.Fatalf("GetColumn(b): %v", err)
	}
	dst, err := renamed.GetColumn("b2")
	if err != nil {
		t.Fatalf("GetColumn(b2): %v", err)
	}
	for i := 0; i < df.Height(); i++ {
		if !reflect.DeepEqual(src.Value(i), dst.Value(i)) {
			t.Fatalf("renamed row %d: got %v, want %v", i, dst.Value(i), src.Value(i))
		}
	}

	dropped, err := df.Drop("c")
	if err != nil {
		t.Fatalf("Drop: %v", err)
	}
	if got := dropped.Columns(); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("dropped columns=%v, want [a b]", got)
	}
	kept, err := dropped.GetColumn("a")
	if err != nil {
		t.Fatalf("GetColumn(a): %v", err)
	}
	origA, err := df.GetColumn("a")
	if err != nil {
		t.Fatalf("GetColumn(a) on source: %v", err)
	}
	for i := 0; i < df.Height(); i++ {
		if !reflect.DeepEqual(origA.Value(i), kept.Value(i)) {
			t.Fatalf("dropped row %d: a=%v, want %v", i, kept.Value(i), origA.Value(i))
		}
	}
}
