package unit

import (
	"math"
	"reflect"
	"testing"

	iarrow "github.com/eugeneshershen/gopolars/pkg/io/arrow"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestV10WaveESeriesMathAndStatsSurface(t *testing.T) {
	t.Parallel()

	s := newV10FloatSeries(t, "x", []any{float64(0.5), float64(1.5)})
	methods := []string{
		"Sin", "Cos", "Tan", "Sinh", "Cosh", "Tanh",
		"Arcsin", "Arccos", "Arctan", "Arcsinh", "Arccosh", "Arctanh",
		"Cbrt", "Ceil", "Floor", "Degrees", "Sign", "Log10", "Log1p", "Round", "Pow",
		"Max", "Min", "Mean", "Median", "Var", "NUnique", "Mode", "Kurtosis", "Skew", "Quantile", "Product", "NanMax", "NanMin",
	}

	for _, name := range methods {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !reflect.ValueOf(s).MethodByName(name).IsValid() {
				t.Fatalf("expected Series method %s to be exposed", name)
			}
		})
	}
}

func TestV10WaveESeriesStructuralSurface(t *testing.T) {
	t.Parallel()

	s := newV10FloatSeries(t, "x", []any{float64(3), float64(1), float64(2)})
	methods := []string{
		"Alias", "Clone", "Clear", "Head", "Tail", "Limit", "Slice", "Sort", "Unique",
		"ArgSort", "ArgMax", "ArgMin", "ArgUnique", "ArgTrue", "Gather", "GatherEvery", "Sample", "Shuffle",
		"Rechunk", "ShrinkToFit", "Shape", "IsEmpty", "IsSorted", "HasNulls", "HasValidity", "NChunks", "ChunkLengths",
	}

	for _, name := range methods {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !reflect.ValueOf(s).MethodByName(name).IsValid() {
				t.Fatalf("expected Series method %s to be exposed", name)
			}
		})
	}
}

func TestV10WaveESeriesBooleanSurface(t *testing.T) {
	t.Parallel()

	s := newV10FloatSeries(t, "x", []any{float64(1), math.NaN(), float64(3)})
	methods := []string{
		"All", "Any", "Not_", "IsNan", "IsNotNan", "IsFinite", "IsInfinite",
		"IsDuplicated", "IsUnique", "IsFirstDistinct", "IsLastDistinct",
		"IsBetween", "IsClose", "IsIn", "EqMissing", "NeMissing",
	}

	for _, name := range methods {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !reflect.ValueOf(s).MethodByName(name).IsValid() {
				t.Fatalf("expected Series method %s to be exposed", name)
			}
		})
	}
}

func TestV10WaveESeriesRollingSurface(t *testing.T) {
	t.Parallel()

	s := newV10FloatSeries(t, "x", []any{float64(1), float64(2), float64(3)})
	methods := []string{
		"CumSum", "CumMax", "CumMin", "CumProd", "CumCount",
		"RollingStd", "RollingVar", "RollingMedian", "RollingQuantile",
		"EwmMean", "EwmStd", "EwmVar", "Diff", "PctChange",
	}

	for _, name := range methods {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !reflect.ValueOf(s).MethodByName(name).IsValid() {
				t.Fatalf("expected Series method %s to be exposed", name)
			}
		})
	}
}

func TestV10WaveESeriesAdvancedSurface(t *testing.T) {
	t.Parallel()

	s := newV10FloatSeries(t, "x", []any{float64(1), float64(2), float64(3)})
	methods := []string{
		"Dot", "Entropy", "ValueCounts", "UniqueCounts", "TopK", "TopKBy", "BottomK", "BottomKBy",
		"PeakMax", "PeakMin", "InterpolateBy", "ForwardFill", "BackwardFill", "Cut", "QCut",
		"Rle", "RleId", "Rank", "SearchSorted", "LowerBound", "UpperBound", "Item", "First", "Last",
		"IndexOf", "MapElements", "Replace", "ReplaceStrict", "Reshape", "RepeatBy", "SetSorted",
	}

	for _, name := range methods {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !reflect.ValueOf(s).MethodByName(name).IsValid() {
				t.Fatalf("expected Series method %s to be exposed", name)
			}
		})
	}
}

