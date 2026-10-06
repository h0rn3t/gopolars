package parquet

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/parquet/file"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// dictionaryFrame holds one column per dictionary decision the writer makes.
func dictionaryFrame(t *testing.T) frame.DataFrame {
	t.Helper()
	const n = 100_000
	cols := map[string][]any{}
	names := []string{"unique_int", "low_int", "unique_float", "low_float", "unique_ts", "low_ts", "label", "unique_label"}
	for _, name := range names {
		cols[name] = make([]any, n)
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range n {
		cols["unique_int"][i] = int64(i)
		cols["low_int"][i] = int64(i % 10)
		cols["unique_float"][i] = float64(i) / 3
		cols["low_float"][i] = []float64{0.5, math.NaN(), 2.5}[i%3]
		cols["unique_ts"][i] = base.Add(time.Duration(i) * time.Second)
		cols["low_ts"][i] = base.Add(time.Duration(i%10) * time.Hour)
		cols["label"][i] = []string{"alpha", "beta", "gamma"}[i%3]
		cols["unique_label"][i] = "user-" + strconv.Itoa(i)
		if i%11 == 0 {
			cols["low_float"][i] = nil
			cols["label"][i] = nil
		}
	}
	dtypeOf := map[string]dtypes.DataType{
		"unique_int": dtypes.Int64, "low_int": dtypes.Int64,
		"unique_float": dtypes.Float64, "low_float": dtypes.Float64,
		"unique_ts": dtypes.Datetime, "low_ts": dtypes.Datetime,
		"label": dtypes.String, "unique_label": dtypes.String,
	}
	inputs := make([]frame.SeriesInput, 0, len(names))
	for _, name := range names {
		inputs = append(inputs, frame.SeriesInput{Name: name, Values: cols[name], DType: dtypeOf[name]})
	}
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: inputs})
	if err != nil {
		t.Fatalf("FromAnyColumns: %v", err)
	}
	return df
}

// dictionaryPages reports, per column name, whether the first row group's
// column chunk carries a dictionary page.
func dictionaryPages(t *testing.T, path string) map[string]bool {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}
	defer func() { _ = f.Close() }() // read-only file
	pf, err := file.NewParquetReader(f)
	if err != nil {
		t.Fatalf("NewParquetReader(%q): %v", path, err)
	}
	rg := pf.MetaData().RowGroup(0)
	out := map[string]bool{}
	for i := range rg.NumColumns() {
		cc, err := rg.ColumnChunk(i)
		if err != nil {
			t.Fatalf("ColumnChunk(%d): %v", i, err)
		}
		out[cc.PathInSchema().String()] = cc.HasDictionaryPage()
	}
	return out
}

func TestWriteDictionaryOnlyWherePaysOff(t *testing.T) {
	t.Parallel()
	df := dictionaryFrame(t)
	path := filepath.Join(t.TempDir(), "dict.parquet")
	if err := Write(df, WriteInput{Path: path}); err != nil {
		t.Fatalf("Write(%q): %v", path, err)
	}
	got := dictionaryPages(t, path)

	want := map[string]bool{
		"unique_int":   false,
		"low_int":      true,
		"unique_float": false,
		"low_float":    true,
		"unique_ts":    false,
		"low_ts":       true,
		"label":        true,
		"unique_label": false,
	}
	for name, wantDict := range want {
		if got[name] != wantDict {
			t.Errorf("Write: column %q has dictionary page = %t, want %t", name, got[name], wantDict)
		}
	}

	again := filepath.Join(t.TempDir(), "dict_again.parquet")
	if err := Write(df, WriteInput{Path: again}); err != nil {
		t.Fatalf("Write(%q): %v", again, err)
	}
	for name, dict := range dictionaryPages(t, again) {
		if dict != got[name] {
			t.Errorf("Write twice: column %q dictionary page = %t then %t, want the same", name, got[name], dict)
		}
	}

	back, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("Read(%q): %v", path, err)
	}
	for _, name := range df.Columns() {
		src, _ := df.Series(name)
		dst, ok := back.Series(name)
		if !ok {
			t.Fatalf("Read: column %q missing", name)
		}
		if dst.DataType() != src.DataType() || dst.Len() != src.Len() {
			t.Fatalf("Read: column %q = %s[%d], want %s[%d]", name, dst.DataType(), dst.Len(), src.DataType(), src.Len())
		}
		for i := range src.Len() {
			if !sameCell(src.Value(i), dst.Value(i)) {
				t.Errorf("Read: column %q row %d = %v, want %v", name, i, dst.Value(i), src.Value(i))
				break
			}
		}
	}
}

