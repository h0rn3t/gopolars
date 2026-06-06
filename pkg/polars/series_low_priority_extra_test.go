package polars

import (
	"testing"
)

// TestSeriesRollingByVariants exercises the *By rolling reductions that scatter
// observations along a sort key (the RollingMeanBy sibling is covered elsewhere).
func TestSeriesRollingByVariants(t *testing.T) {
	v := newFloatSeries(t, "v", []any{1.0, 2.0, 4.0, 8.0})
	by := newInt64Series(t, "t", []any{int64(0), int64(1), int64(2), int64(3)})

	cases := []struct {
		name string
		fn   func() (Series, error)
	}{
		{"RollingSumBy", func() (Series, error) { return v.RollingSumBy(by, 2) }},
		{"RollingMinBy", func() (Series, error) { return v.RollingMinBy(by, 2) }},
		{"RollingMaxBy", func() (Series, error) { return v.RollingMaxBy(by, 2) }},
		{"RollingStdBy", func() (Series, error) { return v.RollingStdBy(by, 2) }},
		{"RollingVarBy", func() (Series, error) { return v.RollingVarBy(by, 2) }},
		{"RollingMedianBy", func() (Series, error) { return v.RollingMedianBy(by, 2) }},
		{"RollingQuantileBy", func() (Series, error) { return v.RollingQuantileBy(by, 2, 0.5) }},
		{"RollingRankBy", func() (Series, error) { return v.RollingRankBy(by, 2) }},
		{"EwmMeanBy", func() (Series, error) { return v.EwmMeanBy(by, 0.5) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.fn()
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if out.Len() != 4 {
				t.Errorf("%s Len = %d, want 4", tc.name, out.Len())
			}
		})
	}
}

// TestSeriesRollingByLengthMismatch confirms the *By variants reject a mismatched
// sort key.
func TestSeriesRollingByLengthMismatch(t *testing.T) {
	v := newFloatSeries(t, "v", []any{1.0, 2.0, 4.0})
	by := newInt64Series(t, "t", []any{int64(0), int64(1)})
	if _, err := v.RollingSumBy(by, 2); err == nil {
		t.Error("RollingSumBy with mismatched lengths: expected error")
	}
	if _, err := v.RollingQuantileBy(by, 2, 0.5); err == nil {
		t.Error("RollingQuantileBy with mismatched lengths: expected error")
	}
	if _, err := v.EwmMeanBy(by, 0.5); err == nil {
		t.Error("EwmMeanBy with mismatched lengths: expected error")
	}
}

// TestSeriesRollingRankSkewKurtosis exercises the windowed rank/skew/kurtosis
// reductions that return a Series directly.
func TestSeriesRollingRankSkewKurtosis(t *testing.T) {
	v := newFloatSeries(t, "v", []any{1.0, 3.0, 2.0, 5.0, 4.0})
	if out := v.RollingRank(3); out.Len() != 5 {
		t.Errorf("RollingRank Len = %d, want 5", out.Len())
	}
	if out := v.RollingSkew(3); out.Len() != 5 {
		t.Errorf("RollingSkew Len = %d, want 5", out.Len())
	}
	if out := v.RollingKurtosis(3); out.Len() != 5 {
		t.Errorf("RollingKurtosis Len = %d, want 5", out.Len())
	}
}

// TestSeriesRollingMap exercises the custom windowed reducer, including the nil
// function guard.
func TestSeriesRollingMap(t *testing.T) {
	v := newFloatSeries(t, "v", []any{1.0, 2.0, 3.0, 4.0})

	// Sum reducer over a window of 2: [1, 3, 5, 7].
	out, err := v.RollingMap(2, func(xs []float64) float64 {
		var sum float64
		for _, x := range xs {
			sum += x
		}
		return sum
	})
	if err != nil {
		t.Fatalf("RollingMap: %v", err)
	}
	want := []float64{1, 3, 5, 7}
	for i, w := range want {
		if got, _ := out.Value(i).(float64); got != w {
			t.Errorf("RollingMap[%d] = %v, want %v", i, out.Value(i), w)
		}
	}

	if _, err := v.RollingMap(2, nil); err == nil {
		t.Error("RollingMap(nil): expected error")
	}
}