func TestV10WaveESeriesIOSurface(t *testing.T) {
	t.Parallel()

	s := newV10FloatSeries(t, "x", []any{float64(1), float64(2), float64(3)})
	methods := []string{
		"ToFrame", "ToDummies", "ToArrow", "ToInitRepr", "ToJax", "ToTorch", "ToPhysical",
		"Rename", "Serialize", "Deserialize", "Hash", "Implode", "Explode", "Flatten",
		"Extend", "ExtendConstant", "NewFromIndex", "Scatter", "Set", "ZipWith",
	}

	for _, name := range methods {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !reflect.ValueOf(s).MethodByName(name).IsValid() {
				t.Fatalf("expected Series method %s to be exposed", name)
			}
		})
	}
}

func TestV10WaveESeriesTrigAndSign(t *testing.T) {
	t.Parallel()

	trig := newV10FloatSeries(t, "angles", []any{0.0, math.Pi / 2})

	type seriesCase struct {
		name string
		want []float64
	}

	cases := []seriesCase{
		{name: "Sin", want: []float64{0, 1}},
		{name: "Cos", want: []float64{1, 0}},
		{name: "Tan", want: []float64{0, math.Tan(math.Pi / 2)}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mustCallSeries(t, trig, tc.name)
			for i, want := range tc.want {
				assertV10FloatApprox(t, got.Value(i), want)
			}
		})
	}

	sign := newV10FloatSeries(t, "sign", []any{float64(-1), float64(0), float64(2)})
	got := mustCallSeries(t, sign, "Sign")
	want := []float64{-1, 0, 1}
	for i, expected := range want {
		assertV10FloatApprox(t, got.Value(i), expected)
	}
}

func TestV10WaveESeriesMathTransforms(t *testing.T) {
	t.Parallel()

	input := newV10FloatSeries(t, "x", []any{float64(8), float64(2.6), float64(45), float64(99), float64(1)})

	type unaryCase struct {
		name  string
		index int
		want  float64
	}

	cases := []unaryCase{
		{name: "Cbrt", index: 0, want: 2},
		{name: "Ceil", index: 1, want: 3},
		{name: "Floor", index: 1, want: 2},
		{name: "Degrees", index: 4, want: 180 / math.Pi},
		{name: "Log10", index: 3, want: math.Log10(99)},
		{name: "Log1p", index: 1, want: math.Log1p(2.6)},
		{name: "Round", index: 1, want: 3},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mustCallSeries(t, input, tc.name)
			assertV10FloatApprox(t, got.Value(tc.index), tc.want)
		})
	}

	pow := mustCallSeries(t, input, "Pow", float64(2))
	assertV10FloatApprox(t, pow.Value(0), 64)
	assertV10FloatApprox(t, pow.Value(1), 6.76)
}

func TestV10WaveESeriesStatAggregates(t *testing.T) {
	t.Parallel()

	stats := newV10FloatSeries(t, "stats", []any{float64(1), float64(2), float64(2), float64(3)})

	scalars := []struct {
		name string
		want float64
	}{
		{name: "Max", want: 3},
		{name: "Min", want: 1},
		{name: "Mean", want: 2},
		{name: "Median", want: 2},
		{name: "Var", want: 2.0 / 3.0},
		{name: "Quantile", want: 2},
		{name: "Product", want: 12},
	}

	for _, tc := range scalars {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out any
			if tc.name == "Quantile" {
				out = mustCallScalar(t, stats, tc.name, float64(0.5))
			} else {
				out = mustCallScalar(t, stats, tc.name)
			}
			assertV10FloatApprox(t, out, tc.want)
		})
	}

	if got := mustCallScalar(t, stats, "NUnique"); got.(int) != 3 {
		t.Fatalf("unexpected NUnique: got %v want 3", got)
	}
	assertV10FloatApprox(t, mustCallScalar(t, stats, "Mode"), 2)

	shape := newV10FloatSeries(t, "shape", []any{float64(-2), float64(0), float64(2)})
	if got := mustCallScalar(t, shape, "Skew"); math.Abs(got.(float64)) > 1e-9 {
		t.Fatalf("unexpected Skew: got %v want 0", got)
	}
	if got := mustCallScalar(t, shape, "Kurtosis"); math.IsNaN(got.(float64)) {
		t.Fatalf("unexpected Kurtosis: got NaN")
	}
}

func TestV10WaveESeriesNaNAggregates(t *testing.T) {
	t.Parallel()

	stats := newV10FloatSeries(t, "nan", []any{math.NaN(), float64(2), float64(5)})
	assertV10FloatApprox(t, mustCallScalar(t, stats, "NanMax"), 5)
	assertV10FloatApprox(t, mustCallScalar(t, stats, "NanMin"), 2)
}

