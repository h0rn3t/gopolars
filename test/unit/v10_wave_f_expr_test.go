package unit

import (
	"math"
	"reflect"
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestV10WaveFExprTrigSurface(t *testing.T) {
	t.Parallel()

	e := polars.Col("x")
	methods := []string{"Sin", "Sinh", "Tan", "Tanh", "Sign", "Radians", "RoundSigFigs"}
	for _, name := range methods {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !reflect.ValueOf(e).MethodByName(name).IsValid() {
				t.Fatalf("expected Expr method %s to be exposed", name)
			}
		})
	}
}

func TestV10WaveFExprTrigEval(t *testing.T) {
	t.Parallel()

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(0.5), float64(1.0)}},
			{Name: "deg", Values: []any{float64(180), float64(90)}},
			{Name: "sign", Values: []any{float64(-2), float64(0)}},
			{Name: "sig", Values: []any{float64(1234.567), float64(0.012345)}},
		},
	})
	if err != nil {
		t.Fatalf("new df failed: %v", err)
	}

	out, err := df.Select(
		polars.Col("x").Sin().Alias("sin"),
		polars.Col("x").Sinh().Alias("sinh"),
		polars.Col("x").Tan().Alias("tan"),
		polars.Col("x").Tanh().Alias("tanh"),
		polars.Col("sign").Sign().Alias("sign_out"),
		polars.Col("deg").Radians().Alias("radians"),
		polars.Col("sig").RoundSigFigs(3).Alias("sig3"),
	)
	if err != nil {
		t.Fatalf("select failed: %v", err)
	}

	assertExprFloatColumn(t, out, "sin", []float64{math.Sin(0.5), math.Sin(1.0)})
	assertExprFloatColumn(t, out, "sinh", []float64{math.Sinh(0.5), math.Sinh(1.0)})
	assertExprFloatColumn(t, out, "tan", []float64{math.Tan(0.5), math.Tan(1.0)})
	assertExprFloatColumn(t, out, "tanh", []float64{math.Tanh(0.5), math.Tanh(1.0)})
	assertExprFloatColumn(t, out, "sign_out", []float64{-1, 0})
	assertExprFloatColumn(t, out, "radians", []float64{math.Pi, math.Pi / 2})
	assertExprFloatColumn(t, out, "sig3", []float64{1230, 0.0123})
}

func TestV10WaveFExprRollingSurface(t *testing.T) {
	t.Parallel()

	e := polars.Col("x")
	methods := []string{
		"RollingMaxBy", "RollingMeanBy", "RollingMinBy", "RollingSumBy", "RollingStdBy", "RollingVarBy",
		"RollingMedian", "RollingMedianBy", "RollingQuantile", "RollingQuantileBy",
		"RollingSkew", "RollingKurtosis", "RollingMap", "Rolling", "RollingRank", "RollingRankBy",
	}
	for _, name := range methods {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !reflect.ValueOf(e).MethodByName(name).IsValid() {
				t.Fatalf("expected Expr method %s to be exposed", name)
			}
		})
	}
}

func TestV10WaveFExprTailSurface(t *testing.T) {
	t.Parallel()

	e := polars.Col("x")
	methods := []string{
		"Sort", "SortBy", "Slice", "Tail", "Unique", "UniqueCounts", "ValueCounts", "Rechunk", "Reinterpret",
		"RepeatBy", "ReplaceStrict", "Reshape", "Rle", "RleId", "Sample", "SearchSorted", "SetSorted",
		"Shift", "ShrinkDtype", "Shuffle", "Skew", "Std", "Sum", "Var", "Where", "Xor",
		"ToPhysical", "TopK", "TopKBy", "Truncate", "TrueDiv", "UpperBound",
	}
	for _, name := range methods {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !reflect.ValueOf(e).MethodByName(name).IsValid() {
				t.Fatalf("expected Expr method %s to be exposed", name)
			}
		})
	}
}

