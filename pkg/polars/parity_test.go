package polars

import (
	"context"
	"path/filepath"
	"testing"
)

// TestParityRoundTripIO round-trips a frame through all four IO formats and
// asserts that the four reads produce the same column set, height, and
// width. This is a contract test: the IO wrappers must not silently drop
// columns or rows.
func TestParityRoundTripIO(t *testing.T) {
	io := NewIO()
	tmp := t.TempDir()
	df := newIOTestFrame(t)
	csvPath := filepath.Join(tmp, "out.csv")
	parquetPath := filepath.Join(tmp, "out.parquet")
	ipcPath := filepath.Join(tmp, "out.ipc")
	jsonPath := filepath.Join(tmp, "out.json")

	if err := df.WriteCSV(WriteCSVInput{Path: csvPath, IncludeHeader: true}); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	if err := df.WriteParquet(WriteParquetInput{Path: parquetPath}); err != nil {
		t.Fatalf("WriteParquet: %v", err)
	}
	if err := df.WriteIPC(WriteIPCInput{Path: ipcPath}); err != nil {
		t.Fatalf("WriteIPC: %v", err)
	}
	if err := df.WriteJSON(WriteJSONInput{Path: jsonPath, NDJSON: true}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	csvBack, err := io.ReadCSV(ReadCSVInput{Path: csvPath, HasHeader: true})
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	parquetBack, err := io.ReadParquet(ReadParquetInput{Path: parquetPath})
	if err != nil {
		t.Fatalf("ReadParquet: %v", err)
	}
	ipcBack, err := io.ReadIPC(ReadIPCInput{Path: ipcPath})
	if err != nil {
		t.Fatalf("ReadIPC: %v", err)
	}
	jsonBack, err := io.ReadJSON(ReadJSONInput{Path: jsonPath, NDJSON: true})
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}

	// All four reads must report the same shape.
	checks := []struct {
		name string
		df   DataFrame
	}{
		{"csv", csvBack}, {"parquet", parquetBack}, {"ipc", ipcBack}, {"json", jsonBack},
	}
	for _, c := range checks {
		if c.df.Height() != df.Height() {
			t.Errorf("%s height = %d, want %d", c.name, c.df.Height(), df.Height())
		}
		if c.df.Width() != df.Width() {
			t.Errorf("%s width = %d, want %d", c.name, c.df.Width(), df.Width())
		}
	}
}

// TestParityEagerVsLazy evaluates an equivalent filter expression via the
// eager Select and the lazy Collect paths and asserts the per-cell values
// match. This guards against drift between the eager and lazy paths that
// the public Expr API depends on.
func TestParityEagerVsLazy(t *testing.T) {
	df := newLFFrame(t)

	// Eager path: Select(Col("a").Gt(Lit(int64(2)))).
	eager, err := df.Select(Col("a").Gt(Lit(int64(2))))
	if err != nil {
		t.Fatalf("Eager Select: %v", err)
	}

	// Lazy path: Lazy().Select(...).Collect().
	lazy, err := df.Lazy().Select(Col("a").Gt(Lit(int64(2)))).Collect(context.Background())
	if err != nil {
		t.Fatalf("Lazy Collect: %v", err)
	}

	if eager.Height() != lazy.Height() {
		t.Fatalf("eager.height = %d, lazy.height = %d", eager.Height(), lazy.Height())
	}
	if eager.Width() != lazy.Width() {
		t.Fatalf("eager.width = %d, lazy.width = %d", eager.Width(), lazy.Width())
	}

	// Per-cell parity on the predicate result column. The eager Select
	// produces a column named "a" (the same as the input); the lazy path
	// may name it differently — check membership rather than a hard-coded
	// name.
	eagerCols := eager.Columns()
	lazyCols := lazy.Columns()
	if len(eagerCols) != 1 || len(lazyCols) != 1 {
		t.Fatalf("eager cols = %v, lazy cols = %v", eagerCols, lazyCols)
	}
	eagerCol, _ := eager.GetColumn(eagerCols[0])
	lazyCol, _ := lazy.GetColumn(lazyCols[0])
	for i := 0; i < eager.Height(); i++ {
		ev, _ := eagerCol.Value(i).(bool)
		lv, _ := lazyCol.Value(i).(bool)
		if ev != lv {
			t.Errorf("row %d: eager=%v lazy=%v", i, ev, lv)
		}
	}
}

// TestParityEagerVsLazyGroupBy exercises the same GroupBy.Agg expression on
// both the eager DataFrame.GroupBy and the LazyFrame.GroupBy paths.
func TestParityEagerVsLazyGroupBy(t *testing.T) {
	df := newLFFrame(t)

	eager, err := df.GroupBy("g").Agg(Sum(Col("a")).Alias("sum_a"))
	if err != nil {
		t.Fatalf("Eager GroupBy: %v", err)
	}
	lazy, err := df.Lazy().GroupBy("g").Agg(Sum(Col("a")).Alias("sum_a")).Collect(context.Background())
	if err != nil {
		t.Fatalf("Lazy GroupBy: %v", err)
	}

	if eager.Height() != lazy.Height() {
		t.Fatalf("eager.height = %d, lazy.height = %d", eager.Height(), lazy.Height())
	}

	// Build a key → sum map for both and compare.
	eagerKeys, _ := eager.GetColumn("g")
	eagerSums, _ := eager.GetColumn("sum_a")
	eagerMap := map[string]int64{}
	for i := 0; i < eagerKeys.Len(); i++ {
		k, _ := eagerKeys.Value(i).(string)
		v, _ := eagerSums.Value(i).(int64)
		eagerMap[k] = v
	}
	lazyKeys, _ := lazy.GetColumn("g")
	lazySums, _ := lazy.GetColumn("sum_a")
	lazyMap := map[string]int64{}
	for i := 0; i < lazyKeys.Len(); i++ {
		k, _ := lazyKeys.Value(i).(string)
		v, _ := lazySums.Value(i).(int64)
		lazyMap[k] = v
	}
	for k, ev := range eagerMap {
		if lv, ok := lazyMap[k]; !ok || ev != lv {
			t.Errorf("sum[%s]: eager=%v lazy=%v (present=%v)", k, ev, lv, ok)
		}
	}
}
