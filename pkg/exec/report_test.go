package exec

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// TestExecuteWithReport verifies the report-producing entry point returns an
// output frame matching standard execution plus a populated report (operator
// list, node counts, source/output rows, stateful flag).
func TestExecuteWithReport(t *testing.T) {
	t.Parallel()

	source := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		frame.SeriesInput{Name: "grp", Values: []any{"a", "a", "b", "b"}},
	)
	engine := New()
	ctx := context.Background()

	nodes := []logical.Node{
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Ge(expr.Lit(int64(2)))}},
		{Type: logical.NodeAggregate, Columns: []string{"grp"}, Exprs: []expr.Expr{expr.Sum(expr.Col("id"))}},
	}

	out, report, err := engine.ExecuteWithReport(ctx, source, nodes)
	if err != nil {
		t.Fatalf("execute with report: %v", err)
	}

	// Output must equal standard execution.
	want, err := engine.Execute(ctx, source, nodes)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	eq, err := want.Equals(out)
	if err != nil || !eq {
		t.Fatalf("report output differs from standard (eq=%v err=%v)", eq, err)
	}

	// Report metadata.
	if report.SchemaVersion != "v2" {
		t.Fatalf("schema version = %q, want v2", report.SchemaVersion)
	}
	if report.SourceRows != 4 {
		t.Fatalf("source rows = %d, want 4", report.SourceRows)
	}
	if report.OutputRows != out.Height() {
		t.Fatalf("output rows = %d, want %d", report.OutputRows, out.Height())
	}
	if report.OptimizedNodes != len(report.Operators) || len(report.Operators) == 0 {
		t.Fatalf("operators=%v optimizedNodes=%d", report.Operators, report.OptimizedNodes)
	}
	if !report.StatefulPipeline {
		t.Fatal("aggregate pipeline should be flagged stateful")
	}
	if report.Metadata == nil {
		t.Fatal("metadata map should be initialized")
	}
}

// TestExecuteWithReportError surfaces execution errors while still returning a
// report.
func TestExecuteWithReportError(t *testing.T) {
	t.Parallel()

	source := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(1)}})
	engine := New()

	// A filter node with no expression is an executor error.
	_, report, err := engine.ExecuteWithReport(context.Background(), source, []logical.Node{
		{Type: logical.NodeFilter},
	})
	if err == nil {
		t.Fatal("expected execution error for empty filter")
	}
	// On error, OutputRows stays at its zero value.
	if report.OutputRows != 0 {
		t.Fatalf("output rows on error = %d, want 0", report.OutputRows)
	}
}
