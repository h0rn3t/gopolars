package polars

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestDataFrameFoldInterpolateUpdate covers Fold, Interpolate, Update, and the
// sequential select/with-columns variants.
func TestDataFrameFoldInterpolateUpdate(t *testing.T) {
	t.Parallel()

	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "b", Values: []any{int64(10), int64(20), int64(30)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}

	// Fold sums columns a,b row-wise into "total".
	folded, err := d.Fold("sum", []string{"a", "b"}, "total")
	if err != nil {
		t.Fatalf("Fold: %v", err)
	}
	if tot, ok := folded.Series("total"); !ok {
		t.Error("Fold: total column missing")
	} else if v, _ := toFloatLocal(tot.Value(0)); v != 11 {
		t.Errorf("Fold total[0] = %v, want 11", v)
	}

	// Interpolate over a gap-free column is a no-op on height.
	interp, err := d.Interpolate("a")
	if err != nil || interp.Height() != 3 {
		t.Fatalf("Interpolate height=%d err=%v", interp.Height(), err)
	}

	// Update == VStack: stacking the frame on itself doubles the rows.
	updated, err := d.Update(d)
	if err != nil || updated.Height() != 6 {
		t.Fatalf("Update height=%d err=%v", updated.Height(), err)
	}

	// SelectSeq / WithColumnsSeq.
	sel, err := d.SelectSeq(Col("a"))
	if err != nil || sel.Width() != 1 {
		t.Fatalf("SelectSeq width=%d err=%v", sel.Width(), err)
	}
	wc, err := d.WithColumnsSeq(Col("a").Add(Col("b")).Alias("sum"))
	if err != nil {
		t.Fatalf("WithColumnsSeq: %v", err)
	}
	if _, ok := wc.Series("sum"); !ok {
		t.Error("WithColumnsSeq: sum column missing")
	}

	// ToStruct returns a column-keyed representation.
	if st := d.ToStruct(); len(st) == 0 {
		t.Error("ToStruct returned empty map")
	}
}

// TestDataFrameSerializeRoundTrip covers Serialize/Deserialize.
func TestDataFrameSerializeRoundTrip(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)

	payload, err := d.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if len(payload) == 0 {
		t.Fatal("Serialize returned empty payload")
	}
	back, err := d.Deserialize(payload)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	// Serialize is JSON-based (json.Marshal of ToDicts), so the round-trip
	// preserves shape and column names but not the int64/float64 distinction
	// (JSON numbers are untyped). Assert the shape contract.
	if back.Height() != d.Height() || back.Width() != d.Width() {
		t.Fatalf("round-trip shape = %dx%d, want %dx%d", back.Height(), back.Width(), d.Height(), d.Width())
	}
	for _, c := range d.Columns() {
		if _, ok := back.Series(c); !ok {
			t.Errorf("round-trip dropped column %q", c)
		}
	}
}

// TestDataFrameShowAndWriters covers Show plus the IPC / NDJSON writers.
func TestDataFrameShowAndWriters(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)
	dir := t.TempDir()

	if s := d.Show(2); s == "" {
		t.Error("Show returned empty string")
	}
	if err := d.WriteIpc(WriteIPCInput{Path: filepath.Join(dir, "out.arrow")}); err != nil {
		t.Errorf("WriteIpc: %v", err)
	}
	if err := d.WriteNDJSON(WriteJSONInput{Path: filepath.Join(dir, "out.ndjson")}); err != nil {
		t.Errorf("WriteNDJSON: %v", err)
	}
}

// TestDataFrameSQL covers the SQL entry point on a DataFrame.
func TestDataFrameSQL(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)
	ctx := context.Background()

	lz, err := d.SQL(ctx, "SELECT id FROM self WHERE id > 1")
	if err != nil {
		t.Fatalf("SQL: %v", err)
	}
	out, err := lz.Collect(ctx)
	if err != nil {
		t.Fatalf("SQL collect: %v", err)
	}
	if out.Height() != 3 {
		t.Errorf("SQL height = %d, want 3 (id 2,3,4)", out.Height())
	}
}

// TestLazyFrameCollectStreaming covers the streaming collect entry point.
func TestLazyFrameCollectStreaming(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)
	ctx := context.Background()

	out, err := d.Lazy().Filter(Col("id").Ge(Lit(int64(2)))).CollectStreaming(ctx, 2)
	if err != nil {
		t.Fatalf("CollectStreaming: %v", err)
	}
	if out.Height() != 3 {
		t.Errorf("CollectStreaming height = %d, want 3", out.Height())
	}
}
