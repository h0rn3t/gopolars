package conformance

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestV09WaveDComputeConformance(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "g", Values: []any{"a", "b"}},
			{Name: "x", Values: []any{float64(1), float64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if _, err := df.SQL(context.Background(), "SELECT x FROM self"); err != nil {
		t.Fatalf("df sql conformance failed: %v", err)
	}
	if _, err := df.Sql(context.Background(), "SELECT x FROM self"); err != nil {
		t.Fatalf("df sql alias conformance failed: %v", err)
	}
	if _, err := df.ToDummies("g"); err != nil {
		t.Fatalf("to_dummies conformance failed: %v", err)
	}
	if err := df.WriteNdjson(polars.WriteJSONInput{Path: "/tmp/gopolars_wave_d_conf.ndjson", NDJSON: true}); err != nil {
		t.Fatalf("write_ndjson conformance failed: %v", err)
	}
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.Float64, Values: []any{float64(-1), float64(1)}})
	if err != nil {
		t.Fatalf("series init failed: %v", err)
	}
	if s.Abs().Len() != 2 || s.Exp().Len() != 2 {
		t.Fatalf("series unary conformance failed")
	}
}
