package dataframe

// Ported from py-polars/tests/unit/dataframe/test_df.py (py-1.28.1)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func helperDF() polars.DataFrame {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "b", Values: []any{float64(4), float64(5), float64(6)}},
			{Name: "c", Values: []any{"x", "y", "z"}},
		},
	})
	if err != nil {
		panic(err)
	}
	return df
}

func TestDFConstruction(t *testing.T) {
	t.Parallel()
	df := helperDF()
	if df.Height() != 3 {
		t.Fatalf("height: got %d, want 3", df.Height())
	}
	if df.Width() != 3 {
		t.Fatalf("width: got %d, want 3", df.Width())
	}
	cols := df.Columns()
	if len(cols) != 3 || cols[0] != "a" || cols[1] != "b" || cols[2] != "c" {
		t.Fatalf("columns: got %v, want [a b c]", cols)
	}
}

func TestDFEmpty(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{}})
	if err != nil {
		t.Fatalf("empty df creation: %v", err)
	}
	if !df.IsEmpty() {
		t.Fatalf("empty df should be empty")
	}
	if df.Height() != 0 {
		t.Fatalf("empty df height: got %d, want 0", df.Height())
	}
	if df.Width() != 0 {
		t.Fatalf("empty df width: got %d, want 0", df.Width())
	}
}

func TestDFSchema(t *testing.T) {
	t.Parallel()
	df := helperDF()
	schema := df.Schema()
	if len(schema) != 3 {
		t.Fatalf("schema length: got %d, want 3", len(schema))
	}
	foundA, foundB, foundC := false, false, false
	for _, f := range schema {
		if f.Name == "a" && f.Type == polars.Int64 {
			foundA = true
		}
		if f.Name == "b" && f.Type == polars.Float64 {
			foundB = true
		}
		if f.Name == "c" && f.Type == polars.String {
			foundC = true
		}
	}
	if !foundA {
		t.Fatalf("schema missing field a with Int64")
	}
	if !foundB {
		t.Fatalf("schema missing field b with Float64")
	}
	if !foundC {
		t.Fatalf("schema missing field c with String")
	}
}

func TestDFGetColumn(t *testing.T) {
	t.Parallel()
	df := helperDF()
	s, err := df.GetColumn("a")
	if err != nil {
		t.Fatalf("get column a: %v", err)
	}
	if s.Len() != 3 {
		t.Fatalf("column a length: got %d, want 3", s.Len())
	}
	if s.Name() != "a" {
		t.Fatalf("column a name: got %s, want a", s.Name())
	}
}

func TestDFGetColumnIndex(t *testing.T) {
	t.Parallel()
	df := helperDF()
	if idx := df.GetColumnIndex("a"); idx != 0 {
		t.Fatalf("column a index: got %d, want 0", idx)
	}
	if idx := df.GetColumnIndex("b"); idx != 1 {
		t.Fatalf("column b index: got %d, want 1", idx)
	}
	if idx := df.GetColumnIndex("c"); idx != 2 {
		t.Fatalf("column c index: got %d, want 2", idx)
	}
}

func TestDFGetColumns(t *testing.T) {
	t.Parallel()
	df := helperDF()
	cols := df.GetColumns()
	if len(cols) != 3 {
		t.Fatalf("columns length: got %d, want 3", len(cols))
	}
	if cols[0].Name() != "a" || cols[1].Name() != "b" || cols[2].Name() != "c" {
		t.Fatalf("column names: got %v", []string{cols[0].Name(), cols[1].Name(), cols[2].Name()})
	}
}

func TestDFDtypes(t *testing.T) {
	t.Parallel()
	df := helperDF()
	dtypes := df.Dtypes()
	if len(dtypes) != 3 {
		t.Fatalf("dtypes length: got %d, want 3", len(dtypes))
	}
	if dtypes[0] != polars.Int64 {
		t.Fatalf("dtype[0]: got %v, want Int64", dtypes[0])
	}
	if dtypes[1] != polars.Float64 {
		t.Fatalf("dtype[1]: got %v, want Float64", dtypes[1])
	}
	if dtypes[2] != polars.String {
		t.Fatalf("dtype[2]: got %v, want String", dtypes[2])
	}
}

