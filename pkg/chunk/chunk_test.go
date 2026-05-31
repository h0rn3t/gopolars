package chunk

import (
	"reflect"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

func TestNewFloat64ValuesAndNulls(t *testing.T) {
	t.Parallel()

	c := NewFloat64([]float64{1.5, 0, 3.5}, []bool{false, true, false})
	if c.Len() != 3 {
		t.Fatalf("Len = %d, want 3", c.Len())
	}
	if c.DataType() != dtypes.Float64 {
		t.Fatalf("DataType = %s, want float64", c.DataType())
	}
	if c.IsNull(0) || !c.IsNull(1) || c.IsNull(2) {
		t.Fatalf("null mask wrong: %v %v %v", c.IsNull(0), c.IsNull(1), c.IsNull(2))
	}
	if v := c.ValueAt(0); v != 1.5 {
		t.Fatalf("ValueAt(0) = %v, want 1.5", v)
	}
	if v := c.ValueAt(1); v != nil {
		t.Fatalf("ValueAt(1) = %v, want nil (null)", v)
	}
	if v := c.ValueAt(2); v != 3.5 {
		t.Fatalf("ValueAt(2) = %v, want 3.5", v)
	}
}

func TestTypedBackingAccessors(t *testing.T) {
	t.Parallel()

	f := NewFloat64([]float64{1, 2}, nil)
	if got, ok := f.Float64s(); !ok || !reflect.DeepEqual(got, []float64{1, 2}) {
		t.Fatalf("Float64s = %v ok=%v", got, ok)
	}
	if _, ok := f.Int64s(); ok {
		t.Fatalf("Int64s should not be ok for float column")
	}

	i := NewInt64([]int64{7, 8, 9}, nil)
	if got, ok := i.Int64s(); !ok || !reflect.DeepEqual(got, []int64{7, 8, 9}) {
		t.Fatalf("Int64s = %v ok=%v", got, ok)
	}

	s := NewString([]string{"a", "b"}, nil)
	if got, ok := s.Strings(); !ok || !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("Strings = %v ok=%v", got, ok)
	}

	b := NewBool([]bool{true, false}, nil)
	if got, ok := b.Bools(); !ok || !reflect.DeepEqual(got, []bool{true, false}) {
		t.Fatalf("Bools = %v ok=%v", got, ok)
	}
}

func TestFromAnyBuildsTypedBacking(t *testing.T) {
	t.Parallel()

	c, err := FromAny(dtypes.Int64, []any{int64(1), nil, int64(3)})
	if err != nil {
		t.Fatalf("FromAny: %v", err)
	}
	got, ok := c.Int64s()
	if !ok {
		t.Fatalf("expected typed int64 backing")
	}
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("int64 backing = %v", got)
	}
	if !c.IsNull(1) {
		t.Fatalf("index 1 should be null")
	}
	if c.ValueAt(1) != nil {
		t.Fatalf("null ValueAt should be nil")
	}
}

func TestFromAnyTypeMismatch(t *testing.T) {
	t.Parallel()

	if _, err := FromAny(dtypes.Int64, []any{"nope"}); err == nil {
		t.Fatal("expected type error for int64 column with string value")
	}
}

func TestFromAnyUnsupportedDtypeIsBoxed(t *testing.T) {
	t.Parallel()

	row := map[string]any{"k": int64(1)}
	c, err := FromAny(dtypes.Struct, []any{row, nil})
	if err != nil {
		t.Fatalf("FromAny struct: %v", err)
	}
	if c.Len() != 2 {
		t.Fatalf("Len = %d, want 2", c.Len())
	}
	if !reflect.DeepEqual(c.ValueAt(0), row) {
		t.Fatalf("ValueAt(0) = %v, want %v", c.ValueAt(0), row)
	}
	if !c.IsNull(1) {
		t.Fatalf("index 1 should be null")
	}
	// typed accessors must report not-ok for boxed columns
	if _, ok := c.Int64s(); ok {
		t.Fatal("boxed column must not expose Int64s")
	}
}

func TestFilterByMask(t *testing.T) {
	t.Parallel()

	c := NewFloat64([]float64{10, 20, 30, 40}, []bool{false, false, true, false})
	out := c.Filter([]bool{true, false, true, true})
	if out.Len() != 3 {
		t.Fatalf("filtered Len = %d, want 3", out.Len())
	}
	if out.ValueAt(0) != 10.0 {
		t.Fatalf("out[0] = %v, want 10", out.ValueAt(0))
	}
	// original index 2 was null and survives the mask -> stays null at new index 1
	if !out.IsNull(1) {
		t.Fatalf("out[1] should preserve null from original index 2")
	}
	if out.ValueAt(2) != 40.0 {
		t.Fatalf("out[2] = %v, want 40", out.ValueAt(2))
	}
}

func TestSliceGatherByIndices(t *testing.T) {
	t.Parallel()

	c := NewInt64([]int64{5, 6, 7, 8}, []bool{false, true, false, false})
	out := c.Slice([]int{3, 1, 0})
	if out.Len() != 3 {
		t.Fatalf("gathered Len = %d, want 3", out.Len())
	}
	if out.ValueAt(0) != int64(8) {
		t.Fatalf("out[0] = %v, want 8", out.ValueAt(0))
	}
	if !out.IsNull(1) {
		t.Fatalf("out[1] should be null (gathered original index 1)")
	}
	if out.ValueAt(2) != int64(5) {
		t.Fatalf("out[2] = %v, want 5", out.ValueAt(2))
	}
}

func TestShiftPositiveAndNegative(t *testing.T) {
	t.Parallel()

	c := NewInt64([]int64{10, 20, 30}, nil)

	down := c.Shift(1)
	if !down.IsNull(0) || down.ValueAt(1) != int64(10) {
		t.Fatalf("shift down: null0=%v v1=%v", down.IsNull(0), down.ValueAt(1))
	}

	up := c.Shift(-1)
	if !up.IsNull(2) || up.ValueAt(0) != int64(20) {
		t.Fatalf("shift up: null2=%v v0=%v", up.IsNull(2), up.ValueAt(0))
	}

	same := c.Shift(0)
	if same.ValueAt(2) != int64(30) {
		t.Fatalf("shift(0) must clone values")
	}
}

func TestCloneIsDeep(t *testing.T) {
	t.Parallel()

	c := NewFloat64([]float64{1, 2, 3}, []bool{false, true, false})
	cl := c.Clone()
	f, _ := cl.Float64s()
	f[0] = 99
	// mutating clone backing must not affect original
	if c.ValueAt(0) != 1.0 {
		t.Fatalf("clone is not deep: original mutated to %v", c.ValueAt(0))
	}
}

func TestDatetimeBacking(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	c, err := FromAny(dtypes.Datetime, []any{ts, nil})
	if err != nil {
		t.Fatalf("FromAny datetime: %v", err)
	}
	got, ok := c.Times()
	if !ok || !got[0].Equal(ts) {
		t.Fatalf("Times = %v ok=%v", got, ok)
	}
	if !c.IsNull(1) {
		t.Fatalf("index 1 should be null")
	}
}
