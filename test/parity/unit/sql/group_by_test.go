package sql

// Ported from py-polars/tests/unit/sql/test_group_by.py (py-1.28.1, representative subset)

import "testing"

// GROUP BY with SUM.
func TestSQLGroupBySum(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT b, SUM(v) AS s FROM t GROUP BY b")
	if out.Height() != 2 {
		t.Fatalf("groups: got %d, want 2", out.Height())
	}
	bcol, _ := out.GetColumn("b")
	scol, _ := out.GetColumn("s")
	got := map[string]float64{}
	for i := 0; i < out.Height(); i++ {
		k, _ := bcol.Value(i).(string)
		switch v := scol.Value(i).(type) {
		case float64:
			got[k] = v
		case int64:
			got[k] = float64(v)
		}
	}
	if got["x"] != 40 { // 10 + 30
		t.Fatalf("sum x: got %v, want 40", got["x"])
	}
	if got["y"] != 60 { // 20 + 40
		t.Fatalf("sum y: got %v, want 60", got["y"])
	}
}

// GROUP BY with COUNT(*).
func TestSQLGroupByCount(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT b, COUNT(*) AS n FROM t GROUP BY b")
	if out.Height() != 2 {
		t.Fatalf("groups: got %d, want 2", out.Height())
	}
}

// GROUP BY with MAX.
func TestSQLGroupByMax(t *testing.T) {
	t.Parallel()
	out := runSQL(t, baseDF(t), "SELECT b, MAX(v) AS m FROM t GROUP BY b")
	bcol, _ := out.GetColumn("b")
	mcol, _ := out.GetColumn("m")
	for i := 0; i < out.Height(); i++ {
		k, _ := bcol.Value(i).(string)
		m := mcol.Value(i)
		var mv float64
		switch x := m.(type) {
		case float64:
			mv = x
		case int64:
			mv = float64(x)
		}
		if k == "x" && mv != 30 {
			t.Fatalf("max x: got %v, want 30", mv)
		}
		if k == "y" && mv != 40 {
			t.Fatalf("max y: got %v, want 40", mv)
		}
	}
}
