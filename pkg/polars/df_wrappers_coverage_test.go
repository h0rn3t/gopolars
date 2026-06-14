package polars

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestDataFrameJoinWhereAndMerge covers JoinWhere (filter form) and MergeSorted
// (vertical concat).
func TestDataFrameJoinWhereAndMerge(t *testing.T) {
	t.Parallel()

	d := lazyCovFrame(t)

	filtered, err := d.JoinWhere(Col("id").Gt(Lit(int64(2))))
	if err != nil || filtered.Height() != 2 {
		t.Fatalf("JoinWhere height=%d err=%v, want 2", filtered.Height(), err)
	}

	other, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(5), int64(6)}},
		{Name: "grp", Values: []any{"c", "c"}},
		{Name: "val", Values: []any{int64(50), int64(60)}},
	}})
	if err != nil {
		t.Fatalf("other frame: %v", err)
	}
	merged, err := d.MergeSorted(other, "id")
	if err != nil || merged.Height() != 6 {
		t.Fatalf("MergeSorted height=%d err=%v, want 6", merged.Height(), err)
	}
}

// TestDataFrameMatchToSchema covers MatchToSchema adding a missing column.
func TestDataFrameMatchToSchema(t *testing.T) {
	t.Parallel()

	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}

	target := dtypes.Schema{
		{Name: "a", Type: dtypes.Int64},
		{Name: "b", Type: dtypes.Float64}, // missing -> should be added
	}
	out, err := d.MatchToSchema(target)
	if err != nil {
		t.Fatalf("MatchToSchema: %v", err)
	}
	if _, ok := out.Series("b"); !ok {
		t.Error("MatchToSchema did not add missing column b")
	}
	if out.Height() != 2 {
		t.Errorf("MatchToSchema height = %d, want 2", out.Height())
	}
}