func TestDFSelect(t *testing.T) {
	t.Parallel()
	df := helperDF()
	result, err := df.Select(polars.Col("a"), polars.Col("c"))
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if result.Width() != 2 {
		t.Fatalf("select width: got %d, want 2", result.Width())
	}
	cols := result.Columns()
	if len(cols) != 2 || cols[0] != "a" || cols[1] != "c" {
		t.Fatalf("select columns: got %v", cols)
	}
}

func TestDFFilter(t *testing.T) {
	t.Parallel()
	df := helperDF()
	result, err := df.Filter(polars.Col("a").Gt(polars.Lit(int64(1))))
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if result.Height() != 2 {
		t.Fatalf("filter height: got %d, want 2", result.Height())
	}
}

func TestDFSort(t *testing.T) {
	t.Parallel()
	df := helperDF()
	sorted, err := df.Sort(polars.SortInput{By: []string{"a"}, Descending: []bool{true}})
	if err != nil {
		t.Fatalf("sort: %v", err)
	}
	s, _ := sorted.GetColumn("a")
	if v, ok := s.Value(0).(int64); !ok || v != 3 {
		t.Fatalf("sort[0]: got %v, want 3", s.Value(0))
	}
}

func TestDFSlice(t *testing.T) {
	t.Parallel()
	df := helperDF()
	sliced := df.Slice(1, 2)
	if sliced.Height() != 2 {
		t.Fatalf("slice height: got %d, want 2", sliced.Height())
	}
}

func TestDFHeadTail(t *testing.T) {
	t.Parallel()
	df := helperDF()
	head := df.Head(2)
	if head.Height() != 2 {
		t.Fatalf("head height: got %d, want 2", head.Height())
	}
	tail := df.Tail(1)
	if tail.Height() != 1 {
		t.Fatalf("tail height: got %d, want 1", tail.Height())
	}
}

func TestDFLimit(t *testing.T) {
	t.Parallel()
	df := helperDF()
	limited := df.Limit(2)
	if limited.Height() != 2 {
		t.Fatalf("limit height: got %d, want 2", limited.Height())
	}
}

func TestDFDrop(t *testing.T) {
	t.Parallel()
	df := helperDF()
	dropped, err := df.Drop("b")
	if err != nil {
		t.Fatalf("drop: %v", err)
	}
	if dropped.Width() != 2 {
		t.Fatalf("drop width: got %d, want 2", dropped.Width())
	}
	cols := dropped.Columns()
	if len(cols) != 2 || cols[0] != "a" || cols[1] != "c" {
		t.Fatalf("drop columns: got %v", cols)
	}
}

