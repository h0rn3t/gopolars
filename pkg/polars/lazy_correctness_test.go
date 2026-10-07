package polars

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// lazyValues returns fmt.Sprint of column name's values in d.
func lazyValues(t *testing.T, d DataFrame, name string) string {
	t.Helper()
	s, err := d.GetColumn(name)
	if err != nil {
		t.Fatalf("GetColumn(%q) error = %v, columns %v", name, err, d.Columns())
	}
	return fmt.Sprint(s.ToList())
}

// lazyFrame builds an in-memory lazy frame from int64 columns given as name,
// values pairs.
func lazyFrame(t *testing.T, cols ...frame.SeriesInput) LazyFrame {
	t.Helper()
	d, err := NewDataFrame(NewDataFrameInput{Columns: cols})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	return d.Lazy()
}

func int64Values(vs ...int64) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = v
	}
	return out
}

// checkResult compares d's column names with wantCols (when not nil) and the
// values of every column in want.
func checkResult(t *testing.T, label string, d DataFrame, wantCols []string, want map[string]string) {
	t.Helper()
	if wantCols != nil && !slices.Equal(d.Columns(), wantCols) {
		t.Errorf("%s columns = %v, want %v", label, d.Columns(), wantCols)
	}
	for name, values := range want {
		if got := lazyValues(t, d, name); got != values {
			t.Errorf("%s column %s = %s, want %s", label, name, got, values)
		}
	}
}

func TestLazyCollectKeepsWrittenOrderSemantics(t *testing.T) {
	ab := func(t *testing.T) LazyFrame {
		return lazyFrame(t,
			frame.SeriesInput{Name: "a", Values: int64Values(5, 1, 4, 2, 3)},
			frame.SeriesInput{Name: "b", Values: int64Values(10, 20, 30, 40, 50)},
		)
	}
	tests := []struct {
		name     string
		plan     func(t *testing.T) LazyFrame
		wantCols []string
		want     map[string]string
	}{
		{
			name:     "limit before sort",
			plan:     func(t *testing.T) LazyFrame { return ab(t).Limit(2).Sort(SortInput{By: []string{"a"}}) },
			wantCols: []string{"a", "b"},
			want:     map[string]string{"a": "[1 5]"},
		},
		{
			name:     "chained selects keep alias",
			plan:     func(t *testing.T) LazyFrame { return ab(t).Select(Col("a").Alias("x")).Select(Col("x")) },
			wantCols: []string{"x"},
			want:     map[string]string{"x": "[5 1 4 2 3]"},
		},
		{
			name: "computed select is not collapsed",
			plan: func(t *testing.T) LazyFrame {
				return lazyFrame(t, frame.SeriesInput{Name: "a", Values: int64Values(1, 2)}).
					Select(Col("a").Add(Lit(int64(1))).Alias("a")).Select(Col("a"))
			},
			wantCols: []string{"a"},
			want:     map[string]string{"a": "[2 3]"},
		},
		{
			name: "filter after renaming select",
			plan: func(t *testing.T) LazyFrame {
				return ab(t).Select(Col("a").Alias("b"), Col("b").Alias("a")).Filter(Col("a").Gt(Lit(int64(25))))
			},
			wantCols: []string{"b", "a"},
			want:     map[string]string{"b": "[4 2 3]", "a": "[30 40 50]"},
		},
		{
			name: "filter after aggregating select",
			plan: func(t *testing.T) LazyFrame {
				return lazyFrame(t, frame.SeriesInput{Name: "v", Values: int64Values(1, 2, 3)}).
					Select(Col("v"), Sum(Col("v")).Alias("s")).Filter(Col("v").Gt(Lit(int64(1))))
			},
			wantCols: []string{"v", "s"},
			want:     map[string]string{"v": "[2 3]", "s": "[6 6]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.plan(t).Collect(context.Background())
			if err != nil {
				t.Fatalf("Collect(%s) error = %v", tt.name, err)
			}
			checkResult(t, "Collect("+tt.name+")", got, tt.wantCols, tt.want)
		})
	}
}

