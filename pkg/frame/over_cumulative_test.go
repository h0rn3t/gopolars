package frame

import (
	"reflect"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

// TestOverCumulativeRestartsPerPartition checks cum_sum, cum_count and rank
// over partitions see only the raw values of their own partition.
func TestOverCumulativeRestartsPerPartition(t *testing.T) {
	t.Parallel()
	g := []any{"a", "b", "a", "b", "a"}
	tests := []struct {
		name string
		v    []any
		e    expr.Expr
		want []any
	}{
		{"cum_sum int64", []any{int64(1), int64(10), int64(2), int64(20), int64(3)},
			expr.Col("v").CumSum().Over("g"), []any{1.0, 10.0, 3.0, 30.0, 6.0}},
		{"cum_sum float64", []any{1.0, 10.0, 2.0, 20.0, 3.0},
			expr.Col("v").CumSum().Over("g"), []any{1.0, 10.0, 3.0, 30.0, 6.0}},
		{"cum_sum int64 with null", []any{int64(1), int64(10), nil, int64(20), int64(3)},
			expr.Col("v").CumSum().Over("g"), []any{1.0, 10.0, 1.0, 30.0, 4.0}},
		{"cum_sum float64 with null", []any{1.0, 10.0, nil, 20.0, 3.0},
			expr.Col("v").CumSum().Over("g"), []any{1.0, 10.0, 1.0, 30.0, 4.0}},
		{"cum_count with null", []any{int64(1), int64(10), nil, int64(20), int64(3)},
			expr.Col("v").CumCount().Over("g"), []any{int64(1), int64(1), int64(1), int64(2), int64(2)}},
		{"rank", []any{int64(3), int64(1), int64(2), int64(5), int64(1)},
			expr.Col("v").Rank().Over("g"), []any{int64(3), int64(1), int64(2), int64(2), int64(1)}},
		{"rank float64", []any{3.5, 1.0, 2.5, 5.0, 1.5},
			expr.Col("v").Rank().Over("g"), []any{int64(3), int64(1), int64(2), int64(2), int64(1)}},
		{"rank string", []any{"c", "a", "b", "e", "a"},
			expr.Col("v").Rank().Over("g"), []any{int64(3), int64(1), int64(2), int64(2), int64(1)}},
		{"cum_sum without partition", []any{int64(1), int64(10), int64(2), int64(20), int64(3)},
			expr.Col("v").CumSum().Over(), []any{1.0, 11.0, 13.0, 33.0, 36.0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			df := mustFrame(t, SeriesInput{Name: "g", Values: g}, SeriesInput{Name: "v", Values: tt.v})
			out, err := df.Select(tt.e.Alias("o"))
			if err != nil {
				t.Fatalf("Select(%s) error = %v", tt.name, err)
			}
			if got := columnValues(t, out, "o"); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Select(%s) with v=%v = %v, want %v", tt.name, tt.v, got, tt.want)
			}
		})
	}
}

// TestOverCumulativeLargeMatchesPerPartitionReference uses a size that takes
// the parallel partition build and compares every row with a running value
// computed partition by partition in row order.
func TestOverCumulativeLargeMatchesPerPartitionReference(t *testing.T) {
	t.Parallel()
	const n, parts = 40000, 4
	g := make([]any, n)
	v := make([]any, n)
	for i := range n {
		g[i] = int64(i % parts)
		if i%11 != 0 {
			v[i] = int64(i%97 - 20)
		}
	}
	df := mustFrame(t, SeriesInput{Name: "g", Values: g}, SeriesInput{Name: "v", Values: v})
	out, err := df.Select(
		expr.Col("v").CumSum().Over("g").Alias("cs"),
		expr.Col("v").CumCount().Over("g").Alias("cc"),
	)
	if err != nil {
		t.Fatalf("Select(cum_sum.over, cum_count.over) error = %v", err)
	}
	cs, cc := columnValues(t, out, "cs"), columnValues(t, out, "cc")
	sums := make([]float64, parts)
	counts := make([]int64, parts)
	for i := range n {
		p := i % parts
		if x, ok := v[i].(int64); ok {
			sums[p] += float64(x)
			counts[p]++
		}
		if cs[i] != sums[p] || cc[i] != counts[p] {
			t.Fatalf("row %d (partition %d): cum_sum = %v, cum_count = %v, want %v, %v", i, p, cs[i], cc[i], sums[p], counts[p])
		}
	}
}
