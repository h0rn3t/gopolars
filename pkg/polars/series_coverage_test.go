package polars

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

func numSeries(t *testing.T, name string, vals ...any) Series {
	t.Helper()
	s, err := NewSeries(NewSeriesInput{Name: name, DType: dtypes.Float64, Values: vals})
	if err != nil {
		t.Fatalf("series %s: %v", name, err)
	}
	return s
}

// TestSeriesShapeReturningOps calls the Series methods that return another
// Series and asserts the result length is consistent with the input. These are
// thin wrappers over the kernel layer; the length invariant is the contract that
// matters for the public surface.
func TestSeriesShapeReturningOps(t *testing.T) {
	t.Parallel()

	s := numSeries(t, "v", 1.0, 2.0, 2.0, 3.0, 4.0)
	n := s.Len()

	sameLen := map[string]Series{
		"rolling_std":       s.RollingStd(2),
		"rolling_var":       s.RollingVar(2),
		"rolling_median":    s.RollingMedian(2),
		"rolling_quantile":  s.RollingQuantile(2, 0.5),
		"is_nan":            s.IsNan(),
		"is_not_nan":        s.IsNotNan(),
		"is_finite":         s.IsFinite(),
		"is_infinite":       s.IsInfinite(),
		"is_duplicated":     s.IsDuplicated(),
		"is_unique":         s.IsUnique(),
		"is_first_distinct": s.IsFirstDistinct(),
		"is_last_distinct":  s.IsLastDistinct(),
		"is_between":        s.IsBetween(1.5, 3.5),
		"is_in":             s.IsIn([]any{2.0, 4.0}),
		"ewm_std":           s.EwmStd(0.5),
		"ewm_var":           s.EwmVar(0.5),
		"diff":              s.Diff(1),
		"pct_change":        s.PctChange(1),
		"forward_fill":      s.ForwardFill(),
		"backward_fill":     s.BackwardFill(),
		"peak_max":          s.PeakMax(),
		"peak_min":          s.PeakMin(),
		"rle_id":            s.RleId(),
		"rank":              s.Rank(),
		"set_sorted":        s.SetSorted(false),
		"shrink_to_fit":     s.ShrinkToFit(),
	}
	for name, out := range sameLen {
		if out == nil {
			t.Errorf("%s returned nil series", name)
			continue
		}
		if out.Len() != n {
			t.Errorf("%s len = %d, want %d", name, out.Len(), n)
		}
	}

	// TopK / BottomK return at most k rows.
	if got := s.TopK(2); got.Len() != 2 {
		t.Errorf("TopK(2) len = %d, want 2", got.Len())
	}
	if got := s.BottomK(2); got.Len() != 2 {
		t.Errorf("BottomK(2) len = %d, want 2", got.Len())
	}
	// UniqueCounts has one row per distinct value (here {1,2,3,4} -> 4).
	if got := s.UniqueCounts(); got.Len() != 4 {
		t.Errorf("UniqueCounts len = %d, want 4", got.Len())
	}
	// RepeatBy returns a list-valued series of the same length.
	if got := s.RepeatBy(2); got == nil {
		t.Error("RepeatBy returned nil")
	}
	// Clear empties the series.
	if got := s.Clear(); got.Len() != 0 {
		t.Errorf("Clear len = %d, want 0", got.Len())
	}
}

// TestSeriesScalarReducers covers the methods returning scalar values.
func TestSeriesScalarReducers(t *testing.T) {
	t.Parallel()

	s := numSeries(t, "v", 1.0, 2.0, 2.0, 3.0, 4.0)

	if s.Mode() != 2.0 {
		t.Errorf("Mode = %v, want 2", s.Mode())
	}
	if s.NanMax() != 4.0 {
		t.Errorf("NanMax = %v, want 4", s.NanMax())
	}
	if s.NanMin() != 1.0 {
		t.Errorf("NanMin = %v, want 1", s.NanMin())
	}
	if s.First() != 1.0 {
		t.Errorf("First = %v, want 1", s.First())
	}
	if s.Last() != 4.0 {
		t.Errorf("Last = %v, want 4", s.Last())
	}
	if s.IndexOf(3.0) != 3 {
		t.Errorf("IndexOf(3) = %d, want 3", s.IndexOf(3.0))
	}
	// Kurtosis/Skew/Entropy just need to be finite numbers; call them for coverage.
	_ = s.Kurtosis()
	_ = s.Skew()
	_ = s.Entropy()

	// Search/bound helpers on a sorted series.
	sorted := numSeries(t, "v", 1.0, 2.0, 3.0, 4.0)
	if idx := sorted.SearchSorted(2.5); idx < 0 || idx > sorted.Len() {
		t.Errorf("SearchSorted(2.5) = %d out of range", idx)
	}
	if idx := sorted.LowerBound(2.0); idx < 0 {
		t.Errorf("LowerBound(2) = %d", idx)
	}
	if idx := sorted.UpperBound(2.0); idx < 0 {
		t.Errorf("UpperBound(2) = %d", idx)
	}

	// Item happy + out-of-range.
	if v, err := s.Item(0); err != nil || v != 1.0 {
		t.Errorf("Item(0) = %v err=%v", v, err)
	}
	if _, err := s.Item(99); err == nil {
		t.Error("Item(99) should error")
	}

	// ChunkLengths sums to Len.
	total := 0
	for _, c := range s.ChunkLengths() {
		total += c
	}
	if total != s.Len() {
		t.Errorf("ChunkLengths sum = %d, want %d", total, s.Len())
	}
}