func TestDFRename(t *testing.T) {
	t.Parallel()
	df := helperDF()
	renamed, err := df.Rename(map[string]string{"a": "x"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	cols := renamed.Columns()
	if len(cols) != 3 || cols[0] != "x" {
		t.Fatalf("rename columns: got %v", cols)
	}
}

func TestDFClone(t *testing.T) {
	t.Parallel()
	df := helperDF()
	cloned := df.Clone()
	if cloned.Height() != df.Height() {
		t.Fatalf("clone height mismatch")
	}
}

func TestDFClear(t *testing.T) {
	t.Parallel()
	df := helperDF()
	cleared := df.Clear()
	if cleared.Height() != 0 {
		t.Fatalf("clear height: got %d, want 0", cleared.Height())
	}
	if cleared.Width() != df.Width() {
		t.Fatalf("clear width: got %d, want %d", cleared.Width(), df.Width())
	}
}

func TestDFReverse(t *testing.T) {
	t.Parallel()
	df := helperDF()
	rev := df.Reverse()
	s, _ := rev.GetColumn("a")
	if v, ok := s.Value(0).(int64); !ok || v != 3 {
		t.Fatalf("reverse[0]: got %v, want 3", s.Value(0))
	}
}

func TestDFAggregations(t *testing.T) {
	t.Parallel()
	df := helperDF()
	sums := df.Sum()
	if sums["a"] != 6.0 {
		t.Fatalf("sum[a]: got %v, want 6.0", sums["a"])
	}
	means := df.Mean()
	if v, ok := means["a"]; !ok {
		t.Fatalf("mean[a]: missing")
	} else {
		if v < 1.99 || v > 2.01 {
			t.Fatalf("mean[a]: got %v, want ~2.0", v)
		}
	}
	maxMap := df.Max()
	if v, ok := maxMap["a"]; !ok {
		t.Fatalf("max[a]: missing")
	} else if vi, ok2 := v.(int64); !ok2 || vi != 3 {
		t.Fatalf("max[a]: got %v, want 3", v)
	}
	minMap := df.Min()
	if v, ok := minMap["a"]; !ok {
		t.Fatalf("min[a]: missing")
	} else if vi, ok2 := v.(int64); !ok2 || vi != 1 {
		t.Fatalf("min[a]: got %v, want 1", v)
	}
}

func TestDFUnique(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(1), int64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	uniq, err := df.Unique("a")
	if err != nil {
		t.Fatalf("unique: %v", err)
	}
	if uniq.Height() != 2 {
		t.Fatalf("unique height: got %d, want 2", uniq.Height())
	}
}

func TestDFIsNullNotNull(t *testing.T) {
	t.Parallel()
	df := helperDF()
	isDup := df.IsDuplicated()
	if isDup.Len() != 3 {
		t.Fatalf("is_duplicated len: got %d, want 3", isDup.Len())
	}
	isUni := df.IsUnique()
	if isUni.Len() != 3 {
		t.Fatalf("is_unique len: got %d, want 3", isUni.Len())
	}
}

func TestDFRow(t *testing.T) {
	t.Parallel()
	df := helperDF()
	row, err := df.Row(1)
	if err != nil {
		t.Fatalf("row: %v", err)
	}
	if row["a"] != int64(2) {
		t.Fatalf("row[1][a]: got %v, want 2", row["a"])
	}
	if row["c"] != "y" {
		t.Fatalf("row[1][c]: got %v, want y", row["c"])
	}
}

func TestDFItem(t *testing.T) {
	t.Parallel()
	df := helperDF()
	val, err := df.Item(1, "a")
	if err != nil {
		t.Fatalf("item: %v", err)
	}
	if v, ok := val.(int64); !ok || v != 2 {
		t.Fatalf("item(1,a): got %v, want 2", val)
	}
}

func TestDFShift(t *testing.T) {
	t.Parallel()
	df := helperDF()
	shifted, err := df.Shift(1)
	if err != nil {
		t.Fatalf("shift: %v", err)
	}
	s, _ := shifted.GetColumn("a")
	if s.Value(0) != nil {
		t.Fatalf("shift[0]: got %v, want nil", s.Value(0))
	}
	if v, ok := s.Value(1).(int64); !ok || v != 1 {
		t.Fatalf("shift[1]: got %v, want 1", s.Value(1))
	}
}

func TestDFFillNull(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{nil, int64(2), int64(3)}},
		},
	})
	if err != nil {
		t.Fatalf("df creation: %v", err)
	}
	filled, err := df.FillNull(int64(0))
	if err != nil {
		t.Fatalf("fill_null: %v", err)
	}
	s, _ := filled.GetColumn("a")
	if v, ok := s.Value(0).(int64); !ok || v != 0 {
		t.Fatalf("fill_null[0]: got %v, want 0", s.Value(0))
	}
}

func TestDFColumnsContain(t *testing.T) {
	t.Parallel()
	df := helperDF()
	cols := df.Columns()
	found := false
	for _, c := range cols {
		if c == "a" {
			found = true
		}
	}
	if !found {
		t.Fatalf("column 'a' should be in df columns")
	}
}

func TestDFWithColumns(t *testing.T) {
	t.Parallel()
	df := helperDF()
	result, err := df.WithColumns(polars.Col("a").Add(polars.Lit(int64(10))).Alias("d"))
	if err != nil {
		t.Fatalf("with_columns: %v", err)
	}
	if result.Width() != 4 {
		t.Fatalf("with_columns width: got %d, want 4", result.Width())
	}
}

func TestDFGatherEvery(t *testing.T) {
	t.Parallel()
	df := helperDF()
	result := df.GatherEvery(2, 0)
	if result.Height() != 2 {
		t.Fatalf("gather_every height: got %d, want 2", result.Height())
	}
}

func TestDFCast(t *testing.T) {
	t.Parallel()
	df := helperDF()
	casted, err := df.Cast(map[string]dtypes.DataType{"a": polars.Float64})
	if err != nil {
		t.Fatalf("cast: %v", err)
	}
	dtypes := casted.Dtypes()
	if dtypes[0] != polars.Float64 {
		t.Fatalf("cast dtype[0]: got %v, want Float64", dtypes[0])
	}
}
