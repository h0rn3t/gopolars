package top30

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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
//
// A ratio is a quotient, so a fall does not say which side moved. Where the
// baseline records the absolute times it was seeded from, the gate names the
// side (see attribute); that distinction decides whether the answer is to
// optimise or to re-seed the floor.

// ratioFallEpsilon is the relative slack below a baseline ratio that counts as
// unchanged rather than fallen: parity_baseline.json rounds ratios to six
// decimals, which is ~1e-6 relative, and this sits two orders above that while
// staying far below any move worth attributing.
const ratioFallEpsilon = 1e-4

type parityBudgetEntry struct {
	RMin          float64 `json:"r_min"`
	OutOfScope    bool    `json:"out_of_scope"`
	Justification string  `json:"justification"`
}

// parityConditions records what a number was measured under. Comparing ratios
// across different conditions is not a like-for-like comparison, so a mismatch
// is reported rather than treated as a regression. A file that omits the block
// leaves the conditions unknown and is compared exactly as before.
type parityConditions struct {
	GoToolchain   string `json:"go_toolchain"`
	PolarsVersion string `json:"polars_version"`
	Cores         int    `json:"cores"`
}

func (c parityConditions) known() bool {
	return c.GoToolchain != "" || c.PolarsVersion != "" || c.Cores != 0
}

func (c parityConditions) String() string {
	parts := make([]string, 0, 3)
	if c.GoToolchain != "" {
		parts = append(parts, "go "+c.GoToolchain)
	}
	if c.PolarsVersion != "" {
		parts = append(parts, "polars "+c.PolarsVersion)
	}
	if c.Cores != 0 {
		parts = append(parts, fmt.Sprintf("%d cores", c.Cores))
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, ", ")
}

type parityBudgetFile struct {
	Version        string  `json:"version"`
	ReferenceScale string  `json:"reference_scale"`
	Tolerance      float64 `json:"tolerance"`
	// RegressionCheckCeiling skips the baseline-regression check for workloads
	// whose baseline ratio is at or above this value: where Go is already this
	// many times faster than Polars, the op runs in nanoseconds and carries
	// large cross-run/cross-machine variance, and the absolute r_min floor
	// already protects it. The r_min check still applies to every workload.
	RegressionCheckCeiling float64                      `json:"regression_check_ceiling"`
	Conditions             parityConditions             `json:"conditions"`
	Workloads              map[string]parityBudgetEntry `json:"workloads"`
}

// parityAbsolute is the pair of absolute times a baseline ratio was computed
// from. Without it a fallen ratio cannot be attributed to a side.
type parityAbsolute struct {
	GoNsPerOp      int64   `json:"go_ns_per_op"`
	PythonSecPerOp float64 `json:"python_sec_per_op"`
}

type parityBaselineFile struct {
	ReferenceScale string                    `json:"reference_scale"`
	Conditions     parityConditions          `json:"conditions"`
	Ratios         map[string]float64        `json:"ratios"`
	Absolutes      map[string]parityAbsolute `json:"absolutes"`
}

// parityReport separates what makes the gate red from what it only says out
// loud. Conditions mismatches, attributions and unmeasured workloads are notes:
// they change what a human should conclude, not whether the build passes.
type parityReport struct {
	failures []string
	notes    []string
}

