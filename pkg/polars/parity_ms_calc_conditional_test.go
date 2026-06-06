package polars

// Parity: conditional expressions (spec: Conditional expression parity).
// Mirrors when(diff < 0).then(0).otherwise(diff) in ../ms-calculations (profiling.py:450).

import "testing"

// TestParityWhenThenOtherwise pins profiling.py:450 — when(diff<0).then(0).otherwise(diff).
func TestParityWhenThenOtherwise(t *testing.T) {
	df := mscFrame(t,
		mscCol("diff_residual_graph", -2.0, 0.0, 3.0, -1.5),
	)
	out, err := df.WithColumns(
		When(
			Col("diff_residual_graph").Lt(Lit(0.0)),
			Lit(0.0),
			Col("diff_residual_graph"),
		).Alias("profile"),
	)
	if err != nil {
		t.Fatalf("when/then/otherwise: %v", err)
	}
	col, err := out.GetColumn("profile")
	if err != nil {
		t.Fatalf("GetColumn: %v", err)
	}
	want := []float64{0, 0, 3, 0}
	for i := 0; i < col.Len(); i++ {
		if got := asFloat(t, col.Value(i)); !mscApprox(got, want[i], 1e-9) {
			t.Errorf("profile[%d] = %v, want %v", i, got, want[i])
		}
	}
}
