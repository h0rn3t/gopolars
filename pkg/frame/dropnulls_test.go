package frame

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// refDropNulls computes the reference survivor rows: a row survives iff none of
// the target columns is null there (matching the index-gather baseline). An
// empty targets slice means all columns, mirroring DropNulls.
func refDropNulls(d DataFrame, targets []string) []int {
	if len(targets) == 0 {
		targets = d.order
	}
	var keep []int
	for r := 0; r < d.height; r++ {
		drop := false
		for _, name := range targets {
			if d.cols[name].IsNull(r) {
				drop = true
				break
			}
		}
		if !drop {
			keep = append(keep, r)
		}
	}
	return keep
}

// assertDropNullsMatchesRef checks DropNulls(targets) equals the reference keep
// set for every column's values (including null placement) and height.
func assertDropNullsMatchesRef(t *testing.T, d DataFrame, targets []string) {
	t.Helper()
	keep := refDropNulls(d, targets)
	got := d.DropNulls(targets...)
	if got.height != len(keep) {
		t.Fatalf("targets=%v: height=%d, want %d", targets, got.height, len(keep))
	}
	for _, name := range d.order {
		src := d.cols[name]
		out := got.cols[name]
		for i, srcRow := range keep {
			gn, wn := out.IsNull(i), src.IsNull(srcRow)
			if gn != wn {
				t.Fatalf("targets=%v col=%s row=%d: null=%v, want %v", targets, name, i, gn, wn)
			}
			if !wn && out.Value(i) != src.Value(srcRow) {
				t.Fatalf("targets=%v col=%s row=%d: %v, want %v", targets, name, i, out.Value(i), src.Value(srcRow))
			}
		}
	}
}

// buildNullFrame builds an n-row frame: "a" is null every aEvery rows (0 = never),
// "b" is null every bEvery rows, "c" is a null-free string label.
func buildNullFrame(t *testing.T, n, aEvery, bEvery int) DataFrame {
	t.Helper()
	a := make([]any, n)
	b := make([]any, n)
	c := make([]any, n)
	for r := 0; r < n; r++ {
		if aEvery > 0 && r%aEvery == 0 {
			a[r] = nil
		} else {
			a[r] = int64(r)
		}
		if bEvery > 0 && r%bEvery == 0 {
			b[r] = nil
		} else {
			b[r] = float64(r) * 0.5
		}
		c[r] = "row"
	}
	return mustFrame(t,
		// Pin dtypes so an all-null column (aEvery/bEvery == 1) still types.
		SeriesInput{Name: "a", Values: a, DType: dtypes.Int64},
		SeriesInput{Name: "b", Values: b, DType: dtypes.Float64},
		SeriesInput{Name: "c", Values: c, DType: dtypes.String},
	)
}

// TestDropNullsBitmapEquality covers sparse, dense, all-null, no-null, and
// multi-column (OR) drops, comparing the bitmap path to the reference keep set
// under the race detector (parallel gather engages at 1M-ish sizes).
func TestDropNullsBitmapEquality(t *testing.T) {
	sizes := []int{1000, parallelFilterThreshold * 2}
	for _, n := range sizes {
		for _, tc := range []struct {
			name           string
			aEvery, bEvery int
			targets        []string
		}{
			{"sparse-a", 10, 0, []string{"a"}},       // ~10% null single col
			{"dense-a", 2, 0, []string{"a"}},         // ~50% null single col
			{"all-null-a", 1, 0, []string{"a"}},      // every row null
			{"no-null-c", 0, 0, []string{"c"}},       // no-null share path
			{"multi-a-b", 10, 7, []string{"a", "b"}}, // OR of two null masks
			{"multi-default", 10, 7, nil},            // all columns targeted
		} {
			t.Run(tc.name, func(t *testing.T) {
				df := buildNullFrame(t, n, tc.aEvery, tc.bEvery)
				assertDropNullsMatchesRef(t, df, tc.targets)
			})
		}
	}
}

// TestDropNullsNoNullShares proves the null-free target shares the frame (no
// gather): the bitmap path returns before building any mask.
func TestDropNullsNoNullShares(t *testing.T) {
	df := buildNullFrame(t, 4096, 0, 0) // no nulls at all
	got := df.DropNulls("a")
	if got.height != df.height {
		t.Fatalf("height=%d, want %d", got.height, df.height)
	}
	// Shared: the underlying column pointer is the same (not a fresh gather).
	if got.cols["a"].Column() != df.cols["a"].Column() {
		t.Fatal("null-free DropNulls should share the column, not gather a copy")
	}
}
