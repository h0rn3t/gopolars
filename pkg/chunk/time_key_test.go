package chunk

import (
	"fmt"
	"math"
	"testing"
	"time"
)

var (
	farDate = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	// wrappedDate is the instant farDate.UnixNano() wraps to.
	wrappedDate = time.Date(1816, 3, 29, 5, 56, 8, 66277376, time.UTC)
	noonUTC     = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	noonPlusTwo = time.Date(2026, 1, 1, 14, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))
)

// datetimeKeys repeats [farDate, wrappedDate, noonUTC, noonPlusTwo, null] to
// n rows: four distinct keys, since both noons are one instant.
func datetimeKeys(n int) *Column {
	base := []time.Time{farDate, wrappedDate, noonUTC, noonPlusTwo, {}}
	vals := make([]time.Time, n)
	nulls := make([]bool, n)
	for i := range vals {
		vals[i] = base[i%len(base)]
		nulls[i] = i%len(base) == len(base)-1
	}
	return NewTime(vals, nulls)
}

func TestGroupIDsDatetimeDistinctInstants(t *testing.T) {
	t.Parallel()
	if farDate.UnixNano() != wrappedDate.UnixNano() {
		t.Fatalf("UnixNano(%v) = %d, want it to wrap to UnixNano(%v) = %d", farDate, farDate.UnixNano(), wrappedDate, wrappedDate.UnixNano())
	}
	for _, n := range []int{5, 65536} {
		keys := datetimeKeys(n)
		other := NewInt64(make([]int64, n), nil)
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			t.Parallel()
			if _, first := GroupIDs([]*Column{keys}, n); len(first) != 4 {
				t.Errorf("GroupIDs(datetime key, n=%d) groups = %d, want 4", n, len(first))
			}
			if _, first := GroupIDs([]*Column{keys, other}, n); len(first) != 4 {
				t.Errorf("GroupIDs(datetime and int keys, n=%d) groups = %d, want 4", n, len(first))
			}
			if first := FirstRows([]*Column{keys}, n); len(first) != 4 {
				t.Errorf("FirstRows(datetime key, n=%d) = %d rows, want 4", n, len(first))
			}
			if first := FirstRows([]*Column{keys, other}, n); len(first) != 4 {
				t.Errorf("FirstRows(datetime and int keys, n=%d) = %d rows, want 4", n, len(first))
			}
			if _, groups := GroupIDsUnordered([]*Column{keys}, n); groups != 4 {
				t.Errorf("GroupIDsUnordered(datetime key, n=%d) groups = %d, want 4", n, groups)
			}
		})
	}
}

func TestAppendRowKeyDatetime(t *testing.T) {
	t.Parallel()
	c := NewTime([]time.Time{farDate, wrappedDate, noonUTC, noonPlusTwo}, nil)
	key := func(row int) string { return string(AppendRowKey(nil, []*Column{c}, row)) }
	if key(0) == key(1) {
		t.Errorf("AppendRowKey(%v) = AppendRowKey(%v) = %x, want different keys", farDate, wrappedDate, key(0))
	}
	if key(2) != key(3) {
		t.Errorf("AppendRowKey(%v) = %x, AppendRowKey(%v) = %x, want equal", noonUTC, key(2), noonPlusTwo, key(3))
	}
}

func TestCanPackJoinKeyDatetime(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		vals  []time.Time
		nulls []bool
		want  bool
	}{
		{"in range", []time.Time{noonUTC, noonPlusTwo}, nil, true},
		{"far future", []time.Time{noonUTC, farDate}, nil, false},
		{"year one", []time.Time{time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)}, nil, false},
		{"far value under null", []time.Time{noonUTC, farDate}, []bool{false, true}, true},
		{"empty", nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CanPackJoinKey(NewTime(tt.vals, tt.nulls)); got != tt.want {
				t.Errorf("CanPackJoinKey(%v, nulls %v) = %t, want %t", tt.vals, tt.nulls, got, tt.want)
			}
		})
	}
}

func TestPackKeyFuncDatetime(t *testing.T) {
	t.Parallel()
	c := NewTime([]time.Time{farDate, wrappedDate, noonUTC, noonPlusTwo, time.Unix(0, math.MinInt64)}, nil)
	keyAt, ok := PackKeyFunc(c)
	if !ok {
		t.Fatal("PackKeyFunc(Datetime) ok = false, want true")
	}
	if keyAt(0) == keyAt(1) {
		t.Errorf("PackKeyFunc: key(%v) = key(%v) = %#x, want different keys", farDate, wrappedDate, keyAt(0))
	}
	if keyAt(2) != keyAt(3) {
		t.Errorf("PackKeyFunc: key(%v) = %#x, key(%v) = %#x, want equal", noonUTC, keyAt(2), noonPlusTwo, keyAt(3))
	}
	if CanPackJoinKey(NewTime([]time.Time{time.Unix(0, math.MinInt64)}, nil)) {
		t.Errorf("CanPackJoinKey(%v) = true, want false: its key is the one reserved for unpackable instants", time.Unix(0, math.MinInt64))
	}
}
