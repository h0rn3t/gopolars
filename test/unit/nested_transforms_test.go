package unit

import (
	"context"
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestNestedExplodeAndFlatten(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "tags", Values: []any{[]any{"a", "b"}, []any{"c"}}},
			{Name: "meta", Values: []any{map[string]any{"x": int64(1), "y": "u"}, map[string]any{"x": int64(2), "y": "v"}}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}
	exploded, err := df.Explode("tags")
	if err != nil {
		t.Fatalf("explode failed: %v", err)
	}
	if exploded.Height() != 3 {
		t.Fatalf("unexpected explode height: %d", exploded.Height())
	}
	flat, err := exploded.FlattenStruct("meta", "meta_")
	if err != nil {
		t.Fatalf("flatten failed: %v", err)
	}
	if _, ok := flat.Series("meta_x"); !ok {
		t.Fatalf("missing flattened column")
	}
}

func TestNestedExprParityEagerLazy(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "tags", Values: []any{[]any{"a", "b"}, []any{"x"}}},
			{Name: "meta", Values: []any{map[string]any{"x": int64(10)}, map[string]any{"x": int64(20)}}},
		},
	})
	eager, err := df.Select(
		polars.Col("tags").ListLen().Alias("n"),
		polars.Col("meta").StructField("x").Alias("x"),
	)
	if err != nil {
		t.Fatalf("eager select failed: %v", err)
	}
	lazy, err := df.Lazy().
		Select(
			polars.Col("tags").ListLen().Alias("n"),
			polars.Col("meta").StructField("x").Alias("x"),
		).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("lazy collect failed: %v", err)
	}
	if eager.Height() != lazy.Height() || eager.Width() != lazy.Width() {
		t.Fatalf("eager/lazy mismatch")
	}
}
