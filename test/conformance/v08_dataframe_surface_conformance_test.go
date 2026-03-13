package conformance

import (
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestDataFrameSurfaceV08Conformance(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "b", Values: []any{float64(4), float64(3), float64(2), float64(1)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}
	if _, err := df.BottomK(2, "a"); err != nil {
		t.Fatalf("bottom_k conformance failed: %v", err)
	}
	if _, err := df.Corr("a", "b"); err != nil {
		t.Fatalf("corr conformance failed: %v", err)
	}
	if _, err := df.Describe(); err != nil {
		t.Fatalf("describe conformance failed: %v", err)
	}
	if _, err := df.HashRows(7); err != nil {
		t.Fatalf("hash_rows conformance failed: %v", err)
	}
	if _, err := df.Fold("sum", []string{"a", "b"}, "sum_ab"); err != nil {
		t.Fatalf("fold conformance failed: %v", err)
	}
}
