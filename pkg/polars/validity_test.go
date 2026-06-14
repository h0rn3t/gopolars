package polars

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

func mkFloatSeries(t testing.TB, vals []any) Series {
	t.Helper()
	s, err := NewSeries(NewSeriesInput{Name: "v", DType: dtypes.Float64, Values: vals})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	return s
}

func TestNullCountFacade(t *testing.T) {
	s := mkFloatSeries(t, []any{1.0, nil, 3.0, nil, 5.0})
	if got := s.NullCount(); got != 2 {
		t.Fatalf("NullCount = %d, want 2", got)
	}
	// Repeated calls return the cached value.
	if got := s.NullCount(); got != 2 {
		t.Fatalf("cached NullCount = %d, want 2", got)
	}
}

func TestIsNullIsNotNullInverse(t *testing.T) {
	s := mkFloatSeries(t, []any{1.0, nil, 3.0, nil, 5.0})
	isNull := s.IsNull()
	isNotNull := s.IsNotNull()

	if isNull.DataType() != dtypes.Boolean || isNotNull.DataType() != dtypes.Boolean {
		t.Fatalf("validity masks must be boolean")
	}
	want := []bool{false, true, false, true, false}
	for i := 0; i < s.Len(); i++ {
		gv := isNull.Value(i).(bool)
		if gv != want[i] {
			t.Errorf("is_null[%d] = %v, want %v", i, gv, want[i])
		}
		// is_not_null must be the exact inverse.
		if isNotNull.Value(i).(bool) == gv {
			t.Errorf("is_not_null[%d] should be the inverse of is_null", i)
		}
	}
	// The masks are built with no validity buffer, so they contain no nulls.
	if isNull.NullCount() != 0 || isNotNull.NullCount() != 0 {
		t.Errorf("validity mask must not contain nulls")
	}
}

func TestIsNotNullNullFreeAllTrue(t *testing.T) {
	s := mkFloatSeries(t, []any{1.0, 2.0, 3.0, 4.0})
	got := s.IsNotNull()
	if got.NullCount() != 0 {
		t.Fatalf("validity mask must not contain nulls")
	}
	for i := 0; i < s.Len(); i++ {
		if !got.Value(i).(bool) {
			t.Errorf("is_not_null[%d] = false, want true on a null-free column", i)
		}
	}
}

// TestIsNotNullAllocationDoesNotScale proves is_not_null builds its result from
// the validity state in a bounded number of allocations rather than boxing per
// row: the alloc count at 64K rows must not exceed the count at 1K rows.
func TestIsNotNullAllocationDoesNotScale(t *testing.T) {
	mk := func(n int) Series {
		vals := make([]any, n)
		for i := range vals {
			vals[i] = float64(i)
		}
		return mkFloatSeries(t, vals)
	}
	small := mk(1024)
	large := mk(1 << 16)
	aSmall := testing.AllocsPerRun(50, func() { _ = small.IsNotNull() })
	aLarge := testing.AllocsPerRun(50, func() { _ = large.IsNotNull() })
	if aLarge > aSmall {
		t.Fatalf("is_not_null allocation scales with rows: n=1024 -> %.0f allocs, n=65536 -> %.0f allocs", aSmall, aLarge)
	}
}

// BenchmarkIsNullNullFree and BenchmarkIsNotNullNullFree run on a null-free
// column — the shape the cross-language top30 suite exercises — so the two can
// be compared directly. After the memmove-backed all-true fill they should sit
// within a small constant factor instead of is_not_null running ~7x slower.
func BenchmarkIsNullNullFree(b *testing.B) {
	vals := make([]any, 1_000_000)
	for i := range vals {
		vals[i] = float64(i)
	}
	s := mkFloatSeries(b, vals)
	_ = s.NullCount()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.IsNull()
	}
}

func BenchmarkIsNotNullNullFree(b *testing.B) {
	vals := make([]any, 1_000_000)
	for i := range vals {
		vals[i] = float64(i)
	}
	s := mkFloatSeries(b, vals)
	_ = s.NullCount()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.IsNotNull()
	}
}

func BenchmarkNullCountCached(b *testing.B) {
	vals := make([]any, 1_000_000)
	for i := range vals {
		if i%10 == 0 {
			vals[i] = nil
		} else {
			vals[i] = float64(i)
		}
	}
	s := mkFloatSeries(b, vals)
	_ = s.NullCount() // warm the cache
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.NullCount()
	}
}

func BenchmarkIsNull(b *testing.B) {
	vals := make([]any, 1_000_000)
	for i := range vals {
		if i%10 == 0 {
			vals[i] = nil
		} else {
			vals[i] = float64(i)
		}
	}
	s := mkFloatSeries(b, vals)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.IsNull()
	}
}
