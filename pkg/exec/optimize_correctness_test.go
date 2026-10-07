package exec

import (
	"context"
	"reflect"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// int64s boxes vs as []any, the form frame inputs and column values take.
func int64s(vs ...int64) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = v
	}
	return out
}

// TestExecuteMatchesWrittenOrder runs plans the optimizer used to rewrite
// incorrectly and checks that Execute returns both the expected values and the
// result of executing the nodes in the order they were written.
func TestExecuteMatchesWrittenOrder(t *testing.T) {
	t.Parallel()

	ab := func(t *testing.T) frame.DataFrame {
		return mustFrame(t,
			frame.SeriesInput{Name: "a", Values: int64s(5, 1, 4, 2, 3)},
			frame.SeriesInput{Name: "b", Values: int64s(10, 20, 30, 40, 50)},
		)
	}
	a, b := expr.Col("a"), expr.Col("b")
	tests := []struct {
		name   string
		source func(t *testing.T) frame.DataFrame
		nodes  []logical.Node
		want   map[string][]any
	}{
		{
			name:   "limit before sort",
			source: ab,
			nodes: []logical.Node{
				{Type: logical.NodeLimit, IntValue: 2},
				{Type: logical.NodeSort, Columns: []string{"a"}},
			},
			want: map[string][]any{"a": int64s(1, 5)},
		},
		{
			name:   "chained selects keep alias",
			source: ab,
			nodes: []logical.Node{
				{Type: logical.NodeSelect, Exprs: []expr.Expr{a.Alias("x")}},
				{Type: logical.NodeSelect, Exprs: []expr.Expr{expr.Col("x")}},
			},
			want: map[string][]any{"x": int64s(5, 1, 4, 2, 3)},
		},
		{
			name: "computed select is not collapsed",
			source: func(t *testing.T) frame.DataFrame {
				return mustFrame(t, frame.SeriesInput{Name: "a", Values: int64s(1, 2)})
			},
			nodes: []logical.Node{
				{Type: logical.NodeSelect, Exprs: []expr.Expr{a.Add(expr.Lit(int64(1))).Alias("a")}},
				{Type: logical.NodeSelect, Exprs: []expr.Expr{a}},
			},
			want: map[string][]any{"a": int64s(2, 3)},
		},
		{
			name:   "filter after renaming select",
			source: ab,
			nodes: []logical.Node{
				{Type: logical.NodeSelect, Exprs: []expr.Expr{a.Alias("b"), b.Alias("a")}},
				{Type: logical.NodeFilter, Exprs: []expr.Expr{a.Gt(expr.Lit(int64(25)))}},
			},
			want: map[string][]any{"b": int64s(4, 2, 3), "a": int64s(30, 40, 50)},
		},
		{
			name: "filter after aggregating select",
			source: func(t *testing.T) frame.DataFrame {
				return mustFrame(t, frame.SeriesInput{Name: "v", Values: int64s(1, 2, 3)})
			},
			nodes: []logical.Node{
				{Type: logical.NodeSelect, Exprs: []expr.Expr{expr.Col("v"), expr.Sum(expr.Col("v")).Alias("s")}},
				{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("v").Gt(expr.Lit(int64(1)))}},
			},
			want: map[string][]any{"v": int64s(2, 3), "s": int64s(6, 6)},
		},
		{
			name: "filter after row_number",
			source: func(t *testing.T) frame.DataFrame {
				return mustFrame(t, frame.SeriesInput{Name: "id", Values: int64s(1, 2, 3)})
			},
			nodes: []logical.Node{
				{Type: logical.NodeWindow, Windows: []logical.WindowSpec{{Func: "row_number", Alias: "rn", OrderBy: []string{"id"}}}},
				{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Gt(expr.Lit(int64(1)))}},
			},
			want: map[string][]any{"id": int64s(2, 3), "rn": int64s(2, 3)},
		},
		{
			name: "filter on partition keys after partitioned window",
			source: func(t *testing.T) frame.DataFrame {
				return mustFrame(t,
					frame.SeriesInput{Name: "p", Values: []any{"x", "y", "x", "y", "x"}},
					frame.SeriesInput{Name: "id", Values: int64s(5, 4, 3, 2, 1)},
				)
			},
			nodes: []logical.Node{
				{Type: logical.NodeWindow, Windows: []logical.WindowSpec{{Func: "row_number", Alias: "rn", PartitionBy: []string{"p"}, OrderBy: []string{"id"}}}},
				{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("p").Eq(expr.Lit("x"))}},
			},
			want: map[string][]any{"id": int64s(5, 3, 1), "rn": int64s(3, 2, 1)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := tt.source(t)
			got, err := New().Execute(context.Background(), source, tt.nodes)
			if err != nil {
				t.Fatalf("Execute(%s) error = %v", tt.name, err)
			}
			for name, want := range tt.want {
				if values := colValues(t, got, name); !reflect.DeepEqual(values, want) {
					t.Errorf("Execute(%s) column %s = %v, want %v", tt.name, name, values, want)
				}
			}
			written, err := executeOptimized(source, tt.nodes)
			if err != nil {
				t.Fatalf("executeOptimized(%s) error = %v", tt.name, err)
			}
			if eq, err := got.Equals(written); err != nil || !eq {
				t.Errorf("Execute(%s) = %v, want written-order result %v", tt.name, got.ToDicts(), written.ToDicts())
			}
		})
	}
}