func TestScanCSVPushdownKeepsPlanSemantics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "abc.csv")
	if err := os.WriteFile(path, []byte("a,b,c\n1,2,3\n4,5,6\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	scan := func(t *testing.T) LazyFrame {
		t.Helper()
		lf, err := NewIO().ScanCSV(ScanCSVInput{Path: path, HasHeader: true})
		if err != nil {
			t.Fatalf("ScanCSV(%q) error = %v", path, err)
		}
		return lf
	}
	tests := []struct {
		name     string
		plan     func(LazyFrame) LazyFrame
		wantCols []string
		want     map[string]string
	}{
		{
			name:     "filter keeps all columns",
			plan:     func(lf LazyFrame) LazyFrame { return lf.Filter(Col("a").Gt(Lit(int64(1)))) },
			wantCols: []string{"a", "b", "c"},
			want:     map[string]string{"a": "[4]", "b": "[5]", "c": "[6]"},
		},
		{
			name:     "computed select reads its inputs",
			plan:     func(lf LazyFrame) LazyFrame { return lf.Select(Col("a"), Col("b").Add(Col("c")).Alias("s")) },
			wantCols: []string{"a", "s"},
			want:     map[string]string{"a": "[1 4]", "s": "[5 11]"},
		},
		{
			name:     "select reads over partitions",
			plan:     func(lf LazyFrame) LazyFrame { return lf.Select(Col("a"), Col("b").CumCount().Over("c").Alias("n")) },
			wantCols: []string{"a", "n"},
			want:     map[string]string{"n": "[1 1]"},
		},
		{
			name: "group-by aggregation reads the aggregated column",
			plan: func(lf LazyFrame) LazyFrame {
				return lf.GroupBy("a").Agg(Sum(Col("b")).Alias("total")).Sort(SortInput{By: []string{"a"}})
			},
			wantCols: []string{"a", "total"},
			want:     map[string]string{"a": "[1 4]", "total": "[2 5]"},
		},
		{
			name: "filter after with_columns sees the new values",
			plan: func(lf LazyFrame) LazyFrame {
				return lf.WithColumns(Col("a").Mul(Lit(int64(10))).Alias("a")).Filter(Col("a").Gt(Lit(int64(5))))
			},
			want: map[string]string{"a": "[10 40]"},
		},
		{
			name: "filter after limit sees the limited rows",
			plan: func(lf LazyFrame) LazyFrame { return lf.Limit(1).Filter(Col("a").Gt(Lit(int64(1)))) },
			want: map[string]string{"a": "[]"},
		},
		{
			name:     "filter before a projection is pushed and pruned",
			plan:     func(lf LazyFrame) LazyFrame { return lf.Filter(Col("a").Gt(Lit(int64(1)))).Select(Col("b")) },
			wantCols: []string{"b"},
			want:     map[string]string{"b": "[5]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.plan(scan(t)).Collect(context.Background())
			if err != nil {
				t.Fatalf("Collect(%s) error = %v", tt.name, err)
			}
			checkResult(t, "Collect("+tt.name+")", got, tt.wantCols, tt.want)
		})
	}
}

