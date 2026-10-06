package polars

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// seriesCharString renders a Series result as its dtype and values, or the
// error text, so float results compare bit-for-bit (-0 and NaN included).
func seriesCharString(s Series, err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	return fmt.Sprintf("%s %v", s.DataType(), s.ToList())
}

// frameCharString renders every column of a DataFrame result.
func frameCharString(d DataFrame, err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	out := fmt.Sprintf("height=%d", d.Height())
	for _, c := range d.GetColumns() {
		out += fmt.Sprintf(" %s:%s", c.Name(), seriesCharString(c, nil))
	}
	return out
}

func TestSeriesRefactorCharacterization(t *testing.T) {
	negZero := math.Copysign(0, -1)
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	f := mustSeries(t, "f", dtypes.Float64, []any{2.5, nil, math.NaN(), negZero, 0.0, -1.5})
	ints := mustSeries(t, "i", dtypes.Int64, []any{int64(5), nil, int64(-3), int64(5), int64(9)})
	strs := mustSeries(t, "s", dtypes.String, []any{"b", nil, "a", "b", "c"})
	times := mustSeries(t, "t", dtypes.Datetime, []any{t0.Add(time.Hour), nil, t0, t0.Add(time.Hour), nil})
	empty := mustSeries(t, "e", dtypes.Float64, []any{})
	allNull := mustSeries(t, "n", dtypes.Float64, []any{nil, nil})
	single := mustSeries(t, "one", dtypes.Float64, []any{4.0})
	flat := mustSeries(t, "k", dtypes.Float64, []any{2.0, 2.0, 2.0, 2.0})
	dense := mustSeries(t, "d", dtypes.Float64, []any{3.0, 1.0, negZero, 7.5, -2.0, 0.5})
	groupVals := mustSeries(t, "g", dtypes.Float64, []any{nil, 1.0, 2.0, nil, 0.5})
	groupKeys := mustSeries(t, "gk", dtypes.String, []any{"a", "b", "a", "b", "a"})
	sum := func(xs []float64) float64 {
		total := 0.0
		for _, x := range xs {
			total += x
		}
		return total
	}

	tests := []struct {
		name string
		got  func() string
		want string
	}{
		{"Abs string", func() string { return seriesCharString(strs.Abs(), nil) }, "float64 [<nil> <nil> <nil> <nil> <nil>]"},
		{"Cot zeros", func() string { return seriesCharString(f.Cot(), nil) }, "float64 [-1.3386481283041514 <nil> NaN -Inf +Inf -0.07091484430265245]"},
		{"Sign zeros", func() string { return seriesCharString(f.Sign(), nil) }, "float64 [1 <nil> 0 0 0 -1]"},
		{"Degrees int", func() string { return seriesCharString(ints.Degrees(), nil) }, "float64 [286.4788975654116 <nil> -171.88733853924697 286.4788975654116 515.662015617741]"},
		{"Pow nulls", func() string { return seriesCharString(f.Pow(2), nil) }, "float64 [6.25 <nil> NaN 0 0 2.25]"},
		{"Pow string", func() string { return seriesCharString(strs.Pow(2), nil) }, "float64 [<nil> <nil> <nil> <nil> <nil>]"},
		{"Clip nulls", func() string { return seriesCharString(f.Clip(-1, 1), nil) }, "float64 [1 <nil> NaN -0 0 -1]"},
		{"Clip negative zero bounds", func() string { return seriesCharString(f.Clip(negZero, negZero), nil) }, "float64 [-0 <nil> NaN -0 0 -0]"},
		{"Clip string", func() string { return seriesCharString(strs.Clip(0, 1), nil) }, "float64 [<nil> <nil> <nil> <nil> <nil>]"},
		{"Var single", func() string { return fmt.Sprint(single.Var()) }, "0"},
		{"Kurtosis single", func() string { return fmt.Sprint(single.Kurtosis()) }, "0"},
		{"Kurtosis flat", func() string { return fmt.Sprint(flat.Kurtosis()) }, "0"},
		{"Kurtosis", func() string { return fmt.Sprint(dense.Kurtosis()) }, "-0.20174639841271347"},
		{"Median", func() string { return fmt.Sprint(dense.Median(), dense.Head(5).Median()) }, "0.75 1"},
		{"Var", func() string { return fmt.Sprint(dense.Var(), ints.Var()) }, "10.766666666666666 25.333333333333332"},
		{"Median empty", func() string { return fmt.Sprint(empty.Median()) }, "0"},
		{"Quantile", func() string { return fmt.Sprint(f.Quantile(0.3), ints.Quantile(2), single.Quantile(0.5)) }, "-0.15000000000000013 9 4"},
		{"RollingVar int window 1", func() string { return seriesCharString(ints.RollingVar(1), nil) }, "float64 [0 <nil> 0 0 0]"},
		{"RollingStd int", func() string { return seriesCharString(ints.RollingStd(2), nil) }, "float64 [<nil> <nil> <nil> 5.656854249492381 2.8284271247461903]"},
		{"RollingMedian odd", func() string { return seriesCharString(dense.RollingMedian(3), nil) }, "float64 [<nil> <nil> 1 1 -0 0.5]"},
		{"RollingMedian even", func() string { return seriesCharString(dense.RollingMedian(2), nil) }, "float64 [<nil> 2 0.5 3.75 2.75 -0.75]"},
		{"RollingMedian int", func() string { return seriesCharString(ints.RollingMedian(2), nil) }, "float64 [<nil> <nil> <nil> 1 7]"},
		{"RollingKurtosis", func() string { return seriesCharString(dense.RollingKurtosis(4), nil) }, "float64 [<nil> <nil> <nil> -1.045272218498303 -0.8653000284315082 -0.8099726647186349]"},
		{"RollingKurtosis flat", func() string { return seriesCharString(flat.RollingKurtosis(4), nil) }, "float64 [<nil> <nil> <nil> 0]"},
		{"RollingQuantile int", func() string { return seriesCharString(ints.RollingQuantile(2, 0.25), nil) }, "float64 [<nil> <nil> <nil> -1 6]"},
		{"RollingMap int", func() string { return seriesCharString(ints.RollingMap(2, sum)) }, "float64 [5 5 -3 2 14]"},
		{"RollingMap string", func() string { return seriesCharString(strs.RollingMap(2, sum)) }, "float64 [<nil> <nil> <nil> <nil> <nil>]"},
		{"Hist int", func() string { return frameCharString(ints.Hist(2)) }, "height=2 bin:float64 [-3 3] count:int64 [1 3]"},
		{"Describe int", func() string { return fmt.Sprint(ints.Describe()) }, "map[dtype:int64 len:5 max:9 mean:4 min:-3 null_count:1]"},
		{"Describe float", func() string { return fmt.Sprint(f.Describe()) }, "map[dtype:float64 len:6 max:2.5 mean:0.25 min:-1.5 null_count:1]"},
		{"Sort nulls ascending", func() string { return seriesCharString(times.Sort(false), nil) }, "datetime [<nil> <nil> 2024-01-01 00:00:00 +0000 UTC 2024-01-01 01:00:00 +0000 UTC 2024-01-01 01:00:00 +0000 UTC]"},
		{"Sort nulls descending", func() string { return seriesCharString(times.Sort(true), nil) }, "datetime [<nil> <nil> 2024-01-01 01:00:00 +0000 UTC 2024-01-01 01:00:00 +0000 UTC 2024-01-01 00:00:00 +0000 UTC]"},
		{"Sort strings descending", func() string { return seriesCharString(strs.Sort(true), nil) }, "string [<nil> c b b a]"},
		{"ArgSort strings", func() string { return seriesCharString(strs.ArgSort(), nil) }, "int64 [1 2 0 3 4]"},
		{"Rank times", func() string { return seriesCharString(times.Rank(), nil) }, "int64 [4 1 3 5 2]"},
		{"IsSorted strings", func() string { return fmt.Sprint(strs.IsSorted(), strs.Sort(false).IsSorted()) }, "false true"},
		{"Bounds strings", func() string {
			sorted := strs.Sort(false)
			return fmt.Sprint(sorted.LowerBound("b"), sorted.UpperBound("b"), sorted.LowerBound(int64(1)))
		}, "2 4 1"},
		{"TopKBy strings", func() string { return seriesCharString(ints.TopKBy(strs, 3)) }, "int64 [9 5 5]"},
		{"BottomKBy times", func() string { return seriesCharString(ints.BottomKBy(times, 10)) }, "int64 [<nil> 9 -3 5 5]"},
		{"ArgMax ArgMin", func() string {
			return fmt.Sprint(empty.ArgMax(), empty.ArgMin(), allNull.ArgMax(), allNull.ArgMin(),
				ints.ArgMax(), ints.ArgMin(), strs.ArgMax(), strs.ArgMin(), f.ArgMax(), f.ArgMin())
		}, "-1 -1 -1 -1 4 2 4 2 0 5"},
		{"MaxBy nulls", func() string { return seriesCharString(f.Head(5).MaxBy(strs)) }, "float64 [2.5 <nil> NaN 0]"},
		{"MinBy nulls", func() string { return seriesCharString(f.Head(5).MinBy(strs)) }, "float64 [-0 <nil> NaN 0]"},
		{"MaxBy null rows in group", func() string { return seriesCharString(groupVals.MaxBy(groupKeys)) }, "float64 [2 1]"},
		{"MinBy null rows in group", func() string { return seriesCharString(groupVals.MinBy(groupKeys)) }, "float64 [0.5 1]"},
		{"MaxBy length mismatch", func() string { return seriesCharString(f.MaxBy(strs)) }, "error: max_by/min_by: length mismatch"},
		{"Truncate nulls", func() string { return seriesCharString(strs.Truncate(0)) }, "string [ <nil>   ]"},
		{"BitwiseCountOnes nulls", func() string { return seriesCharString(ints.BitwiseCountOnes()) }, "int64 [2 <nil> 63 2 2]"},
		{"Add int nulls", func() string { return seriesCharString(ints.Add(ints.Reverse())) }, "int64 [14 <nil> -6 <nil> 14]"},
		{"Div int nulls", func() string { return seriesCharString(ints.Div(ints.Reverse())) }, "float64 [0.5555555555555556 <nil> 1 <nil> 1.8]"},
		{"Add string", func() string { return seriesCharString(ints.Add(strs)) }, "error: numeric operations require numeric values"},
		{"Eq nulls", func() string { return seriesCharString(strs.Eq(strs.Reverse())) }, "bool [false <nil> true <nil> false]"},
		{"EqMissing", func() string { return seriesCharString(strs.EqMissing(strs.Reverse())) }, "bool [false false true false false]"},
		{"NeMissing", func() string { return seriesCharString(strs.NeMissing(strs.Reverse())) }, "bool [true true false true true]"},
		{"Rle empty", func() string { return frameCharString(empty.Rle()) }, "error: cannot infer data type"},
		{"RleId empty", func() string { return seriesCharString(empty.RleId(), nil) }, "int64 []"},
		{"QCut flat", func() string { return seriesCharString(flat.QCut(3)) }, "string [<= 2 <= 2 <= 2 <= 2]"},
		{"QCut empty", func() string { return seriesCharString(empty.QCut(2)) }, "string []"},
		{"EwmVar nulls", func() string { return seriesCharString(f.EwmVar(0.5), nil) }, "float64 [0 <nil> NaN NaN NaN NaN]"},
		{"Diff", func() string { return seriesCharString(ints.Diff(0), nil) }, "float64 [<nil> <nil> <nil> 8 4]"},
		{"PctChange", func() string { return seriesCharString(f.PctChange(2), nil) }, "float64 [<nil> <nil> NaN <nil> NaN <nil>]"},
		{"PctChange default period", func() string { return seriesCharString(ints.PctChange(0), nil) }, "float64 [<nil> <nil> <nil> -2.6666666666666665 0.8]"},
		{"Not_ int nulls", func() string { return seriesCharString(ints.Not_(), nil) }, "int64 [-6 <nil> 2 -6 -10]"},
		{"Filter null mask", func() string {
			mask := mustSeries(t, "m", dtypes.Boolean, []any{true, nil, false, true, nil})
			return seriesCharString(ints.Filter(mask))
		}, "int64 [5 5]"},
		{"MapElements nulls", func() string {
			return seriesCharString(ints.MapElements(func(v any) any { return v.(int64) * 2 }))
		}, "int64 [10 <nil> -6 10 18]"},
		{"Deserialize nulls", func() string {
			return seriesCharString(ints.Deserialize([]byte(`{"dtype":"int64","values":[1,null,2.5]}`)))
		}, "int64 [1 <nil> 2]"},
		{"Cast nulls", func() string { return seriesCharString(ints.Cast(dtypes.String)) }, "string [5 <nil> -3 5 9]"},
		{"Cast error", func() string { return seriesCharString(strs.Cast(dtypes.Datetime)) }, "error: parsing time \"b\" as \"2006-01-02T15:04:05.999999999Z07:00\": cannot parse \"b\" as \"2006\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.got(); got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