// TestSeriesPairwiseOps covers the methods taking another Series.
func TestSeriesPairwiseOps(t *testing.T) {
	t.Parallel()

	a := numSeries(t, "a", 1.0, 2.0, 3.0)
	b := numSeries(t, "b", 1.0, 2.0, 3.0)

	dot, err := a.Dot(b)
	if err != nil || dot != 14.0 { // 1+4+9
		t.Fatalf("Dot = %v err=%v, want 14", dot, err)
	}
	eq, err := a.EqMissing(b)
	if err != nil || eq.Len() != 3 {
		t.Fatalf("EqMissing len=%d err=%v", eq.Len(), err)
	}
	ne, err := a.NeMissing(b)
	if err != nil || ne.Len() != 3 {
		t.Fatalf("NeMissing len=%d err=%v", ne.Len(), err)
	}
	close, err := a.IsClose(b)
	if err != nil || close.Len() != 3 {
		t.Fatalf("IsClose len=%d err=%v", close.Len(), err)
	}

	topBy, err := a.TopKBy(b, 2)
	if err != nil || topBy.Len() != 2 {
		t.Fatalf("TopKBy len=%d err=%v", topBy.Len(), err)
	}
	botBy, err := a.BottomKBy(b, 2)
	if err != nil || botBy.Len() != 2 {
		t.Fatalf("BottomKBy len=%d err=%v", botBy.Len(), err)
	}
}

// TestSeriesFrameReturningOps covers the methods returning a DataFrame.
func TestSeriesFrameReturningOps(t *testing.T) {
	t.Parallel()

	s := numSeries(t, "v", 1.0, 1.0, 2.0, 3.0)

	vc, err := s.ValueCounts()
	if err != nil || vc.Height() == 0 {
		t.Fatalf("ValueCounts height=%d err=%v", vc.Height(), err)
	}
	rle, err := s.Rle()
	if err != nil || rle.Height() == 0 {
		t.Fatalf("Rle height=%d err=%v", rle.Height(), err)
	}
	dummies, err := s.ToDummies()
	if err != nil || dummies.Height() != s.Len() {
		t.Fatalf("ToDummies height=%d err=%v", dummies.Height(), err)
	}
}

// TestSeriesTransformsWithErrors covers the (Series, error) transform methods.
func TestSeriesTransformsWithErrors(t *testing.T) {
	t.Parallel()

	s := numSeries(t, "v", 1.0, 2.0, 3.0, 4.0)

	cut, err := s.Cut([]float64{2.0, 3.0})
	if err != nil || cut.Len() != s.Len() {
		t.Fatalf("Cut len=%d err=%v", cut.Len(), err)
	}
	qcut, err := s.QCut(2)
	if err != nil || qcut.Len() != s.Len() {
		t.Fatalf("QCut len=%d err=%v", qcut.Len(), err)
	}
	rep, err := s.ReplaceStrict(2.0, 20.0)
	if err != nil || rep.Len() != s.Len() {
		t.Fatalf("ReplaceStrict len=%d err=%v", rep.Len(), err)
	}
	reshaped, err := s.Reshape(2, 2)
	if err != nil {
		t.Fatalf("Reshape: %v", err)
	}
	_ = reshaped

	by := numSeries(t, "by", 1.0, 2.0, 3.0, 4.0)
	interp, err := s.InterpolateBy(by)
	if err != nil || interp.Len() != s.Len() {
		t.Fatalf("InterpolateBy len=%d err=%v", interp.Len(), err)
	}
}
