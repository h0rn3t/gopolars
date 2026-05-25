package conformance

import (
	"math"
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestV10WaveFExpr(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(4), float64(2)}},
			{Name: "by", Values: []any{float64(1), float64(2)}},
			{Name: "deg", Values: []any{float64(180), float64(90)}},
			{Name: "sign", Values: []any{float64(-2), float64(0)}},
			{Name: "sig", Values: []any{float64(1234.567), float64(0.012345)}},
			{Name: "b", Values: []any{true, false}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}

	out, err := df.Select(
		polars.Col("x").Sin().Alias("sin"),
		polars.Col("deg").Radians().Alias("radians"),
		polars.Col("sign").Sign().Alias("sign_out"),
		polars.Col("sig").RoundSigFigs(3).Alias("sig3"),
		polars.Col("x").RollingMaxBy(polars.Col("by"), 2).Alias("rolling_max_by"),
		polars.Col("x").RollingMedian(2).Alias("rolling_median"),
		polars.Col("x").Sort(false).Alias("sort"),
		polars.Col("x").Slice(0, 1).Alias("slice"),
		polars.Col("x").Tail(1).Alias("tail"),
		polars.Col("x").Unique().Alias("unique"),
		polars.Col("x").ValueCounts().Alias("value_counts"),
		polars.Col("x").Std().Alias("std"),
		polars.Col("x").Sum().Alias("sum"),
		polars.Col("x").Var().Alias("var"),
		polars.Col("x").Where(polars.Col("b")).Alias("where"),
		polars.Col("b").Xor(polars.Lit(true)).Alias("xor"),
		polars.Col("x").TrueDiv(polars.Lit(float64(2))).Alias("true_div"),
		polars.Col("x").TopK(1).Alias("top_k"),
		polars.Col("x").TopKBy(polars.Col("by"), 1).Alias("top_k_by"),
		polars.Col("x").ToPhysical().Alias("to_physical"),
		polars.Col("x").UpperBound().Alias("upper_bound"),
	)
	if err != nil {
		t.Fatalf("expr conformance select failed: %v", err)
	}

	sin, err := out.GetColumn("sin")
	if err != nil {
		t.Fatalf("get sin failed: %v", err)
	}
	if got := sin.Value(0).(float64); math.Abs(got-math.Sin(4)) > 1e-9 {
		t.Fatalf("unexpected sin value: got %.12f want %.12f", got, math.Sin(4))
	}
	radians, err := out.GetColumn("radians")
	if err != nil {
		t.Fatalf("get radians failed: %v", err)
	}
	if got := radians.Value(0).(float64); math.Abs(got-math.Pi) > 1e-9 {
		t.Fatalf("unexpected radians value: got %.12f want %.12f", got, math.Pi)
	}
	where, err := out.GetColumn("where")
	if err != nil {
		t.Fatalf("get where failed: %v", err)
	}
	if where.Value(1) != nil {
		t.Fatalf("unexpected where value: got %v want nil", where.Value(1))
	}
	xor, err := out.GetColumn("xor")
	if err != nil {
		t.Fatalf("get xor failed: %v", err)
	}
	if xor.Value(0) != false || xor.Value(1) != true {
		t.Fatalf("unexpected xor values: got [%v %v] want [false true]", xor.Value(0), xor.Value(1))
	}
	trueDiv, err := out.GetColumn("true_div")
	if err != nil {
		t.Fatalf("get true_div failed: %v", err)
	}
	if trueDiv.Value(0).(float64) != 2 || trueDiv.Value(1).(float64) != 1 {
		t.Fatalf("unexpected true_div values: got [%v %v] want [2 1]", trueDiv.Value(0), trueDiv.Value(1))
	}
	for _, name := range []string{
		"rolling_max_by", "rolling_median", "sort", "slice", "tail", "unique",
		"value_counts", "std", "sum", "var", "top_k", "top_k_by", "to_physical", "upper_bound",
	} {
		col, err := out.GetColumn(name)
		if err != nil {
			t.Fatalf("get %s failed: %v", name, err)
		}
		if col.Len() != 2 {
			t.Fatalf("unexpected %s len: got %d want 2", name, col.Len())
		}
	}
}
