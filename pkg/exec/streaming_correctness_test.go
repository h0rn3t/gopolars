package exec

import (
	"context"
	"fmt"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// TestExecuteStreamingMatchesExecute checks that ExecuteStreaming returns what
// Execute returns for plans whose result for a row depends on other rows, and
// for row-local plans, at chunk sizes smaller than the input.
func TestExecuteStreamingMatchesExecute(t *testing.T) {
	t.Parallel()

	source := func(t *testing.T) frame.DataFrame {
		return mustFrame(t,
			frame.SeriesInput{Name: "a", Values: int64s(5, 1, 4, 2, 3)},
			frame.SeriesInput{Name: "b", Values: int64s(10, 20, 30, 40, 50)},
			frame.SeriesInput{Name: "g", Values: int64s(1, 1, 2, 1, 2)},
		)
	}
	a := expr.Col("a")
	aGt1 := a.Gt(expr.Lit(int64(1)))
	tests := []struct {
		name    string
		source  func(t *testing.T) frame.DataFrame
		nodes   []logical.Node
		want    map[string]string // column -> fmt.Sprint of its values
		wantErr bool              // Execute fails, so ExecuteStreaming must fail too
	}{
		{
			name:  "row index",
			nodes: []logical.Node{{Type: logical.NodeWithRowIdx, Strings: []string{"i", "0"}}},
			want:  map[string]string{"i": "[0 1 2 3 4]"},
		},
		{
			name:  "tail",
			nodes: []logical.Node{{Type: logical.NodeTail, IntValue: 2}},
			want:  map[string]string{"a": "[2 3]"},
		},
		{
			name: "limit then filter",
			nodes: []logical.Node{
				{Type: logical.NodeLimit, IntValue: 3},
				{Type: logical.NodeFilter, Exprs: []expr.Expr{aGt1}},
			},
			want: map[string]string{"a": "[5 4]"},
		},
		{
			name: "whole-column expressions",
			source: func(t *testing.T) frame.DataFrame {
				return mustFrame(t, frame.SeriesInput{Name: "a", Values: int64s(1, 2, 3, 4)})
			},
			nodes: []logical.Node{{Type: logical.NodeWithCols, Exprs: []expr.Expr{a.CumSum().Alias("c"), expr.Sum(a).Alias("s")}}},
			want:  map[string]string{"c": "[1 3 6 10]", "s": "[10 10 10 10]"},
		},
		{
			name:  "slice",
			nodes: []logical.Node{{Type: logical.NodeSlice, IntValue: 1, Strings: []string{"3"}}},
			want:  map[string]string{"a": "[1 4 2]"},
		},
		{
			name:  "gather every",
			nodes: []logical.Node{{Type: logical.NodeGatherEvery, Strings: []string{"2"}}},
			want:  map[string]string{"a": "[5 4 3]"},
		},
		{
			name:  "reverse",
			nodes: []logical.Node{{Type: logical.NodeReverse}},
			want:  map[string]string{"a": "[3 2 4 1 5]"},
		},
		{
			name:  "unique",
			nodes: []logical.Node{{Type: logical.NodeUnique, Columns: []string{"g"}}},
			want:  map[string]string{"g": "[1 2]"},
		},
		{
			name:  "shift",
			nodes: []logical.Node{{Type: logical.NodeShift, IntValue: 1}},
			want:  map[string]string{"a": "[<nil> 5 1 4 2]"},
		},
		{
			name: "interpolate",
			source: func(t *testing.T) frame.DataFrame {
				return mustFrame(t, frame.SeriesInput{Name: "f", Values: []any{1.0, nil, 3.0, nil, 5.0}})
			},
			nodes: []logical.Node{{Type: logical.NodeInterpolate, Columns: []string{"f"}}},
			want:  map[string]string{"f": "[1 2 3 4 5]"},
		},
		{
			name:  "melt",
			nodes: []logical.Node{{Type: logical.NodeMelt, Columns: []string{"g", "a", "b"}, Strings: []string{"variable", "value", "1"}}},
		},
		{
			name:  "frame aggregation",
			nodes: []logical.Node{{Type: logical.NodeFrameAgg, Strings: []string{"max"}}},
			want:  map[string]string{"a": "[5]"},
		},
		{
			name:  "window expression in select",
			nodes: []logical.Node{{Type: logical.NodeSelect, Exprs: []expr.Expr{a.CumCount().Over("g").Alias("n")}}},
			want:  map[string]string{"n": "[1 2 1 3 2]"},
		},
		{
			name:  "sort",
			nodes: []logical.Node{{Type: logical.NodeSort, Columns: []string{"a"}}},
			want:  map[string]string{"a": "[1 2 3 4 5]"},
		},
		{
			name: "row-local plan",
			nodes: []logical.Node{
				{Type: logical.NodeFilter, Exprs: []expr.Expr{aGt1}},
				{Type: logical.NodeSelect, Exprs: []expr.Expr{a.Mul(expr.Lit(int64(2))).Alias("d")}},
			},
			want: map[string]string{"d": "[10 8 4 6]"},
		},
		{
			name: "computed column all null in a chunk",
			source: func(t *testing.T) frame.DataFrame {
				return mustFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1), nil, int64(3)}})
			},
			nodes: []logical.Node{{Type: logical.NodeWithCols, Exprs: []expr.Expr{a.Add(expr.Lit(int64(1))).Alias("b")}}},
			want:  map[string]string{"b": "[2 <nil> 4]"},
		},
		{
			name: "struct fields differ between chunks",
			source: func(t *testing.T) frame.DataFrame {
				return mustFrame(t, frame.SeriesInput{Name: "s", Values: []any{
					map[string]any{"x": int64(1), "y": int64(2)}, map[string]any{"x": int64(3)}, map[string]any{"y": int64(4)},
				}})
			},
			nodes: []logical.Node{{Type: logical.NodeUnnest, Columns: []string{"s"}}},
			want:  map[string]string{"x": "[1 3 <nil>]", "y": "[2 <nil> 4]"},
		},
		{
			name: "chunk dtypes differ from the whole input",
			nodes: []logical.Node{{Type: logical.NodeWithCols, Exprs: []expr.Expr{
				expr.When(a.Gt(expr.Lit(int64(4))), expr.Lit(2.5), expr.Lit(int64(7))).Alias("w"),
			}}},
			wantErr: true,
		},
		{
			name: "row-local plan with trailing limits",
			nodes: []logical.Node{
				{Type: logical.NodeFilter, Exprs: []expr.Expr{aGt1}},
				{Type: logical.NodeLimit, IntValue: 3},
				{Type: logical.NodeLimit, IntValue: 2},
			},
			want: map[string]string{"a": "[5 4]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := source
			if tt.source != nil {
				src = tt.source
			}
			df := src(t)
			ctx := context.Background()
			want, err := New().Execute(ctx, df, tt.nodes)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Execute(%s) error = nil, want an error", tt.name)
				}
				for _, chunk := range []int{1, 2, 3} {
					if got, err := New().ExecuteStreaming(ctx, df, tt.nodes, chunk); err == nil {
						t.Errorf("ExecuteStreaming(%s, %d) = %v, want an error like Execute", tt.name, chunk, got.ToDicts())
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute(%s) error = %v", tt.name, err)
			}
			for name, values := range tt.want {
				if got := fmt.Sprint(colValues(t, want, name)); got != values {
					t.Errorf("Execute(%s) column %s = %s, want %s", tt.name, name, got, values)
				}
			}
			for _, chunk := range []int{1, 2, 3} {
				got, err := New().ExecuteStreaming(ctx, df, tt.nodes, chunk)
				if err != nil {
					t.Fatalf("ExecuteStreaming(%s, %d) error = %v", tt.name, chunk, err)
				}
				if eq, err := want.Equals(got); err != nil || !eq {
					t.Errorf("ExecuteStreaming(%s, %d) = %v, want %v", tt.name, chunk, got.ToDicts(), want.ToDicts())
				}
			}
		})
	}
}