func TestV10WaveESeriesStructuralMethods(t *testing.T) {
	t.Parallel()

	base := newV10FloatSeries(t, "x", []any{float64(3), float64(1), float64(2), float64(1)})

	alias := mustCallSeries(t, base, "Alias", "renamed")
	if alias.Name() != "renamed" {
		t.Fatalf("unexpected alias name: got %q want %q", alias.Name(), "renamed")
	}

	assertV10SeriesValues(t, mustCallSeries(t, base, "Clone"), []any{float64(3), float64(1), float64(2), float64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Clear"), []any{})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Head", 2), []any{float64(3), float64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Tail", 2), []any{float64(2), float64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Limit", 3), []any{float64(3), float64(1), float64(2)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Slice", 1, 2), []any{float64(1), float64(2)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Sort", false), []any{float64(1), float64(1), float64(2), float64(3)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Unique"), []any{float64(3), float64(1), float64(2)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "ArgSort"), []any{int64(1), int64(3), int64(2), int64(0)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "ArgUnique"), []any{int64(0), int64(1), int64(2)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Gather", []int{2, 0}), []any{float64(2), float64(3)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "GatherEvery", 2), []any{float64(3), float64(2)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Sample", 2, int64(7)), []any{float64(2), float64(3)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Shuffle", int64(7)), []any{float64(2), float64(3), float64(1), float64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "Rechunk"), []any{float64(3), float64(1), float64(2), float64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "ShrinkToFit"), []any{float64(3), float64(1), float64(2), float64(1)})

	if got := mustCallScalar(t, base, "ArgMax"); got.(int) != 0 {
		t.Fatalf("unexpected ArgMax: got %v want 0", got)
	}
	if got := mustCallScalar(t, base, "ArgMin"); got.(int) != 1 {
		t.Fatalf("unexpected ArgMin: got %v want 1", got)
	}
	if got := mustCallScalar(t, base, "Shape"); got != [1]int{4} {
		t.Fatalf("unexpected Shape: got %v want %v", got, [1]int{4})
	}
	if got := mustCallScalar(t, base, "IsEmpty"); got.(bool) {
		t.Fatalf("unexpected IsEmpty: got true want false")
	}
	if got := mustCallScalar(t, base, "IsSorted"); got.(bool) {
		t.Fatalf("unexpected IsSorted: got true want false")
	}
	if got := mustCallScalar(t, base, "HasNulls"); got.(bool) {
		t.Fatalf("unexpected HasNulls: got true want false")
	}
	if got := mustCallScalar(t, base, "HasValidity"); got.(bool) {
		t.Fatalf("unexpected HasValidity: got true want false")
	}
	if got := mustCallScalar(t, base, "NChunks"); got.(int) != 1 {
		t.Fatalf("unexpected NChunks: got %v want 1", got)
	}
	if got := mustCallScalar(t, base, "ChunkLengths"); !reflect.DeepEqual(got, []int{4}) {
		t.Fatalf("unexpected ChunkLengths: got %v want %v", got, []int{4})
	}

	withNulls, err := polars.NewSeries(polars.NewSeriesInput{Name: "nullable", DType: polars.Float64, Values: []any{float64(1), nil}})
	if err != nil {
		t.Fatalf("new nullable series failed: %v", err)
	}
	if got := mustCallScalar(t, withNulls, "HasNulls"); !got.(bool) {
		t.Fatalf("unexpected HasNulls: got false want true")
	}
	if got := mustCallScalar(t, withNulls, "HasValidity"); !got.(bool) {
		t.Fatalf("unexpected HasValidity: got false want true")
	}

	bools, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Boolean, Values: []any{false, true, true}})
	if err != nil {
		t.Fatalf("new bool series failed: %v", err)
	}
	assertV10SeriesValues(t, mustCallSeries(t, bools, "ArgTrue"), []any{int64(1), int64(2)})
}

