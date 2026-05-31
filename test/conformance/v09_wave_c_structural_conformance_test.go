package conformance

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestV09WaveCStructuralConformance(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "k", Values: []any{"a", "a", "b"}},
			{Name: "x", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if _, err := df.MeanHorizontal("avg"); err != nil {
		t.Fatalf("horizontal conformance failed: %v", err)
	}
	if _, err := df.Remove("x"); err != nil {
		t.Fatalf("remove conformance failed: %v", err)
	}
	ctx := context.Background()
	if _, err := df.Lazy().ApproxNUnique("k").Collect(ctx); err != nil {
		t.Fatalf("lazy approx_n_unique conformance failed: %v", err)
	}
}
