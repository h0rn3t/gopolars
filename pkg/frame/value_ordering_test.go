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

// hours returns one Datetime per offset, in hours after 2026-01-01T00:00Z.
func hours(offsets ...int) []time.Time {
	out := make([]time.Time, len(offsets))
	for i, h := range offsets {
		out[i] = time.Date(2026, 1, 1, h, 0, 0, 0, time.UTC)
	}
	return out
}

func timeFrame(t *testing.T, ts []time.Time) DataFrame {
	t.Helper()
	seq := make([]int64, len(ts))
	for i := range seq {
		seq[i] = int64(i)
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromColumn("ts", chunk.NewTime(ts, nil)),
		series.FromInt64("g", make([]int64, len(ts)), nil),
		series.FromInt64("seq", seq, nil),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return df
}

func columnTimes(t *testing.T, df DataFrame, name string) []time.Time {
	t.Helper()
	s, ok := df.Series(name)
	if !ok {
		t.Fatalf("column %q not found", name)
	}
	vals, ok := s.Column().Times()
	if !ok {
		t.Fatalf("column %q is %s, want datetime", name, s.DataType())
	}
	return vals
}

func columnInt64s(t *testing.T, df DataFrame, name string) []int64 {
	t.Helper()
	s, ok := df.Series(name)
	if !ok {
		t.Fatalf("column %q not found", name)
	}
	vals, ok := s.Column().Int64s()
	if !ok {
		t.Fatalf("column %q is %s, want int64", name, s.DataType())
	}
	return vals
}

func TestSortOrdersDatetimeByInstant(t *testing.T) {
	t.Parallel()
	for _, n := range []int{3, 300} {
		ts := make([]time.Time, n)
		for i := range ts {
			// Three hours cycled with a decreasing minute offset: unsorted, distinct.
			ts[i] = hours(3, 1, 2)[i%3].Add(time.Duration(n-i) * time.Minute)
		}
		df := timeFrame(t, ts)
		for _, desc := range []bool{false, true} {
			got, err := df.Sort(SortInput{By: []string{"ts"}, Descending: []bool{desc}})
			if err != nil {
				t.Fatalf("Sort(ts, desc=%v): %v", desc, err)
			}
			want := slices.SortedFunc(slices.Values(ts), time.Time.Compare)
			if desc {
				slices.Reverse(want)
			}
			if vals := columnTimes(t, got, "ts"); !slices.Equal(vals, want) {
				t.Errorf("Sort(ts, desc=%v) on %d rows is not ordered by instant: first %v, want %v", desc, n, vals[0], want[0])
			}
		}
	}
}

func TestSortOrdersDurationByLength(t *testing.T) {
	t.Parallel()
	df := mustFrame(t, SeriesInput{Name: "d", Values: []any{3 * time.Second, time.Second, 2 * time.Second}})
	got, err := df.Sort(SortInput{By: []string{"d"}})
	if err != nil {
		t.Fatalf("Sort(d): %v", err)
	}
	want := []any{time.Second, 2 * time.Second, 3 * time.Second}
	s, _ := got.Series("d")
	for i, w := range want {
		if v := s.Value(i); v != w {
			t.Errorf("Sort(d)[%d] = %v, want %v", i, v, w)
		}
	}
}

func TestGroupByMinMaxDatetime(t *testing.T) {
	t.Parallel()
	df := timeFrame(t, hours(2, 0, 5))
	got, err := df.GroupBy("g").Agg(expr.Min(expr.Col("ts")).Alias("mn"), expr.Max(expr.Col("ts")).Alias("mx"))
	if err != nil {
		t.Fatalf("GroupBy(g).Agg(min, max): %v", err)
	}
	if mn := columnTimes(t, got, "mn")[0]; !mn.Equal(hours(0)[0]) {
		t.Errorf("min(ts) of [02:00 00:00 05:00] = %v, want 00:00", mn)
	}
	if mx := columnTimes(t, got, "mx")[0]; !mx.Equal(hours(5)[0]) {
		t.Errorf("max(ts) of [02:00 00:00 05:00] = %v, want 05:00", mx)
	}
}

func TestRankDatetime(t *testing.T) {
	t.Parallel()
	df := timeFrame(t, hours(3, 1, 2))
	tests := []struct {
		name string
		e    expr.Expr
	}{
		{"plain", expr.Col("ts").Rank().Alias("r")},
		{"over", expr.Col("ts").Rank().Over("g").Alias("r")},
	}
	for _, tt := range tests {
		got, err := df.WithColumns(tt.e)
		if err != nil {
			t.Fatalf("WithColumns(%s rank): %v", tt.name, err)
		}
		if ranks := columnInt64s(t, got, "r"); !slices.Equal(ranks, []int64{3, 1, 2}) {
			t.Errorf("%s rank of [03:00 01:00 02:00] = %v, want [3 1 2]", tt.name, ranks)
		}
	}
}

func TestFlagsSortedDatetime(t *testing.T) {
	t.Parallel()
	df := timeFrame(t, hours(3, 1))
	if df.Flags()["ts"]["sorted_asc"] {
		t.Error("Flags()[ts][sorted_asc] of [03:00 01:00] = true, want false")
	}
}

func TestPivotMinMaxBoolean(t *testing.T) {
	t.Parallel()
	df := mustFrame(t,
		SeriesInput{Name: "i", Values: []any{int64(1), int64(1)}},
		SeriesInput{Name: "c", Values: []any{"x", "x"}},
		SeriesInput{Name: "b", Values: []any{true, false}},
	)
	for agg, want := range map[string]bool{"min": false, "max": true} {
		got, err := df.Pivot([]string{"i"}, "c", "b", agg)
		if err != nil {
			t.Fatalf("Pivot(%s): %v", agg, err)
		}
		s, ok := got.Series("x")
		if !ok {
			t.Fatalf("Pivot(%s) columns = %v, want an x column", agg, got.Columns())
		}
		if v := s.Value(0); v != want {
			t.Errorf("Pivot(%s) of [true false] = %v, want %v", agg, v, want)
		}
	}
}

func TestSortSameInstantInTwoZonesTies(t *testing.T) {
	t.Parallel()
	for _, ts := range [][]time.Time{{noonUTC, noonPlusTwo}, {noonPlusTwo, noonUTC}} {
		got, err := timeFrame(t, ts).Sort(SortInput{By: []string{"ts"}})
		if err != nil {
			t.Fatalf("Sort(ts): %v", err)
		}
		if seq := columnInt64s(t, got, "seq"); !slices.Equal(seq, []int64{0, 1}) {
			t.Errorf("Sort(ts) of %v = rows %v, want [0 1]", fmt.Sprint(ts), seq)
		}
	}
}