func TestV10WaveESeriesBooleanMethods(t *testing.T) {
	t.Parallel()

	bools, err := polars.NewSeries(polars.NewSeriesInput{Name: "b", DType: polars.Boolean, Values: []any{true, false, true}})
	if err != nil {
		t.Fatalf("new bool series failed: %v", err)
	}
	if got := mustCallScalar(t, bools, "All"); got.(bool) {
		t.Fatalf("unexpected All: got true want false")
	}
	if got := mustCallScalar(t, bools, "Any"); !got.(bool) {
		t.Fatalf("unexpected Any: got false want true")
	}
	assertV10SeriesValues(t, mustCallSeries(t, bools, "Not_"), []any{false, true, false})

	floats := newV10FloatSeries(t, "f", []any{math.NaN(), math.Inf(1), float64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, floats, "IsNan"), []any{true, false, false})
	assertV10SeriesValues(t, mustCallSeries(t, floats, "IsNotNan"), []any{false, true, true})
	assertV10SeriesValues(t, mustCallSeries(t, floats, "IsFinite"), []any{false, false, true})
	assertV10SeriesValues(t, mustCallSeries(t, floats, "IsInfinite"), []any{false, true, false})

	dupes := newV10FloatSeries(t, "d", []any{float64(1), float64(2), float64(2), float64(3), float64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, dupes, "IsDuplicated"), []any{true, true, true, false, true})
	assertV10SeriesValues(t, mustCallSeries(t, dupes, "IsUnique"), []any{false, false, false, true, false})
	assertV10SeriesValues(t, mustCallSeries(t, dupes, "IsFirstDistinct"), []any{true, true, false, true, false})
	assertV10SeriesValues(t, mustCallSeries(t, dupes, "IsLastDistinct"), []any{false, false, true, true, true})

	between := newV10FloatSeries(t, "between", []any{float64(0), float64(1), float64(2), float64(3)})
	assertV10SeriesValues(t, mustCallSeries(t, between, "IsBetween", float64(1), float64(2)), []any{false, true, true, false})

	other := newV10FloatSeries(t, "other", []any{float64(1 + 1e-10), float64(2.1), float64(2), float64(3)})
	assertV10SeriesValues(t, mustCallSeriesResult(t, between, "IsClose", other), []any{false, false, true, true})
	assertV10SeriesValues(t, mustCallSeries(t, between, "IsIn", []any{float64(1), float64(3)}), []any{false, true, false, true})

	left, err := polars.NewSeries(polars.NewSeriesInput{Name: "left", DType: polars.Float64, Values: []any{float64(1), nil, float64(3)}})
	if err != nil {
		t.Fatalf("new left series failed: %v", err)
	}
	right, err := polars.NewSeries(polars.NewSeriesInput{Name: "right", DType: polars.Float64, Values: []any{float64(1), nil, float64(4)}})
	if err != nil {
		t.Fatalf("new right series failed: %v", err)
	}
	assertV10SeriesValues(t, mustCallSeriesResult(t, left, "EqMissing", right), []any{true, true, false})
	assertV10SeriesValues(t, mustCallSeriesResult(t, left, "NeMissing", right), []any{false, false, true})
}

func TestV10WaveESeriesRollingMethods(t *testing.T) {
	t.Parallel()

	base := newV10FloatSeries(t, "x", []any{float64(1), float64(2), float64(3)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "CumSum"), []any{float64(1), float64(3), float64(6)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "CumMax"), []any{float64(1), float64(2), float64(3)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "CumMin"), []any{float64(1), float64(1), float64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "CumProd"), []any{float64(1), float64(2), float64(6)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "CumCount"), []any{float64(1), float64(2), float64(3)})

	assertV10SeriesValues(t, mustCallSeries(t, base, "RollingStd", 2), []any{float64(0), math.Sqrt(0.5), math.Sqrt(0.5)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "RollingVar", 2), []any{float64(0), float64(0.5), float64(0.5)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "RollingMedian", 2), []any{float64(1), float64(1.5), float64(2.5)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "RollingQuantile", 2, float64(0.5)), []any{float64(1), float64(1.5), float64(2.5)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "EwmMean", float64(0.5)), []any{float64(1), float64(1.5), float64(2.25)})

	ewmStd := mustCallSeries(t, base, "EwmStd", float64(0.5))
	assertV10FloatApprox(t, ewmStd.Value(0), 0)
	assertV10FloatApprox(t, ewmStd.Value(1), math.Sqrt(0.125))
	assertV10FloatApprox(t, ewmStd.Value(2), math.Sqrt(0.34375))

	ewmVar := mustCallSeries(t, base, "EwmVar", float64(0.5))
	assertV10FloatApprox(t, ewmVar.Value(0), 0)
	assertV10FloatApprox(t, ewmVar.Value(1), 0.125)
	assertV10FloatApprox(t, ewmVar.Value(2), 0.34375)

	diff := newV10FloatSeries(t, "diff", []any{float64(1), float64(3), float64(6)})
	assertV10SeriesValues(t, mustCallSeries(t, diff, "Diff", 1), []any{nil, float64(2), float64(3)})

	pct := newV10FloatSeries(t, "pct", []any{float64(1), float64(2), float64(4)})
	assertV10SeriesValues(t, mustCallSeries(t, pct, "PctChange", 1), []any{nil, float64(1), float64(1)})
}

