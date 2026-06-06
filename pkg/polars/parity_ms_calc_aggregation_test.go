package polars

// Parity: aggregation invariants and the floating-point precision pattern
// (spec: Aggregation invariant parity, Floating-point precision aggregation parity).
// Mirrors ../ms-calculations/app/services/balance/volume_invariants.py and the
// (col.fill_nan(0) * 1000).sum() / 1000 pattern from profiling.py:218.

import (
	"testing"
)

// mscStatsFrame builds an hourly-quantity fixture and reproduces the volume_invariants.py:26
// aggregation: group_by(["z_id","kyiv_date"]).agg(max, sum, first(volume_per_day)).
//
// gopolars GroupBy.Agg has no `first` aggregate; volume_per_day is invariant within a group, so
// Max yields the same per-group value (see TestParityAggregationFirstAggUnsupported for the gap).
func mscStatsFrame(t *testing.T, df DataFrame) DataFrame {
	t.Helper()
	stats, err := df.GroupBy("z_id", "kyiv_date").Agg(
		Max(Col("z_quantity")).Alias("max_z_quantity"),
		Sum(Col("z_quantity")).Alias("sum_z_quantity"),
		Max(Col("volume_per_day")).Alias("volume_per_day"),
	)
	if err != nil {
		t.Fatalf("group_by.agg: %v", err)
	}
	return stats
}

// TestParityAggregationMaxSumFirst pins volume_invariants.py:26 — one row per (z_id, kyiv_date) with
// max, sum, and the group's volume_per_day.
func TestParityAggregationMaxSumFirst(t *testing.T) {
	df := mscFrame(t,
		mscCol("z_id", int64(1), int64(1), int64(2)),
		mscCol("kyiv_date", "2024-01-01", "2024-01-01", "2024-01-01"),
		mscCol("z_quantity", 10.0, 12.0, 4.0),
		mscCol("volume_per_day", 20.0, 20.0, 20.0),
	)
	stats := mscStatsFrame(t, df)

	if stats.Height() != 2 {
		t.Fatalf("stats height = %d, want 2 (groups (1,A) and (2,A))", stats.Height())
	}

	// Index group rows by z_id for assertions.
	byID := map[int64]map[string]any{}
	for _, r := range stats.ToDicts() {
		id, _ := r["z_id"].(int64)
		byID[id] = r
	}
	if g := byID[1]; asFloat(t, g["max_z_quantity"]) != 12 || asFloat(t, g["sum_z_quantity"]) != 22 || asFloat(t, g["volume_per_day"]) != 20 {
		t.Errorf("group(1) = max %v sum %v vpd %v, want 12/22/20", g["max_z_quantity"], g["sum_z_quantity"], g["volume_per_day"])
	}
	if g := byID[2]; asFloat(t, g["max_z_quantity"]) != 4 || asFloat(t, g["sum_z_quantity"]) != 4 || asFloat(t, g["volume_per_day"]) != 20 {
		t.Errorf("group(2) = max %v sum %v vpd %v, want 4/4/20", g["max_z_quantity"], g["sum_z_quantity"], g["volume_per_day"])
	}
}

// TestParityAggregationInflatedFilter pins volume_invariants.py:32 — the inflated-volume filter
// selects only violating groups and is empty when none violate.
func TestParityAggregationInflatedFilter(t *testing.T) {
	df := mscFrame(t,
		mscCol("z_id", int64(1), int64(1), int64(2), int64(3)),
		mscCol("kyiv_date", "2024-01-01", "2024-01-01", "2024-01-01", "2024-01-01"),
		mscCol("z_quantity", 10.0, 12.0, 4.0, 5.0),
		// group 1: vpd=0.01 -> sum 22 > 0.01*1000=10 -> VIOLATE
		// group 2: vpd=20   -> sum 4  > 20000 -> no
		// group 3: vpd=null -> excluded by is_not_null
		mscCol("volume_per_day", 0.01, 0.01, 20.0, nil),
	)
	stats := mscStatsFrame(t, df)

	inflatedPred := Col("volume_per_day").IsNotNull().
		And(Col("volume_per_day").Gt(Lit(0.0))).
		And(Col("sum_z_quantity").Gt(Col("volume_per_day").Mul(Lit(1000.0))))

	inflated, err := stats.Filter(inflatedPred)
	if err != nil {
		t.Fatalf("inflated filter: %v", err)
	}
	if inflated.Height() != 1 {
		t.Fatalf("inflated height = %d, want 1 (group 1)", inflated.Height())
	}
	row, err := inflated.Row(0)
	if err != nil {
		t.Fatalf("Row(0): %v", err)
	}
	if id, _ := row["z_id"].(int64); id != 1 {
		t.Errorf("inflated z_id = %v, want 1", row["z_id"])
	}

	// No-violation subset (groups 2 and 3 only) -> empty.
	clean := mscFrame(t,
		mscCol("z_id", int64(2), int64(3)),
		mscCol("kyiv_date", "2024-01-01", "2024-01-01"),
		mscCol("z_quantity", 4.0, 5.0),
		mscCol("volume_per_day", 20.0, 30.0),
	)
	cleanStats := mscStatsFrame(t, clean)
	cleanInflated, err := cleanStats.Filter(inflatedPred)
	if err != nil {
		t.Fatalf("clean inflated filter: %v", err)
	}
	if !cleanInflated.IsEmpty() {
		t.Errorf("clean inflated IsEmpty = false, want true (no violations)")
	}
}

