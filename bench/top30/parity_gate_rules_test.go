package top30

import (
	"encoding/json"
	"strings"
	"testing"
)

// These exercise the gate's rules on synthetic artifacts. TestParityBudgets runs
// the same evaluator against the committed files; here the inputs are built so
// each rule can be triggered on its own.

func entry(object, op string, goNs int64, pySec float64) summaryEntry {
	return summaryEntry{
		Object:         object,
		Op:             op,
		Size:           "1M",
		Elements:       1_000_000,
		GoNsPerOp:      goNs,
		PythonSecPerOp: pySec,
		Ratio:          pySec / (float64(goNs) / 1e9),
	}
}

func budgetFile(conds parityConditions, workloads map[string]parityBudgetEntry) parityBudgetFile {
	return parityBudgetFile{
		Version:                "test",
		ReferenceScale:         "1M",
		Tolerance:              0.4,
		RegressionCheckCeiling: 3.0,
		Conditions:             conds,
		Workloads:              workloads,
	}
}

func hasLine(lines []string, want string) bool {
	for _, l := range lines {
		if strings.Contains(l, want) {
			return true
		}
	}
	return false
}

// Requirement: a recorded budget carries the conditions it was measured under —
// and a file predating the block still parses and still gates.
func TestParityConditionsBothFileForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		budgetsRaw string
		wantKnown  bool
		wantCores  int
	}{
		{
			name: "with conditions block",
			budgetsRaw: `{"version":"v1","reference_scale":"1M","tolerance":0.4,
				"conditions":{"go_toolchain":"1.27.0","polars_version":"1.27.1","cores":12},
				"workloads":{"Expr/rank":{"r_min":1.0}}}`,
			wantKnown: true,
			wantCores: 12,
		},
		{
			name: "without conditions block (legacy form)",
			budgetsRaw: `{"version":"v1","reference_scale":"1M","tolerance":0.4,
				"workloads":{"Expr/rank":{"r_min":1.0}}}`,
			wantKnown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var budgets parityBudgetFile
			if err := json.Unmarshal([]byte(tt.budgetsRaw), &budgets); err != nil {
				t.Fatalf("parse budgets: %v", err)
			}
			if got := budgets.Conditions.known(); got != tt.wantKnown {
				t.Errorf("conditions known = %v, want %v", got, tt.wantKnown)
			}
			if got := budgets.Conditions.Cores; got != tt.wantCores {
				t.Errorf("cores = %d, want %d", got, tt.wantCores)
			}

			// Either form must still evaluate, and a met budget must stay green.
			summary := []summaryEntry{entry("Expr", "rank", 7_000_000, 0.0140)}
			rep := evaluateParity(budgets, parityBaselineFile{ReferenceScale: "1M"}, summary)
			if len(rep.failures) != 0 {
				t.Errorf("legacy/new form must not fail on a met budget, got %v", rep.failures)
			}
		})
	}
}

// Requirement: a toolchain or Polars change is reported as such, not as a
// regression — while the absolute r_min floor keeps failing exactly as before.
func TestParityConditionsMismatchReportsAndStillEnforcesFloor(t *testing.T) {
	t.Parallel()

	budgets := budgetFile(
		parityConditions{GoToolchain: "1.27.0", PolarsVersion: "1.27.1", Cores: 12},
		map[string]parityBudgetEntry{
			"Expr/rolling_mean": {RMin: 1.3277},
		},
	)
	baseline := parityBaselineFile{
		ReferenceScale: "1M",
		Conditions:     parityConditions{GoToolchain: "1.26.1", PolarsVersion: "1.26.0", Cores: 12},
		Ratios:         map[string]float64{"Expr/rolling_mean": 2.043},
	}
	// Ratio 0.869: below both the baseline and the r_min floor.
	summary := []summaryEntry{entry("Expr", "rolling_mean", 12_580_000, 0.01093)}

	rep := evaluateParity(budgets, baseline, summary)

	if !hasLine(rep.notes, "CONDITIONS DIFFER") {
		t.Errorf("want a conditions note, got notes %v", rep.notes)
	}
	if !hasLine(rep.notes, "1.26.1") || !hasLine(rep.notes, "1.27.0") {
		t.Errorf("conditions note must name both toolchains, got %v", rep.notes)
	}
	if !hasLine(rep.notes, "1.26.0") || !hasLine(rep.notes, "1.27.1") {
		t.Errorf("conditions note must name both Polars versions, got %v", rep.notes)
	}
	if hasLine(rep.failures, "CONDITIONS") {
		t.Errorf("a conditions difference must not fail the gate, got %v", rep.failures)
	}
	if !hasLine(rep.failures, "PARITY BREACH Expr/rolling_mean") {
		t.Errorf("the absolute r_min floor must still fail, got %v", rep.failures)
	}
}