func TestV10WaveESeriesAdvancedMethods(t *testing.T) {
	t.Parallel()

	left := newV10FloatSeries(t, "left", []any{float64(1), float64(2), float64(3)})
	right := newV10FloatSeries(t, "right", []any{float64(4), float64(5), float64(6)})
	if got := mustCallScalarResult(t, left, "Dot", right); math.Abs(got.(float64)-32) > 1e-9 {
		t.Fatalf("unexpected Dot: got %v want 32", got)
	}

	entropy := newV10FloatSeries(t, "entropy", []any{float64(1), float64(1), float64(2), float64(2)})
	assertV10FloatApprox(t, mustCallScalar(t, entropy, "Entropy"), math.Log(2))

	counts := newV10FloatSeries(t, "counts", []any{float64(1), float64(1), float64(2), float64(3), float64(2)})
	valueCounts := mustCallDataFrameResult(t, counts, "ValueCounts")
	assertV10FrameColumnValues(t, valueCounts, "value", []any{float64(1), float64(2), float64(3)})
	assertV10FrameColumnValues(t, valueCounts, "count", []any{int64(2), int64(2), int64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, counts, "UniqueCounts"), []any{int64(2), int64(2), int64(1)})

	order := newV10FloatSeries(t, "order", []any{float64(3), float64(1), float64(4), float64(2)})
	assertV10SeriesValues(t, mustCallSeries(t, order, "TopK", 2), []any{float64(4), float64(3)})
	assertV10SeriesValues(t, mustCallSeries(t, order, "BottomK", 2), []any{float64(1), float64(2)})

	base, err := polars.NewSeries(polars.NewSeriesInput{Name: "base", DType: polars.String, Values: []any{"a", "b", "c"}})
	if err != nil {
		t.Fatalf("new string series failed: %v", err)
	}
	by := newV10FloatSeries(t, "by", []any{float64(2), float64(3), float64(1)})
	assertV10SeriesValues(t, mustCallSeriesResult(t, base, "TopKBy", by, 2), []any{"b", "a"})
	assertV10SeriesValues(t, mustCallSeriesResult(t, base, "BottomKBy", by, 2), []any{"c", "a"})

	peaks := newV10FloatSeries(t, "peaks", []any{float64(1), float64(3), float64(1), float64(2), float64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, peaks, "PeakMax"), []any{false, true, false, true, false})
	valleys := newV10FloatSeries(t, "valleys", []any{float64(3), float64(1), float64(3), float64(0), float64(2)})
	assertV10SeriesValues(t, mustCallSeries(t, valleys, "PeakMin"), []any{false, true, false, true, false})

	withNulls, err := polars.NewSeries(polars.NewSeriesInput{Name: "nulls", DType: polars.Float64, Values: []any{float64(1), nil, nil, float64(4)}})
	if err != nil {
		t.Fatalf("new nullable series failed: %v", err)
	}
	assertV10SeriesValues(t, mustCallSeries(t, withNulls, "ForwardFill"), []any{float64(1), float64(1), float64(1), float64(4)})
	assertV10SeriesValues(t, mustCallSeries(t, withNulls, "BackwardFill"), []any{float64(1), float64(4), float64(4), float64(4)})

	bySeries := newV10FloatSeries(t, "idx", []any{float64(0), float64(1), float64(2), float64(3)})
	assertV10SeriesValues(t, mustCallSeriesResult(t, withNulls, "InterpolateBy", bySeries), []any{float64(1), float64(2), float64(3), float64(4)})

	cut := mustCallSeriesResult(t, counts, "Cut", []float64{1.5, 2.5})
	assertV10SeriesValues(t, cut, []any{"<= 1.5", "<= 1.5", "<= 2.5", "> 2.5", "<= 2.5"})

	qcut := mustCallSeriesResult(t, counts, "QCut", 2)
	if qcut.Len() != counts.Len() {
		t.Fatalf("unexpected QCut len: got %d want %d", qcut.Len(), counts.Len())
	}
	if _, ok := qcut.Value(0).(string); !ok {
		t.Fatalf("expected QCut values to be strings, got %T", qcut.Value(0))
	}

	rleInput := newV10FloatSeries(t, "runs", []any{float64(1), float64(1), float64(2), float64(2), float64(1)})
	rle := mustCallDataFrameResult(t, rleInput, "Rle")
	assertV10FrameColumnValues(t, rle, "value", []any{float64(1), float64(2), float64(1)})
	assertV10FrameColumnValues(t, rle, "count", []any{int64(2), int64(2), int64(1)})
	assertV10SeriesValues(t, mustCallSeries(t, rleInput, "RleId"), []any{int64(0), int64(0), int64(1), int64(1), int64(2)})

	rankInput := newV10FloatSeries(t, "rank", []any{float64(1), float64(3), float64(2)})
	assertV10SeriesValues(t, mustCallSeries(t, rankInput, "Rank"), []any{int64(1), int64(3), int64(2)})

	sorted := newV10FloatSeries(t, "sorted", []any{float64(1), float64(2), float64(2), float64(4)})
	if got := mustCallScalar(t, sorted, "SearchSorted", float64(2)); got.(int) != 1 {
		t.Fatalf("unexpected SearchSorted: got %v want 1", got)
	}
	if got := mustCallScalar(t, sorted, "LowerBound", float64(2)); got.(int) != 1 {
		t.Fatalf("unexpected LowerBound: got %v want 1", got)
	}
	if got := mustCallScalar(t, sorted, "UpperBound", float64(2)); got.(int) != 3 {
		t.Fatalf("unexpected UpperBound: got %v want 3", got)
	}

	if got := mustCallScalarResult(t, counts, "Item", 2); got.(float64) != 2 {
		t.Fatalf("unexpected Item: got %v want 2", got)
	}
	if got := mustCallScalar(t, counts, "First"); got.(float64) != 1 {
		t.Fatalf("unexpected First: got %v want 1", got)
	}
	if got := mustCallScalar(t, counts, "Last"); got.(float64) != 2 {
		t.Fatalf("unexpected Last: got %v want 2", got)
	}
	if got := mustCallScalar(t, counts, "IndexOf", float64(3)); got.(int) != 3 {
		t.Fatalf("unexpected IndexOf: got %v want 3", got)
	}

	mapped := mustCallSeriesResult(t, counts, "MapElements", func(v any) any {
		if f, ok := v.(float64); ok {
			return f * 10
		}
		return v
	})
	assertV10SeriesValues(t, mapped, []any{float64(10), float64(10), float64(20), float64(30), float64(20)})

	assertV10SeriesValues(t, mustCallSeries(t, counts, "Replace", float64(2), float64(20)), []any{float64(1), float64(1), float64(20), float64(3), float64(20)})
	assertV10SeriesValues(t, mustCallSeriesResult(t, counts, "ReplaceStrict", float64(3), float64(30)), []any{float64(1), float64(1), float64(2), float64(30), float64(2)})

	reshaped := mustCallSeriesResult(t, counts, "Reshape", 5, 1)
	assertV10SeriesValues(t, reshaped, []any{float64(1), float64(1), float64(2), float64(3), float64(2)})

	assertV10SeriesValues(t, mustCallSeries(t, counts, "RepeatBy", 2), []any{float64(1), float64(1), float64(1), float64(1), float64(2), float64(2), float64(3), float64(3), float64(2), float64(2)})
	assertV10SeriesValues(t, mustCallSeries(t, counts, "SetSorted", true), []any{float64(1), float64(1), float64(2), float64(3), float64(2)})
}

