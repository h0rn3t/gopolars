package polars

// Parity: joins (spec: Join parity).
// Mirrors the left (with coalesce), anti, and cross joins in ../ms-calculations
// (mga_balance.py:320, profiling.py:399, profiling.py:539).

import "testing"

// TestParityLeftJoin pins mga_balance.py:320 — a left join preserves every left row and fills matched
// right columns. polars uses coalesce=True so the key appears once; gopolars JoinInput has no coalesce
// option, so the right key surfaces as "<key>_right" (documented divergence).
func TestParityLeftJoin(t *testing.T) {
	left := mscFrame(t,
		mscCol("yd_time", int64(1), int64(2), int64(3)),
		mscCol("base", 10.0, 20.0, 30.0),
	)
	right := mscFrame(t,
		mscCol("yd_time", int64(2), int64(3)),
		mscCol("loss_factor", 0.5, 0.7),
	)
	out, err := left.Join(JoinInput{
		Other:   right,
		LeftOn:  []string{"yd_time"},
		RightOn: []string{"yd_time"},
		How:     JoinTypeLeft,
	})
	if err != nil {
		t.Fatalf("left join: %v", err)
	}
	if out.Height() != 3 {
		t.Fatalf("left join height = %d, want 3 (all left rows preserved)", out.Height())
	}

	// Matched rows carry the right column; the unmatched left row (yd_time=1) is null there.
	byTime := map[int64]map[string]any{}
	for _, r := range out.ToDicts() {
		id, _ := r["yd_time"].(int64)
		byTime[id] = r
	}
	if lf := byTime[2]["loss_factor"]; asFloat(t, lf) != 0.5 {
		t.Errorf("loss_factor(yd_time=2) = %v, want 0.5", lf)
	}
	if lf := byTime[3]["loss_factor"]; asFloat(t, lf) != 0.7 {
		t.Errorf("loss_factor(yd_time=3) = %v, want 0.7", lf)
	}
	if lf := byTime[1]["loss_factor"]; lf != nil {
		t.Errorf("loss_factor(yd_time=1) = %v, want nil (no match)", lf)
	}

	// Divergence: without coalesce, the right key is duplicated as "yd_time_right".
	hasRightKey := false
	for _, c := range out.Columns() {
		if c == "yd_time_right" {
			hasRightKey = true
		}
	}
	if !hasRightKey {
		t.Logf("note: left join did not emit yd_time_right; gopolars may now coalesce keys — revisit the coalesce gap")
	}
}

// TestParityAntiJoin pins profiling.py:399 — an anti join returns only the unmatched left rows and
// keeps just the left columns.
func TestParityAntiJoin(t *testing.T) {
	left := mscFrame(t,
		mscCol("z_id", int64(1), int64(2), int64(3)),
		mscCol("kyiv_date", "A", "A", "A"),
		mscCol("z_quantity", 1.0, 2.0, 3.0),
	)
	matched := mscFrame(t,
		mscCol("z_id", int64(2), int64(3)),
		mscCol("kyiv_date", "A", "A"),
	)
	out, err := left.Join(JoinInput{
		Other:   matched,
		LeftOn:  []string{"z_id", "kyiv_date"},
		RightOn: []string{"z_id", "kyiv_date"},
		How:     JoinTypeAnti,
	})
	if err != nil {
		t.Fatalf("anti join: %v", err)
	}
	if out.Height() != 1 {
		t.Fatalf("anti join height = %d, want 1 (only z_id=1 unmatched)", out.Height())
	}
	row, _ := out.Row(0)
	if id, _ := row["z_id"].(int64); id != 1 {
		t.Errorf("anti join z_id = %v, want 1", row["z_id"])
	}
	// Only left columns survive an anti join.
	for _, c := range out.Columns() {
		if c != "z_id" && c != "kyiv_date" && c != "z_quantity" {
			t.Errorf("unexpected column %q in anti-join output", c)
		}
	}
}

// TestParityCrossJoin pins profiling.py:539 — a cross join yields the cartesian product.
func TestParityCrossJoin(t *testing.T) {
	hourly := mscFrame(t,
		mscCol("z_time", int64(1), int64(2), int64(3)),
	)
	apIDs := mscFrame(t,
		mscCol("ap_id", int64(100), int64(200)),
	)
	out, err := hourly.Join(JoinInput{Other: apIDs, How: JoinTypeCross})
	if err != nil {
		t.Fatalf("DataFrame.Join cross: %v", err)
	}
	if out.Height() != hourly.Height()*apIDs.Height() {
		t.Errorf("cross join height = %d, want %d (3*2)", out.Height(), hourly.Height()*apIDs.Height())
	}
}