func TestV10WaveFExprRollingAndTailEval(t *testing.T) {
	t.Parallel()

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(4), float64(2)}},
			{Name: "by", Values: []any{float64(1), float64(2)}},
			{Name: "b", Values: []any{true, false}},
		},
	})
	if err != nil {
		t.Fatalf("new df failed: %v", err)
	}

	out, err := df.Select(
		polars.Col("x").RollingMaxBy(polars.Col("by"), 2).Alias("rolling_max_by"),
		polars.Col("x").RollingMedian(2).Alias("rolling_median"),
		polars.Col("x").RollingQuantile(2, 0.5).Alias("rolling_quantile"),
		polars.Col("x").RollingRank(2).Alias("rolling_rank"),
		polars.Col("x").Sort(false).Alias("sort"),
		polars.Col("x").Slice(0, 1).Alias("slice"),
		polars.Col("x").Tail(1).Alias("tail"),
		polars.Col("x").Unique().Alias("unique"),
		polars.Col("x").ValueCounts().Alias("value_counts"),
		polars.Col("x").Std().Alias("std"),
		polars.Col("x").Sum().Alias("sum"),
		polars.Col("x").Var().Alias("var"),
		polars.Col("x").Skew().Alias("skew"),
		polars.Col("x").Where(polars.Col("b")).Alias("where"),
		polars.Col("b").Xor(polars.Lit(true)).Alias("xor"),
		polars.Col("x").TrueDiv(polars.Lit(float64(2))).Alias("true_div"),
		polars.Col("x").TopK(1).Alias("top_k"),
		polars.Col("x").TopKBy(polars.Col("by"), 1).Alias("top_k_by"),
		polars.Col("x").ToPhysical().Alias("to_physical"),
		polars.Col("x").UpperBound().Alias("upper_bound"),
	)
	if err != nil {
		t.Fatalf("select failed: %v", err)
	}

	assertExprFloatColumn(t, out, "rolling_max_by", []float64{4, 2})
	assertExprFloatColumn(t, out, "rolling_median", []float64{4, 2})
	assertExprFloatColumn(t, out, "rolling_quantile", []float64{4, 2})
	assertExprFloatColumn(t, out, "rolling_rank", []float64{4, 2})
	assertExprFloatColumn(t, out, "sort", []float64{4, 2})
	assertExprFloatColumn(t, out, "slice", []float64{4, 2})
	assertExprFloatColumn(t, out, "tail", []float64{4, 2})
	assertExprFloatColumn(t, out, "unique", []float64{4, 2})
	assertExprFloatColumn(t, out, "value_counts", []float64{4, 2})
	assertExprFloatColumn(t, out, "std", []float64{4, 2})
	assertExprFloatColumn(t, out, "sum", []float64{4, 2})
	assertExprFloatColumn(t, out, "var", []float64{4, 2})
	assertExprFloatColumn(t, out, "skew", []float64{4, 2})
	assertExprColumnAny(t, out, "where", []any{float64(4), nil})
	assertExprColumnAny(t, out, "xor", []any{false, true})
	assertExprFloatColumn(t, out, "true_div", []float64{2, 1})
	assertExprFloatColumn(t, out, "top_k", []float64{4, 2})
	assertExprFloatColumn(t, out, "top_k_by", []float64{4, 2})
	assertExprFloatColumn(t, out, "to_physical", []float64{4, 2})
	assertExprFloatColumn(t, out, "upper_bound", []float64{4, 2})
}

func assertExprFloatColumn(t *testing.T, df polars.DataFrame, column string, want []float64) {
	t.Helper()

	got, err := df.GetColumn(column)
	if err != nil {
		t.Fatalf("get column %s failed: %v", column, err)
	}
	if got.Len() != len(want) {
		t.Fatalf("unexpected len for %s: got %d want %d", column, got.Len(), len(want))
	}
	for i, expected := range want {
		value, ok := got.Value(i).(float64)
		if !ok {
			t.Fatalf("expected float64 in %s[%d], got %T", column, i, got.Value(i))
		}
		if math.Abs(value-expected) > 1e-9 {
			t.Fatalf("unexpected %s[%d]: got %.12f want %.12f", column, i, value, expected)
		}
	}
}

func assertExprColumnAny(t *testing.T, df polars.DataFrame, column string, want []any) {
	t.Helper()

	got, err := df.GetColumn(column)
	if err != nil {
		t.Fatalf("get column %s failed: %v", column, err)
	}
	if got.Len() != len(want) {
		t.Fatalf("unexpected len for %s: got %d want %d", column, got.Len(), len(want))
	}
	for i, expected := range want {
		if !reflect.DeepEqual(got.Value(i), expected) {
			t.Fatalf("unexpected %s[%d]: got %v want %v", column, i, got.Value(i), expected)
		}
	}
}