func TestV10WaveESeriesIOMethods(t *testing.T) {
	t.Parallel()

	base := newV10FloatSeries(t, "x", []any{float64(1), float64(2), float64(3)})

	frame := mustCallDataFrameResult(t, base, "ToFrame")
	assertV10FrameColumnValues(t, frame, "x", []any{float64(1), float64(2), float64(3)})

	cats, err := polars.NewSeries(polars.NewSeriesInput{Name: "cat", DType: polars.String, Values: []any{"a", "b", "a"}})
	if err != nil {
		t.Fatalf("new string series failed: %v", err)
	}
	dummies := mustCallDataFrameResult(t, cats, "ToDummies")
	assertV10FrameColumnValues(t, dummies, "cat_a", []any{true, false, true})
	assertV10FrameColumnValues(t, dummies, "cat_b", []any{false, true, false})

	table := mustCallArrowResult(t, base, "ToArrow")
	if !reflect.DeepEqual(table.Columns["x"], []any{float64(1), float64(2), float64(3)}) {
		t.Fatalf("unexpected ToArrow payload: got %v", table.Columns["x"])
	}
	if repr := mustCallScalar(t, base, "ToInitRepr").(string); repr == "" {
		t.Fatalf("expected non-empty ToInitRepr")
	}
	if got := mustCallScalar(t, base, "ToJax"); !reflect.DeepEqual(got, []float64{1, 2, 3}) {
		t.Fatalf("unexpected ToJax: got %v", got)
	}
	if got := mustCallScalar(t, base, "ToTorch"); !reflect.DeepEqual(got, []float64{1, 2, 3}) {
		t.Fatalf("unexpected ToTorch: got %v", got)
	}
	assertV10SeriesValues(t, mustCallSeries(t, base, "ToPhysical"), []any{float64(1), float64(2), float64(3)})

	renamed := mustCallSeries(t, base, "Rename", "y")
	if renamed.Name() != "y" {
		t.Fatalf("unexpected Rename result: got %q want %q", renamed.Name(), "y")
	}

	payload := mustCallScalarResult(t, base, "Serialize").([]byte)
	decoded := mustCallSeriesResult(t, base, "Deserialize", payload)
	assertV10SeriesValues(t, decoded, []any{float64(1), float64(2), float64(3)})

	hashed := mustCallSeries(t, base, "Hash", uint64(7))
	if hashed.Len() != base.Len() {
		t.Fatalf("unexpected Hash len: got %d want %d", hashed.Len(), base.Len())
	}
	if _, ok := hashed.Value(0).(int64); !ok {
		t.Fatalf("expected Hash values to be int64, got %T", hashed.Value(0))
	}

	imploded := mustCallSeries(t, base, "Implode")
	if imploded.Len() != 1 {
		t.Fatalf("unexpected Implode len: got %d want 1", imploded.Len())
	}
	if got, ok := imploded.Value(0).([]any); !ok || !reflect.DeepEqual(got, []any{float64(1), float64(2), float64(3)}) {
		t.Fatalf("unexpected Implode payload: got %v", imploded.Value(0))
	}

	listSeries, err := polars.NewSeries(polars.NewSeriesInput{Name: "list", DType: polars.List, Values: []any{[]any{int64(1), int64(2)}, []any{int64(3)}}})
	if err != nil {
		t.Fatalf("new list series failed: %v", err)
	}
	assertV10SeriesValues(t, mustCallSeries(t, listSeries, "Explode"), []any{int64(1), int64(2), int64(3)})
	assertV10SeriesValues(t, mustCallSeries(t, listSeries, "Flatten"), []any{int64(1), int64(2), int64(3)})

	other := newV10FloatSeries(t, "x", []any{float64(4), float64(5)})
	assertV10SeriesValues(t, mustCallSeriesResult(t, base, "Extend", other), []any{float64(1), float64(2), float64(3), float64(4), float64(5)})
	assertV10SeriesValues(t, mustCallSeries(t, base, "ExtendConstant", float64(9), 2), []any{float64(1), float64(2), float64(3), float64(9), float64(9)})
	assertV10SeriesValues(t, mustCallSeriesResult(t, base, "NewFromIndex", 1, 3), []any{float64(2), float64(2), float64(2)})
	assertV10SeriesValues(t, mustCallSeriesResult(t, base, "Scatter", []int{0, 2}, []any{float64(10), float64(30)}), []any{float64(10), float64(2), float64(30)})

	mask, err := polars.NewSeries(polars.NewSeriesInput{Name: "mask", DType: polars.Boolean, Values: []any{true, false, true}})
	if err != nil {
		t.Fatalf("new bool series failed: %v", err)
	}
	assertV10SeriesValues(t, mustCallSeriesResult(t, base, "Set", mask, float64(99)), []any{float64(99), float64(2), float64(99)})

	zipOther := newV10FloatSeries(t, "other", []any{float64(7), float64(8), float64(9)})
	assertV10SeriesValues(t, mustCallSeriesResult(t, base, "ZipWith", mask, zipOther), []any{float64(1), float64(8), float64(3)})
}

