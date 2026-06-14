package top30

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The parity gate enforces the "performance parity on all workloads" contract:
// every workload tracked by bench/top30 at the reference scale must run within
// its budgeted ratio versus Python Polars and must not have regressed below its
// committed baseline. It reads the committed top30_summary.json (no Python or
// benchmark run required), so it executes as part of the normal `go test ./...`
// gate on every push and locks the recorded performance in place.
//
// ratio = python_time / go_time, so a higher ratio is faster; r_min is the
// minimum acceptable ratio. Seed/ratchet the budgets via ./run-bench.sh
// --python, which regenerates top30_summary.json.

type parityBudgetEntry struct {
	RMin          float64 `json:"r_min"`
	OutOfScope    bool    `json:"out_of_scope"`
	Justification string  `json:"justification"`
}

type parityBudgetFile struct {
	Version        string                       `json:"version"`
	ReferenceScale string                       `json:"reference_scale"`
	Tolerance      float64                      `json:"tolerance"`
	// RegressionCheckCeiling skips the baseline-regression check for workloads
	// whose baseline ratio is at or above this value: where Go is already this
	// many times faster than Polars, the op runs in nanoseconds and carries
	// large cross-run/cross-machine variance, and the absolute r_min floor
	// already protects it. The r_min check still applies to every workload.
	RegressionCheckCeiling float64                      `json:"regression_check_ceiling"`
	Workloads              map[string]parityBudgetEntry `json:"workloads"`
}

type parityBaselineFile struct {
	ReferenceScale string             `json:"reference_scale"`
	Ratios         map[string]float64 `json:"ratios"`
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}

func TestParityBudgets(t *testing.T) {
	var budgets parityBudgetFile
	readJSON(t, filepath.Join("..", "..", "docs", "performance", "parity_budgets.json"), &budgets)
	if len(budgets.Workloads) == 0 {
		t.Fatal("no workloads in parity_budgets.json")
	}

	var baseline parityBaselineFile
	readJSON(t, filepath.Join("baselines", "parity_baseline.json"), &baseline)

	var summary []summaryEntry
	readJSON(t, "top30_summary.json", &summary)

	// Best (fastest) ratio per workload at the reference scale, mirroring how
	// comparison_table.md reports the min ns/op across calibration rounds.
	best := map[string]float64{}
	for _, e := range summary {
		if e.Size != budgets.ReferenceScale {
			continue
		}
		key := e.Object + "/" + e.Op
		if r, ok := best[key]; !ok || e.Ratio > r {
			best[key] = e.Ratio
		}
	}
	if len(best) == 0 {
		t.Fatalf("top30_summary.json has no %s-scale entries", budgets.ReferenceScale)
	}

	// Every tracked workload must carry a budget entry.
	for key := range best {
		if _, ok := budgets.Workloads[key]; !ok {
			t.Errorf("workload %q has no parity budget entry in docs/performance/parity_budgets.json", key)
		}
	}

	// Sub-parity budgets must be justified.
	for key, b := range budgets.Workloads {
		if b.RMin < 1.0 && !b.OutOfScope && strings.TrimSpace(b.Justification) == "" {
			t.Errorf("workload %q has r_min %.4f < 1.0 but no justification", key, b.RMin)
		}
	}

	// Budget breach + regression checks.
	for key, measured := range best {
		b, ok := budgets.Workloads[key]
		if !ok || b.OutOfScope {
			continue
		}
		if measured < b.RMin {
			t.Errorf("PARITY BREACH %s: measured ratio %.4f < budget r_min %.4f", key, measured, b.RMin)
		}
		if base, ok := baseline.Ratios[key]; ok {
			// Skip the regression check where Go is already far ahead: such ops
			// are nanosecond-scale and noisy across runs, and r_min protects them.
			if budgets.RegressionCheckCeiling > 0 && base >= budgets.RegressionCheckCeiling {
				continue
			}
			floor := base * (1 - budgets.Tolerance)
			if measured < floor {
				t.Errorf("REGRESSION %s: measured ratio %.4f is below baseline %.4f by more than %.0f%% (floor %.4f)",
					key, measured, base, budgets.Tolerance*100, floor)
			}
		}
	}
}
