package polars

import (
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// newDFTestFrame builds a 4-row, 3-column frame used by DataFrame tests.
func newDFTestFrame(t *testing.T) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		{Name: "b", Values: []any{"w", "x", "y", "z"}},
		{Name: "g", Values: []any{"p", "p", "q", "q"}},
	}})
	if err != nil {
		t.Fatalf("newDFTestFrame: %v", err)
	}
	return df
}

// TestDataFrameAccessors pins the documented schema accessors.
func TestDataFrameAccessors(t *testing.T) {
	df := newDFTestFrame(t)
	if df.Height() != 4 {
		t.Errorf("Height = %d, want 4", df.Height())
	}
	if df.Width() != 3 {
		t.Errorf("Width = %d, want 3", df.Width())
	}
	if df.IsEmpty() {
		t.Errorf("IsEmpty = true, want false (4 rows)")
	}
	cols := df.Columns()
	if len(cols) != 3 || cols[0] != "a" || cols[1] != "b" || cols[2] != "g" {
		t.Errorf("Columns = %v, want [a b g]", cols)
	}
	if len(df.Dtypes()) != 3 {
		t.Errorf("Dtypes len = %d, want 3", len(df.Dtypes()))
	}
	if len(df.Schema()) != 3 {
		t.Errorf("Schema len = %d, want 3", len(df.Schema()))
	}
	// GetColumn happy + missing path.
	if _, err := df.GetColumn("a"); err != nil {
		t.Errorf("GetColumn(a): %v", err)
	}
	if _, err := df.GetColumn("missing"); err == nil {
		t.Errorf("GetColumn(missing) returned nil error, want non-nil")
	}
	// GetColumns returns the full slice.
	all := df.GetColumns()
	if len(all) != 3 {
		t.Errorf("GetColumns len = %d, want 3", len(all))
	}
	// GetColumnIndex.
	if i := df.GetColumnIndex("a"); i != 0 {
		t.Errorf("GetColumnIndex(a) = %d, want 0", i)
	}
	// Series accessor (Series, bool) form.
	if _, ok := df.Series("a"); !ok {
		t.Errorf("Series(a) returned ok=false, want true")
	}
	if _, ok := df.Series("missing"); ok {
		t.Errorf("Series(missing) returned ok=true, want false")
	}
}

// TestDataFrameHeadTail exercises the row-prefix/suffix methods.
func TestDataFrameHeadTail(t *testing.T) {
	df := newDFTestFrame(t)
	if h := df.Head(2); h.Height() != 2 {
		t.Errorf("Head(2) height = %d, want 2", h.Height())
	}
	if tail := df.Tail(2); tail.Height() != 2 {
		t.Errorf("Tail(2) height = %d, want 2", tail.Height())
	}
	if s := df.Slice(1, 2); s.Height() != 2 {
		t.Errorf("Slice(1,2) height = %d, want 2", s.Height())
	}
	if l := df.Limit(3); l.Height() != 3 {
		t.Errorf("Limit(3) height = %d, want 3", l.Height())
	}
}

// TestDataFrameSelectWithColumnsDropRename exercises column projection.
func TestDataFrameSelectWithColumnsDropRename(t *testing.T) {
	df := newDFTestFrame(t)

	// Select
	out, err := df.Select(Col("a"))
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if out.Width() != 1 || out.Columns()[0] != "a" {
		t.Errorf("Select = %v, want [a]", out.Columns())
	}

	// WithColumns
	out, err = df.WithColumns(Lit(int64(0)).Alias("z"))
	if err != nil {
		t.Fatalf("WithColumns: %v", err)
	}
	if out.Width() != 4 {
		t.Errorf("WithColumns width = %d, want 4", out.Width())
	}

	// Drop
	out, err = df.Drop("g")
	if err != nil {
		t.Fatalf("Drop: %v", err)
	}
	if out.Width() != 2 {
		t.Errorf("Drop width = %d, want 2", out.Width())
	}

	// Rename
	out, err = df.Rename(map[string]string{"a": "alpha"})
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if out.Columns()[0] != "alpha" {
		t.Errorf("Rename columns[0] = %q, want alpha", out.Columns()[0])
	}

	// Cast
	out, err = df.Cast(map[string]dtypes.DataType{"a": dtypes.Float64})
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	casted := false
	for _, f := range out.Schema() {
		if f.Name == "a" && f.Type == dtypes.Float64 {
			casted = true
		}
	}
	if !casted {
		t.Errorf("Cast(a) did not produce Float64, schema=%v", out.Schema())
	}
}

// TestDataFrameFilterSort exercises row filtering and sorting.
func TestDataFrameFilterSort(t *testing.T) {
	df := newDFTestFrame(t)
	out, err := df.Filter(Col("a").Gt(Lit(int64(2))))
	if err != nil {
		t.Fatalf("Filter: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("Filter height = %d, want 2 (a > 2)", out.Height())
	}

	out, err = df.Sort(SortInput{By: []string{"a"}, Descending: []bool{true}})
	if err != nil {
		t.Fatalf("Sort: %v", err)
	}
	col, _ := out.GetColumn("a")
	if v, _ := col.Value(0).(int64); v != 4 {
		t.Errorf("Sort[0] = %v, want 4 (descending)", col.Value(0))
	}
}

// TestDataFrameGroupBy exercises the GroupBy + Agg path.
func TestDataFrameGroupBy(t *testing.T) {
	df := newDFTestFrame(t)
	out, err := df.GroupBy("g").Agg(Sum(Col("a")).Alias("sum_a"))
	if err != nil {
		t.Fatalf("GroupBy.Agg: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("GroupBy height = %d, want 2 (groups p, q)", out.Height())
	}
	keys, _ := out.GetColumn("g")
	sums, _ := out.GetColumn("sum_a")
	idxByKey := map[string]int{}
	for i := 0; i < keys.Len(); i++ {
		k, _ := keys.Value(i).(string)
		idxByKey[k] = i
	}
	if v, _ := sums.Value(idxByKey["p"]).(int64); v != 3 {
		t.Errorf("sum_a(p) = %d, want 3 (1+2)", v)
	}
	if v, _ := sums.Value(idxByKey["q"]).(int64); v != 7 {
		t.Errorf("sum_a(q) = %d, want 7 (3+4)", v)
	}
}

// TestDataFrameJoin exercises the inner join path.
func TestDataFrameJoin(t *testing.T) {
	left, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(2), int64(3)}},
	}})
	right, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(2), int64(3), int64(4)}},
	}})
	out, err := left.Join(JoinInput{
		Other:   right,
		LeftOn:  []string{"k"},
		RightOn: []string{"k"},
		How:     JoinTypeInner,
	})
	if err != nil {
		t.Fatalf("Join: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("inner join height = %d, want 2 (k=2, k=3)", out.Height())
	}
}

