package dataframe

// Ported from py-polars/tests/unit/dataframe/test_show.py (py-1.28.1)

import (
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDFShow(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{"x", "y", "z"}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	s := df.Show(5)
	if !strings.Contains(s, "a") || !strings.Contains(s, "b") {
		t.Fatalf("show should contain column names, got: %s", s)
	}
}

func TestDFShowEmpty(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{}})
	if err != nil {
		t.Fatalf("empty df creation: %v", err)
	}
	s := df.Show(5)
	_ = s // Should not panic on empty df
}
