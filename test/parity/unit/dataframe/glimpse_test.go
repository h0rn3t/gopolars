package dataframe

// Ported from py-polars/tests/unit/dataframe/test_glimpse.py (py-1.28.1)

import (
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFGlimpse(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{float64(4), float64(5), float64(6)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	g := df.Glimpse(5)
	if !strings.Contains(g, "a") || !strings.Contains(g, "b") {
		t.Fatalf("glimpse should contain column names, got: %s", g)
	}
}