// TestDataFrameNullCountAndFill exercises the null/replace family.
func TestDataFrameNullCountAndFill(t *testing.T) {
	df, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), nil, int64(3), nil}},
	}})
	nc := df.NullCount()
	if nc["a"] != 2 {
		t.Errorf("NullCount[a] = %d, want 2", nc["a"])
	}
	filled, err := df.FillNull(int64(0))
	if err != nil {
		t.Fatalf("FillNull: %v", err)
	}
	nc2 := filled.NullCount()
	if nc2["a"] != 0 {
		t.Errorf("FillNull NullCount[a] = %d, want 0", nc2["a"])
	}
	dropped := df.DropNulls("a")
	if dropped.Height() != 2 {
		t.Errorf("DropNulls height = %d, want 2", dropped.Height())
	}
}

// TestDataFrameEquals verifies the documented equality contract.
func TestDataFrameEquals(t *testing.T) {
	a := newDFTestFrame(t)
	b := newDFTestFrame(t)
	eq, err := a.Equals(b)
	if err != nil {
		t.Fatalf("Equals: %v", err)
	}
	if !eq {
		t.Errorf("a.Equals(b) = false, want true (same data)")
	}
}

// TestDataFrameClone verifies the documented clone semantics.
func TestDataFrameClone(t *testing.T) {
	a := newDFTestFrame(t)
	b := a.Clone()
	if b == nil {
		t.Fatalf("Clone returned nil")
	}
	if b.Height() != a.Height() || b.Width() != a.Width() {
		t.Errorf("Clone shape = (%d, %d), want (%d, %d)", b.Height(), b.Width(), a.Height(), a.Width())
	}
}

// TestDataFrameLazy exercises the Lazy() path.
func TestDataFrameLazy(t *testing.T) {
	df := newDFTestFrame(t)
	if lf := df.Lazy(); lf == nil {
		t.Errorf("Lazy() returned nil")
	}
}

// TestDataFrameToArrow converts to an arrow table and back.
func TestDataFrameToArrow(t *testing.T) {
	df := newDFTestFrame(t)
	tbl, err := df.ToArrow(ToArrowInput{})
	if err != nil {
		t.Fatalf("ToArrow: %v", err)
	}
	if len(tbl.Columns) != 3 {
		t.Errorf("arrow Columns len = %d, want 3", len(tbl.Columns))
	}
	back, err := NewDataFrameFromArrow(tbl)
	if err != nil {
		t.Fatalf("NewDataFrameFromArrow: %v", err)
	}
	if back.Height() != df.Height() {
		t.Errorf("round-trip height = %d, want %d", back.Height(), df.Height())
	}
}

// TestDataFrameWriteCSVReadCSV covers a write-then-read round-trip.
func TestDataFrameWriteCSVReadCSV(t *testing.T) {
	df := newDFTestFrame(t)
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.csv")
	if err := df.WriteCSV(WriteCSVInput{Path: path, IncludeHeader: true}); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	io := NewIO()
	back, err := io.ReadCSV(ReadCSVInput{Path: path, HasHeader: true})
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	if back.Height() != df.Height() {
		t.Errorf("round-trip height = %d, want %d", back.Height(), df.Height())
	}
}

// TestDataFrameIterColumns returns a slice of Series.
func TestDataFrameIterColumns(t *testing.T) {
	df := newDFTestFrame(t)
	cols := df.IterColumns()
	if len(cols) != 3 {
		t.Errorf("IterColumns len = %d, want 3", len(cols))
	}
}

// TestDataFrameItem returns a single cell value.
func TestDataFrameItem(t *testing.T) {
	df := newDFTestFrame(t)
	v, err := df.Item(0, "a")
	if err != nil {
		t.Fatalf("Item: %v", err)
	}
	if got, _ := v.(int64); got != 1 {
		t.Errorf("Item(0, a) = %v, want 1", v)
	}
	// Out-of-bounds row.
	if _, err := df.Item(100, "a"); err == nil {
		t.Errorf("Item(100, a) returned nil error, want non-nil")
	}
}

// TestDataFrameGlimpse exercises the printable representation.
func TestDataFrameGlimpse(t *testing.T) {
	df := newDFTestFrame(t)
	g := df.Glimpse(2)
	if g == "" {
		t.Errorf("Glimpse returned empty string")
	}
}

// TestDataFrameEstimatedSize verifies the documented size estimate.
func TestDataFrameEstimatedSize(t *testing.T) {
	df := newDFTestFrame(t)
	if v := df.EstimatedSize(); v <= 0 {
		t.Errorf("EstimatedSize = %d, want > 0", v)
	}
}
