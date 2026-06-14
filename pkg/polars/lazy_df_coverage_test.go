package polars

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// lazyCovFrame builds a frame with a null and groups for the lazy chain tests.
func lazyCovFrame(t *testing.T) DataFrame {
	t.Helper()
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		{Name: "grp", Values: []any{"a", "a", "b", "b"}},
		{Name: "val", Values: []any{int64(10), nil, int64(30), int64(40)}},
	}})
	if err != nil {
		t.Fatalf("lazyCovFrame: %v", err)
	}
	return d
}

// TestLazyFrameChainMethods drives a broad set of LazyFrame transforms through
// Collect, asserting a basic property for each.
func TestLazyFrameChainMethods(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)
	ctx := context.Background()

	collect := func(name string, lz LazyFrame) DataFrame {
		out, err := lz.Collect(ctx)
		if err != nil {
			t.Fatalf("%s collect: %v", name, err)
		}
		return out
	}

	// GatherEvery(2,0) keeps rows 0,2.
	if got := collect("gather_every", d.Lazy().GatherEvery(2, 0)); got.Height() != 2 {
		t.Errorf("gather_every height = %d, want 2", got.Height())
	}
	// Clone is an identity copy.
	if got := collect("clone", d.Lazy().Clone()); got.Height() != 4 {
		t.Errorf("clone height = %d, want 4", got.Height())
	}
	// Rename id -> ident.
	renamed := collect("rename", d.Lazy().Rename(map[string]string{"id": "ident"}))
	if _, ok := renamed.Series("ident"); !ok {
		t.Error("rename: ident column missing")
	}
	// FillNull replaces the null in val.
	filled := collect("fill_null", d.Lazy().FillNull(int64(0)))
	if vc, _ := filled.Series("val"); vc.Value(1) != int64(0) {
		t.Errorf("fill_null val[1] = %v, want 0", vc.Value(1))
	}
	// DropNulls on val drops the null row.
	if got := collect("drop_nulls", d.Lazy().DropNulls("val")); got.Height() != 3 {
		t.Errorf("drop_nulls height = %d, want 3", got.Height())
	}
	// DropNaNs is a no-op for integer columns here.
	if got := collect("drop_nans", d.Lazy().DropNaNs()); got.Height() != 4 {
		t.Errorf("drop_nans height = %d, want 4", got.Height())
	}
	// Drop removes a column.
	dropped := collect("drop", d.Lazy().Drop("val"))
	if _, ok := dropped.Series("val"); ok {
		t.Error("drop: val should be gone")
	}
	// Remove is the single-column form.
	removed := collect("remove", d.Lazy().Remove("grp"))
	if _, ok := removed.Series("grp"); ok {
		t.Error("remove: grp should be gone")
	}
	// SelectSeq / WithColumnsSeq sequential variants.
	sel := collect("select_seq", d.Lazy().SelectSeq(Col("id")))
	if sel.Width() != 1 {
		t.Errorf("select_seq width = %d, want 1", sel.Width())
	}
	wc := collect("with_columns_seq", d.Lazy().WithColumnsSeq(Col("id").Mul(Lit(int64(10))).Alias("id10")))
	if _, ok := wc.Series("id10"); !ok {
		t.Error("with_columns_seq: id10 missing")
	}
	// Cache / Inspect are pass-through optimizer hints.
	if got := collect("cache", d.Lazy().Cache()); got.Height() != 4 {
		t.Errorf("cache height = %d", got.Height())
	}
	if got := collect("inspect", d.Lazy().Inspect()); got.Height() != 4 {
		t.Errorf("inspect height = %d", got.Height())
	}
	// Pipe applies a user function to the lazy frame.
	piped := collect("pipe", d.Lazy().Pipe(func(l LazyFrame) LazyFrame { return l.Head(2) }))
	if piped.Height() != 2 {
		t.Errorf("pipe height = %d, want 2", piped.Height())
	}
}

// TestLazyFrameExplainAndDescribe covers the diagnostic methods.
func TestLazyFrameExplainAndDescribe(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)

	if got := d.Lazy().Filter(Col("id").Gt(Lit(int64(1)))).Explain(true); got == "" {
		t.Error("Explain(true) returned empty string")
	}
	if got := d.Lazy().Explain(false); got == "" {
		t.Error("Explain(false) returned empty string")
	}
	desc, err := d.Lazy().Describe()
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if desc.Height() == 0 {
		t.Error("Describe returned no rows")
	}
}

