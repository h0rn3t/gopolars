package polars

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestDataFrameIterators exercises IterRows, IterSlices, ToDicts, SubSelectColumns.
func TestDataFrameIterators(t *testing.T) {
	df := newDFTestFrame(t)

	rows := df.IterRows()
	if len(rows) != 4 {
		t.Errorf("IterRows len = %d, want 4", len(rows))
	}
	// IterSlices splits into chunks of `size`.
	if slices := df.IterSlices(2); len(slices) == 0 {
		t.Errorf("IterSlices(2) returned no slices")
	}

	dicts := df.ToDicts()
	if len(dicts) != 4 {
		t.Errorf("ToDicts len = %d, want 4", len(dicts))
	}
	// Each row dict has the column names as keys.
	if v, ok := dicts[0]["a"]; !ok {
		t.Errorf("dicts[0] missing key 'a'")
	} else if v, _ := v.(int64); v != 1 {
		t.Errorf("dicts[0]['a'] = %v, want 1", v)
	}

	// SubSelectColumns projects to a subset.
	out, err := df.SubSelectColumns("a", "b")
	if err != nil {
		t.Fatalf("SubSelectColumns: %v", err)
	}
	if out.Width() != 2 {
		t.Errorf("SubSelectColumns width = %d, want 2", out.Width())
	}
	// SubSelectColumns with a missing column returns an error.
	if _, err := df.SubSelectColumns("a", "missing"); err == nil {
		t.Errorf("SubSelectColumns with missing col returned nil error, want non-nil")
	}
}

// TestDataFrameUniqueAndDuplicated exercises the boolean mask family.
func TestDataFrameUniqueAndDuplicated(t *testing.T) {
	df, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(1), int64(3)}},
	}})
	dup := df.IsDuplicated()
	if dup.Len() != 4 {
		t.Errorf("IsDuplicated len = %d, want 4", dup.Len())
	}
	uniq := df.IsUnique()
	if uniq.Len() != 4 {
		t.Errorf("IsUnique len = %d, want 4", uniq.Len())
	}
}

// TestDataFrameCountNUniqueApproxNUnique exercises the typed reductions.
func TestDataFrameCountNUniqueApproxNUnique(t *testing.T) {
	df, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(1), int64(3)}},
	}})
	// Count returns map[name]non_null_count.
	c := df.Count()
	if v := c["a"]; v != 4 {
		t.Errorf("Count[a] = %d, want 4 (no nulls)", v)
	}
	// NUnique returns total distinct values across the given columns.
	if n, err := df.NUnique("a"); err != nil {
		t.Errorf("NUnique: %v", err)
	} else if n != 3 {
		t.Errorf("NUnique(a) = %d, want 3 (values 1,2,3)", n)
	}
	// NUnique on missing column returns an error.
	if _, err := df.NUnique("missing"); err == nil {
		t.Errorf("NUnique(missing) returned nil error, want non-nil")
	}
	// ApproxNUnique delegates.
	if n, err := df.ApproxNUnique("a"); err != nil {
		t.Errorf("ApproxNUnique: %v", err)
	} else if n != 3 {
		t.Errorf("ApproxNUnique(a) = %d, want 3", n)
	}
}

// TestDataFrameFlags exercises the flags map.
func TestDataFrameFlags(t *testing.T) {
	df := newDFTestFrame(t)
	flags := df.Flags()
	if flags == nil {
		t.Errorf("Flags returned nil")
	}
}

// TestDataFrameWithRowCount and WithRowIndex add an index column.
func TestDataFrameWithRowCountAndIndex(t *testing.T) {
	df := newDFTestFrame(t)
	out, err := df.WithRowCount("rc", 0)
	if err != nil {
		t.Fatalf("WithRowCount: %v", err)
	}
	if out.Width() != 4 {
		t.Errorf("WithRowCount width = %d, want 4", out.Width())
	}

	out, err = df.WithRowIndex("ri", 1)
	if err != nil {
		t.Fatalf("WithRowIndex: %v", err)
	}
	if out.Width() != 4 {
		t.Errorf("WithRowIndex width = %d, want 4", out.Width())
	}
	col, _ := out.GetColumn("ri")
	if v, _ := col.Value(0).(int64); v != 1 {
		t.Errorf("WithRowIndex offset=1: row 0 = %v, want 1", col.Value(0))
	}
}

// TestDataFrameConcat concatenates two frames vertically.
func TestDataFrameConcat(t *testing.T) {
	a := newDFTestFrame(t)
	b := newDFTestFrame(t)
	out, err := a.Concat(ConcatInput{Others: []DataFrame{b}, How: "vertical"})
	if err != nil {
		t.Fatalf("Concat: %v", err)
	}
	if out.Height() != 8 {
		t.Errorf("Concat height = %d, want 8 (4+4)", out.Height())
	}
}

// TestDataFrameSample draws a random subset.
func TestDataFrameSample(t *testing.T) {
	df := newDFTestFrame(t)
	out := df.Sample(2, 42)
	if out.Height() != 2 {
		t.Errorf("Sample(2) height = %d, want 2", out.Height())
	}
}

