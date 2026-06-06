package polars

// Parity: row and scalar access (spec: Row and scalar access parity).
// Mirrors row(0, named=True) and df.select(pl.sum(...)).item() in ../ms-calculations
// (volume_invariants.py:38, mga_balance.py:369).

import "testing"

// TestParityRowNamed pins volume_invariants.py:38 — row(0, named=True) returns a column->value map.
func TestParityRowNamed(t *testing.T) {
	df := mscFrame(t,
		mscCol("z_id", int64(7), int64(8)),
		mscCol("kyiv_date", "2024-01-01", "2024-01-02"),
		mscCol("sum_z_quantity", 22.5, 4.0),
	)
	row, err := df.Row(0)
	if err != nil {
		t.Fatalf("Row(0): %v", err)
	}
	if id, _ := row["z_id"].(int64); id != 7 {
		t.Errorf("row[z_id] = %v, want 7", row["z_id"])
	}
	if d, _ := row["kyiv_date"].(string); d != "2024-01-01" {
		t.Errorf("row[kyiv_date] = %v, want 2024-01-01", row["kyiv_date"])
	}
	if !mscApprox(asFloat(t, row["sum_z_quantity"]), 22.5, 1e-9) {
		t.Errorf("row[sum_z_quantity] = %v, want 22.5", row["sum_z_quantity"])
	}
}

// TestParityItemScalarAggregate pins mga_balance.py:369 — reading a single-cell aggregate via item().
func TestParityItemScalarAggregate(t *testing.T) {
	df := mscFrame(t,
		mscCol("grp", int64(1), int64(1), int64(1)),
		mscCol("residual_profile", 2.0, 3.5, 4.5),
	)
	agg, err := df.GroupBy("grp").Agg(Sum(Col("residual_profile")).Alias("total"))
	if err != nil {
		t.Fatalf("agg: %v", err)
	}
	if agg.Height() != 1 {
		t.Fatalf("agg height = %d, want 1", agg.Height())
	}
	v, err := agg.Item(0, "total")
	if err != nil {
		t.Fatalf("Item: %v", err)
	}
	if got := asFloat(t, v); !mscApprox(got, 10.0, 1e-9) {
		t.Errorf("item() = %v, want 10.0 (2.0+3.5+4.5)", got)
	}
}