func newV10FloatSeries(t *testing.T, name string, values []any) polars.Series {
	t.Helper()

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.Float64, Values: values})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	return s
}

func mustCallSeries(t *testing.T, s polars.Series, method string, args ...any) polars.Series {
	t.Helper()

	results := callV10SeriesMethod(t, s, method, args...)
	if len(results) != 1 {
		t.Fatalf("expected %s to return one value, got %d", method, len(results))
	}
	out, ok := results[0].Interface().(polars.Series)
	if !ok {
		t.Fatalf("expected %s to return polars.Series, got %T", method, results[0].Interface())
	}
	return out
}

func mustCallSeriesResult(t *testing.T, s polars.Series, method string, args ...any) polars.Series {
	t.Helper()

	results := callV10SeriesMethod(t, s, method, args...)
	if len(results) != 2 {
		t.Fatalf("expected %s to return (polars.Series, error), got %d values", method, len(results))
	}
	if err, _ := results[1].Interface().(error); err != nil {
		t.Fatalf("%s returned error: %v", method, err)
	}
	out, ok := results[0].Interface().(polars.Series)
	if !ok {
		t.Fatalf("expected %s to return polars.Series, got %T", method, results[0].Interface())
	}
	return out
}

func mustCallDataFrameResult(t *testing.T, s polars.Series, method string, args ...any) polars.DataFrame {
	t.Helper()

	results := callV10SeriesMethod(t, s, method, args...)
	if len(results) != 2 {
		t.Fatalf("expected %s to return (polars.DataFrame, error), got %d values", method, len(results))
	}
	if err, _ := results[1].Interface().(error); err != nil {
		t.Fatalf("%s returned error: %v", method, err)
	}
	out, ok := results[0].Interface().(polars.DataFrame)
	if !ok {
		t.Fatalf("expected %s to return polars.DataFrame, got %T", method, results[0].Interface())
	}
	return out
}

