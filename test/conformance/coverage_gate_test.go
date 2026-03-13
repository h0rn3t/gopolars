package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type coverageMatrix struct {
	MinimumOverallCoverage float64            `json:"minimum_overall_coverage"`
	Capabilities           map[string]float64 `json:"capabilities"`
}

func TestParityCoverageThreshold(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "parity", "v0_2_coverage.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read coverage matrix failed: %v", err)
	}
	var matrix coverageMatrix
	if err := json.Unmarshal(raw, &matrix); err != nil {
		t.Fatalf("decode coverage matrix failed: %v", err)
	}
	if len(matrix.Capabilities) == 0 {
		t.Fatalf("coverage matrix has no capabilities")
	}

	min := matrix.MinimumOverallCoverage
	if envMin := os.Getenv("GOPOLARS_PARITY_MIN"); envMin != "" {
		parsed, err := strconv.ParseFloat(envMin, 64)
		if err != nil {
			t.Fatalf("invalid GOPOLARS_PARITY_MIN: %v", err)
		}
		min = parsed
	}

	var sum float64
	for name, value := range matrix.Capabilities {
		if value <= 0 {
			t.Fatalf("capability %s coverage must be > 0", name)
		}
		if value > 1 {
			t.Fatalf("capability %s coverage must be <= 1", name)
		}
		sum += value
	}
	overall := sum / float64(len(matrix.Capabilities))
	if overall < min {
		t.Fatalf("overall parity coverage %.3f is below threshold %.3f", overall, min)
	}
}
