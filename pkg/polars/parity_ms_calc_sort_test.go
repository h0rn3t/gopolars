package polars

// Parity: sorting (spec: Sorting parity).
// Mirrors df.sort(["z_id","z_time"]) and .sort(by=[...], descending=True) in ../ms-calculations
// (utils.py:182, profiling.py:260).

import "testing"

func mscSortFixture(t *testing.T) DataFrame {
	t.Helper()
	return mscFrame(t,
		mscCol("z_id", int64(2), int64(1), int64(1), int64(2)),
		mscCol("z_time", int64(5), int64(9), int64(3), int64(1)),
	)
}

func col64(t *testing.T, df DataFrame, name string) []int64 {
	t.Helper()
	c, err := df.GetColumn(name)
	if err != nil {
		t.Fatalf("GetColumn(%s): %v", name, err)
	}
	out := make([]int64, c.Len())
	for i := 0; i < c.Len(); i++ {
		v, _ := c.Value(i).(int64)
		out[i] = v
	}
	return out
}

func eqInt64(a, b []int64) bool {
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

// TestParitySortAscendingMultiColumn pins utils.py:182 — ascending sort by ["z_id","z_time"].
func TestParitySortAscendingMultiColumn(t *testing.T) {
	out, err := mscSortFixture(t).Sort(SortInput{By: []string{"z_id", "z_time"}})
	if err != nil {
		t.Fatalf("Sort asc: %v", err)
	}
	if got := col64(t, out, "z_id"); !eqInt64(got, []int64{1, 1, 2, 2}) {
		t.Errorf("z_id order = %v, want [1 1 2 2]", got)
	}
	if got := col64(t, out, "z_time"); !eqInt64(got, []int64{3, 9, 1, 5}) {
		t.Errorf("z_time order = %v, want [3 9 1 5]", got)
	}
}

// TestParitySortDescendingMultiColumn pins profiling.py:260 — descending multi-column sort.
func TestParitySortDescendingMultiColumn(t *testing.T) {
	out, err := mscSortFixture(t).Sort(SortInput{
		By:         []string{"z_id", "z_time"},
		Descending: []bool{true, true},
	})
	if err != nil {
		t.Fatalf("Sort desc: %v", err)
	}
	if got := col64(t, out, "z_id"); !eqInt64(got, []int64{2, 2, 1, 1}) {
		t.Errorf("z_id order = %v, want [2 2 1 1]", got)
	}
	if got := col64(t, out, "z_time"); !eqInt64(got, []int64{5, 1, 9, 3}) {
		t.Errorf("z_time order = %v, want [5 1 9 3]", got)
	}
}
