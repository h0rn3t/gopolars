package frame

import (
	"fmt"
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// signedZeroFrame repeats k = [0, -0, NaN, NaN', 1] to n rows, next to v = 1,
// j = 0 and s = "x": k has three distinct keys (zero, NaN, one) at any n.
func signedZeroFrame(t *testing.T, n int) DataFrame {
	t.Helper()
	base := []float64{0, math.Copysign(0, -1), math.NaN(), math.Float64frombits(0xfff8000000000001), 1}
	k := make([]float64, n)
	v := make([]int64, n)
	j := make([]int64, n)
	s := make([]string, n)
	for i := range n {
		k[i] = base[i%len(base)]
		v[i] = 1
		s[i] = "x"
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromFloat64("k", k, nil),
		series.FromInt64("v", v, nil),
		series.FromInt64("j", j, nil),
		series.FromString("s", s, nil),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return df
}

// signedZeroKey returns the canonical key class of row i of signedZeroFrame.
func signedZeroKey(i int) int {
	return []int{0, 0, 1, 1, 2}[i%5]
}

func TestSignedZeroAndNaNAreOneKey(t *testing.T) {
	t.Parallel()
	for _, n := range []int{5, 65536} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			t.Parallel()
			df := signedZeroFrame(t, n)
			zeros := 0
			for i := range n {
				if signedZeroKey(i) == 0 {
					zeros++
				}
			}

			for _, keys := range [][]string{{"k"}, {"k", "s"}, {"j", "k"}} {
				out, err := df.GroupBy(keys...).Agg(expr.Sum(expr.Col("v")).Alias("v"))
				if err != nil {
					t.Fatalf("GroupBy(%v) error = %v", keys, err)
				}
				if out.Height() != 3 {
					t.Errorf("GroupBy(%v).Agg(sum) groups = %d, want 3", keys, out.Height())
				}
				if n == 5 {
					if got := columnValues(t, out, "v"); fmt.Sprint(got) != "[2 2 1]" {
						t.Errorf("GroupBy(%v).Agg(sum) sums = %v, want [2 2 1]", keys, got)
					}
				}
			}

			if out, err := df.Unique("k"); err != nil || out.Height() != 3 {
				t.Errorf("Unique(k) = %d rows, %v; want 3 rows", out.Height(), err)
			}
			if got, err := df.NUnique("k"); err != nil || got != 3 {
				t.Errorf("NUnique(k) = %d, %v; want 3", got, err)
			}
			if out, err := df.Select(expr.NUnique(expr.Col("k")).Alias("n")); err != nil || fmt.Sprint(columnValues(t, out, "n")) != "[3]" {
				t.Errorf("Select(n_unique(k)) = %v, %v; want [3]", out.ToDicts(), err)
			}
			if out, err := df.GroupBy("j").Agg(expr.NUnique(expr.Col("k")).Alias("n")); err != nil || fmt.Sprint(columnValues(t, out, "n")) != "[3]" {
				t.Errorf("GroupBy(j).Agg(n_unique(k)) = %v, %v; want [3]", out.ToDicts(), err)
			}

			right, err := New(NewInput{Series: []series.Series{
				series.FromFloat64("k", []float64{0, math.Copysign(0, -1)}, nil),
				series.FromString("s", []string{"x", "x"}, nil),
			}})
			if err != nil {
				t.Fatalf("New(right): %v", err)
			}
			for _, on := range [][]string{{"k"}, {"k", "s"}} {
				out, err := df.Join(JoinInput{Other: right, LeftOn: on, RightOn: on, How: JoinTypeInner})
				if err != nil {
					t.Fatalf("Join(on %v) error = %v", on, err)
				}
				if out.Height() != 2*zeros {
					t.Errorf("Join(on %v) rows = %d, want %d", on, out.Height(), 2*zeros)
				}
			}

			over, err := df.Select(expr.Col("v").CumCount().Over("k").Alias("c"))
			if err != nil {
				t.Fatalf("Select(cum_count.over(k)) error = %v", err)
			}
			counts := [3]int64{}
			for i, got := range columnValues(t, over, "c") {
				counts[signedZeroKey(i)]++
				if got != counts[signedZeroKey(i)] {
					t.Errorf("cum_count.over(k) row %d = %v, want %d", i, got, counts[signedZeroKey(i)])
					break
				}
			}

			byIndex, err := df.Pivot([]string{"k"}, "s", "v", "sum")
			if err != nil {
				t.Fatalf("Pivot(index k) error = %v", err)
			}
			if byIndex.Height() != 3 {
				t.Errorf("Pivot(index k) rows = %d, want 3", byIndex.Height())
			}
			byColumn, err := df.Pivot([]string{"j"}, "k", "v", "sum")
			if err != nil {
				t.Fatalf("Pivot(columns k) error = %v", err)
			}
			if got := byColumn.Columns(); fmt.Sprint(got) != "[j 0 NaN 1]" {
				t.Errorf("Pivot(columns k) columns = %v, want [j 0 NaN 1]", got)
			}
		})
	}
}
