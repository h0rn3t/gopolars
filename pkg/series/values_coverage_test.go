package series

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestTypedValuesSkipNulls exercises the null-skip branch of every typed value
// extractor (the non-null path is covered elsewhere).
func TestTypedValuesSkipNulls(t *testing.T) {
	f, err := New("f", dtypes.Float64, []any{1.5, nil, 3.5})
	if err != nil {
		t.Fatalf("New float: %v", err)
	}
	if got := Float64Values(f); got[1] != 0 || got[0] != 1.5 {
		t.Errorf("Float64Values = %v, want [1.5 0 3.5]", got)
	}

	i, _ := New("i", dtypes.Int64, []any{int64(7), nil, int64(9)})
	if got := Int64Values(i); got[1] != 0 || got[0] != 7 {
		t.Errorf("Int64Values = %v", got)
	}

	s, _ := New("s", dtypes.String, []any{"a", nil})
	if got := StringValues(s); got[0] != "a" || got[1] != "" {
		t.Errorf("StringValues = %v", got)
	}

	b, _ := New("b", dtypes.Boolean, []any{true, nil})
	if got := BoolValues(b); got[0] != true || got[1] != false {
		t.Errorf("BoolValues = %v", got)
	}

	now := time.Now()
	d, _ := New("d", dtypes.Datetime, []any{now, nil})
	got := DatetimeValues(d)
	if !got[0].Equal(now) || !got[1].IsZero() {
		t.Errorf("DatetimeValues = %v", got)
	}
}

// TestZeroValueSeriesAccessors covers the nil-column guards of DataType and Clone.
func TestZeroValueSeriesAccessors(t *testing.T) {
	var z Series
	if z.DataType() != "" {
		t.Errorf("zero Series DataType = %q, want empty", z.DataType())
	}
	cloned := z.Clone()
	if cloned.DataType() != "" {
		t.Errorf("zero Series Clone().DataType = %q, want empty", cloned.DataType())
	}
}
