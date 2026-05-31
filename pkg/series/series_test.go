package series

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

func TestNewAndTypedExtractors(t *testing.T) {
	t.Parallel()

	intS, err := New("id", dtypes.Int64, []any{int64(1), nil, int64(3)})
	if err != nil {
		t.Fatalf("int series: %v", err)
	}
	if got := Int64Values(intS); got[0] != 1 || got[2] != 3 {
		t.Fatalf("Int64Values: %+v", got)
	}

	floatS, err := New("score", dtypes.Float64, []any{float64(1.5), float64(2.5)})
	if err != nil {
		t.Fatalf("float series: %v", err)
	}
	if got := Float64Values(floatS); got[1] != 2.5 {
		t.Fatalf("Float64Values: %+v", got)
	}

	strS, err := New("city", dtypes.String, []any{"kyiv", "lviv"})
	if err != nil {
		t.Fatalf("string series: %v", err)
	}
	if got := StringValues(strS); got[0] != "kyiv" {
		t.Fatalf("StringValues: %+v", got)
	}

	boolS, err := New("flag", dtypes.Boolean, []any{true, false})
	if err != nil {
		t.Fatalf("bool series: %v", err)
	}
	if got := BoolValues(boolS); !got[0] || got[1] {
		t.Fatalf("BoolValues: %+v", got)
	}

	ts := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	dtS, err := New("ts", dtypes.Datetime, []any{ts})
	if err != nil {
		t.Fatalf("datetime series: %v", err)
	}
	if got := DatetimeValues(dtS); !got[0].Equal(ts) {
		t.Fatalf("DatetimeValues: %+v", got)
	}
}

func TestShiftPositiveAndNegative(t *testing.T) {
	t.Parallel()

	s, err := New("v", dtypes.Int64, []any{int64(10), int64(20), int64(30)})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	down := s.Shift(1)
	if !down.IsNull(0) || down.Value(1) != int64(10) {
		t.Fatalf("shift down: row0 null=%v row1=%v", down.IsNull(0), down.Value(1))
	}

	up := s.Shift(-1)
	if !up.IsNull(2) || up.Value(0) != int64(20) {
		t.Fatalf("shift up: row2 null=%v row0=%v", up.IsNull(2), up.Value(0))
	}

	clone := s.Shift(0)
	if clone.Value(2) != int64(30) {
		t.Fatalf("shift(0) має клонувати значення")
	}
}

func TestValidateTypeRejectsMismatch(t *testing.T) {
	t.Parallel()

	_, err := New("bad", dtypes.Int64, []any{"not-int"})
	if err == nil {
		t.Fatal("очікували помилку типу для Int64")
	}

	_, err = New("bad", dtypes.Decimal, []any{int64(1)})
	if err == nil {
		t.Fatal("очікували помилку типу для Decimal")
	}

	_, err = New("bad", dtypes.List, []any{"not-list"})
	if err == nil {
		t.Fatal("очікували помилку типу для List")
	}
}
