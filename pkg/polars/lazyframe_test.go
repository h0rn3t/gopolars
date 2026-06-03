package polars

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// newLFFrame builds a 4-row, 3-column frame used by LazyFrame tests.
func newLFFrame(t *testing.T) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		{Name: "b", Values: []any{1.0, 2.0, 3.0, 4.0}},
		{Name: "g", Values: []any{"x", "x", "y", "y"}},
	}})
	if err != nil {
		t.Fatalf("newLFFrame: %v", err)
	}
	return df
}

// TestLazyAccessors pins the documented schema accessors.
func TestLazyAccessors(t *testing.T) {
	df := newLFFrame(t)
	lf := df.Lazy()
	if lf == nil {
		t.Fatalf("Lazy() returned nil")
	}
	cols := lf.Columns()
	if len(cols) != 3 {
		t.Errorf("Columns len = %d, want 3", len(cols))
	}
	if w := lf.Width(); w != 3 {
		t.Errorf("Width = %d, want 3", w)
	}
	sch := lf.Schema()
	if len(sch) != 3 {
		t.Errorf("Schema len = %d, want 3", len(sch))
	}
	dts := lf.Dtypes()
	if len(dts) != 3 {
		t.Errorf("Dtypes len = %d, want 3", len(dts))
	}
	cs := lf.CollectSchema()
	if len(cs) != 3 {
		t.Errorf("CollectSchema len = %d, want 3", len(cs))
	}
}

// TestLazySelectCollect exercises Select → Collect.
func TestLazySelectCollect(t *testing.T) {
	df := newLFFrame(t)
	out, err := df.Lazy().Select(Col("a")).Collect(context.Background())
	if err != nil {
		t.Fatalf("Select Collect: %v", err)
	}
	if out.Width() != 1 {
		t.Errorf("width = %d, want 1", out.Width())
	}
}