// TestDataFrameMiscMethods covers assorted DataFrame helpers.
func TestDataFrameMiscMethods(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)

	if sch := d.CollectSchema(); len(sch) != 3 {
		t.Errorf("CollectSchema len = %d, want 3", len(sch))
	}
	if n := d.NChunks(); n < 1 {
		t.Errorf("NChunks = %d, want >= 1", n)
	}
	if repr := d.ToInitRepr(); repr == "" {
		t.Error("ToInitRepr returned empty string")
	}
	if got := d.ShrinkToFit(); got.Height() != 4 {
		t.Errorf("ShrinkToFit height = %d", got.Height())
	}

	// RowsByKey groups rows by a key column.
	byKey := d.RowsByKey("grp")
	if len(byKey["a"]) != 2 {
		t.Errorf("RowsByKey[a] = %d rows, want 2", len(byKey["a"]))
	}

	// HashRows returns one hash per row.
	hashes, err := d.HashRows(0)
	if err != nil || len(hashes) != 4 {
		t.Fatalf("HashRows len=%d err=%v", len(hashes), err)
	}

	// Pipe / MapColumns / MapRows.
	piped, err := d.Pipe(func(in DataFrame) (DataFrame, error) { return in, nil })
	if err != nil || piped.Height() != 4 {
		t.Fatalf("Pipe height=%d err=%v", piped.Height(), err)
	}
	mapped, err := d.MapColumns(func(name string, s Series) (Series, error) { return s, nil })
	if err != nil || mapped.Width() != 3 {
		t.Fatalf("MapColumns width=%d err=%v", mapped.Width(), err)
	}
	rows, err := d.MapRows(func(row map[string]any) (map[string]any, error) { return row, nil })
	if err != nil || rows.Height() != 4 {
		t.Fatalf("MapRows height=%d err=%v", rows.Height(), err)
	}

	// SetSorted / Remove / DropInPlace.
	if _, err := d.SetSorted("id"); err != nil {
		t.Errorf("SetSorted: %v", err)
	}
	rem, err := d.Remove("val")
	if err != nil {
		t.Errorf("Remove: %v", err)
	} else if _, ok := rem.Series("val"); ok {
		t.Error("Remove: val should be gone")
	}
	dip, err := d.DropInPlace("grp")
	if err != nil {
		t.Errorf("DropInPlace: %v", err)
	} else if _, ok := dip.Series("grp"); ok {
		t.Error("DropInPlace: grp should be gone")
	}
}

// TestDataFrameMelt covers Melt and its Unpivot alias.
func TestDataFrameMelt(t *testing.T) {
	t.Parallel()

	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2)}},
		{Name: "x", Values: []any{int64(10), int64(20)}},
		{Name: "y", Values: []any{int64(30), int64(40)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}

	input := MeltInput{IDVars: []string{"id"}, ValueVars: []string{"x", "y"}, VariableCol: "variable", ValueCol: "value"}
	melted, err := d.Melt(input)
	if err != nil {
		t.Fatalf("Melt: %v", err)
	}
	// 2 rows * 2 value vars = 4 rows.
	if melted.Height() != 4 {
		t.Errorf("Melt height = %d, want 4", melted.Height())
	}
	unpiv, err := d.Unpivot(input)
	if err != nil || unpiv.Height() != 4 {
		t.Fatalf("Unpivot height=%d err=%v", unpiv.Height(), err)
	}
}

// TestDataFrameWriteMethods covers the file writers via a temp directory.
func TestDataFrameWriteMethods(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)
	dir := t.TempDir()

	if err := d.WriteCsv(WriteCSVInput{Path: filepath.Join(dir, "out.csv"), IncludeHeader: true, Separator: ','}); err != nil {
		t.Errorf("WriteCsv: %v", err)
	}
	if err := d.WriteJson(WriteJSONInput{Path: filepath.Join(dir, "out.json")}); err != nil {
		t.Errorf("WriteJson: %v", err)
	}
}
