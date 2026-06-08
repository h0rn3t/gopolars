package operations

// Divergences surfaced while porting Bucket A operations files (py-1.28.1).
// Each test asserts gopolars' ACTUAL behavior and documents how Python differs,
// mirroring the convention used elsewhere in this suite.
//
// Sources: test_gather.py, test_interpolate.py, test_is_sorted.py.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_gather: gopolars Series.Gather SKIPS out-of-range and negative indices,
// so a gather containing them returns a shorter Series. Python instead treats
// negative indices as from-the-end (gather([0,-1]) -> first & last) and raises
// on true out-of-bounds.
func TestGatherOutOfBoundsSkipped(t *testing.T) {
	t.Parallel()
	s := mkInts(t, int64(10), int64(20), int64(30), int64(40))
	// indices 0 (ok), -1 (skipped), 5 (skipped), 2 (ok) -> [10, 30]
	out := s.Gather([]int{0, -1, 5, 2})
	if out.Len() != 2 {
		t.Fatalf("gather len: got %d, want 2 (gopolars skips -1 and 5; Python keeps both)", out.Len())
	}
	if v, _ := out.Value(0).(int64); v != 10 {
		t.Fatalf("gather[0]: got %v, want 10", out.Value(0))
	}
	if v, _ := out.Value(1).(int64); v != 30 {
		t.Fatalf("gather[1]: got %v, want 30 (Python gather([0,-1]) would yield 40 at this spot)", out.Value(1))
	}
}

// test_interpolate: gopolars fills LEADING and TRAILING nulls with the nearest
// known value (forward/backward fill), whereas Python linear `interpolate`
// leaves boundary nulls as null. Interior nulls are linearly interpolated by both.
func TestInterpolateLeadingTrailingFilled(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64,
		Values: []any{nil, 1.0, nil, 3.0, nil}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Interpolate()
	// gopolars: [1, 1, 2, 3, 3]; Python: [null, 1, 2, 3, null]
	want := []float64{1.0, 1.0, 2.0, 3.0, 3.0}
	for i, w := range want {
		v, ok := out.Value(i).(float64)
		if !ok || v != w {
			t.Fatalf("interpolate[%d]: got %v, want %v (gopolars fills boundaries; Python leaves them null)", i, out.Value(i), w)
		}
	}
}

// test_is_sorted: gopolars Series.IsSorted() is ascending-only and takes no
// descending/nulls_last parameters, so a descending-sorted Series reports false.
func TestIsSortedNoDescendingParam(t *testing.T) {
	t.Parallel()
	desc := mkInts(t, int64(5), int64(3), int64(1))
	if desc.IsSorted() {
		t.Fatal("descending series: gopolars IsSorted()=true, but it only checks ascending (Python is_sorted(descending=True)=true via a param gopolars lacks)")
	}
	asc := mkInts(t, int64(1), int64(3), int64(5))
	if !asc.IsSorted() {
		t.Fatal("ascending series should report IsSorted()=true")
	}
}
