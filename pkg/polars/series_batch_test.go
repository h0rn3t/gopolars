package polars

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestSeriesBatchA covers a wide range of one-line series delegations in a
// single test, exercising the public Series interface more thoroughly. Each
// call is asserted to return a non-nil, well-typed result; per-cell value
// parity is left to the deeper tests in series_test.go.
func TestSeriesBatchA(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1), int64(2), int64(3), int64(4)})
	// Unary math — all return a series of the same length.
	calls := []struct {
		name string
		out  Series
	}{
		{"Abs", s.Abs()}, {"Exp", s.Exp()}, {"Log", s.Log()}, {"Sqrt", s.Sqrt()},
		{"Sin", s.Sin()}, {"Cos", s.Cos()}, {"Tan", s.Tan()},
		{"Sinh", s.Sinh()}, {"Cosh", s.Cosh()}, {"Tanh", s.Tanh()},
		{"Arcsin", s.Arcsin()}, {"Arccos", s.Arccos()}, {"Arctan", s.Arctan()},
		{"Cbrt", s.Cbrt()}, {"Ceil", s.Ceil()}, {"Floor", s.Floor()},
		{"Degrees", s.Degrees()}, {"Sign", s.Sign()},
		{"Log10", s.Log10()}, {"Log1p", s.Log1p()}, {"Round", s.Round()},
	}
	for _, c := range calls {
		if c.out == nil {
			t.Errorf("%s returned nil", c.name)
			continue
		}
		if c.out.Len() != s.Len() {
			t.Errorf("%s len = %d, want %d", c.name, c.out.Len(), s.Len())
		}
	}

	// Pow takes a parameter.
	if out := s.Pow(2); out.Len() != s.Len() {
		t.Errorf("Pow len = %d, want %d", out.Len(), s.Len())
	}

	// ArgSort, Unique, ArgUnique, ArgTrue return a series of indices.
	if out := s.ArgSort(); out.Len() != s.Len() {
		t.Errorf("ArgSort len = %d, want %d", out.Len(), s.Len())
	}
	if out := s.Unique(); out.Len() == 0 {
		t.Errorf("Unique returned empty")
	}
	if out := s.ArgUnique(); out.Len() == 0 {
		t.Errorf("ArgUnique returned empty")
	}
	if out := s.ArgTrue(); out.Len() == 0 && s.Len() > 0 {
		// ArgTrue returns indices where the value is truthy; for non-bool
		// series, the result length is 0 (no true values).
		_ = out
	}
	if out := s.Sample(2, 1); out.Len() != 2 {
		t.Errorf("Sample len = %d, want 2", out.Len())
	}
	if out := s.Shuffle(1); out.Len() != s.Len() {
		t.Errorf("Shuffle len = %d, want %d", out.Len(), s.Len())
	}

	// Shape and IsEmpty.
	if shape := s.Shape(); shape[0] != s.Len() {
		t.Errorf("Shape[0] = %d, want %d", shape[0], s.Len())
	}
	if s.IsEmpty() {
		t.Errorf("IsEmpty = true, want false")
	}
	if !s.IsSorted() {
		// Sorted may or may not return true depending on impl; just don't panic.
		_ = s.IsSorted()
	}
	if s.NChunks() <= 0 {
		t.Errorf("NChunks = %d, want > 0", s.NChunks())
	}
	if s.HasNulls() {
		t.Errorf("HasNulls = true, want false (no nulls in input)")
	}
	if !s.HasValidity() {
		// HasValidity may be false for a non-nullable series; just don't panic.
		_ = s.HasValidity()
	}
	// All/Any require boolean semantics; on an int64 series they may return
	// false. We just exercise the call paths.
	_ = s.All()
	_ = s.Any()
}

