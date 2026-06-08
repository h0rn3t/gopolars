package polars

// Parity: banking round-half-to-even with sequential residue carry
// (spec: Banking round-half-to-even parity).
// Mirrors round_bankers / round_for_nparray / process_group_vectorized in
// ../ms-calculations/app/docs/banking_rounding2.py.

import (
	"testing"
	"time"
)

// TestParityBankersSampleVectors pins banking_rounding2.py:67 — the documented sample vectors that
// the Python check_round_sample() asserts.
func TestParityBankersSampleVectors(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0.0115, 0.012},
		{0.0155, 0.016},
		{0.0175, 0.018},
		{0.0173, 0.017},
		{0.0176, 0.018},
		{0.0111, 0.011},
	}
	for _, c := range cases {
		if got := roundHalfEvenN(c.in, 3); !mscApprox(got, c.want, 1e-9) {
			t.Errorf("roundHalfEvenN(%v, 3) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestParityBankersResidueCarryConservesSum pins banking_rounding2.py:43/59 — applying the residue
// carry over a gopolars-sorted, gopolars-partitioned group conserves the group total to within one
// rounding unit (0.001). gopolars handles the sort/partition/extraction; the rounding kernel is the
// helper (gopolars has no native round-to-3-decimals — see TestParityRoundDecimalsModeUnsupported).
func TestParityBankersResidueCarryConservesSum(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	df := mscFrame(t,
		// Two meters, deliberately out of time order so Sort matters.
		mscCol("z_id", int64(1), int64(1), int64(1), int64(2), int64(2)),
		mscCol("z_time",
			base.Add(2*time.Hour), base, base.Add(time.Hour),
			base.Add(time.Hour), base),
		mscCol("z_quantity", 0.0124, 0.0115, 0.0155, 0.0136, 0.0175),
	)

	sorted, err := df.Sort(SortInput{By: []string{"z_id", "z_time"}})
	if err != nil {
		t.Fatalf("Sort: %v", err)
	}
	groups, err := sorted.PartitionBy("z_id")
	if err != nil {
		t.Fatalf("PartitionBy: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("PartitionBy groups = %d, want 2", len(groups))
	}

	for _, g := range groups {
		col, err := g.GetColumn("z_quantity")
		if err != nil {
			t.Fatalf("GetColumn: %v", err)
		}
		orig := make([]float64, col.Len())
		for i := 0; i < col.Len(); i++ {
			orig[i] = asFloat(t, col.Value(i))
		}
		rounded := bankersResidueCarry(orig)
		if len(rounded) != len(orig) {
			t.Fatalf("rounded len = %d, want %d", len(rounded), len(orig))
		}
		if diff := sumFloats(orig) - sumFloats(rounded); !mscApprox(diff, 0, 0.001) {
			t.Errorf("residue carry sum drift = %v, want within one rounding unit (0.001) of 0", diff)
		}
		// Every rounded value is a clean multiple of 0.001.
		for i, r := range rounded {
			if !mscApprox(r, roundHalfEvenN(r, 3), 1e-9) {
				t.Errorf("rounded[%d] = %v is not a 3-decimal value", i, r)
			}
		}
	}
}
