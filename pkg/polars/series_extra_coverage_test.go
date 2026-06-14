package polars

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

func intSeries(t *testing.T, name string, vals ...any) Series {
	t.Helper()
	s, err := NewSeries(NewSeriesInput{Name: name, DType: dtypes.Int64, Values: vals})
	if err != nil {
		t.Fatalf("int series %s: %v", name, err)
	}
	return s
}

// TestSeriesExportAndReprMethods covers the export/representation helpers.
func TestSeriesExportAndReprMethods(t *testing.T) {
	t.Parallel()

	s := numSeries(t, "v", 1.0, 2.0, 3.0)

	if s.ToInitRepr() == "" {
		t.Error("ToInitRepr returned empty string")
	}
	if got := s.ToJax(); len(got) != 3 {
		t.Errorf("ToJax len = %d, want 3", len(got))
	}
	if got := s.ToTorch(); len(got) != 3 {
		t.Errorf("ToTorch len = %d, want 3", len(got))
	}
	if got := s.ToPandas(); len(got) != 3 {
		t.Errorf("ToPandas len = %d, want 3", len(got))
	}
	if got := s.ToPhysical(); got.Len() != 3 {
		t.Errorf("ToPhysical len = %d, want 3", got.Len())
	}
	if desc := s.Describe(); len(desc) == 0 {
		t.Error("Describe returned empty map")
	}
	hist, err := s.Hist(2)
	if err != nil || hist.Height() == 0 {
		t.Fatalf("Hist height=%d err=%v", hist.Height(), err)
	}
	if h := s.Hash(7); h.Len() != 3 {
		t.Errorf("Hash len = %d, want 3", h.Len())
	}
}

// TestSeriesSerializeRoundTrip covers Series Serialize/Deserialize.
func TestSeriesSerializeRoundTrip(t *testing.T) {
	t.Parallel()

	s := numSeries(t, "v", 1.0, 2.0, 3.0)
	payload, err := s.Serialize()
	if err != nil || len(payload) == 0 {
		t.Fatalf("Serialize: payload=%d err=%v", len(payload), err)
	}
	back, err := s.Deserialize(payload)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	if back.Len() != 3 {
		t.Errorf("round-trip len = %d, want 3", back.Len())
	}
}

// TestSeriesImplodeExplodeFlatten covers the list-shaping helpers.
func TestSeriesImplodeExplodeFlatten(t *testing.T) {
	t.Parallel()

	s := numSeries(t, "v", 1.0, 2.0, 3.0)

	// Implode collapses the series into a single list element.
	imploded := s.Implode()
	if imploded.Len() != 1 {
		t.Fatalf("Implode len = %d, want 1", imploded.Len())
	}
	// Explode expands it back to the original length.
	exploded := imploded.Explode()
	if exploded.Len() != 3 {
		t.Fatalf("Explode len = %d, want 3", exploded.Len())
	}
	// Flatten on a flat series is a no-op.
	if got := s.Flatten(); got.Len() != 3 {
		t.Errorf("Flatten len = %d, want 3", got.Len())
	}
}

// TestSeriesNumericLowPriority covers the low-priority numeric helpers.
func TestSeriesNumericLowPriority(t *testing.T) {
	t.Parallel()

	f := numSeries(t, "f", 0.5, 1.0, 1.5)
	if got := f.Cot(); got.Len() != 3 {
		t.Errorf("Cot len = %d, want 3", got.Len())
	}
	if got := f.Not_(); got.Len() != 3 {
		t.Errorf("Not_ len = %d, want 3", got.Len())
	}

	by := numSeries(t, "by", 1.0, 2.0, 3.0)
	rm, err := f.RollingMeanBy(by, 2)
	if err != nil || rm.Len() != 3 {
		t.Fatalf("RollingMeanBy len=%d err=%v", rm.Len(), err)
	}

	a := intSeries(t, "a", int64(6), int64(12), int64(8))
	b := intSeries(t, "b", int64(3), int64(10), int64(8))
	and, err := a.BitwiseAnd(b)
	if err != nil || and.Len() != 3 {
		t.Fatalf("BitwiseAnd len=%d err=%v", and.Len(), err)
	}
	shr, err := a.ShrinkDType()
	if err != nil {
		t.Fatalf("ShrinkDType: %v", err)
	}
	if shr.Len() != 3 {
		t.Fatalf("ShrinkDType len = %d, want 3", shr.Len())
	}
	// Reinterpret is intentionally unsupported in v1 and must report an error.
	if _, err := a.Reinterpret(dtypes.Int64); err == nil {
		t.Fatal("Reinterpret should return an unsupported error")
	}
}

// TestSeriesListNamespace covers the .Arr()/.List() list namespace ListLen.
func TestSeriesListNamespace(t *testing.T) {
	t.Parallel()

	s, err := NewSeries(NewSeriesInput{
		Name:   "lists",
		DType:  dtypes.List,
		Values: []any{[]any{int64(1), int64(2)}, []any{int64(3)}},
	})
	if err != nil {
		t.Fatalf("list series: %v", err)
	}

	lens, err := s.Arr().ListLen()
	if err != nil {
		t.Fatalf("Arr().ListLen: %v", err)
	}
	if lens.Value(0) != int64(2) || lens.Value(1) != int64(1) {
		t.Fatalf("ListLen = [%v %v], want [2 1]", lens.Value(0), lens.Value(1))
	}

	// .List() is the same namespace.
	if _, err := s.List().ListLen(); err != nil {
		t.Fatalf("List().ListLen: %v", err)
	}
}
