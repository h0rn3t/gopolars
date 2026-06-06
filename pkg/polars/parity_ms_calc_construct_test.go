package polars

// Parity: DataFrame construction (spec: DataFrame construction parity).
// Mirrors pl.from_records(..., orient="row") and pl.DataFrame({...}, schema=...) usage in
// ../ms-calculations (mga_balance.py:91, profiling.py:80).

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestParityConstructFromRowRecords mirrors mga_balance.py:91 — pl.from_records over row tuples
// with an explicit column list. Asserts column order, dtypes, and height.
func TestParityConstructFromRowRecords(t *testing.T) {
	df := mscFrame(t,
		mscCol("ap_id", int64(10), int64(11), int64(12)),
		mscCol("quantity_type", "in", "out", "in"),
		mscCol("interchange_type", "a", "b", "c"),
	)

	if got := df.Columns(); len(got) != 3 || got[0] != "ap_id" || got[1] != "quantity_type" || got[2] != "interchange_type" {
		t.Fatalf("Columns = %v, want [ap_id quantity_type interchange_type]", got)
	}
	if df.Height() != 3 {
		t.Errorf("Height = %d, want 3", df.Height())
	}

	wantTypes := map[string]dtypes.DataType{
		"ap_id":            dtypes.Int64,
		"quantity_type":    dtypes.String,
		"interchange_type": dtypes.String,
	}
	for _, f := range df.Schema() {
		if want, ok := wantTypes[f.Name]; ok && f.Type != want {
			t.Errorf("dtype(%s) = %s, want %s", f.Name, f.Type, want)
		}
	}
}

// TestParityConstructEmptyFromSchema mirrors profiling.py:80 — pl.DataFrame(schema={...}) builds an
// empty frame that still carries its declared dtypes. gopolars infers dtype from values, so a direct
// empty construction fails ("cannot infer data type"); the achievable equivalent (a typed frame
// emptied via Head(0)) is asserted to still carry schema + dtypes.
func TestParityConstructEmptyFromSchema(t *testing.T) {
	// Gap: gopolars cannot construct an empty column without value-based dtype inference.
	if _, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "z_id", Values: []any{}},
	}}); err == nil {
		t.Errorf("empty NewDataFrame unexpectedly succeeded; gopolars now supports schema-only construction — update this gap note")
	}

	// Equivalent empty-but-typed frame: build typed, then take zero rows.
	typed := mscFrame(t,
		mscCol("z_id", int64(1)),
		mscCol("z_quantity", 1.5),
	)
	empty := typed.Head(0)

	if !empty.IsEmpty() {
		t.Errorf("IsEmpty = false, want true")
	}
	if empty.Height() != 0 {
		t.Errorf("Height = %d, want 0", empty.Height())
	}
	cols := empty.Columns()
	if len(cols) != 2 || cols[0] != "z_id" || cols[1] != "z_quantity" {
		t.Errorf("Columns = %v, want [z_id z_quantity]", cols)
	}
	wantTypes := map[string]dtypes.DataType{"z_id": dtypes.Int64, "z_quantity": dtypes.Float64}
	for _, f := range empty.Schema() {
		if want, ok := wantTypes[f.Name]; ok && f.Type != want {
			t.Errorf("dtype(%s) = %s, want %s (preserved on empty frame)", f.Name, f.Type, want)
		}
	}
}
