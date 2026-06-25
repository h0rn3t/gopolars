package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type v10CoverageGate struct {
	Version      string                           `json:"version"`
	FullMatrix   v10CoverageGateSummary           `json:"full_matrix"`
	ReleaseScope v10CoverageGateSummary           `json:"release_scope"`
	Objects      map[string]v10CoverageGateObject `json:"objects"`
}

type v10CoverageGateSummary struct {
	ImplementedMethods int     `json:"implemented_methods"`
	TotalMethods       int     `json:"total_methods"`
	RemainingMethods   int     `json:"remaining_methods"`
	Coverage           float64 `json:"coverage"`
}

type v10CoverageGateObject struct {
	ReleaseScope v10CoverageGateSummary `json:"release_scope"`
}

func TestV10FinalParityGate(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "parity", "v1_0_coverage.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read v1.0 coverage failed: %v", err)
	}

	var report v10CoverageGate
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode v1.0 coverage failed: %v", err)
	}
	if report.Version != "v1.0" {
		t.Fatalf("unexpected coverage version: got %q want %q", report.Version, "v1.0")
	}
	if report.ReleaseScope.RemainingMethods != 0 {
		t.Fatalf("release-scope remaining methods: got %d want 0", report.ReleaseScope.RemainingMethods)
	}
	if report.ReleaseScope.TotalMethods == 0 || report.ReleaseScope.ImplementedMethods != report.ReleaseScope.TotalMethods {
		t.Fatalf(
			"release-scope totals mismatch: implemented=%d total=%d",
			report.ReleaseScope.ImplementedMethods,
			report.ReleaseScope.TotalMethods,
		)
	}
	if report.ReleaseScope.Coverage < 1.0 {
		t.Fatalf("release-scope coverage below 1.0: got %.6f", report.ReleaseScope.Coverage)
	}

	for _, name := range []string{"DataFrame", "LazyFrame", "Expr", "Series"} {
		object, ok := report.Objects[name]
		if !ok {
			t.Fatalf("missing object coverage for %s", name)
		}
		if object.ReleaseScope.RemainingMethods != 0 {
			t.Fatalf("%s release-scope remaining methods: got %d want 0", name, object.ReleaseScope.RemainingMethods)
		}
	}
}
