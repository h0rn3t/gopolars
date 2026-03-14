package conformance

import (
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestV09WaveDExprTailConformance(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(0.5)}},
			{Name: "i", Values: []any{int64(3)}},
			{Name: "b", Values: []any{true}},
			{Name: "list", Values: []any{[]any{true, false}}},
		},
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	_, err = df.Select(
		polars.Col("x").Arctan(),
		polars.Col("i").BitwiseCountOnes(),
		polars.Col("b").And_(polars.Lit(true)),
		polars.Col("list").Append(polars.Lit(int64(1))),
		polars.Col("i").BitwiseOr(polars.Lit(int64(8))),
		polars.Col("x").Cbrt(),
		polars.Col("x").Degrees(),
		polars.Col("x").Dot(polars.Lit(float64(2))),
		polars.Col("x").Entropy(),
		polars.Col("x").Floor(),
		polars.Col("x").FloorDiv(polars.Lit(float64(0.2))),
		polars.Col("x").Floordiv(polars.Lit(float64(0.2))),
		polars.Lit("{\"a\":1}").FromJson(),
		polars.Col("x").IsClose(polars.Lit(float64(0.5))),
		polars.Col("x").IsBetween(polars.Lit(float64(0)), polars.Lit(float64(1))),
		polars.Col("x").Log10(),
		polars.Col("x").MapBatches(),
		polars.Col("x").MapElements(),
		polars.Col("x").Max(),
		polars.Col("x").Min(),
		polars.Col("x").Mod(polars.Lit(float64(0.2))),
		polars.Col("x").Neg(),
		polars.Col("b").Not_(),
		polars.Col("b").Or_(polars.Lit(false)),
		polars.Col("x").Quantile(),
		polars.Col("list").Gather(polars.Lit(int64(0))),
		polars.Col("x").Implode(),
	)
	if err != nil {
		t.Fatalf("expr tail conformance failed: %v", err)
	}
	if _, err := df.Vstack(df); err != nil {
		t.Fatalf("vstack alias conformance failed: %v", err)
	}
}
