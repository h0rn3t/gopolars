package unit

import (
	"context"
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestAdvancedJoinModes(t *testing.T) {
	left, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "l", Values: []any{"a", "b", "c"}},
		},
	})
	right, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(2), int64(4)}},
			{Name: "r", Values: []any{"x", "y"}},
		},
	})
	semi, err := left.Join(polars.JoinInput{Other: right, LeftOn: []string{"id"}, RightOn: []string{"id"}, How: polars.JoinTypeSemi})
	if err != nil {
		t.Fatalf("semi join failed: %v", err)
	}
	if semi.Height() != 1 || semi.Width() != 2 {
		t.Fatalf("unexpected semi shape")
	}
	anti, err := left.Join(polars.JoinInput{Other: right, LeftOn: []string{"id"}, RightOn: []string{"id"}, How: polars.JoinTypeAnti})
	if err != nil {
		t.Fatalf("anti join failed: %v", err)
	}
	if anti.Height() != 2 || anti.Width() != 2 {
		t.Fatalf("unexpected anti shape")
	}
	cross, err := left.Join(polars.JoinInput{Other: right, How: polars.JoinTypeCross})
	if err != nil {
		t.Fatalf("cross join failed: %v", err)
	}
	if cross.Height() != 6 {
		t.Fatalf("unexpected cross rows")
	}
}

func TestAsofJoinAndLazyParity(t *testing.T) {
	left, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "ts", Values: []any{int64(10), int64(20), int64(30)}},
			{Name: "value", Values: []any{int64(1), int64(2), int64(3)}},
		},
	})
	right, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "ts", Values: []any{int64(5), int64(18), int64(29)}},
			{Name: "price", Values: []any{int64(100), int64(200), int64(300)}},
		},
	})
	eager, err := left.Join(polars.JoinInput{
		Other:         right,
		LeftOn:        []string{"ts"},
		RightOn:       []string{"ts"},
		How:           polars.JoinTypeAsof,
		AsofDirection: "backward",
		AsofTolerance: 3,
	})
	if err != nil {
		t.Fatalf("asof join failed: %v", err)
	}
	lazy, err := left.Lazy().
		Join(polars.JoinInput{
			Other:         right,
			LeftOn:        []string{"ts"},
			RightOn:       []string{"ts"},
			How:           polars.JoinTypeAsof,
			AsofDirection: "backward",
			AsofTolerance: 3,
		}).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("lazy asof join failed: %v", err)
	}
	if eager.Height() != lazy.Height() || eager.Width() != lazy.Width() {
		t.Fatalf("eager/lazy asof mismatch")
	}
	price, _ := eager.Series("price")
	if price.Value(0) != nil {
		t.Fatalf("unexpected first asof match")
	}
}
