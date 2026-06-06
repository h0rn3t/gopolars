package series

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestSeriesFilter pins the row-keeping contract of Series.Filter: rows where
// mask[i] is true are retained, preserving order, name, and nulls.
func TestSeriesFilter(t *testing.T) {
	s, err := New("id", dtypes.Int64, []any{int64(10), int64(20), int64(30), int64(40)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out := s.Filter([]bool{true, false, true, false})
	if out.Name() != "id" {
		t.Errorf("Name = %q, want id", out.Name())
	}
	if out.Len() != 2 {
		t.Fatalf("Len = %d, want 2", out.Len())
	}
	got := Int64Values(out)
	if got[0] != 10 || got[1] != 30 {
		t.Errorf("Filter values = %v, want [10 30]", got)
	}
}

// TestSeriesFilterPreservesNulls confirms the validity mask survives filtering.
func TestSeriesFilterPreservesNulls(t *testing.T) {
	s, err := New("v", dtypes.Int64, []any{int64(1), nil, int64(3)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	out := s.Filter([]bool{false, true, true})
	if out.Len() != 2 {
		t.Fatalf("Len = %d, want 2", out.Len())
	}
	if !out.IsNull(0) {
		t.Errorf("IsNull(0) = false, want true (kept null row)")
	}
	if out.IsNull(1) {
		t.Errorf("IsNull(1) = true, want false")
	}
}
