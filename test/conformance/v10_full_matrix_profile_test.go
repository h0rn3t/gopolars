package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestV10FullMatrixRemainderProfile перевіряє узгодженість лічильників повної матриці
// після зміни polars-full-matrix-remainder-v2 (не змінює критерії TestV10FinalParityGate).
func TestV10FullMatrixRemainderProfile(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "parity", "v1_0_coverage.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read coverage: %v", err)
	}
	var report v10CoverageGate
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode: %v", err)
	}
	fm := report.FullMatrix
	if fm.TotalMethods != 673 {
		t.Fatalf("full_matrix total: got %d want 673", fm.TotalMethods)
	}
	if fm.ImplementedMethods+fm.RemainingMethods != fm.TotalMethods {
		t.Fatalf("full_matrix implemented+remaining != total: %d+%d vs %d",
			fm.ImplementedMethods, fm.RemainingMethods, fm.TotalMethods)
	}
	// Після remainder-v2 та series-low-priority-apis залишок low-priority у full matrix ≤ 10.
	if fm.RemainingMethods > 10 {
		t.Fatalf("expected remainder <= 10, got %d", fm.RemainingMethods)
	}
}
