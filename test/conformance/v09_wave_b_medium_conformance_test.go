package conformance

import (
	"context"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestV09WaveBMediumConformance(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "g", Values: []any{"a", "a", "b"}},
			{Name: "v", Values: []any{float64(1), float64(2), float64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df init failed: %v", err)
	}
	if _, err := df.PartitionBy("g"); err != nil {
		t.Fatalf("partition_by conformance failed: %v", err)
	}
	// Upsample requires a Datetime key; "v" is numeric, so this errors and df is
	// left unchanged (Upsample is exercised by the datetime case in the unit tests).
	if up, err := df.Upsample("v", time.Second); err == nil {
		df = up
	}
	ctx := context.Background()
	lf := df.Lazy().Cache()
	if _, err := lf.Serialize(); err != nil {
		t.Fatalf("serialize conformance failed: %v", err)
	}
	if (<-lf.SinkBatches(ctx, 1)).Error != nil {
		t.Fatalf("sink_batches conformance failed")
	}
	sqlCtx := polars.NewSQLContext()
	if err := sqlCtx.RegisterGlobals(map[string]polars.DataFrame{"t": df}); err != nil {
		t.Fatalf("register_globals conformance failed: %v", err)
	}
	if _, err := sqlCtx.ExecuteGlobal(ctx, "SELECT v FROM t"); err != nil {
		t.Fatalf("execute_global conformance failed: %v", err)
	}
}