// TestDataFrameReverse and Rechunk are simple pass-throughs.
func TestDataFrameReverseAndRechunk(t *testing.T) {
	df := newDFTestFrame(t)
	if out := df.Reverse(); out.Height() != 4 {
		t.Errorf("Reverse height = %d, want 4", out.Height())
	}
	if out := df.Rechunk(); out.Height() != 4 {
		t.Errorf("Rechunk height = %d, want 4", out.Height())
	}
}

// TestDataFrameGatherEvery steps through the rows.
func TestDataFrameGatherEvery(t *testing.T) {
	df := newDFTestFrame(t)
	out := df.GatherEvery(2, 0)
	if out.Height() != 2 {
		t.Errorf("GatherEvery(2, 0) height = %d, want 2 (4/2)", out.Height())
	}
}

// TestDataFrameClear is a pass-through in the current implementation that
// returns a frame; we just assert it doesn't panic and is non-nil.
func TestDataFrameClear(t *testing.T) {
	df := newDFTestFrame(t)
	out := df.Clear()
	if out == nil {
		t.Errorf("Clear returned nil")
	}
}

// TestDataFrameDistinct returns the unique rows.
func TestDataFrameDistinct(t *testing.T) {
	df, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(1), int64(2), int64(2)}},
	}})
	// Unique with no subset distincts on all columns.
	out, err := df.Unique()
	if err != nil {
		t.Fatalf("Unique: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("Unique height = %d, want 2", out.Height())
	}
	// Unique with a subset.
	out, err = df.Unique("a")
	if err != nil {
		t.Fatalf("Unique(a): %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("Unique(a) height = %d, want 2", out.Height())
	}
}

// TestDataFrameToSeries converts a single column to a Series.
func TestDataFrameToSeries(t *testing.T) {
	df := newDFTestFrame(t)
	s, err := df.ToSeries("a")
	if err != nil {
		t.Fatalf("ToSeries: %v", err)
	}
	if s.Len() != 4 {
		t.Errorf("ToSeries len = %d, want 4", s.Len())
	}
}

// TestDataFrameRow returns a single row as a map.
func TestDataFrameRow(t *testing.T) {
	df := newDFTestFrame(t)
	row, err := df.Row(0)
	if err != nil {
		t.Fatalf("Row(0): %v", err)
	}
	if row["a"] == nil {
		t.Errorf("Row(0)['a'] is nil")
	}
	if row, err := df.Row(100); err == nil {
		t.Errorf("Row(100) returned nil error, want non-nil; row=%v", row)
	}
}

// TestDataFrameRows returns all rows as a slice of maps.
func TestDataFrameRows(t *testing.T) {
	df := newDFTestFrame(t)
	rows := df.Rows()
	if len(rows) != 4 {
		t.Errorf("Rows len = %d, want 4", len(rows))
	}
}

// TestDataFrameShape returns [height, width].
func TestDataFrameShape(t *testing.T) {
	df := newDFTestFrame(t)
	shape := df.Shape()
	if shape[0] != 4 || shape[1] != 3 {
		t.Errorf("Shape = %v, want [4 3]", shape)
	}
}

// TestDataFrameShift shifts rows down by `periods` positions.
func TestDataFrameShift(t *testing.T) {
	df := newDFTestFrame(t)
	out, err := df.Shift(1)
	if err != nil {
		t.Fatalf("Shift: %v", err)
	}
	if out.Height() != 4 {
		t.Errorf("Shift height = %d, want 4", out.Height())
	}
}

// TestDataFrameSumMaxMinMean exercises the per-column reductions.
func TestDataFrameReductions(t *testing.T) {
	df := newDFTestFrame(t)
	sum := df.Sum()
	if v, ok := sum["a"]; !ok {
		t.Errorf("Sum has no key 'a'")
	} else if v != 10 {
		t.Errorf("Sum[a] = %v, want 10", v)
	}
	max := df.Max()
	if v, ok := max["a"]; !ok {
		t.Errorf("Max has no key 'a'")
	} else if v, _ := v.(int64); v != 4 {
		t.Errorf("Max[a] = %v, want 4", v)
	}
	min := df.Min()
	if v, ok := min["a"]; !ok {
		t.Errorf("Min has no key 'a'")
	} else if v, _ := v.(int64); v != 1 {
		t.Errorf("Min[a] = %v, want 1", v)
	}
	mean := df.Mean()
	if v, ok := mean["a"]; !ok {
		t.Errorf("Mean has no key 'a'")
	} else if v != 2.5 {
		t.Errorf("Mean[a] = %v, want 2.5", v)
	}
}

// TestDataFrameDropColumn drops a single column by name.
func TestDataFrameDropColumn(t *testing.T) {
	df := newDFTestFrame(t)
	out, err := df.Drop("g")
	if err != nil {
		t.Fatalf("Drop: %v", err)
	}
	if out.Width() != 2 {
		t.Errorf("Drop width = %d, want 2", out.Width())
	}
}
