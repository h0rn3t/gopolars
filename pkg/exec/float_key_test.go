package exec

import (
	"fmt"
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// floatKeyFrame repeats k = [0, -0, NaN, NaN', 1] to n rows: three distinct
// keys (zero, NaN, one) at any n.
func floatKeyFrame(t *testing.T, n int) frame.DataFrame {
	t.Helper()
	base := []float64{0, math.Copysign(0, -1), math.NaN(), math.Float64frombits(0xfff8000000000001), 1}
	k := make([]float64, n)
	for i := range n {
		k[i] = base[i%len(base)]
	}
	df, err := frame.New(frame.NewInput{Series: []series.Series{series.FromFloat64("k", k, nil)}})
	if err != nil {
		t.Fatalf("frame.New: %v", err)
	}
	return df
}

func TestSetOpsSignedZeroAndNaN(t *testing.T) {
	t.Parallel()
	for _, n := range []int{5, 65536} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			t.Parallel()
			left := floatKeyFrame(t, n)
			right, err := frame.New(frame.NewInput{Series: []series.Series{
				series.FromFloat64("k", []float64{math.Copysign(0, -1), math.NaN()}, nil),
			}})
			if err != nil {
				t.Fatalf("frame.New(right): %v", err)
			}
			// Rows with i%5 < 4 hold a zero or a NaN; the rest hold 1.
			ones := n / 5
			tests := []struct {
				op   string
				want int
			}{
				{"intersect", n - ones},
				{"except", ones},
				{"union", 3},
			}
			for _, tt := range tests {
				out, err := applySetOp(left, right, tt.op)
				if err != nil {
					t.Fatalf("applySetOp(%s) error = %v", tt.op, err)
				}
				if out.Height() != tt.want {
					t.Errorf("applySetOp(%s) rows = %d, want %d", tt.op, out.Height(), tt.want)
				}
			}
		})
	}
}

func TestWindowPartitionSignedZero(t *testing.T) {
	t.Parallel()
	out, err := applyWindows(floatKeyFrame(t, 5), []logical.WindowSpec{{
		Func:        "row_number",
		Alias:       "rn",
		PartitionBy: []string{"k"},
	}})
	if err != nil {
		t.Fatalf("applyWindows(row_number over k) error = %v", err)
	}
	want := []any{int64(1), int64(2), int64(1), int64(2), int64(1)}
	if got := colValues(t, out, "rn"); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("applyWindows(row_number over k=[0 -0 NaN NaN 1]) = %v, want %v", got, want)
	}
}