// sameCell compares two cell values, treating NaN as equal to NaN and
// datetimes as equal instants.
func sameCell(want, got any) bool {
	switch w := want.(type) {
	case float64:
		g, ok := got.(float64)
		return ok && (g == w || math.IsNaN(g) && math.IsNaN(w))
	case time.Time:
		g, ok := got.(time.Time)
		return ok && g.Equal(w)
	default:
		return want == got
	}
}

// dictionaryEntries reports, per row group, how many values the dictionary page
// of column holds, or -1 when the column chunk has no dictionary page.
func dictionaryEntries(t *testing.T, path string, column int) []int32 {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}
	defer func() { _ = f.Close() }() // read-only file
	pf, err := file.NewParquetReader(f)
	if err != nil {
		t.Fatalf("NewParquetReader(%q): %v", path, err)
	}
	out := make([]int32, pf.NumRowGroups())
	for rg := range out {
		pages, err := pf.RowGroup(rg).GetColumnPageReader(column)
		if err != nil {
			t.Fatalf("GetColumnPageReader(%d) in row group %d: %v", column, rg, err)
		}
		out[rg] = -1
		if pages.Next() && pages.Page().Type() == file.PageTypeDictionaryPage {
			out[rg] = pages.Page().NumValues()
		}
	}
	return out
}

func TestWriteStringDictionary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		values   []any
		rowGroup int
		want     []int32 // dictionary entries per row group, -1 for none
	}{
		{name: "empty", values: nil, want: []int32{-1}},
		{name: "all null", values: []any{nil, nil, nil, nil}, want: []int32{0}},
		{name: "repeated values with nulls", values: []any{"a", "b", nil, "a", "b", "c", nil, "a", "b", "c", "a", "b"}, want: []int32{3}},
		{name: "row groups with disjoint values", values: []any{"x", "y", "x", "y", "x", "p", "q", "p", "q", nil}, rowGroup: 5, want: []int32{2, 2}},
		{name: "unique values", values: []any{"a", "b", "c", "d", "e", "f", "g", "h"}, want: []int32{-1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: []frame.SeriesInput{
				{Name: "s", Values: tt.values, DType: dtypes.String},
			}})
			if err != nil {
				t.Fatalf("FromAnyColumns(%v): %v", tt.values, err)
			}
			path := filepath.Join(t.TempDir(), "s.parquet")
			if err := Write(df, WriteInput{Path: path, RowGroupSize: tt.rowGroup}); err != nil {
				t.Fatalf("Write(%v, RowGroupSize %d): %v", tt.values, tt.rowGroup, err)
			}
			if got := dictionaryEntries(t, path, 0); !slices.Equal(got, tt.want) {
				t.Errorf("Write(%v, RowGroupSize %d) dictionary entries per row group = %v, want %v", tt.values, tt.rowGroup, got, tt.want)
			}

			back, err := Read(ReadInput{Path: path})
			if err != nil {
				t.Fatalf("Read(%q): %v", path, err)
			}
			if back.Height() != len(tt.values) {
				t.Fatalf("Read(Write(%v)) height = %d, want %d", tt.values, back.Height(), len(tt.values))
			}
			// An empty file reads back with no columns at all, so only a
			// non-empty column has a dtype to check.
			s, ok := back.Series("s")
			if len(tt.values) > 0 && (!ok || s.DataType() != dtypes.String) {
				t.Fatalf("Read(Write(%v)) column s = %v, want String", tt.values, s.DataType())
			}
			for i, want := range tt.values {
				if got := s.Value(i); got != want {
					t.Errorf("Read(Write(%v)) row %d = %v, want %v", tt.values, i, got, want)
				}
			}
		})
	}
}
