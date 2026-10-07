package polars

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// aggFormsFrame has n rows in 3 groups "g", a Float64 "v" with every 7th row
// null, and a String "name".
func aggFormsFrame(t *testing.T, n int) DataFrame {
	t.Helper()
	g := make([]any, n)
	v := make([]any, n)
	name := make([]any, n)
	for i := range n {
		g[i] = int64(i % 3)
		if i%7 != 0 {
			v[i] = float64(i%11) + 0.5
		}
		name[i] = fmt.Sprintf("n%d", i%5)
	}
	return mscFrame(t, frame.SeriesInput{Name: "g", Values: g}, frame.SeriesInput{Name: "v", Values: v}, frame.SeriesInput{Name: "name", Values: name})
}

func TestAggMethodFormEqualsFunctionForm(t *testing.T) {
	t.Parallel()
	v, name := Col("v"), Col("name")
	tests := []struct {
		op             string
		method, called Expr
	}{
		{"sum", v.Sum(), expr.Sum(v)},
		{"mean", v.Mean(), expr.Mean(v)},
		{"min", v.Min(), expr.Min(v)},
		{"max", v.Max(), expr.Max(v)},
		{"first", name.First(), expr.First(name)},
		{"last", name.Last(), expr.Last(name)},
		{"n_unique", name.NUnique(), expr.NUnique(name)},
		{"median", v.Median(), expr.Median(v)},
		{"std", v.Std(), expr.Std(v)},
		{"var", v.Var(), expr.Var(v)},
	}
	for _, n := range []int{30, 40_000} {
		d := aggFormsFrame(t, n)
		for _, tt := range tests {
			want, err := d.GroupBy("g").Agg(tt.called)
			if err != nil {
				t.Fatalf("Agg(%s function form) on %d rows: %v", tt.op, n, err)
			}
			got, err := d.GroupBy("g").Agg(tt.method)
			if err != nil {
				t.Errorf("Agg(%s method form) on %d rows error = %v, want nil", tt.op, n, err)
				continue
			}
			if !slices.Equal(got.Columns(), want.Columns()) || !reflect.DeepEqual(got.ToDicts(), want.ToDicts()) {
				t.Errorf("Agg(%s method form) on %d rows = %v %v, want %v %v", tt.op, n, got.Columns(), got.Head(3).ToDicts(), want.Columns(), want.Head(3).ToDicts())
			}
			lazy, err := d.Lazy().GroupBy("g").Agg(tt.method).Collect(t.Context())
			if err != nil {
				t.Errorf("Lazy().GroupBy.Agg(%s method form) on %d rows error = %v, want nil", tt.op, n, err)
				continue
			}
			if !reflect.DeepEqual(lazy.ToDicts(), want.ToDicts()) {
				t.Errorf("Lazy().GroupBy.Agg(%s method form) on %d rows = %v, want %v", tt.op, n, lazy.Head(3).ToDicts(), want.Head(3).ToDicts())
			}
		}
	}
}

func TestAggMethodFormAliasAndCount(t *testing.T) {
	t.Parallel()
	d := mscFrame(t, mscCol("g", int64(1), int64(1), int64(2)), mscCol("name", "a", "x", "b"))
	got, err := d.GroupBy("g").Agg(Col("name").First().Alias("first_name"))
	if err != nil {
		t.Fatalf("Agg(name.First().Alias) error = %v", err)
	}
	if vals := frameColumnValues(t, got, "first_name"); !slices.Equal(vals, []any{"a", "b"}) {
		t.Errorf("Agg(name.First().Alias(first_name)) = %v, want [a b]", vals)
	}
	if _, err := d.GroupBy("g").Agg(Col("name").Count()); err == nil || !strings.Contains(err.Error(), "not aggregate") {
		t.Errorf("Agg(name.Count()) error = %v, want a not-aggregate error", err)
	}
}

func TestFacadeFirstLast(t *testing.T) {
	t.Parallel()
	d := mscFrame(t, mscCol("g", int64(1), int64(1), int64(1), int64(2), int64(2)), mscCol("v", int64(10), int64(20), int64(30), nil, int64(5)))
	got, err := d.GroupBy("g").Agg(First(Col("v")).Alias("first"), Last(Col("v")).Alias("last"))
	if err != nil {
		t.Fatalf("Agg(First, Last) error = %v", err)
	}
	if vals := frameColumnValues(t, got, "first"); !slices.Equal(vals, []any{int64(10), nil}) {
		t.Errorf("First(v) per group = %v, want [10 <nil>]", vals)
	}
	if vals := frameColumnValues(t, got, "last"); !slices.Equal(vals, []any{int64(30), int64(5)}) {
		t.Errorf("Last(v) per group = %v, want [30 5]", vals)
	}
}
