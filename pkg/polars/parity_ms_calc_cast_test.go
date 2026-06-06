package polars

// Parity: casting and null/NaN handling (spec: Casting and null/NaN handling parity).
// Mirrors cast(Float64).fill_nan(0).fill_null(0) and clip(lower_bound=0) in ../ms-calculations
// (mga_balance.py:253, mga_balance.py:365).

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestParityCastFillChain pins mga_balance.py:253 — cast(Float64).fill_nan(0).fill_null(0) zeroes NaN
// and null entries and yields a Float64 column.
func TestParityCastFillChain(t *testing.T) {
	df := mscFrame(t,
		mscCol("interchange_in", 1.5, math.NaN(), nil, 4.0),
	)
	out, err := df.WithColumns(
		Col("interchange_in").Cast(Float64).FillNaN(Lit(0.0)).FillNull(Lit(0.0)).Alias("interchange_in"),
	)
	if err != nil {
		t.Fatalf("cast/fill chain: %v", err)
	}

	col, err := out.GetColumn("interchange_in")
	if err != nil {
		t.Fatalf("GetColumn: %v", err)
	}
	want := []float64{1.5, 0, 0, 4.0}
	for i := 0; i < col.Len(); i++ {
		got := asFloat(t, col.Value(i))
		if math.IsNaN(got) {
			t.Errorf("row %d is NaN after fill_nan", i)
			continue
		}
		if !mscApprox(got, want[i], 1e-9) {
			t.Errorf("row %d = %v, want %v", i, got, want[i])
		}
	}
	if out.NullCount()["interchange_in"] != 0 {
		t.Errorf("null count = %d, want 0 after fill_null", out.NullCount()["interchange_in"])
	}
	for _, f := range out.Schema() {
		if f.Name == "interchange_in" && f.Type != dtypes.Float64 {
			t.Errorf("dtype = %s, want Float64", f.Type)
		}
	}
}

// TestParityClipLowerBound pins mga_balance.py:365 — clip(lower_bound=0) floors negatives at 0 and
// leaves non-negatives unchanged. gopolars Series.Clip takes an explicit upper bound, so a very large
// upper (MaxFloat64) reproduces a lower-only clip.
func TestParityClipLowerBound(t *testing.T) {
	df := mscFrame(t,
		mscCol("residual_graph", -5.0, -0.1, 0.0, 3.2),
	)
	col, err := df.GetColumn("residual_graph")
	if err != nil {
		t.Fatalf("GetColumn: %v", err)
	}
	clipped := col.Clip(0, math.MaxFloat64)
	want := []float64{0, 0, 0, 3.2}
	for i := 0; i < clipped.Len(); i++ {
		if got := asFloat(t, clipped.Value(i)); !mscApprox(got, want[i], 1e-9) {
			t.Errorf("clipped[%d] = %v, want %v", i, got, want[i])
		}
	}
}