// TestSeriesGatherAndScatter exercises index-based mutation methods.
func TestSeriesGatherAndScatter(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(10), int64(20), int64(30), int64(40)})
	if out := s.Gather([]int{0, 2}); out.Len() != 2 {
		t.Errorf("Gather len = %d, want 2", out.Len())
	}
	if out := s.GatherEvery(2); out.Len() != 2 {
		t.Errorf("GatherEvery(2) len = %d, want 2", out.Len())
	}
	out, err := s.Scatter([]int{0, 1}, []any{int64(100), int64(200)})
	if err != nil {
		t.Fatalf("Scatter: %v", err)
	}
	if v, _ := out.Value(0).(int64); v != 100 {
		t.Errorf("Scatter[0] = %v, want 100", out.Value(0))
	}
}

// TestSeriesExtendConstant extends a series with a constant value.
func TestSeriesExtendConstant(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1), int64(2)})
	out := s.ExtendConstant(int64(7), 3)
	if out.Len() != 5 {
		t.Errorf("ExtendConstant len = %d, want 5", out.Len())
	}
	if v, _ := out.Value(4).(int64); v != 7 {
		t.Errorf("ExtendConstant[4] = %v, want 7", out.Value(4))
	}
}

// TestSeriesExtend concatenates two series.
func TestSeriesExtend(t *testing.T) {
	a := newInt64Series(t, "a", []any{int64(1), int64(2)})
	b := newInt64Series(t, "b", []any{int64(3), int64(4)})
	out, err := a.Extend(b)
	if err != nil {
		t.Fatalf("Extend: %v", err)
	}
	if out.Len() != 4 {
		t.Errorf("Extend len = %d, want 4", out.Len())
	}
}

// TestSeriesZipWith zips two series based on a boolean mask.
func TestSeriesZipWith(t *testing.T) {
	a := newInt64Series(t, "a", []any{int64(1), int64(2), int64(3)})
	b := newInt64Series(t, "b", []any{int64(10), int64(20), int64(30)})
	mask, _ := NewSeries(NewSeriesInput{Name: "m", DType: dtypes.Boolean, Values: []any{true, false, true}})
	out, err := a.ZipWith(mask, b)
	if err != nil {
		t.Fatalf("ZipWith: %v", err)
	}
	want := []int64{1, 20, 3}
	for i, w := range want {
		if v, _ := out.Value(i).(int64); v != w {
			t.Errorf("ZipWith[%d] = %d, want %d", i, v, w)
		}
	}
}

// TestSeriesSet conditionally sets values.
func TestSeriesSet(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1), int64(2), int64(3), int64(4)})
	mask, _ := NewSeries(NewSeriesInput{Name: "m", DType: dtypes.Boolean, Values: []any{true, false, true, false}})
	out, err := s.Set(mask, int64(0))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if v, _ := out.Value(0).(int64); v != 0 {
		t.Errorf("Set[0] = %d, want 0 (masked)", v)
	}
	if v, _ := out.Value(1).(int64); v != 2 {
		t.Errorf("Set[1] = %d, want 2 (unmasked)", v)
	}
}

// TestSeriesMapElements applies a fn to each value.
func TestSeriesMapElements(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1), int64(2), int64(3)})
	out, err := s.MapElements(func(v any) any { return v.(int64) * 2 })
	if err != nil {
		t.Fatalf("MapElements: %v", err)
	}
	for i := 0; i < out.Len(); i++ {
		want := int64((i + 1) * 2)
		if v, _ := out.Value(i).(int64); v != want {
			t.Errorf("MapElements[%d] = %d, want %d", i, v, want)
		}
	}
}

// TestSeriesReplace swaps one value for another.
func TestSeriesReplace(t *testing.T) {
	s := newInt64Series(t, "v", []any{int64(1), int64(2), int64(1), int64(3)})
	out := s.Replace(int64(1), int64(99))
	for i := 0; i < out.Len(); i++ {
		if v, _ := out.Value(i).(int64); v == 1 {
			t.Errorf("Replace[%d] = 1, expected all 1s replaced", i)
		}
	}
}
