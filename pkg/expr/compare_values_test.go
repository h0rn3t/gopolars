package expr

import (
	"math"
	"testing"
	"time"
)

func TestCompareValues(t *testing.T) {
	t.Parallel()
	nan := math.NaN()
	l := func(v ...any) []any { return v }
	type st = map[string]any
	tests := []struct {
		name        string
		left, right any
		want        int
	}{
		{"list by first element", l(int64(1)), l(int64(2)), -1},
		{"list prefix first", l(int64(1)), l(int64(1), int64(2)), -1},
		{"equal lists", l(int64(1), "a"), l(int64(1), "a"), 0},
		{"empty list first", l(), l(int64(0)), -1},
		{"nested lists", l(l(int64(1))), l(l(int64(2))), -1},
		{"null element last", l(nil), l(int64(1)), 1},
		{"null elements equal", l(nil), l(nil), 0},
		{"nan element last", l(nan), l(1.0), 1},
		{"nan elements equal", l(nan), l(nan), 0},
		{"signed zeros equal", l(math.Copysign(0, -1)), l(0.0), 0},
		{"strings", l("a"), l("b"), -1},
		{"bools", l(false), l(true), -1},
		{"times", l(time.Unix(1, 0)), l(time.Unix(2, 0)), -1},
		{"durations", l(time.Second), l(time.Minute), -1},
		{"different element types", l(int64(1)), l("1"), -1},
		{"int and float elements", l(int64(1)), l(1.0), 1},
		{"struct first field decides", st{"b": int64(9), "a": int64(1)}, st{"b": int64(0), "a": int64(2)}, -1},
		{"struct second field", st{"a": int64(1), "b": int64(2)}, st{"a": int64(1), "b": int64(3)}, -1},
		{"equal structs", st{"a": int64(1), "b": "x"}, st{"b": "x", "a": int64(1)}, 0},
		{"struct with fewer fields first", st{"a": int64(1)}, st{"a": int64(1), "b": int64(0)}, -1},
		{"struct field names", st{"a": int64(5)}, st{"b": int64(0)}, -1},
		{"list of structs", l(st{"a": int64(1)}), l(st{"a": int64(2)}), -1},
		{"binary", []byte("ab"), []byte("b"), -1},
		{"binary prefix", []byte("a"), []byte("ab"), -1},
		{"list against struct", l(), st{}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CompareValues(tt.left, tt.right); got != tt.want {
				t.Errorf("CompareValues(%v, %v) = %d, want %d", tt.left, tt.right, got, tt.want)
			}
			if got := CompareValues(tt.right, tt.left); got != -tt.want {
				t.Errorf("CompareValues(%v, %v) = %d, want %d", tt.right, tt.left, got, -tt.want)
			}
		})
	}
}

func TestEvalBinEqualityOnNestedValues(t *testing.T) {
	t.Parallel()
	l := func(v ...any) []any { return v }
	tests := []struct {
		op          string
		left, right any
		want        any
	}{
		{"eq", l(int64(1)), l(int64(1)), true},
		{"eq", l(int64(1)), l(int64(1), int64(2)), false},
		{"eq", map[string]any{"a": int64(1)}, map[string]any{"a": int64(1)}, true},
		{"eq", []byte("x"), []byte("y"), false},
		{"eq", l(int64(1)), int64(1), false},
		{"eq", int64(1), l(int64(1)), false},
		{"eq", l(int64(1)), nil, nil},
		{"ne", l(int64(1)), l(int64(2)), true},
		{"ne", map[string]any{"a": int64(1)}, map[string]any{"a": int64(1)}, false},
		{"eq_missing", l(int64(1)), l(int64(1)), true},
		{"eq_missing", l(int64(1)), nil, false},
		{"ne_missing", l(int64(1)), l(int64(1)), false},
	}
	for _, tt := range tests {
		got, err := EvalBin(tt.op, tt.left, tt.right)
		if err != nil {
			t.Errorf("EvalBin(%s, %v, %v) error = %v", tt.op, tt.left, tt.right, err)
			continue
		}
		if got != tt.want {
			t.Errorf("EvalBin(%s, %v, %v) = %v, want %v", tt.op, tt.left, tt.right, got, tt.want)
		}
	}
}
