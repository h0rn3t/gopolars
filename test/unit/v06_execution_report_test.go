package unit

import (
	"context"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/exec"
	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/plan/logical"
)

func TestExecutionReportSchemaV2(t *testing.T) {
	source, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "ts", Values: []any{
				time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 10, 11, 0, 0, 0, time.UTC),
			}},
			{Name: "v", Values: []any{int64(1), int64(2)}},
		},
	})
	if err != nil {
		t.Fatalf("source build failed: %v", err)
	}
	nodes := []logical.Node{
		{
			Type:    logical.NodeRolling,
			Columns: []string{"ts", "v", "roll"},
			Strings: []string{(time.Hour).String(), "1", "both"},
		},
	}
	nodes[0].Strings[0] = "3600000000000"
	engine := exec.New()
	_, report, err := engine.ExecuteWithReport(context.Background(), source, nodes)
	if err != nil {
		t.Fatalf("execute with report failed: %v", err)
	}
	if report.SchemaVersion != "v2" {
		t.Fatalf("unexpected report schema")
	}
	if report.TemporalOps == 0 {
		t.Fatalf("expected temporal ops marker")
	}
	if report.MemoryBytes == 0 {
		t.Fatalf("expected memory marker")
	}
}
