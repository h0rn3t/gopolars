package frame

import (
	"fmt"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

var (
	farDate = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	// wrappedDate is the instant farDate.UnixNano() wraps to.
	wrappedDate = time.Date(1816, 3, 29, 5, 56, 8, 66277376, time.UTC)
	noonUTC     = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	noonPlusTwo = time.Date(2026, 1, 1, 14, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))
)

// datetimeKeyFrame repeats [farDate, wrappedDate, noonUTC, noonPlusTwo] to n
// rows of key column "ts", with a constant "id" and a "v" of ones: three
// distinct instants.
func datetimeKeyFrame(t *testing.T, n int) DataFrame {
	t.Helper()
	base := []time.Time{farDate, wrappedDate, noonUTC, noonPlusTwo}
	ts := make([]time.Time, n)
	v := make([]int64, n)
	for i := range n {
		ts[i] = base[i%len(base)]
		v[i] = 1
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromColumn("ts", chunk.NewTime(ts, nil)),
		series.FromInt64("id", make([]int64, n), nil),
		series.FromInt64("v", v, nil),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return df
}

func TestDatetimeKeysDistinguishInstants(t *testing.T) {
	t.Parallel()
	for _, n := range []int{4, 2 * parallelGroupByThreshold} {
		df := datetimeKeyFrame(t, n)
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			t.Parallel()
			for _, keys := range [][]string{{"ts"}, {"id", "ts"}} {
				got, err := df.GroupBy(keys...).Agg(expr.Sum(expr.Col("v")))
				if err != nil {
					t.Fatalf("GroupBy(%v).Agg: %v", keys, err)
				}
				if got.Height() != 3 {
					t.Errorf("GroupBy(%v) on %d rows = %d groups, want 3", keys, n, got.Height())
				}
				uniq, err := df.Unique(keys...)
				if err != nil {
					t.Fatalf("Unique(%v): %v", keys, err)
				}
				if uniq.Height() != 3 {
					t.Errorf("Unique(%v) on %d rows = %d rows, want 3", keys, n, uniq.Height())
				}
			}
			if got, err := df.NUnique("ts"); err != nil || got != 3 {
				t.Errorf("NUnique(ts) on %d rows = %d, %v, want 3, nil", n, got, err)
			}
		})
	}
}

func TestJoinDatetimeKeysMatchInstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		left, right []time.Time
		want        int
	}{
		{"far and wrapped", []time.Time{farDate}, []time.Time{wrappedDate}, 0},
		{"wrapped and far", []time.Time{wrappedDate}, []time.Time{farDate}, 0},
		{"far and far", []time.Time{farDate}, []time.Time{farDate}, 1},
		{"in-range left, far right", []time.Time{wrappedDate, noonUTC}, []time.Time{farDate, noonPlusTwo}, 1},
		{"same instant, two zones", []time.Time{noonUTC}, []time.Time{noonPlusTwo}, 1},
	}
	frameOf := func(t *testing.T, ts []time.Time) DataFrame {
		t.Helper()
		df, err := New(NewInput{Series: []series.Series{series.FromColumn("k", chunk.NewTime(ts, nil))}})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		return df
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := frameOf(t, tt.left).Join(JoinInput{Other: frameOf(t, tt.right), LeftOn: []string{"k"}, RightOn: []string{"k"}, How: JoinTypeInner})
			if err != nil {
				t.Fatalf("Join(%v, %v): %v", tt.left, tt.right, err)
			}
			if got.Height() != tt.want {
				t.Errorf("inner Join(%v, %v) = %d rows, want %d", tt.left, tt.right, got.Height(), tt.want)
			}
		})
	}
}
