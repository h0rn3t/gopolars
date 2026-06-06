package chunk

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestFromAnyAllDtypes covers FromAny across every dtype, including a null entry
// (validity branch) and the unsupported-dtype default.
func TestFromAnyAllDtypes(t *testing.T) {
	now := time.Now()
	valid := []struct {
		dt   dtypes.DataType
		vals []any
	}{
		{dtypes.Int64, []any{int64(1), nil}},
		{dtypes.Float64, []any{1.5, nil}},
		{dtypes.String, []any{"a", nil}},
		{dtypes.Boolean, []any{true, nil}},
		{dtypes.Datetime, []any{now, nil}},
		{dtypes.Decimal, []any{dtypes.DecimalValue("1.50"), "2.50", nil}},
		{dtypes.List, []any{[]any{int64(1)}, nil}},
		{dtypes.Struct, []any{map[string]any{"x": int64(1)}, nil}},
	}
	for _, tc := range valid {
		c, err := FromAny(tc.dt, tc.vals)
		if err != nil {
			t.Errorf("FromAny(%s): %v", tc.dt, err)
			continue
		}
		if c.Len() != len(tc.vals) {
			t.Errorf("FromAny(%s) Len = %d, want %d", tc.dt, c.Len(), len(tc.vals))
		}
		if !c.IsNull(len(tc.vals) - 1) {
			t.Errorf("FromAny(%s): last row should be null", tc.dt)
		}
	}

	// Wrong-type value per dtype → error branch.
	bad := []struct {
		dt  dtypes.DataType
		val any
	}{
		{dtypes.Int64, "x"},
		{dtypes.Float64, "x"},
		{dtypes.String, int64(1)},
		{dtypes.Boolean, "x"},
		{dtypes.Datetime, "x"},
		{dtypes.Decimal, int64(1)},
		{dtypes.List, "x"},
		{dtypes.Struct, "x"},
	}
	for _, tc := range bad {
		if _, err := FromAny(tc.dt, []any{tc.val}); err == nil {
			t.Errorf("FromAny(%s, %v): expected type error", tc.dt, tc.val)
		}
	}

	if _, err := FromAny(dtypes.DataType("nonsense"), []any{1}); err == nil {
		t.Error("FromAny(unsupported): expected error")
	}
}

// TestNormalizeNulls covers all three branches: nil, length mismatch, and a
// matching-length passthrough.
func TestNormalizeNulls(t *testing.T) {
	if got := normalizeNulls(nil, 3); len(got) != 3 {
		t.Errorf("nil → len %d, want 3", len(got))
	}
	// Mismatched length → padded copy.
	if got := normalizeNulls([]bool{true}, 3); len(got) != 3 || !got[0] || got[1] {
		t.Errorf("mismatch → %v, want [true false false]", got)
	}
	in := []bool{true, false}
	if got := normalizeNulls(in, 2); len(got) != 2 {
		t.Errorf("match → len %d, want 2", len(got))
	}
}

// TestNilColumnGuards covers the nil-receiver guards.
func TestNilColumnGuards(t *testing.T) {
	var c *Column
	if c.Len() != 0 {
		t.Errorf("nil Len = %d, want 0", c.Len())
	}
	if c.NullCount() != 0 {
		t.Errorf("nil NullCount = %d, want 0", c.NullCount())
	}
}

// TestViewAllDtypes covers the View slice for every dtype, including a column
// with a validity mask.
func TestViewAllDtypes(t *testing.T) {
	now := time.Now()
	cols := []*Column{
		NewInt64([]int64{1, 2, 3}, []bool{false, true, false}),
		NewFloat64([]float64{1, 2, 3}, nil),
		NewString([]string{"a", "b", "c"}, nil),
		NewBool([]bool{true, false, true}, nil),
		NewTime([]time.Time{now, now, now}, nil),
		NewBoxed(dtypes.Struct, []any{map[string]any{"x": 1}, nil, "z"}, nil),
	}
	for _, c := range cols {
		v := c.View(1, 3)
		if v.Len() != 2 {
			t.Errorf("View(%s) Len = %d, want 2", c.DataType(), v.Len())
		}
	}
}

// TestNormWindowParams covers the clamping branches via RollingSum.
func TestNormWindowParams(t *testing.T) {
	vals := []float64{1, 2, 3}
	// window <= 0 and minPeriods < 1 both get clamped.
	out, nulls := RollingSum(vals, nil, 0, 0, true)
	if len(out) != 3 || len(nulls) != 3 {
		t.Fatalf("RollingSum len = %d/%d, want 3/3", len(out), len(nulls))
	}
}

// TestAppendRowKeyAllDtypes covers the per-dtype key encoding, including the
// null tag and the boxed fallback.
func TestAppendRowKeyAllDtypes(t *testing.T) {
	now := time.Now()
	cols := []*Column{
		NewInt64([]int64{1, 0}, []bool{false, true}), // row 1 null
		NewFloat64([]float64{1.5, 2.5}, nil),
		NewString([]string{"a", "b"}, nil),
		NewBool([]bool{true, false}, nil),
		NewTime([]time.Time{now, now}, nil),
		NewBoxed(dtypes.Struct, []any{map[string]any{"x": 1}, map[string]any{"y": 2}}, nil),
	}
	for _, c := range cols {
		for row := 0; row < c.Len(); row++ {
			key := appendRowKey(nil, c, row)
			if len(key) == 0 {
				t.Errorf("appendRowKey(%s, %d) returned empty key", c.DataType(), row)
			}
		}
	}

	// A NaN float key must encode via the canonical-NaN path.
	nanCol := NewFloat64([]float64{nanFloat()}, nil)
	if key := appendRowKey(nil, nanCol, 0); len(key) == 0 {
		t.Error("appendRowKey(NaN) returned empty key")
	}
}

func nanFloat() float64 { var z float64; return z / z }
