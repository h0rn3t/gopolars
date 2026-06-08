package operations

// Ported from py-polars/tests/unit/operations/test_diff.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// diff(1): first element null, rest are consecutive differences.
func TestDiffDefault(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(3), int64(6), int64(10)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Diff(1)
	if out.Value(0) != nil {
		t.Fatalf("diff[0]: got %v, want nil", out.Value(0))
	}
	for i, w := range []int64{2, 3, 4} {
		switch v := out.Value(i + 1).(type) {
		case int64:
			if v != w {
				t.Fatalf("diff[%d]: got %d, want %d", i+1, v, w)
			}
		case float64:
			if v != float64(w) {
				t.Fatalf("diff[%d]: got %v, want %d", i+1, v, w)
			}
		default:
			t.Fatalf("diff[%d]: unexpected type %T", i+1, out.Value(i+1))
		}
	}
}

// diff(2): first two elements null.
func TestDiffN2(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(4), int64(8)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Diff(2)
	if out.Value(0) != nil || out.Value(1) != nil {
		t.Fatalf("diff(2) head: got %v,%v, want nil,nil", out.Value(0), out.Value(1))
	}
}
