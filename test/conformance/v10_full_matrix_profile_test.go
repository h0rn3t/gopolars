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
	// Обидві константи — виміряні проти Polars 1.41.2 генератором gen_parity_table.py
	// (див. POLARS_PARITY_TABLE.md). Попередні 673/10 походили з курованої матриці,
	// де клас звірявся з методами всього пакета, тож зараховував чуже.
	if fm.TotalMethods != 670 {
		t.Fatalf("full_matrix total: got %d want 670", fm.TotalMethods)
	}
	if fm.ImplementedMethods+fm.RemainingMethods != fm.TotalMethods {
		t.Fatalf("full_matrix implemented+remaining != total: %d+%d vs %d",
			fm.ImplementedMethods, fm.RemainingMethods, fm.TotalMethods)
	}
	if fm.RemainingMethods > 11 {
		t.Fatalf("expected remainder <= 11, got %d", fm.RemainingMethods)
	}
}
