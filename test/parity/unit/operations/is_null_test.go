package operations

// Ported from py-polars/tests/unit/operations/test_is_null.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_is_null_null: is_null/is_not_null over an all-null int series.
func TestIsNullAllNull(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{nil, nil}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	isNull := s.IsNull()
	for i := 0; i < 2; i++ {
		if v, ok := isNull.Value(i).(bool); !ok || !v {
			t.Fatalf("is_null[%d]: got %v, want true", i, isNull.Value(i))
		}
	}
	isNotNull := s.IsNotNull()
	for i := 0; i < 2; i++ {
		if v, ok := isNotNull.Value(i).(bool); !ok || v {
			t.Fatalf("is_not_null[%d]: got %v, want false", i, isNotNull.Value(i))
		}
	}
}

// is_null and is_not_null are inverses; is_null has no nulls itself.
func TestIsNullInverse(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), nil, int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	isNull := s.IsNull()
	isNotNull := s.IsNotNull()
	if isNull.NullCount() != 0 {
		t.Fatalf("is_null has nulls: %d", isNull.NullCount())
	}
	for i := 0; i < 3; i++ {
		n, _ := isNull.Value(i).(bool)
		nn, _ := isNotNull.Value(i).(bool)
		if n == nn {
			t.Fatalf("is_null and is_not_null not inverse at %d: %v / %v", i, n, nn)
		}
	}
	// middle element is the null
	if v, _ := isNull.Value(1).(bool); !v {
		t.Fatalf("is_null[1]: got %v, want true", isNull.Value(1))
	}
}
