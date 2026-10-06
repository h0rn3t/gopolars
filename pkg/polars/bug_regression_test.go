package polars

import (
	"math"
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

func TestCumMinAfterLeadingNull(t *testing.T) {
	t.Parallel()

	s := newInt64Series(t, "v", []any{nil, int64(3), int64(1), int64(2)})
	want := []any{nil, float64(3), float64(1), float64(1)}
	if got := s.CumMin().ToList(); !slices.Equal(got, want) {
		t.Errorf("CumMin([null 3 1 2]) = %v, want %v", got, want)
	}
	wantMax := []any{nil, float64(3), float64(3), float64(3)}
	if got := s.CumMax().ToList(); !slices.Equal(got, wantMax) {
		t.Errorf("CumMax([null 3 1 2]) = %v, want %v", got, wantMax)
	}
}

func TestShrinkDTypeKeepsFloatAtInt64Overflow(t *testing.T) {
	t.Parallel()

	// float64(math.MaxInt64) rounds up to 2^63, which int64 cannot hold.
	s := mustSeries(t, "v", dtypes.Float64, []any{float64(1), math.Ldexp(1, 63)})
	out, err := s.ShrinkDType()
	if err != nil {
		t.Fatalf("ShrinkDType() error = %v", err)
	}
	if out.DataType() != dtypes.Float64 {
		t.Errorf("ShrinkDType([1 2^63]) dtype = %s, want %s", out.DataType(), dtypes.Float64)
	}
}

func TestPartitionByFirstAppearanceOrder(t *testing.T) {
	t.Parallel()

	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"c", "a", "c", "b", "a", "d", "e", "f"}},
		{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6), int64(7), int64(8)}},
	}})
	if err != nil {
		t.Fatalf("NewDataFrame() error = %v", err)
	}
	want := []string{"c", "a", "b", "d", "e", "f"}
	// Repeat so a map-ordered implementation cannot pass by chance.
	for range 20 {
		parts, err := df.PartitionBy("g")
		if err != nil {
			t.Fatalf("PartitionBy(g) error = %v", err)
		}
		got := make([]string, len(parts))
		for i, p := range parts {
			col, err := p.GetColumn("g")
			if err != nil {
				t.Fatalf("GetColumn(g) error = %v", err)
			}
			got[i], _ = col.Value(0).(string)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("PartitionBy(g) group order = %v, want %v", got, want)
		}
	}
}