// TestParityAggregationToleranceFilter pins volume_invariants.py:57 — abs(sum - vpd) > eps selects
// only out-of-tolerance groups.
func TestParityAggregationToleranceFilter(t *testing.T) {
	df := mscFrame(t,
		mscCol("z_id", int64(1), int64(2), int64(3)),
		mscCol("kyiv_date", "2024-01-01", "2024-01-01", "2024-01-01"),
		mscCol("z_quantity", 22.0, 4.0, 5.0),
		// group 1: sum 22 vs vpd 22.0   -> diff 0      -> ok
		// group 2: sum 4  vs vpd 4.5    -> diff 0.5    -> mismatch
		// group 3: sum 5  vs vpd null   -> excluded
		mscCol("volume_per_day", 22.0, 4.5, nil),
	)
	stats := mscStatsFrame(t, df)

	// Divergence: gopolars `Abs` errors on a null input rather than propagating null, and `.And()`
	// does not short-circuit, so the null-vpd row must be excluded before the abs comparison. In
	// polars the chained `is_not_null & ... & abs(...)` predicate handles nulls in one pass. The
	// two-step filter below is selection-equivalent and keeps Abs off null rows.
	nonNull, err := stats.Filter(
		Col("volume_per_day").IsNotNull().And(Col("volume_per_day").Gt(Lit(0.0))),
	)
	if err != nil {
		t.Fatalf("non-null filter: %v", err)
	}
	mismatch, err := nonNull.Filter(
		Col("sum_z_quantity").Sub(Col("volume_per_day")).Abs().Gt(Lit(mscEpsKWh)),
	)
	if err != nil {
		t.Fatalf("tolerance filter: %v", err)
	}
	if mismatch.Height() != 1 {
		t.Fatalf("mismatch height = %d, want 1 (group 2)", mismatch.Height())
	}
	row, _ := mismatch.Row(0)
	if id, _ := row["z_id"].(int64); id != 2 {
		t.Errorf("mismatch z_id = %v, want 2", row["z_id"])
	}
}

// TestParityPrecisionScaledSum pins the profiling.py:218 precision pattern:
// (value.fill_nan(0) * 1000).sum() / 1000 rounded to 3 decimals equals the decimal-exact sum.
func TestParityPrecisionScaledSum(t *testing.T) {
	vals := []any{0.1, 0.2, 0.3, 0.123, 0.004}
	df := mscFrame(t, mscCol("z_quantity", vals...))

	// Scaled column then whole-frame sum: (z_quantity.fill_nan(0) * 1000).sum().
	scaledDf, err := df.WithColumns(
		Col("z_quantity").FillNaN(Lit(0.0)).Mul(Lit(1000.0)).Alias("scaled"),
	)
	if err != nil {
		t.Fatalf("WithColumns: %v", err)
	}
	got := roundHalfEvenN(scaledDf.Sum()["scaled"]/1000.0, 3)

	// Decimal-exact expected sum of the literal inputs.
	want := roundHalfEvenN(0.1+0.2+0.3+0.123+0.004, 3) // 0.727
	if !mscApprox(got, want, 1e-9) {
		t.Errorf("scaled-sum/1000 rounded = %v, want %v", got, want)
	}
	if !mscApprox(got, 0.727, 1e-9) {
		t.Errorf("scaled-sum/1000 rounded = %v, want 0.727", got)
	}
}

// TestParityAggregationFirstAggUnsupported documents the gap: gopolars GroupBy.Agg supports
// count/sum/mean/min/max/n_unique but not `first` (volume_invariants.py:29 uses .first()).
func TestParityAggregationFirstAggUnsupported(t *testing.T) {
	df := mscFrame(t,
		mscCol("z_id", int64(1), int64(1)),
		mscCol("v", 7.0, 9.0),
	)
	_, err := df.GroupBy("z_id").Agg(Col("v").First().Alias("first_v"))
	if err == nil {
		t.Errorf("GroupBy.Agg(First) unexpectedly succeeded; gopolars now supports a `first` aggregate — replace the Max() proxy in mscStatsFrame")
	}
}
