package polars

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestSeriesCount verifies the documented count = length - nulls.
func TestSeriesCount(t *testing.T) {
	s := newFloatSeries(t, "v", []any{1.0, nil, 3.0, nil, 5.0})
	if v := s.Count(); v != 3 {
		t.Errorf("Count = %d, want 3 (length 5 minus 2 nulls)", v)
	}
}

// TestSeriesApproxNUnique delegates to NUnique.
func TestSeriesApproxNUnique(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1), int64(2), int64(2), int64(3)})
	if v := s.ApproxNUnique(); v != 3 {
		t.Errorf("ApproxNUnique = %d, want 3", v)
	}
}

// TestSeriesEquals verifies the documented equality semantics.
func TestSeriesEquals(t *testing.T) {
	a := newInt64Series(t, "a", []any{int64(1), int64(2), int64(3)})
	b := newInt64Series(t, "b", []any{int64(1), int64(2), int64(3)})
	c := newInt64Series(t, "c", []any{int64(1), int64(2), int64(4)})
	d := newStringSeries(t, "d", []any{"x", "y"})

	eq, err := a.Equals(b)
	if err != nil || !eq {
		t.Errorf("a.Equals(b) = (%v, %v), want (true, nil)", eq, err)
	}
	eq, err = a.Equals(c)
	if err != nil || eq {
		t.Errorf("a.Equals(c) = (%v, %v), want (false, nil)", eq, err)
	}
	// Different dtype → false (not an error).
	eq, err = a.Equals(d)
	if err != nil || eq {
		t.Errorf("a.Equals(d) = (%v, %v), want (false, nil)", eq, err)
	}
}

// TestSeriesEstimatedSize verifies the documented size estimate.
func TestSeriesEstimatedSize(t *testing.T) {
	s := newFloatSeries(t, "v", []any{1.0, 2.0, 3.0})
	if v := s.EstimatedSize(); v <= 0 {
		t.Errorf("EstimatedSize = %d, want > 0", v)
	}
	empty := newFloatSeries(t, "e", nil)
	if v := empty.EstimatedSize(); v != 0 {
		t.Errorf("EstimatedSize(empty) = %d, want 0", v)
	}
}

// TestSeriesFilterMismatchedLength returns an error.
func TestSeriesFilterMismatchedLength(t *testing.T) {
	s := newInt64Series(t, "a", []any{int64(1), int64(2), int64(3)})
	mask, _ := NewSeries(NewSeriesInput{Name: "m", DType: dtypes.Boolean, Values: []any{true}})
	if _, err := s.Filter(mask); err == nil {
		t.Errorf("Filter with mismatched mask length returned nil error, want non-nil")
	}
}

// TestSeriesTruncate covers the documented string-truncation behavior.
func TestSeriesTruncate(t *testing.T) {
	s := newStringSeries(t, "s", []any{"hello", "world!"})
	out, err := s.Truncate(3)
	if err != nil {
		t.Fatalf("Truncate: %v", err)
	}
	if v, _ := out.Value(0).(string); v != "hel" {
		t.Errorf("Truncate(3)[0] = %v, want hel", out.Value(0))
	}
	// Truncate on int64 returns an error.
	if _, err := newInt64Series(t, "n", []any{int64(1)}).Truncate(1); err == nil {
		t.Errorf("Truncate on int64 returned nil error, want non-nil")
	}
	// Negative maxLen returns an error.
	if _, err := s.Truncate(-1); err == nil {
		t.Errorf("Truncate(-1) returned nil error, want non-nil")
	}
}

// TestSeriesRoundSigFigs covers the documented significant-figures rounding.
func TestSeriesRoundSigFigs(t *testing.T) {
	s := newFloatSeries(t, "v", []any{1.234, 5.678, 0.0001234})
	out, err := s.RoundSigFigs(2)
	if err != nil {
		t.Fatalf("RoundSigFigs: %v", err)
	}
	if v, _ := out.Value(0).(float64); math.Abs(v-1.2) > 1e-9 {
		t.Errorf("RoundSigFigs(2)[0] = %v, want 1.2", v)
	}
}