// TestSeriesBitwiseBinary exercises the elementwise Or/Xor (the And sibling is
// covered elsewhere).
func TestSeriesBitwiseBinary(t *testing.T) {
	a := newInt64Series(t, "a", []any{int64(0b1010), int64(0b1100), int64(0b0011)})
	b := newInt64Series(t, "b", []any{int64(0b0110), int64(0b1010), int64(0b0101)})

	or, err := a.BitwiseOr(b)
	if err != nil {
		t.Fatalf("BitwiseOr: %v", err)
	}
	wantOr := []int64{0b1110, 0b1110, 0b0111}
	for i, w := range wantOr {
		if v, _ := or.Value(i).(int64); v != w {
			t.Errorf("BitwiseOr[%d] = %b, want %b", i, v, w)
		}
	}

	xor, err := a.BitwiseXor(b)
	if err != nil {
		t.Fatalf("BitwiseXor: %v", err)
	}
	wantXor := []int64{0b1100, 0b0110, 0b0110}
	for i, w := range wantXor {
		if v, _ := xor.Value(i).(int64); v != w {
			t.Errorf("BitwiseXor[%d] = %b, want %b", i, v, w)
		}
	}
}

// TestSeriesBitwiseBinaryNonInt confirms the binary ops reject non-int64 inputs.
func TestSeriesBitwiseBinaryNonInt(t *testing.T) {
	a := newFloatSeries(t, "a", []any{1.0, 2.0, 3.0})
	b := newInt64Series(t, "b", []any{int64(1), int64(2), int64(3)})
	if _, err := a.BitwiseOr(b); err == nil {
		t.Error("BitwiseOr on float series: expected error")
	}
	if _, err := a.BitwiseXor(b); err == nil {
		t.Error("BitwiseXor on float series: expected error")
	}
}

// TestSeriesBitwiseUnaryCounts exercises the unary bit-counting reductions
// (the CountOnes sibling is covered elsewhere).
func TestSeriesBitwiseUnaryCounts(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(0b1010), int64(0b1111), int64(1)})

	countZeros, err := s.BitwiseCountZeros()
	if err != nil {
		t.Fatalf("BitwiseCountZeros: %v", err)
	}
	// 64-bit words: 0b1010 has 2 set bits → 62 zeros; 0b1111 → 60; 1 → 63.
	wantZeros := []int64{62, 60, 63}
	for i, w := range wantZeros {
		if v, _ := countZeros.Value(i).(int64); v != w {
			t.Errorf("BitwiseCountZeros[%d] = %d, want %d", i, v, w)
		}
	}

	// Leading/trailing reductions: assert shape and self-consistency rather than
	// pin every bit pattern.
	for _, tc := range []struct {
		name string
		fn   func() (Series, error)
	}{
		{"BitwiseLeadingOnes", s.BitwiseLeadingOnes},
		{"BitwiseLeadingZeros", s.BitwiseLeadingZeros},
		{"BitwiseTrailingOnes", s.BitwiseTrailingOnes},
		{"BitwiseTrailingZeros", s.BitwiseTrailingZeros},
	} {
		out, err := tc.fn()
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if out.Len() != 3 {
			t.Errorf("%s Len = %d, want 3", tc.name, out.Len())
		}
	}

	// TrailingZeros of 1 (0b...0001) is 0; TrailingZeros of 0b1010 is 1.
	tz, _ := s.BitwiseTrailingZeros()
	if v, _ := tz.Value(0).(int64); v != 1 {
		t.Errorf("BitwiseTrailingZeros(0b1010) = %d, want 1", v)
	}
	if v, _ := tz.Value(2).(int64); v != 0 {
		t.Errorf("BitwiseTrailingZeros(1) = %d, want 0", v)
	}
}

// TestSeriesBitwiseUnaryNonInt confirms the unary ops reject non-int64 inputs.
func TestSeriesBitwiseUnaryNonInt(t *testing.T) {
	s := newFloatSeries(t, "v", []any{1.0, 2.0})
	if _, err := s.BitwiseCountZeros(); err == nil {
		t.Error("BitwiseCountZeros on float series: expected error")
	}
	if _, err := s.BitwiseLeadingZeros(); err == nil {
		t.Error("BitwiseLeadingZeros on float series: expected error")
	}
}
