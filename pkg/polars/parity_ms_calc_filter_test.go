package polars

// Parity: filtering and deduplication (spec: Filtering and deduplication parity).
// Mirrors pl.struct([...]).is_duplicated() and unique(subset=...) in ../ms-calculations
// (profiling.py:233, profiling.py:532).

import (
	"time"

	"testing"
)

// TestParityStructKeyIsDuplicated pins profiling.py:233 — filter(struct([z_id, z_time]).is_duplicated()).
// gopolars has no struct-key is_duplicated; the subset-frame mask reproduces it: IsDuplicated() over a
// frame of just the key columns, attached to the original via Hstack and used as the filter predicate.
func TestParityStructKeyIsDuplicated(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	df := mscFrame(t,
		mscCol("z_id", int64(1), int64(1), int64(1), int64(2)),
		mscCol("z_time", base, base, base.Add(time.Hour), base),
		mscCol("z_quantity", 1.0, 1.0, 2.0, 3.0),
	)

	subset, err := df.SubSelectColumns("z_id", "z_time")
	if err != nil {
		skipGap(t, "struct-key is_duplicated", "polars supports pl.struct([...]).is_duplicated(); gopolars has no subset duplicate detection")
		return
	}
	mask := subset.IsDuplicated().Rename("__dup")
	withMask, err := df.Hstack(mask)
	if err != nil {
		t.Fatalf("Hstack mask: %v", err)
	}
	dups, err := withMask.Filter(Col("__dup"))
	if err != nil {
		t.Fatalf("filter mask: %v", err)
	}
	// The (z_id=1, z_time=base) pair repeats -> both rows flagged.
	if dups.Height() != 2 {
		t.Fatalf("duplicated rows = %d, want 2 ((1, base) appears twice)", dups.Height())
	}
	for _, r := range dups.ToDicts() {
		if id, _ := r["z_id"].(int64); id != 1 {
			t.Errorf("duplicated row z_id = %v, want 1", r["z_id"])
		}
	}
}

// TestParityUniqueSubset pins profiling.py:532 — unique(subset=["z_id","kyiv_date"]) keeps exactly one
// row per distinct key combination.
func TestParityUniqueSubset(t *testing.T) {
	df := mscFrame(t,
		mscCol("z_id", int64(1), int64(1), int64(1), int64(2)),
		mscCol("kyiv_date", "A", "A", "B", "A"),
		mscCol("volume_per_day", 10.0, 10.0, 11.0, 20.0),
	)
	out, err := df.Unique("z_id", "kyiv_date")
	if err != nil {
		t.Fatalf("Unique: %v", err)
	}
	// Distinct (z_id, kyiv_date) combos: (1,A), (1,B), (2,A) -> 3 rows.
	if out.Height() != 3 {
		t.Fatalf("unique height = %d, want 3 distinct (z_id, kyiv_date) combos", out.Height())
	}
	seen := map[string]bool{}
	for _, r := range out.ToDicts() {
		id, _ := r["z_id"].(int64)
		date, _ := r["kyiv_date"].(string)
		key := date + ":" + string(rune('0'+id))
		if seen[key] {
			t.Errorf("duplicate combo in unique output: z_id=%d kyiv_date=%s", id, date)
		}
		seen[key] = true
	}
}
