package polars

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestLazyFrameMeltAndMapBatches covers the lazy Melt, MapBatches, MatchToSchema,
// SubSelectColumns and WithContext methods through Collect.
func TestLazyFrameMeltAndMapBatches(t *testing.T) {
	t.Parallel()

	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2)}},
		{Name: "x", Values: []any{int64(10), int64(20)}},
		{Name: "y", Values: []any{int64(30), int64(40)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	ctx := context.Background()

	melted, err := d.Lazy().Melt(MeltInput{
		IDVars: []string{"id"}, ValueVars: []string{"x", "y"},
		VariableCol: "variable", ValueCol: "value",
	}).Collect(ctx)
	if err != nil || melted.Height() != 4 {
		t.Fatalf("lazy Melt height=%d err=%v, want 4", melted.Height(), err)
	}

	// MapBatches applies a function to the collected frame.
	mapped, err := d.Lazy().MapBatches(func(in DataFrame) (DataFrame, error) {
		return in.Head(1), nil
	}).Collect(ctx)
	if err != nil || mapped.Height() != 1 {
		t.Fatalf("MapBatches height=%d err=%v, want 1", mapped.Height(), err)
	}

	// MatchToSchema adds a missing column.
	matched, err := d.Lazy().MatchToSchema(dtypes.Schema{
		{Name: "id", Type: dtypes.Int64},
		{Name: "z", Type: dtypes.Float64},
	}).Collect(ctx)
	if err != nil {
		t.Fatalf("MatchToSchema: %v", err)
	}
	if _, ok := matched.Series("z"); !ok {
		t.Error("MatchToSchema did not add z")
	}

	// SubSelectColumns narrows to the named columns.
	sub, err := d.Lazy().SubSelectColumns("id", "x").Collect(ctx)
	if err != nil || sub.Width() != 2 {
		t.Fatalf("SubSelectColumns width=%d err=%v, want 2", sub.Width(), err)
	}

	// WithContext attaches another frame's columns as context (no row change).
	ctxOut, err := d.Lazy().WithContext(d.Lazy()).Collect(ctx)
	if err != nil || ctxOut.Height() != 2 {
		t.Fatalf("WithContext height=%d err=%v, want 2", ctxOut.Height(), err)
	}
}

// TestLazyFrameDiagnostics covers the lazy Show / ShowGraph / Serialize /
// Deserialize / Lazy diagnostic surface.
func TestLazyFrameDiagnostics(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)
	lz := d.Lazy()

	if lz.Show(2) == "" {
		t.Error("lazy Show returned empty string")
	}
	if lz.ShowGraph() == "" {
		t.Error("ShowGraph returned empty string")
	}
	if lz.Lazy() == nil {
		t.Error("Lazy() returned nil")
	}

	payload, err := lz.Serialize()
	if err != nil || len(payload) == 0 {
		t.Fatalf("Serialize payload=%d err=%v", len(payload), err)
	}
	if _, err := lz.Deserialize(payload); err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
}
