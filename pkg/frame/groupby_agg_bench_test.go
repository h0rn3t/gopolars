package frame

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// aggBenchFrame has n rows: "id" with 500 groups, a distinct "seq", a Float64
// "v", a String "name", a Boolean "flag", a Datetime "ts" null on every 9th
// row, and a Float64 "w" and Int64 "iw" null throughout every 50th group.
func aggBenchFrame(tb testing.TB, n int) DataFrame {
	tb.Helper()
	id := make([]int64, n)
	seq := make([]int64, n)
	v := make([]float64, n)
	name := make([]string, n)
	flag := make([]bool, n)
	ts := make([]time.Time, n)
	tsNulls := make([]bool, n)
	w := make([]float64, n)
	iw := make([]int64, n)
	wNulls := make([]bool, n)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range n {
		id[i] = int64(i % 500)
		seq[i] = int64(i)
		v[i] = float64(i) * 1.5
		name[i] = fmt.Sprintf("zone-%d", i%300)
		flag[i] = i%3 == 0
		ts[i] = base.Add(time.Duration(i) * time.Minute)
		tsNulls[i] = i%9 == 0
		w[i] = float64(i%17) - 8
		iw[i] = int64(i%13) - 6
		wNulls[i] = id[i]%50 == 0
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromInt64("id", id, nil),
		series.FromInt64("seq", seq, nil),
		series.FromFloat64("v", v, nil),
		series.FromString("name", name, nil),
		series.FromBool("flag", flag, nil),
		series.FromColumn("ts", chunk.NewTime(ts, tsNulls)),
		series.FromFloat64("w", w, wNulls),
		series.FromInt64("iw", iw, slices.Clone(wNulls)),
	}})
	if err != nil {
		tb.Fatalf("New: %v", err)
	}
	return df
}

func BenchmarkGroupByAgg(b *testing.B) {
	df := aggBenchFrame(b, 100_000)
	cases := []struct {
		name string
		keys []string
		aggs []expr.Expr
	}{
		{"sum+first/500groups", []string{"id"}, []expr.Expr{expr.Sum(expr.Col("v")), expr.First(expr.Col("name"))}},
		{"sum/100k-groups", []string{"seq"}, []expr.Expr{expr.Sum(expr.Col("v"))}},
		{"last-datetime/500groups", []string{"id"}, []expr.Expr{expr.Last(expr.Col("ts"))}},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := df.GroupBy(c.keys...).Agg(c.aggs...); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestGroupByParallelFirstLastAndNullGroups(t *testing.T) {
	df := aggBenchFrame(t, 100_000)
	cases := []struct {
		name  string
		exprs []expr.Expr
	}{
		{"sum+first", []expr.Expr{expr.Sum(expr.Col("v")), expr.First(expr.Col("name"))}},
		{"first+last datetime", []expr.Expr{expr.First(expr.Col("ts")), expr.Last(expr.Col("ts"))}},
		{"first+last string and bool", []expr.Expr{expr.First(expr.Col("name")), expr.Last(expr.Col("name")), expr.First(expr.Col("flag")), expr.Last(expr.Col("flag"))}},
		{"all-null groups", []expr.Expr{
			expr.Sum(expr.Col("w")), expr.Mean(expr.Col("w")), expr.Min(expr.Col("w")), expr.Max(expr.Col("w")),
			expr.Sum(expr.Col("iw")), expr.Mean(expr.Col("iw")), expr.Min(expr.Col("iw")), expr.Max(expr.Col("iw")),
			expr.Count(),
		}},
	}
	keyCols, err := keyColumns(df, []string{"id"}, "group key")
	if err != nil {
		t.Fatalf("keyColumns: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GOPOLARS_TYPED_STORAGE", "1")
			if _, ok, err := df.GroupBy("id").aggParallel(keyCols, tc.exprs); err != nil || !ok {
				t.Fatalf("aggParallel(%s) ok = %v, err = %v, want the parallel path to run", tc.name, ok, err)
			}
			par, err := df.GroupBy("id").Agg(tc.exprs...)
			if err != nil {
				t.Fatalf("parallel Agg(%s): %v", tc.name, err)
			}
			t.Setenv("GOPOLARS_TYPED_STORAGE", "0")
			seq, err := df.GroupBy("id").Agg(tc.exprs...)
			if err != nil {
				t.Fatalf("sequential Agg(%s): %v", tc.name, err)
			}
			assertGroupFramesEqual(t, tc.name, par, seq)
		})
	}
	t.Setenv("GOPOLARS_TYPED_STORAGE", "1")
	got, err := df.GroupBy("id").Agg(expr.Sum(expr.Col("w")).Alias("s"), expr.Last(expr.Col("ts")).Alias("last"))
	if err != nil {
		t.Fatalf("Agg(sum w, last ts): %v", err)
	}
	// Group 0 is null throughout w; group 4's last row, 99504, has a null ts.
	s, _ := got.Series("s")
	if !s.IsNull(0) {
		t.Errorf("sum(w) of the all-null group 0 = %v, want null", s.Value(0))
	}
	last, _ := got.Series("last")
	if !last.IsNull(4) {
		t.Errorf("last(ts) of group 4, whose last row 99504 is null = %v, want null", last.Value(4))
	}
}
