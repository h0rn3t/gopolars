package exec

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

func streamingSource(t *testing.T, n int) frame.DataFrame {
	t.Helper()
	ids := make([]any, n)
	vals := make([]any, n)
	for i := 0; i < n; i++ {
		ids[i] = int64(i)
		vals[i] = float64(n - i) // descending, so sort order is observable
	}
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: ids},
			{Name: "val", Values: vals},
		},
	})
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	return df
}

// TestExecuteStreamingEquivalence asserts ExecuteStreaming produces the same
// schema, height, and values as the standard Execute across both non-stateful
// plans (which take the split/concat streaming path) and a stateful plan (which
// falls back to whole-frame execution). chunkSize is below the source height so
// the streaming split is genuinely exercised.
func TestExecuteStreamingEquivalence(t *testing.T) {
	t.Parallel()

	source := streamingSource(t, 100)
	engine := New()
	ctx := context.Background()
	const chunk = 16

	plans := map[string][]logical.Node{
		"filter": {
			{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Lt(expr.Lit(int64(50)))}},
		},
		"projection": {
			{Type: logical.NodeSelect, Exprs: []expr.Expr{expr.Col("id")}},
		},
		"filter_then_limit": {
			{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Ge(expr.Lit(int64(10)))}},
			{Type: logical.NodeLimit, IntValue: 5},
		},
		"sort_stateful": {
			{Type: logical.NodeSort, Columns: []string{"val"}, Descending: []bool{false}},
		},
	}

	for name, nodes := range plans {
		t.Run(name, func(t *testing.T) {
			want, err := engine.Execute(ctx, source, nodes)
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			got, err := engine.ExecuteStreaming(ctx, source, nodes, chunk)
			if err != nil {
				t.Fatalf("execute streaming: %v", err)
			}
			eq, err := want.Equals(got)
			if err != nil {
				t.Fatalf("equals: %v", err)
			}
			if !eq {
				t.Fatalf("streaming differs from standard\n want=%v\n got=%v", want.ToDicts(), got.ToDicts())
			}
		})
	}
}

// TestExecuteStreamingFallsBackOnSmallInput covers the chunkSize >= height and
// chunkSize <= 0 short-circuits, which delegate straight to Execute.
func TestExecuteStreamingFallsBackOnSmallInput(t *testing.T) {
	t.Parallel()

	source := streamingSource(t, 8)
	engine := New()
	ctx := context.Background()
	nodes := []logical.Node{
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Lt(expr.Lit(int64(4)))}},
	}

	for _, chunk := range []int{0, 100} {
		got, err := engine.ExecuteStreaming(ctx, source, nodes, chunk)
		if err != nil {
			t.Fatalf("streaming chunk=%d: %v", chunk, err)
		}
		if got.Height() != 4 {
			t.Fatalf("chunk=%d height = %d, want 4", chunk, got.Height())
		}
	}
}