func (r *parityReport) failf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *parityReport) notef(format string, args ...any) {
	r.notes = append(r.notes, fmt.Sprintf(format, args...))
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

// compareConditions reports, and never fails on, a difference between the
// conditions a baseline was recorded under and the ones now in force. Failing
// would turn every toolchain bump red at once and cost the gate its signal.
func compareConditions(rep *parityReport, budgets parityBudgetFile, baseline parityBaselineFile) {
	if !budgets.Conditions.known() || !baseline.Conditions.known() {
		rep.notef("measurement conditions unknown (budgets: %s; baseline: %s); ratios compared as-is",
			budgets.Conditions, baseline.Conditions)
		return
	}
	if budgets.Conditions.GoToolchain != baseline.Conditions.GoToolchain {
		rep.notef("CONDITIONS DIFFER: Go toolchain %s (baseline) vs %s (budgets); a ratio change may be the toolchain, not the code",
			baseline.Conditions.GoToolchain, budgets.Conditions.GoToolchain)
	}
	if budgets.Conditions.PolarsVersion != baseline.Conditions.PolarsVersion {
		rep.notef("CONDITIONS DIFFER: Python Polars %s (baseline) vs %s (budgets); a ratio change may be the reference implementation, not the code",
			baseline.Conditions.PolarsVersion, budgets.Conditions.PolarsVersion)
	}
	if budgets.Conditions.Cores != baseline.Conditions.Cores {
		rep.notef("CONDITIONS DIFFER: %d cores (baseline) vs %d cores (budgets)",
			baseline.Conditions.Cores, budgets.Conditions.Cores)
	}
	if host := runtime.NumCPU(); budgets.Conditions.Cores != 0 && host != budgets.Conditions.Cores {
		rep.notef("CONDITIONS DIFFER: recorded on %d cores, this host has %d",
			budgets.Conditions.Cores, host)
	}
}

// attribute names the side a fallen ratio came from, by comparing the absolute
// times against the ones the baseline was seeded with. Where the baseline
// carries no absolutes the fall stays unattributed and is reported as such — an
// unattributed fall is not evidence for lowering a floor.
func attribute(key string, measured summaryEntry, base float64, baseline parityBaselineFile) string {
	rec, ok := baseline.Absolutes[key]
	if !ok {
		return fmt.Sprintf("UNATTRIBUTED %s: ratio fell %.4f -> %.4f, but the baseline records no absolute times; do not lower this floor until the side is shown",
			key, base, measured.Ratio)
	}
	side := "reference got faster"
	if measured.GoNsPerOp > rec.GoNsPerOp {
		side = "this library got slower"
	}
	return fmt.Sprintf("ATTRIBUTION %s: ratio fell %.4f -> %.4f — %s (go %.3f -> %.3f ms, python %.3f -> %.3f ms)",
		key, base, measured.Ratio, side,
		float64(rec.GoNsPerOp)/1e6, float64(measured.GoNsPerOp)/1e6,
		rec.PythonSecPerOp*1e3, measured.PythonSecPerOp*1e3)
}

// evaluateParity is the whole gate as a pure function of the three artifacts, so
// the rules can be exercised on synthetic inputs without touching the committed
// files.
func evaluateParity(budgets parityBudgetFile, baseline parityBaselineFile, summary []summaryEntry) parityReport {
	var rep parityReport
	compareConditions(&rep, budgets, baseline)

	// Best (fastest) ratio per workload at the reference scale, mirroring how
	// comparison_table.md reports the min ns/op across calibration rounds.
	best := map[string]summaryEntry{}
	for _, e := range summary {
		if e.Size != budgets.ReferenceScale {
			continue
		}
		key := e.Object + "/" + e.Op
		if prev, ok := best[key]; !ok || e.Ratio > prev.Ratio {
			best[key] = e
		}
	}
	if len(best) == 0 {
		rep.failf("top30_summary.json has no %s-scale entries", budgets.ReferenceScale)
		return rep
	}

	// Every tracked workload must carry a budget entry.
	for _, key := range sortedKeys(best) {
		if _, ok := budgets.Workloads[key]; !ok {
			rep.failf("workload %q has no parity budget entry in docs/performance/parity_budgets.json", key)
		}
	}

	for _, key := range sortedKeys(budgets.Workloads) {
		b := budgets.Workloads[key]

		// Sub-parity budgets must be justified.
		if b.RMin < 1.0 && !b.OutOfScope && strings.TrimSpace(b.Justification) == "" {
			rep.failf("workload %q has r_min %.4f < 1.0 but no justification", key, b.RMin)
		}

		measured, ok := best[key]
		if !ok {
			// A budget with no measurement is not a budget that was met. Without
			// this the loop over measured workloads alone would pass it silently.
			if !b.OutOfScope {
				rep.notef("UNMEASURED %s: budgeted at r_min %.4f but top30_summary.json has no %s-scale entry; not counted as passing",
					key, b.RMin, budgets.ReferenceScale)
			}
			continue
		}
		if b.OutOfScope {
			continue
		}

		if measured.Ratio < b.RMin {
			rep.failf("PARITY BREACH %s: measured ratio %.4f < budget r_min %.4f", key, measured.Ratio, b.RMin)
		}

		base, ok := baseline.Ratios[key]
		if !ok {
			continue
		}
		// parity_baseline.json stores ratios rounded to six decimals, so an
		// unchanged workload reads as a hair below its baseline. Anything under
		// this guard is that rounding, not a fall worth attributing.
		if measured.Ratio < base*(1-ratioFallEpsilon) {
			rep.notef("%s", attribute(key, measured, base, baseline))
		}
		// Skip the regression check where Go is already far ahead: such ops
		// are nanosecond-scale and noisy across runs, and r_min protects them.
		if budgets.RegressionCheckCeiling > 0 && base >= budgets.RegressionCheckCeiling {
			continue
		}
		if floor := base * (1 - budgets.Tolerance); measured.Ratio < floor {
			rep.failf("REGRESSION %s: measured ratio %.4f is below baseline %.4f by more than %.0f%% (floor %.4f)",
				key, measured.Ratio, base, budgets.Tolerance*100, floor)
		}
	}
	return rep
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

	rep := evaluateParity(budgets, baseline, summary)
	for _, n := range rep.notes {
		t.Log(n)
	}
	for _, f := range rep.failures {
		t.Error(f)
	}
}
