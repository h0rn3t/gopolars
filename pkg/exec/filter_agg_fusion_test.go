package exec

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// TestFilterFrameAggFusionParity verifies the lazy executor's filter+frame-agg
// fusion (fuseFilterFrameAgg + the FilterAggregate dispatch) produces results
// identical to running the filter and the aggregation as separate, unfused
// executions — for both fused-supported ops and an op that must fall back
// (median). The size is above parallelFilterThreshold so the parallel fused
// reduction is exercised through the engine.
func TestFilterFrameAggFusionParity(t *testing.T) {
	t.Parallel()

	const n = 50_000
	vals := make([]any, n)
	for i := range vals {
		if i%9 == 0 {
			vals[i] = nil
		} else {
			vals[i] = float64((i % 201) - 100) // spans [-100, 100], ~50% > 0
		}
	}
	source, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{{Name: "a", Values: vals}},
	})
	if err != nil {
		t.Fatalf("source: %v", err)
	}

	engine := New()
	ctx := context.Background()
	pred := expr.Col("a").Gt(expr.Lit(0.0))

	// median is not handled by the fused path -> exercises the fallback branch.
	for _, op := range []string{"sum", "min", "max", "count", "mean", "median"} {
		t.Run(op, func(t *testing.T) {
			// Reference: filter and aggregate as two separate single-node plans,
			// so no adjacency exists for fuseFilterFrameAgg to rewrite.
			filtered, err := engine.Execute(ctx, source, []logical.Node{
				{Type: logical.NodeFilter, Exprs: []expr.Expr{pred}},
			})
			if err != nil {
				t.Fatalf("reference filter: %v", err)
			}
			ref, err := engine.Execute(ctx, filtered, []logical.Node{
				{Type: logical.NodeFrameAgg, Strings: []string{op}},
			})
			if err != nil {
				t.Fatalf("reference agg: %v", err)
			}

			// Fused: the combined plan triggers fusion in executeOptimized.
			fused, err := engine.Execute(ctx, source, []logical.Node{
				{Type: logical.NodeFilter, Exprs: []expr.Expr{pred}},
				{Type: logical.NodeFrameAgg, Strings: []string{op}},
			})
			if err != nil {
				t.Fatalf("fused: %v", err)
			}

			eq, err := ref.Equals(fused)
			if err != nil {
				t.Fatalf("equals: %v", err)
			}
			if !eq {
				t.Fatalf("op %s: fused result differs from reference\n ref=%v\n fused=%v",
					op, ref.ToDicts(), fused.ToDicts())
			}
		})
	}
}
