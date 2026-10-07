package polars

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

func auditFrame(t *testing.T, cols ...frame.SeriesInput) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: cols})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	return df
}

func auditValues(t *testing.T, df DataFrame, name string) []any {
	t.Helper()
	s, ok := df.Series(name)
	if !ok {
		t.Fatalf("column %s missing from %v", name, df.Columns())
	}
	out := make([]any, s.Len())
	for i := range out {
		out[i] = s.Value(i)
	}
	return out
}

// auditRows reports an error unless err is nil and df has want rows.
func auditRows(t *testing.T, call string, df DataFrame, err error, want int) {
	t.Helper()
	if err != nil {
		t.Errorf("%s error = %v, want nil", call, err)
		return
	}
	if df.Height() != want {
		t.Errorf("%s rows = %d, want %d", call, df.Height(), want)
	}
}

func TestInsertColumnValidatesLikePolars(t *testing.T) {
	t.Parallel()
	tests := []struct {
		index   int
		name    string
		want    []string
		wantErr string
	}{
		{2, "c", []string{"a", "b", "c"}, ""},
		{1, "c", []string{"a", "c", "b"}, ""},
		{3, "c", nil, "out of bounds"},
		{-1, "c", nil, "out of bounds"},
		{2, "a", nil, "duplicate"},
	}
	for _, tt := range tests {
		df := auditFrame(t,
			frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}},
			frame.SeriesInput{Name: "b", Values: []any{"x", "y"}},
		)
		col, err := NewSeries(NewSeriesInput{Name: tt.name, DType: Int64, Values: []any{int64(7), int64(8)}})
		if err != nil {
			t.Fatalf("NewSeries: %v", err)
		}
		out, err := df.InsertColumn(tt.index, col)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("InsertColumn(%d, %s) error = %v, want error containing %q", tt.index, tt.name, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("InsertColumn(%d, %s) error = %v, want nil", tt.index, tt.name, err)
			continue
		}
		if got := out.Columns(); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("InsertColumn(%d, %s) columns = %v, want %v", tt.index, tt.name, got, tt.want)
		}
	}
}

func TestNestedColumnsSortAndCompare(t *testing.T) {
	t.Parallel()
	df := auditFrame(t, frame.SeriesInput{Name: "l", Values: []any{[]any{int64(2)}, []any{int64(1), int64(2)}, nil, []any{int64(1)}}})
	sorted, err := df.Sort(SortInput{By: []string{"l"}, NullsLast: true})
	if err != nil {
		t.Fatalf("Sort(by list) error = %v", err)
	}
	want := []any{[]any{int64(1)}, []any{int64(1), int64(2)}, []any{int64(2)}, nil}
	if got := auditValues(t, sorted, "l"); !reflect.DeepEqual(got, want) {
		t.Errorf("Sort(by list, nulls last) = %v, want %v", got, want)
	}
	kept, err := df.Filter(Col("l").Eq(Col("l")))
	if err != nil {
		t.Fatalf("Filter(l == l) error = %v", err)
	}
	if kept.Height() != 3 {
		t.Errorf("Filter(l == l) rows = %d, want 3 (the null row is dropped)", kept.Height())
	}
}

func TestCumulativeOverPartitions(t *testing.T) {
	t.Parallel()
	df := auditFrame(t,
		frame.SeriesInput{Name: "g", Values: []any{"a", "b", "a", "b", "a"}},
		frame.SeriesInput{Name: "v", Values: []any{int64(1), int64(10), nil, int64(20), int64(3)}},
	)
	out, err := df.Select(
		Col("v").CumSum().Over("g").Alias("cs"),
		Col("v").CumCount().Over("g").Alias("cc"),
	)
	if err != nil {
		t.Fatalf("Select(cum_sum.over, cum_count.over) error = %v", err)
	}
	if got, want := auditValues(t, out, "cs"), []any{1.0, 10.0, 1.0, 30.0, 4.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("cum_sum().over(g) = %v, want %v", got, want)
	}
	if got, want := auditValues(t, out, "cc"), []any{int64(1), int64(1), int64(1), int64(2), int64(2)}; !reflect.DeepEqual(got, want) {
		t.Errorf("cum_count().over(g) = %v, want %v", got, want)
	}
}

func TestSignedZeroKeysThroughFacade(t *testing.T) {
	t.Parallel()
	base := []any{0.0, math.Copysign(0, -1), math.NaN(), math.Float64frombits(0xfff8000000000001), 1.0}
	for _, n := range []int{5, 65536} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			t.Parallel()
			k := make([]any, n)
			v := make([]any, n)
			for i := range n {
				k[i] = base[i%len(base)]
				v[i] = int64(1)
			}
			df := auditFrame(t, frame.SeriesInput{Name: "k", Values: k}, frame.SeriesInput{Name: "v", Values: v})
			s, _ := df.Series("k")
			if got := s.NUnique(); got != 3 {
				t.Errorf("Series.NUnique() = %d, want 3", got)
			}
			counts, err := s.ValueCounts()
			auditRows(t, "Series.ValueCounts()", counts, err, 3)
			groups, err := df.GroupBy("k").Agg(Sum(Col("v")))
			auditRows(t, "GroupBy(k).Agg(sum(v))", groups, err, 3)
			unique, err := df.Unique("k")
			auditRows(t, "Unique(k)", unique, err, 3)
			if n == 5 {
				keys, err := df.Drop("v")
				if err != nil {
					t.Fatalf("Drop(v): %v", err)
				}
				dup := keys.IsDuplicated()
				got := make([]any, dup.Len())
				for i := range got {
					got[i] = dup.Value(i)
				}
				if fmt.Sprint(got) != "[true true true true false]" {
					t.Errorf("IsDuplicated() on k=%v = %v, want [true true true true false]", k, got)
				}
				out, err := keys.ToDummies("k")
				if err != nil {
					t.Fatalf("ToDummies(k) error = %v", err)
				}
				if got := out.Columns(); !reflect.DeepEqual(got, []string{"k_0", "k_1", "k_NaN"}) {
					t.Errorf("ToDummies(k) columns = %v, want [k_0 k_1 k_NaN]", got)
				}
			}
		})
	}
}

func TestAnalyticsSignedZeroKeys(t *testing.T) {
	t.Parallel()
	negZero := math.Copysign(0, -1)
	df := auditFrame(t,
		frame.SeriesInput{Name: "k", Values: []any{0.0, negZero, 0.0}},
		frame.SeriesInput{Name: "c", Values: []any{negZero, 0.0, 0.0}},
		frame.SeriesInput{Name: "o", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "v", Values: []any{1.0, 1.0, 1.0}},
	)
	summed, err := WindowSum(df, WindowSumInput{PartitionBy: []string{"k"}, OrderBy: "o", Value: "v", Output: "s"})
	if err != nil {
		t.Fatalf("WindowSum(partition by k) error = %v", err)
	}
	if got, want := auditValues(t, summed, "s"), []any{1.0, 2.0, 3.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("WindowSum(partition by k=[0 -0 0]) = %v, want %v", got, want)
	}
	pivoted, err := Pivot(df, PivotInput{Index: "k", Columns: "c", Values: "v", Agg: "sum"})
	if err != nil {
		t.Fatalf("Pivot(index k, columns c) error = %v", err)
	}
	if pivoted.Height() != 1 || pivoted.Width() != 2 {
		t.Errorf("Pivot(index k=[0 -0 0], columns c=[-0 0 0]) shape = %dx%d (columns %v), want 1x2", pivoted.Height(), pivoted.Width(), pivoted.Columns())
	}
}
