package polars

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestDataFrameExplode covers Explode on a list column.
func TestDataFrameExplode(t *testing.T) {
	t.Parallel()

	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2)}},
		{Name: "tags", DType: dtypes.List, Values: []any{
			[]any{int64(10), int64(11)},
			[]any{int64(20)},
		}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}

	out, err := d.Explode("tags")
	if err != nil {
		t.Fatalf("Explode: %v", err)
	}
	// 2 + 1 list elements -> 3 rows.
	if out.Height() != 3 {
		t.Errorf("Explode height = %d, want 3", out.Height())
	}
}

// TestDataFrameStructOps covers FlattenStruct and Unnest on a struct column.
func TestDataFrameStructOps(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T) DataFrame {
		t.Helper()
		d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "rec", DType: dtypes.Struct, Values: []any{
				map[string]any{"x": int64(10), "y": int64(11)},
				map[string]any{"x": int64(20), "y": int64(21)},
			}},
		}})
		if err != nil {
			t.Fatalf("frame: %v", err)
		}
		return d
	}

	flat, err := build(t).FlattenStruct("rec", "rec_")
	if err != nil {
		t.Fatalf("FlattenStruct: %v", err)
	}
	if _, ok := flat.Series("rec_x"); !ok {
		t.Error("FlattenStruct did not produce rec_x")
	}

	un, err := build(t).Unnest("rec")
	if err != nil {
		t.Fatalf("Unnest: %v", err)
	}
	if _, ok := un.Series("x"); !ok {
		t.Error("Unnest did not produce x")
	}
}

// TestDataFrameUnstack covers Unstack (partition into sub-frames by key).
func TestDataFrameUnstack(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t) // grp has two distinct values
	parts, err := d.Unstack("grp")
	if err != nil {
		t.Fatalf("Unstack: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("Unstack produced %d partitions, want 2", len(parts))
	}
}

// TestDataFrameSqlAlias covers the lowercase Sql alias.
func TestDataFrameSqlAlias(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)
	ctx := context.Background()
	lz, err := d.Sql(ctx, "SELECT id FROM self")
	if err != nil {
		t.Fatalf("Sql: %v", err)
	}
	out, err := lz.Collect(ctx)
	if err != nil || out.Width() != 1 {
		t.Fatalf("Sql collect width=%d err=%v", out.Width(), err)
	}
}

// TestDataFrameRemainingWriters covers WriteIpcStream and WriteNdjson.
func TestDataFrameRemainingWriters(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)
	dir := t.TempDir()

	if err := d.WriteIpcStream(WriteIPCInput{Path: filepath.Join(dir, "s.arrows")}); err != nil {
		t.Errorf("WriteIpcStream: %v", err)
	}
	if err := d.WriteNdjson(WriteJSONInput{Path: filepath.Join(dir, "s.ndjson")}); err != nil {
		t.Errorf("WriteNdjson: %v", err)
	}
}
