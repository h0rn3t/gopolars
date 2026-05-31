package unit

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDataFrameMVPOperations(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(1), int64(2), nil}},
			{Name: "city", Values: []any{"kyiv", "kyiv", "lviv", "odesa"}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}
	uniq, err := df.Unique("id")
	if err != nil {
		t.Fatalf("unique failed: %v", err)
	}
	if uniq.Height() != 3 {
		t.Fatalf("unexpected unique height: %d", uniq.Height())
	}
	sliced := df.Slice(1, 2)
	if sliced.Height() != 2 {
		t.Fatalf("unexpected slice height")
	}
	filled, err := df.FillNull(int64(0))
	if err != nil {
		t.Fatalf("fill null failed: %v", err)
	}
	idCol, _ := filled.Series("id")
	if idCol.Value(3) != int64(0) {
		t.Fatalf("fill null did not replace null value")
	}
	dropped := df.DropNulls("id")
	if dropped.Height() != 3 {
		t.Fatalf("drop nulls failed")
	}
	concat, err := df.Concat(polars.ConcatInput{Others: []polars.DataFrame{df}})
	if err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	if concat.Height() != 8 {
		t.Fatalf("unexpected concat height")
	}
}

func TestJoinRightAndFull(t *testing.T) {
	left, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "l", Values: []any{"a", "b"}},
		},
	})
	right, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(2), int64(3)}},
			{Name: "r", Values: []any{"x", "y"}},
		},
	})
	rj, err := left.Join(polars.JoinInput{
		Other:   right,
		LeftOn:  []string{"id"},
		RightOn: []string{"id"},
		How:     polars.JoinTypeRight,
	})
	if err != nil {
		t.Fatalf("right join failed: %v", err)
	}
	if rj.Height() != 2 {
		t.Fatalf("unexpected right join height")
	}
	fj, err := left.Join(polars.JoinInput{
		Other:   right,
		LeftOn:  []string{"id"},
		RightOn: []string{"id"},
		How:     polars.JoinTypeFull,
	})
	if err != nil {
		t.Fatalf("full join failed: %v", err)
	}
	if fj.Height() != 3 {
		t.Fatalf("unexpected full join height")
	}
}
