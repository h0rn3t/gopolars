package operations

// Ported from several small py-polars/tests/unit/operations files (py-1.28.1):
// test_is_sorted.py, test_is_first_last_distinct.py, test_has_nulls.py,
// test_empty.py, test_clear.py, test_gather.py, test_search_sorted.py,
// test_pct_change.py, test_hash.py, test_interpolate.py, test_extend_constant.py,
// test_rolling.py — representative subsets, grouped to keep one file per concern.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func mkInts(t *testing.T, vals ...any) polars.Series {
	t.Helper()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: vals})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	return s
}

// test_is_sorted
func TestIsSorted(t *testing.T) {
	t.Parallel()
	if !mkInts(t, int64(1), int64(2), int64(3)).IsSorted() {
		t.Fatal("ascending series should be sorted")
	}
	if mkInts(t, int64(3), int64(1), int64(2)).IsSorted() {
		t.Fatal("unsorted series should not be sorted")
	}
}

// test_is_first_last_distinct
func TestIsFirstLastDistinct(t *testing.T) {
	t.Parallel()
	s := mkInts(t, int64(1), int64(1), int64(2), int64(3), int64(2))
	first := s.IsFirstDistinct()
	for i, w := range []bool{true, false, true, true, false} {
		if v, _ := first.Value(i).(bool); v != w {
			t.Fatalf("is_first_distinct[%d]: got %v, want %v", i, first.Value(i), w)
		}
	}
	last := s.IsLastDistinct()
	for i, w := range []bool{false, true, false, true, true} {
		if v, _ := last.Value(i).(bool); v != w {
			t.Fatalf("is_last_distinct[%d]: got %v, want %v", i, last.Value(i), w)
		}
	}
}

// test_has_nulls
func TestHasNulls(t *testing.T) {
	t.Parallel()
	if mkInts(t, int64(1), int64(2)).HasNulls() {
		t.Fatal("series without nulls reported HasNulls=true")
	}
	if !mkInts(t, int64(1), nil, int64(3)).HasNulls() {
		t.Fatal("series with nulls reported HasNulls=false")
	}
}

// test_empty
func TestEmpty(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if !s.IsEmpty() {
		t.Fatal("empty series IsEmpty=false")
	}
	if mkInts(t, int64(1)).IsEmpty() {
		t.Fatal("non-empty series IsEmpty=true")
	}
}

// test_clear
func TestClear(t *testing.T) {
	t.Parallel()
	out := mkInts(t, int64(1), int64(2), int64(3)).Clear()
	if out.Len() != 0 {
		t.Fatalf("clear len: got %d, want 0", out.Len())
	}
	if out.DataType() != polars.Int64 {
		t.Fatalf("clear dtype: got %v, want Int64 (preserved)", out.DataType())
	}
}

// test_gather
func TestGather(t *testing.T) {
	t.Parallel()
	out := mkInts(t, int64(10), int64(20), int64(30), int64(40)).Gather([]int{0, 2, 3})
	for i, w := range []int64{10, 30, 40} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("gather[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}

// test_search_sorted
func TestSearchSorted(t *testing.T) {
	t.Parallel()
	s := mkInts(t, int64(1), int64(3), int64(5), int64(7))
	if idx := s.SearchSorted(int64(5)); idx != 2 {
		t.Fatalf("search_sorted(5): got %d, want 2", idx)
	}
	if idx := s.SearchSorted(int64(4)); idx != 2 {
		t.Fatalf("search_sorted(4): got %d, want 2 (insertion point)", idx)
	}
}

// test_pct_change
func TestPctChange(t *testing.T) {
	t.Parallel()
	s := mkInts(t, int64(10), int64(11), int64(12))
	out := s.PctChange(1)
	if out.Value(0) != nil {
		t.Fatalf("pct_change[0]: got %v, want nil", out.Value(0))
	}
	if v, _ := out.Value(1).(float64); v < 0.09 || v > 0.11 {
		t.Fatalf("pct_change[1]: got %v, want ~0.1", out.Value(1))
	}
}

// test_extend_constant
func TestExtendConstant(t *testing.T) {
	t.Parallel()
	out := mkInts(t, int64(1), int64(2)).ExtendConstant(int64(9), 3)
	if out.Len() != 5 {
		t.Fatalf("len: got %d, want 5", out.Len())
	}
	for i, w := range []int64{1, 2, 9, 9, 9} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Fatalf("extend[%d]: got %v, want %d", i, out.Value(i), w)
		}
	}
}

// test_interpolate
func TestInterpolate(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{1.0, nil, 3.0}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Interpolate()
	if v, _ := out.Value(1).(float64); v != 2.0 {
		t.Fatalf("interpolate[1]: got %v, want 2.0", out.Value(1))
	}
}

// test_rolling (Series rolling mean): min_periods defaults to the window size,
// so the leading window-1 outputs are null (matching Polars).
func TestRollingMean(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{1.0, 2.0, 3.0, 4.0}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.RollingMean(2)
	if out.Value(0) != nil {
		t.Fatalf("rolling[0]: got %v, want nil (min_periods=window)", out.Value(0))
	}
	if v, _ := out.Value(1).(float64); v != 1.5 {
		t.Fatalf("rolling[1]: got %v, want 1.5", out.Value(1))
	}
	if v, _ := out.Value(3).(float64); v != 3.5 {
		t.Fatalf("rolling[3]: got %v, want 3.5", out.Value(3))
	}
}