func mustCallArrowResult(t *testing.T, s polars.Series, method string, args ...any) iarrow.Table {
	t.Helper()

	results := callV10SeriesMethod(t, s, method, args...)
	if len(results) != 2 {
		t.Fatalf("expected %s to return (arrow.Table, error), got %d values", method, len(results))
	}
	if err, _ := results[1].Interface().(error); err != nil {
		t.Fatalf("%s returned error: %v", method, err)
	}
	out, ok := results[0].Interface().(iarrow.Table)
	if !ok {
		t.Fatalf("expected %s to return arrow.Table, got %T", method, results[0].Interface())
	}
	return out
}

func mustCallScalarResult(t *testing.T, s polars.Series, method string, args ...any) any {
	t.Helper()

	results := callV10SeriesMethod(t, s, method, args...)
	if len(results) != 2 {
		t.Fatalf("expected %s to return (<scalar>, error), got %d values", method, len(results))
	}
	if err, _ := results[1].Interface().(error); err != nil {
		t.Fatalf("%s returned error: %v", method, err)
	}
	return results[0].Interface()
}

func mustCallScalar(t *testing.T, s polars.Series, method string, args ...any) any {
	t.Helper()

	results := callV10SeriesMethod(t, s, method, args...)
	if len(results) != 1 {
		t.Fatalf("expected %s to return one value, got %d", method, len(results))
	}
	return results[0].Interface()
}

func callV10SeriesMethod(t *testing.T, s polars.Series, method string, args ...any) []reflect.Value {
	t.Helper()

	target := reflect.ValueOf(s)
	fn := target.MethodByName(method)
	if !fn.IsValid() {
		t.Fatalf("expected Series method %s to be exposed", method)
	}

	callArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		callArgs[i] = reflect.ValueOf(arg)
	}
	return fn.Call(callArgs)
}

func assertV10FloatApprox(t *testing.T, got any, want float64) {
	t.Helper()

	value, ok := got.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", got)
	}
	if math.IsNaN(want) {
		if !math.IsNaN(value) {
			t.Fatalf("expected NaN, got %v", value)
		}
		return
	}
	if math.Abs(value-want) > 1e-9 {
		t.Fatalf("unexpected value: got %.12f want %.12f", value, want)
	}
}

func assertV10SeriesValues(t *testing.T, got polars.Series, want []any) {
	t.Helper()

	if got.Len() != len(want) {
		t.Fatalf("unexpected series length: got %d want %d", got.Len(), len(want))
	}
	for i, expected := range want {
		actual := got.Value(i)
		switch exp := expected.(type) {
		case float64:
			assertV10FloatApprox(t, actual, exp)
		default:
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("unexpected value at %d: got %v want %v", i, actual, expected)
			}
		}
	}
}

func assertV10FrameColumnValues(t *testing.T, df polars.DataFrame, column string, want []any) {
	t.Helper()

	got, err := df.GetColumn(column)
	if err != nil {
		t.Fatalf("get column %s failed: %v", column, err)
	}
	assertV10SeriesValues(t, got, want)
}
