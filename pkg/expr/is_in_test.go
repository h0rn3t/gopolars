package expr

import (
	"testing"
	"time"
)

func TestIsInMembership(t *testing.T) {
	t.Parallel()
	noonUTC := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	noonPlusTwo := time.Date(2026, 1, 1, 14, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))
	tests := []struct {
		name  string
		value any
		list  any
		want  bool
	}{
		{"int64 in []int64", int64(3), []int64{1, 3}, true},
		{"int64 not in []int64", int64(2), []int64{1, 3}, false},
		{"int64 in []any", int64(3), []any{int64(1), int64(3)}, true},
		{"float64 in []float64", 2.5, []float64{2.5}, true},
		{"string in []string", "b", []string{"a", "b"}, true},
		{"string not in []string", "c", []string{"a", "b"}, false},
		{"bool in []bool", false, []bool{false}, true},
		{"datetime in []time.Time, other zone", noonUTC, []time.Time{noonPlusTwo}, true},
		{"datetime in []any, other zone", noonPlusTwo, []any{noonUTC}, true},
		{"datetime not in []time.Time", noonUTC, []time.Time{noonUTC.Add(time.Hour)}, false},
		{"null not in list without nil", nil, []int64{1}, false},
		{"null in list with nil", nil, []any{int64(1), nil}, true},
		{"int64 not in list of another type", int64(1), []float64{1}, false},
	}
	for _, tt := range tests {
		got, err := Eval(Col("v").IsIn(Lit(tt.list)), mapRow{"v": tt.value})
		if err != nil {
			t.Errorf("Eval(is_in) %s: error = %v", tt.name, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Eval(%v is_in %v) = %v, want %v (%s)", tt.value, tt.list, got, tt.want, tt.name)
		}
	}
}