// TestScanParquetPartitionPruningRespectsOrder checks that a filter on a hive
// partition column prunes partitions only when it is not behind a node that
// changes the rows or the partition column.
func TestScanParquetPartitionPruningRespectsOrder(t *testing.T) {
	root := t.TempDir()
	for p, ids := range map[string][]any{"1": int64Values(1, 2), "2": int64Values(3, 4)} {
		dir := filepath.Join(root, "p="+p)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
		part, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{{Name: "id", Values: ids}}})
		if err != nil {
			t.Fatalf("NewDataFrame: %v", err)
		}
		if err := part.WriteParquet(WriteParquetInput{Path: filepath.Join(dir, "data.parquet")}); err != nil {
			t.Fatalf("WriteParquet: %v", err)
		}
	}
	isTwo := Col("p").Eq(Lit("2"))
	tests := []struct {
		name string
		plan func(LazyFrame) LazyFrame
		want string // id values
	}{
		{"leading filter prunes", func(lf LazyFrame) LazyFrame { return lf.Filter(isTwo) }, "[3 4]"},
		{"filter after limit", func(lf LazyFrame) LazyFrame { return lf.Limit(1).Filter(isTwo) }, "[]"},
		{"filter after with_columns", func(lf LazyFrame) LazyFrame {
			return lf.WithColumns(Lit("2").Alias("p")).Filter(isTwo).Sort(SortInput{By: []string{"id"}})
		}, "[1 2 3 4]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lf, err := NewIO().ScanParquet(ScanParquetInput{Path: root})
			if err != nil {
				t.Fatalf("ScanParquet(%q) error = %v", root, err)
			}
			got, err := tt.plan(lf).Collect(context.Background())
			if err != nil {
				t.Fatalf("Collect(%s) error = %v", tt.name, err)
			}
			checkResult(t, "Collect("+tt.name+")", got, nil, map[string]string{"id": tt.want})
		})
	}
}

func TestCollectStreamingMatchesCollect(t *testing.T) {
	a51423 := func(t *testing.T) LazyFrame {
		return lazyFrame(t, frame.SeriesInput{Name: "a", Values: int64Values(5, 1, 4, 2, 3)})
	}
	aGt1 := Col("a").Gt(Lit(int64(1)))
	tests := []struct {
		name string
		plan func(t *testing.T) LazyFrame
		want map[string]string
	}{
		{
			name: "row index continues across chunks",
			plan: func(t *testing.T) LazyFrame { return a51423(t).WithRowIndex("i", 0) },
			want: map[string]string{"i": "[0 1 2 3 4]"},
		},
		{
			name: "tail of the whole input",
			plan: func(t *testing.T) LazyFrame { return a51423(t).Tail(2) },
			want: map[string]string{"a": "[2 3]"},
		},
		{
			name: "limit then filter",
			plan: func(t *testing.T) LazyFrame { return a51423(t).Limit(3).Filter(aGt1) },
			want: map[string]string{"a": "[5 4]"},
		},
		{
			name: "whole-column expressions",
			plan: func(t *testing.T) LazyFrame {
				a := Col("a")
				return lazyFrame(t, frame.SeriesInput{Name: "a", Values: int64Values(1, 2, 3, 4)}).
					WithColumns(a.CumSum().Alias("c"), Sum(a).Alias("s"))
			},
			want: map[string]string{"c": "[1 3 6 10]", "s": "[10 10 10 10]"},
		},
		{
			name: "update from a reordered frame",
			plan: func(t *testing.T) LazyFrame {
				lf := a51423(t)
				return lf.Update(lf.Reverse())
			},
			want: map[string]string{"a": "[3 2 4 1 5]"},
		},
		{
			name: "row-local plan",
			plan: func(t *testing.T) LazyFrame {
				return a51423(t).Filter(aGt1).Select(Col("a").Mul(Lit(int64(2))).Alias("d"))
			},
			want: map[string]string{"d": "[10 8 4 6]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			lf := tt.plan(t)
			want, err := lf.Collect(ctx)
			if err != nil {
				t.Fatalf("Collect(%s) error = %v", tt.name, err)
			}
			checkResult(t, "Collect("+tt.name+")", want, nil, tt.want)
			for _, chunk := range []int{1, 2, 3} {
				got, err := lf.CollectStreaming(ctx, chunk)
				if err != nil {
					t.Fatalf("CollectStreaming(%s, %d) error = %v", tt.name, chunk, err)
				}
				if eq, err := want.Equals(got); err != nil || !eq {
					t.Errorf("CollectStreaming(%s, %d) = %v, want %v", tt.name, chunk, got, want)
				}
			}
		})
	}
}
