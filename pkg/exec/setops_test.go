package exec

import (
	"context"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/expr"
	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/plan/logical"
)

func TestSetOpsIntersectAndExcept(t *testing.T) {
	source, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "city", Values: []any{"kyiv", "lviv", "odesa"}},
		},
	})
	if err != nil {
		t.Fatalf("побудова dataframe: %v", err)
	}

	engine := New()
	intersectNodes := []logical.Node{
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Le(expr.Lit(int64(2)))}},
		{
			Type:    logical.NodeSetOp,
			Strings: []string{"intersect"},
			Plan: []logical.Node{
				{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Ge(expr.Lit(int64(2)))}},
			},
		},
	}
	intersected, err := engine.Execute(context.Background(), source, intersectNodes)
	if err != nil {
		t.Fatalf("intersect: %v", err)
	}
	if intersected.Height() != 1 {
		t.Fatalf("intersect: очікували 1 рядок, отримали %d", intersected.Height())
	}
	idCol, _ := intersected.Series("id")
	if idCol.Value(0) != int64(2) {
		t.Fatalf("intersect: неочікуваний id %v", idCol.Value(0))
	}

	exceptNodes := []logical.Node{
		{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Le(expr.Lit(int64(2)))}},
		{
			Type:    logical.NodeSetOp,
			Strings: []string{"except"},
			Plan: []logical.Node{
				{Type: logical.NodeFilter, Exprs: []expr.Expr{expr.Col("id").Ge(expr.Lit(int64(2)))}},
			},
		},
	}
	excepted, err := engine.Execute(context.Background(), source, exceptNodes)
	if err != nil {
		t.Fatalf("except: %v", err)
	}
	if excepted.Height() != 1 {
		t.Fatalf("except: очікували 1 рядок, отримали %d", excepted.Height())
	}
	idCol, _ = excepted.Series("id")
	if idCol.Value(0) != int64(1) {
		t.Fatalf("except: неочікуваний id %v", idCol.Value(0))
	}
}

func TestFrameAggMaxMinAndScheduler(t *testing.T) {
	ts := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	source, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "label", Values: []any{"b", "a", "c"}},
			{Name: "ts", Values: []any{ts, ts.Add(time.Hour), ts.Add(2 * time.Hour)}},
		},
	})
	if err != nil {
		t.Fatalf("побудова dataframe: %v", err)
	}

	engine := New()
	maxOut, err := engine.Execute(context.Background(), source, []logical.Node{
		{Type: logical.NodeFrameAgg, Strings: []string{"max"}},
	})
	if err != nil {
		t.Fatalf("frame max: %v", err)
	}
	label, _ := maxOut.Series("label")
	if label.Value(0) != "c" {
		t.Fatalf("max label: %v", label.Value(0))
	}

	minOut, err := engine.Execute(context.Background(), source, []logical.Node{
		{Type: logical.NodeFrameAgg, Strings: []string{"min"}},
	})
	if err != nil {
		t.Fatalf("frame min: %v", err)
	}
	label, _ = minOut.Series("label")
	if label.Value(0) != "a" {
		t.Fatalf("min label: %v", label.Value(0))
	}

	sched := NewScheduler()
	if sched.Workers <= 0 {
		t.Fatal("scheduler workers має бути > 0")
	}
}
