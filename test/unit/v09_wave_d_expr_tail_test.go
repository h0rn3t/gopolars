package unit

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestV09WaveDExprTailMethods(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(0.5), float64(0.2)}},
			{Name: "i", Values: []any{int64(3), int64(7)}},
			{Name: "b", Values: []any{true, false}},
			{Name: "list", Values: []any{[]any{true, false}, []any{true, true}}},
		},
	})
	if err != nil {
		t.Fatalf("new df failed: %v", err)
	}
	out, err := df.Select(
		polars.Col("x").Arccos().Alias("arccos"),
		polars.Col("x").Arccosh().Alias("arccosh"),
		polars.Col("x").Arcsin().Alias("arcsin"),
		polars.Col("x").Arcsinh().Alias("arcsinh"),
		polars.Col("x").Arctan().Alias("arctan"),
		polars.Col("x").Arctanh().Alias("arctanh"),
		polars.Col("i").BitwiseAnd(polars.Lit(int64(1))).Alias("bw_and"),
		polars.Col("i").BitwiseCountOnes().Alias("bw_ones"),
		polars.Col("i").BitwiseCountZeros().Alias("bw_zeros"),
		polars.Col("i").BitwiseLeadingOnes().Alias("bw_leading_ones"),
		polars.Col("b").And_(polars.Lit(true)).Alias("and_"),
		polars.Col("list").Append(polars.Lit(int64(1))).Alias("append"),
		polars.Col("b").All().Alias("all"),
		polars.Col("b").Any().Alias("any"),
		polars.Col("x").AggGroups().Alias("agg_groups"),
		polars.Col("x").ApproxNUnique().Alias("approx_n_unique"),
		polars.Col("x").ArgMax().Alias("arg_max"),
		polars.Col("x").ArgMin().Alias("arg_min"),
		polars.Col("x").ArgSort().Alias("arg_sort"),
		polars.Col("x").ArgTrue().Alias("arg_true"),
		polars.Col("x").ArgUnique().Alias("arg_unique"),
		polars.Col("x").Arr().Alias("arr"),
		polars.Col("x").BackwardFill().Alias("backward_fill"),
		polars.Col("x").Bin().Alias("bin"),
		polars.Col("i").BitwiseLeadingZeros().Alias("bw_leading_zeros"),
		polars.Col("i").BitwiseTrailingOnes().Alias("bw_trailing_ones"),
		polars.Col("i").BitwiseTrailingZeros().Alias("bw_trailing_zeros"),
		polars.Col("i").BitwiseOr(polars.Lit(int64(8))).Alias("bw_or"),
		polars.Col("i").BitwiseXor(polars.Lit(int64(1))).Alias("bw_xor"),
		polars.Col("x").BottomK(1).Alias("bottom_k"),
		polars.Col("x").BottomKBy(polars.Col("i"), 1).Alias("bottom_k_by"),
		polars.Col("x").Cat().Alias("cat"),
		polars.Col("x").Cbrt().Alias("cbrt"),
		polars.Col("x").Ceil().Alias("ceil"),
		polars.Col("x").Cos().Alias("cos"),
		polars.Col("x").Cosh().Alias("cosh"),
		polars.Col("x").Cot().Alias("cot"),
		polars.Col("x").Count().Alias("count"),
		polars.Col("x").CumMax().Alias("cum_max"),
		polars.Col("x").CumMin().Alias("cum_min"),
		polars.Col("x").CumProd().Alias("cum_prod"),
		polars.Col("x").CumulativeEval().Alias("cumulative_eval"),
		polars.Col("x").Cut().Alias("cut"),
		polars.Col("x").Degrees().Alias("degrees"),
		polars.Col("x").Deserialize().Alias("deserialize"),
		polars.Col("x").Diff().Alias("diff"),
		polars.Col("x").Dot(polars.Lit(float64(2))).Alias("dot"),
		polars.Col("x").DropNans().Alias("drop_nans"),
		polars.Col("x").DropNulls().Alias("drop_nulls"),
		polars.Col("x").Entropy().Alias("entropy"),
		polars.Col("x").EqMissing(polars.Lit(float64(0.5))).Alias("eq_missing"),
		polars.Col("x").EwmMean().Alias("ewm_mean"),
		polars.Col("x").EwmMeanBy(polars.Col("i")).Alias("ewm_mean_by"),
		polars.Col("x").EwmStd().Alias("ewm_std"),
		polars.Col("x").EwmVar().Alias("ewm_var"),
		polars.Col("x").Exclude("i").Alias("exclude"),
		polars.Col("x").Explode().Alias("explode"),
		polars.Col("x").Ext().Alias("ext"),
		polars.Col("x").ExtendConstant(polars.Lit(float64(1))).Alias("extend_constant"),
		polars.Col("x").Filter(polars.Col("b")).Alias("filter"),
		polars.Col("x").First().Alias("first"),
		polars.Col("x").Flatten().Alias("flatten"),
		polars.Col("x").Floor().Alias("floor"),
		polars.Col("x").FloorDiv(polars.Lit(float64(0.2))).Alias("floordiv"),
		polars.Col("x").Floordiv(polars.Lit(float64(0.2))).Alias("floordiv_alias"),
		polars.Col("x").ForwardFill().Alias("forward_fill"),
		polars.Lit("{\"a\":1}").FromJSON().Alias("from_json"),
		polars.Lit("{\"a\":1}").FromJson().Alias("from_json_alias"),
		polars.Col("list").Gather(polars.Lit(int64(0))).Alias("gather"),
		polars.Col("x").GatherEvery(1).Alias("gather_every"),
		polars.Col("list").Get(polars.Lit(int64(1))).Alias("get"),
		polars.Col("list").HasNulls().Alias("has_nulls"),
		polars.Col("x").Hash().Alias("hash"),
		polars.Col("x").Head(1).Alias("head"),
		polars.Col("x").Hist().Alias("hist"),
		polars.Col("x").Implode().Alias("implode"),
		polars.Col("list").IndexOf(polars.Lit(true)).Alias("index_of"),
		polars.Col("x").Inspect().Alias("inspect"),
		polars.Col("x").Interpolate().Alias("interpolate"),
		polars.Col("x").InterpolateBy(polars.Col("i")).Alias("interpolate_by"),
		polars.Col("x").IsBetween(polars.Lit(float64(0.1)), polars.Lit(float64(1.0))).Alias("is_between"),
		polars.Col("x").IsClose(polars.Lit(float64(0.5))).Alias("is_close"),
		polars.Col("x").IsDuplicated().Alias("is_duplicated"),
		polars.Col("x").IsFinite().Alias("is_finite"),
		polars.Col("x").IsFirstDistinct().Alias("is_first_distinct"),
		polars.Col("x").IsIn(polars.Lit([]any{float64(0.5), float64(9)})).Alias("is_in"),
		polars.Col("x").IsInfinite().Alias("is_infinite"),
		polars.Col("x").IsLastDistinct().Alias("is_last_distinct"),
		polars.Col("x").IsNan().Alias("is_nan"),
		polars.Col("x").IsNotNan().Alias("is_not_nan"),
		polars.Col("x").IsUnique().Alias("is_unique"),
		polars.Col("x").Item().Alias("item"),
		polars.Col("x").Kurtosis().Alias("kurtosis"),
		polars.Col("x").Last().Alias("last"),
		polars.Col("x").Limit(1).Alias("limit"),
		polars.Col("x").Log10().Alias("log10"),
		polars.Col("x").Log1p().Alias("log1p"),
		polars.Col("x").LowerBound().Alias("lower_bound"),
		polars.Col("x").MapBatches().Alias("map_batches"),
		polars.Col("x").MapElements().Alias("map_elements"),
		polars.Col("x").Max().Alias("max"),
		polars.Col("x").MaxBy(polars.Col("i")).Alias("max_by"),
		polars.Col("x").Mean().Alias("mean"),
		polars.Col("x").Median().Alias("median"),
		polars.Col("x").Meta().Alias("meta"),
		polars.Col("x").Min().Alias("min"),
		polars.Col("x").MinBy(polars.Col("i")).Alias("min_by"),
		polars.Col("x").Mod(polars.Lit(float64(0.2))).Alias("mod"),
		polars.Col("x").Mode().Alias("mode"),
		polars.Col("x").NUnique().Alias("n_unique"),
		polars.Col("x").NanMax().Alias("nan_max"),
		polars.Col("x").NanMin().Alias("nan_min"),
		polars.Col("x").NeMissing(polars.Lit(float64(0.9))).Alias("ne_missing"),
		polars.Col("x").Neg().Alias("neg"),
		polars.Col("b").Not_().Alias("not_"),
		polars.Col("x").NullCount().Alias("null_count"),
		polars.Col("b").Or_(polars.Lit(false)).Alias("or_"),
		polars.Col("x").PctChange().Alias("pct_change"),
		polars.Col("x").PeakMax().Alias("peak_max"),
		polars.Col("x").PeakMin().Alias("peak_min"),
		polars.Col("x").Pipe().Alias("pipe"),
		polars.Col("x").Product().Alias("product"),
		polars.Col("x").QCut().Alias("qcut"),
		polars.Col("x").Quantile().Alias("quantile"),
	)
	if err != nil {
		t.Fatalf("expr tail select failed: %v", err)
	}
	if out.Height() != df.Height() {
		t.Fatalf("expr tail output mismatch")
	}
	col, _ := out.GetColumn("bw_and")
	if col.Value(0).(int64) != 1 {
		t.Fatalf("bitwise_and mismatch")
	}
	orCol, _ := out.GetColumn("bw_or")
	if orCol.Value(0).(int64) != 11 {
		t.Fatalf("bitwise_or mismatch")
	}
	arc, _ := out.GetColumn("arccos")
	if math.IsNaN(arc.Value(0).(float64)) {
		t.Fatalf("arccos mismatch")
	}
	fd, _ := out.GetColumn("floordiv")
	if fd.Value(0).(float64) != 2 {
		t.Fatalf("floordiv mismatch")
	}
	ib, _ := out.GetColumn("is_between")
	if ib.Value(0).(bool) != true {
		t.Fatalf("is_between mismatch")
	}
	ng, _ := out.GetColumn("neg")
	if ng.Value(0).(float64) != -0.5 {
		t.Fatalf("neg mismatch")
	}
	mod, _ := out.GetColumn("mod")
	if math.Abs(mod.Value(0).(float64)-0.1) > 1e-9 {
		t.Fatalf("mod mismatch")
	}
}