// TestLazyFilterCollect exercises Filter → Collect.
func TestLazyFilterCollect(t *testing.T) {
	df := newLFFrame(t)
	out, err := df.Lazy().Filter(Col("a").Gt(Lit(int64(2)))).Collect(context.Background())
	if err != nil {
		t.Fatalf("Filter Collect: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("filtered height = %d, want 2", out.Height())
	}
}

// TestLazyWithColumnsCollect exercises WithColumns → Collect.
func TestLazyWithColumnsCollect(t *testing.T) {
	df := newLFFrame(t)
	out, err := df.Lazy().WithColumns(Lit(int64(0)).Alias("z")).Collect(context.Background())
	if err != nil {
		t.Fatalf("WithColumns Collect: %v", err)
	}
	if out.Width() != 4 {
		t.Errorf("width = %d, want 4", out.Width())
	}
}

// TestLazyGroupByAgg exercises GroupBy(...).Agg(...) on a LazyFrame.
func TestLazyGroupByAgg(t *testing.T) {
	df := newLFFrame(t)
	out, err := df.Lazy().GroupBy("g").Agg(Sum(Col("a")).Alias("sum_a")).Collect(context.Background())
	if err != nil {
		t.Fatalf("GroupBy Agg Collect: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("height = %d, want 2 (groups x, y)", out.Height())
	}
}

// TestLazyJoinCollect exercises Join → Collect (inner).
func TestLazyJoinCollect(t *testing.T) {
	left, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "lv", Values: []any{10.0, 20.0, 30.0}},
	}})
	if err != nil {
		t.Fatalf("left: %v", err)
	}
	right, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(2), int64(3), int64(4)}},
		{Name: "rv", Values: []any{200.0, 300.0, 400.0}},
	}})
	if err != nil {
		t.Fatalf("right: %v", err)
	}
	out, err := left.Lazy().Join(JoinInput{
		Other:   right,
		LeftOn:  []string{"k"},
		RightOn: []string{"k"},
		How:     JoinTypeInner,
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("Lazy Join Collect: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("inner join height = %d, want 2 (k=2, k=3)", out.Height())
	}
}

// TestLazySortCollect exercises Sort → Collect.
func TestLazySortCollect(t *testing.T) {
	df := newLFFrame(t)
	out, err := df.Lazy().Sort(SortInput{By: []string{"a"}, Descending: []bool{true}}).Collect(context.Background())
	if err != nil {
		t.Fatalf("Lazy Sort Collect: %v", err)
	}
	col, _ := out.GetColumn("a")
	if v, _ := col.Value(0).(int64); v != 4 {
		t.Errorf("sort[0] = %v, want 4 (descending)", col.Value(0))
	}
}

// TestLazyLimitCollect exercises Limit → Collect.
func TestLazyLimitCollect(t *testing.T) {
	df := newLFFrame(t)
	out, err := df.Lazy().Limit(2).Collect(context.Background())
	if err != nil {
		t.Fatalf("Lazy Limit Collect: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("limit height = %d, want 2", out.Height())
	}
}

// TestLazySliceCollect exercises Slice → Collect.
func TestLazySliceCollect(t *testing.T) {
	df := newLFFrame(t)
	out, err := df.Lazy().Slice(1, 2).Collect(context.Background())
	if err != nil {
		t.Fatalf("Lazy Slice Collect: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("slice height = %d, want 2", out.Height())
	}
}

// TestLazyCollectMissingColumnReturnsError exercises the failure path.
func TestLazyCollectMissingColumnReturnsError(t *testing.T) {
	df := newLFFrame(t)
	_, err := df.Lazy().Select(Col("missing")).Collect(context.Background())
	if err == nil {
		t.Fatalf("Select on missing column returned nil error, want non-nil")
	}
}

// TestLazyCollectAsync verifies CollectAsync returns a channel of results.
func TestLazyCollectAsync(t *testing.T) {
	df := newLFFrame(t)
	ch := df.Lazy().CollectAsync(context.Background())
	got := <-ch
	if got.Error != nil {
		t.Fatalf("CollectAsync error: %v", got.Error)
	}
	if got.DataFrame == nil {
		t.Fatalf("CollectAsync result has nil DataFrame")
	}
	if got.DataFrame.Height() != 4 {
		t.Errorf("async result height = %d, want 4", got.DataFrame.Height())
	}
}

// TestLazySinkParquet exercises the sink path.
func TestLazySinkParquet(t *testing.T) {
	df := newLFFrame(t)
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.parquet")
	if err := df.Lazy().SinkParquet(context.Background(), WriteParquetInput{Path: path}); err != nil {
		t.Fatalf("SinkParquet: %v", err)
	}
	// Read it back via IO.
	io := NewIO()
	got, err := io.ReadParquet(ReadParquetInput{Path: path})
	if err != nil {
		t.Fatalf("ReadParquet after sink: %v", err)
	}
	if got.Height() != df.Height() {
		t.Errorf("sink→read height = %d, want %d", got.Height(), df.Height())
	}
}

// TestLazySinkToUnwritablePathReturnsError exercises the sink error path.
func TestLazySinkToUnwritablePathReturnsError(t *testing.T) {
	df := newLFFrame(t)
	err := df.Lazy().SinkParquet(context.Background(), WriteParquetInput{Path: "/this/does/not/exist/out.parquet"})
	if err == nil {
		t.Fatalf("SinkParquet to unwritable path returned nil error, want non-nil")
	}
}

// TestParseSQLRequiresTable verifies the table argument is honored.
func TestParseSQLRequiresTable(t *testing.T) {
	df := newLFFrame(t)
	src := unwrapFrame(t, df)
	// Empty query → parse error.
	if _, err := ParseSQL(ParseSQLInput{Query: "", Source: src}); err == nil {
		t.Fatalf("ParseSQL with empty query returned nil error, want non-nil")
	}
	// Missing table is also an error (parsed.Table == "" → "table is required").
	lf, err := ParseSQL(ParseSQLInput{
		Query:  "SELECT 1 AS x",
		Source: src,
		Table:  "",
	})
	if err == nil && lf == nil {
		t.Fatalf("ParseSQL with no table returned nil LF and no err, want one or the other")
	}
}

// TestSQLFromDataFrameFromEagerDF exercises the SQLFromDataFrame path.
func TestSQLFromDataFrameFromEagerDF(t *testing.T) {
	df := newLFFrame(t)
	lf, err := SQLFromDataFrame(context.Background(), df, "SELECT * FROM t", "t")
	if err != nil {
		t.Fatalf("SQLFromDataFrame: %v", err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if out.Height() != df.Height() {
		t.Errorf("result height = %d, want %d", out.Height(), df.Height())
	}
}

// unwrapFrame returns the underlying frame.DataFrame that powers a *df
// wrapper. Used by ParseSQL tests that need the internal type.
func unwrapFrame(t *testing.T, in DataFrame) frame.DataFrame {
	t.Helper()
	d, ok := in.(*df)
	if !ok {
		t.Fatalf("DataFrame is not *df")
	}
	return d.value
}

// avoid "imported and not used" if dtypes becomes unused.
var _ = dtypes.Int64