// TestSeriesClip covers the documented clamping.
func TestSeriesClip(t *testing.T) {
	s := newFloatSeries(t, "v", []any{-5.0, 0.0, 5.0, 10.0})
	out := s.Clip(0.0, 5.0)
	want := []float64{0, 0, 5, 5}
	for i, w := range want {
		if v, _ := out.Value(i).(float64); v != w {
			t.Errorf("Clip[%d] = %v, want %v", i, v, w)
		}
	}
}

// TestSeriesList covers the documented list-alias.
func TestSeriesList(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1)})
	ns := s.List()
	if ns == (SeriesArrNS{}) {
		t.Errorf("List returned a zero-value SeriesArrNS")
	}
}

// TestSeriesAppend concatenates two series and returns a series of length
// len(a) + len(b).
func TestSeriesAppend(t *testing.T) {
	a := newInt64Series(t, "a", []any{int64(1), int64(2)})
	b := newInt64Series(t, "b", []any{int64(3), int64(4)})
	out, err := a.Append(b)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if out.Len() != 4 {
		t.Errorf("Append len = %d, want 4", out.Len())
	}
}

// TestSeriesGetChunksAndFlags exercises the chunk and flags accessors.
func TestSeriesGetChunksAndFlags(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1), int64(2)})
	chunks, err := s.GetChunks()
	if err != nil {
		t.Fatalf("GetChunks: %v", err)
	}
	if len(chunks) == 0 {
		t.Errorf("GetChunks returned no chunks")
	}
	flags := s.Flags()
	if flags == nil {
		t.Errorf("Flags returned nil map")
	}
}

// TestSeriesMaxByMinBy exercises the by-aggregation methods. With all
// distinct keys in `by`, each row is its own group, so MaxBy/MinBy return
// the original `v` values.
func TestSeriesMaxByMinBy(t *testing.T) {
	v := newInt64Series(t, "v", []any{int64(10), int64(20), int64(30)})
	by := newInt64Series(t, "by", []any{int64(1), int64(3), int64(2)})
	out, err := v.MaxBy(by)
	if err != nil {
		t.Fatalf("MaxBy: %v", err)
	}
	for i, w := range []int64{10, 20, 30} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Errorf("MaxBy[%d] = %d, want %d", i, v, w)
		}
	}
	out, err = v.MinBy(by)
	if err != nil {
		t.Fatalf("MinBy: %v", err)
	}
	for i, w := range []int64{10, 20, 30} {
		if v, _ := out.Value(i).(int64); v != w {
			t.Errorf("MinBy[%d] = %d, want %d", i, v, w)
		}
	}

	// Grouped case: by=[x,x,y] means rows 0,1 are group x → max(v[0..1])=20,
	// row 2 is group y → max(v[2])=30.
	v = newInt64Series(t, "v", []any{int64(10), int64(20), int64(30)})
	by, _ = NewSeries(NewSeriesInput{Name: "by", DType: dtypes.String, Values: []any{"x", "x", "y"}})
	out, _ = v.MaxBy(by)
	if out.Len() != 2 {
		t.Errorf("grouped MaxBy len = %d, want 2", out.Len())
	}
}

// TestSeriesBitwiseCountOnes exercises the bitwise namespace.
func TestSeriesBitwiseCountOnes(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(0b1010), int64(0b1111), int64(0b0001)})
	out, err := s.Bin().CountOnes()
	if err != nil {
		t.Fatalf("Bin.CountOnes: %v", err)
	}
	want := []int64{2, 4, 1}
	for i, w := range want {
		if v, _ := out.Value(i).(int64); v != w {
			t.Errorf("Bin.CountOnes[%d] = %d, want %d", i, v, w)
		}
	}
}
