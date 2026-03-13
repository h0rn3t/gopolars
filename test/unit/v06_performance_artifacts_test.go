package unit

import (
	"encoding/json"
	"os"
	"testing"
)

func TestV06PerformanceArtifactsExist(t *testing.T) {
	paths := []string{
		"../../docs/performance/v0_6_budgets.json",
		"../../docs/parity/v0_6_coverage.json",
		"../../docs/v0_6_migration.md",
		"../../docs/release_checklist_v0_6.md",
		"../../scripts/check_performance_budgets.sh",
		"../../scripts/perf_regression_report.sh",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected artifact missing: %s", p)
		}
	}
}

func TestV06PerformanceBudgetSchema(t *testing.T) {
	raw, err := os.ReadFile("../../docs/performance/v0_6_budgets.json")
	if err != nil {
		t.Fatalf("read budgets failed: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("invalid budgets json: %v", err)
	}
	if doc["version"] != "v0.6" {
		t.Fatalf("unexpected version in budgets")
	}
}
