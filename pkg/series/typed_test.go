package series

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

func TestFromInt64ZeroCopyConstructor(t *testing.T) {
	t.Parallel()

	s := FromInt64("id", []int64{1, 2, 3}, []bool{false, true, false})
	if s.Name() != "id" || s.DataType() != dtypes.Int64 || s.Len() != 3 {
		t.Fatalf("unexpected series: name=%s dtype=%s len=%d", s.Name(), s.DataType(), s.Len())
	}
	if s.Value(0) != int64(1) {
		t.Fatalf("Value(0) = %v, want 1", s.Value(0))
	}
	if !s.IsNull(1) || s.Value(1) != nil {
		t.Fatalf("index 1 should be null")
	}
	if s.Value(2) != int64(3) {
		t.Fatalf("Value(2) = %v, want 3", s.Value(2))
	}
}

func TestFromFloat64AndString(t *testing.T) {
	t.Parallel()

	f := FromFloat64("score", []float64{1.5, 2.5}, nil)
	if f.DataType() != dtypes.Float64 || f.Value(1) != 2.5 {
		t.Fatalf("float series wrong: %v", f.Value(1))
	}
	if got := Float64Values(f); got[0] != 1.5 || got[1] != 2.5 {
		t.Fatalf("Float64Values = %v", got)
	}

	s := FromString("city", []string{"kyiv", "lviv"}, nil)
	if s.DataType() != dtypes.String || s.Value(0) != "kyiv" {
		t.Fatalf("string series wrong: %v", s.Value(0))
	}
}

func TestColumnAccessorExposesTypedBacking(t *testing.T) {
	t.Parallel()

	s := FromFloat64("x", []float64{4, 5, 6}, nil)
	col := s.Column()
	if col == nil {
		t.Fatal("Column() returned nil")
	}
	vals, ok := col.Float64s()
	if !ok || len(vals) != 3 || vals[2] != 6 {
		t.Fatalf("Float64s = %v ok=%v", vals, ok)
	}
}

func TestFromColumnWrapsChunk(t *testing.T) {
	t.Parallel()

	col := chunk.NewInt64([]int64{9, 8, 7}, nil)
	s := FromColumn("c", col)
	if s.DataType() != dtypes.Int64 || s.Value(0) != int64(9) {
		t.Fatalf("FromColumn wrong: dtype=%s v0=%v", s.DataType(), s.Value(0))
	}
}

func TestNewBuildsTypedChunk(t *testing.T) {
	t.Parallel()

	s, err := New("id", dtypes.Int64, []any{int64(1), nil, int64(3)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// canonical storage must be a typed chunk, not boxed []any
	col := s.Column()
	if col == nil {
		t.Fatal("series.New must produce a typed chunk")
	}
	if _, ok := col.Int64s(); !ok {
		t.Fatal("series.New must build typed int64 backing")
	}
}