// Requirement: a fallen ratio is attributed to a side before it is acted on.
func TestParityAttributesFallenRatioToASide(t *testing.T) {
	t.Parallel()

	// Recorded: go 0.48 ms, python 0.91 ms -> ratio 1.891 (the real fill_null case).
	const (
		recGoNs  = 480_000
		recPySec = 0.00091
	)

	tests := []struct {
		name         string
		measured     summaryEntry
		withAbsolute bool
		want         string
	}{
		{
			name: "reference got faster: Go improved while the ratio fell",
			// go 0.48 -> 0.35 ms (faster), python 0.91 -> 0.36 ms (faster still).
			measured:     entry("DataFrame", "fill_null", 350_000, 0.00036),
			withAbsolute: true,
			want:         "reference got faster",
		},
		{
			name:         "this library got slower",
			measured:     entry("DataFrame", "fill_null", 900_000, 0.00091),
			withAbsolute: true,
			want:         "this library got slower",
		},
		{
			name:         "no recorded absolutes leaves the fall unattributed",
			measured:     entry("DataFrame", "fill_null", 350_000, 0.00036),
			withAbsolute: false,
			want:         "UNATTRIBUTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			budgets := budgetFile(parityConditions{}, map[string]parityBudgetEntry{
				"DataFrame/fill_null": {RMin: 0.5, Justification: "test"},
			})
			baseline := parityBaselineFile{
				ReferenceScale: "1M",
				Ratios:         map[string]float64{"DataFrame/fill_null": 1.891},
			}
			if tt.withAbsolute {
				baseline.Absolutes = map[string]parityAbsolute{
					"DataFrame/fill_null": {GoNsPerOp: recGoNs, PythonSecPerOp: recPySec},
				}
			}

			rep := evaluateParity(budgets, baseline, []summaryEntry{tt.measured})

			if !hasLine(rep.notes, tt.want) {
				t.Errorf("want a note containing %q, got %v", tt.want, rep.notes)
			}
		})
	}
}

// Requirement: the gate distinguishes an unmeasured workload from a passing one.
func TestParityReportsUnmeasuredWorkloads(t *testing.T) {
	t.Parallel()

	budgets := budgetFile(parityConditions{}, map[string]parityBudgetEntry{
		"Expr/rank":        {RMin: 1.0},
		"Expr/rolling_std": {RMin: 1.0},
		"DataFrame/vstack": {RMin: 0.0, OutOfScope: true, Justification: "not calibrated"},
	})
	// Only Expr/rank was measured at the reference scale; the 1K row for
	// rolling_std must not count as a measurement either.
	summary := []summaryEntry{
		entry("Expr", "rank", 7_000_000, 0.0140),
		func() summaryEntry { e := entry("Expr", "rolling_std", 7_000, 0.000014); e.Size = "1K"; return e }(),
	}

	rep := evaluateParity(budgets, parityBaselineFile{ReferenceScale: "1M"}, summary)

	if !hasLine(rep.notes, "UNMEASURED Expr/rolling_std") {
		t.Errorf("want an unmeasured note for Expr/rolling_std, got %v", rep.notes)
	}
	if hasLine(rep.notes, "UNMEASURED DataFrame/vstack") {
		t.Errorf("out_of_scope workloads must not be reported as unmeasured, got %v", rep.notes)
	}
	if hasLine(rep.notes, "UNMEASURED Expr/rank") {
		t.Errorf("a measured workload must not be reported as unmeasured, got %v", rep.notes)
	}
	if len(rep.failures) != 0 {
		t.Errorf("an unmeasured workload is reported, not failed, got %v", rep.failures)
	}
}
