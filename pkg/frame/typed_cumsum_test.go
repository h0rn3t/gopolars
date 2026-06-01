package frame

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

func TestCumSumTypedParity(t *testing.T) {
	v := series.FromFloat64("v", []float64{1, 2, 0, 4}, []bool{false, false, true, false})
	df, _ := New(NewInput{Series: []series.Series{v}})

	got, err := df.Select(expr.Col("v").CumSum())
	if err != nil {
		t.Fatalf("cum_sum: %v", err)
	}
	out := got.cols[got.order[0]]
	// null at index 2 contributes nothing; cumulative carries forward.
	want := []float64{1, 3, 3, 7}
	for i, w := range want {
		if got := out.Value(i).(float64); got != w {
			t.Errorf("cum_sum[%d] = %v, want %v", i, got, w)
		}
	}
}

func TestCumSumNaNPropagates(t *testing.T) {
	v := series.FromFloat64("v", []float64{1, math.NaN(), 3}, nil)
	df, _ := New(NewInput{Series: []series.Series{v}})
	got, err := df.Select(expr.Col("v").CumSum())
	if err != nil {
		t.Fatalf("cum_sum: %v", err)
	}
	out := got.cols[got.order[0]]
	if out.Value(0).(float64) != 1 {
		t.Errorf("cum_sum[0] = %v, want 1", out.Value(0))
	}
	if !math.IsNaN(out.Value(1).(float64)) || !math.IsNaN(out.Value(2).(float64)) {
		t.Errorf("NaN should propagate through the running sum")
	}
}

func TestCumSumIntInput(t *testing.T) {
	v := series.FromInt64("v", []int64{10, 20, 30}, nil)
	df, _ := New(NewInput{Series: []series.Series{v}})
	got, err := df.Select(expr.Col("v").CumSum())
	if err != nil {
		t.Fatalf("cum_sum: %v", err)
	}
	out := got.cols[got.order[0]]
	want := []float64{10, 30, 60}
	for i, w := range want {
		if got := out.Value(i).(float64); got != w {
			t.Errorf("cum_sum[%d] = %v, want %v", i, got, w)
		}
	}
}

func BenchmarkCumSum(b *testing.B) {
	n := 1_000_000
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = float64(i)
	}
	df, _ := New(NewInput{Series: []series.Series{series.FromFloat64("v", vals, nil)}})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Select(expr.Col("v").CumSum()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRollingMean(b *testing.B) {
	n := 1_000_000
	vals := make([]float64, n)
	for i := range vals {
		vals[i] = float64(i%1000) - 500
	}
	df, _ := New(NewInput{Series: []series.Series{series.FromFloat64("v", vals, nil)}})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := df.Select(expr.Col("v").RollingMean(100)); err != nil {
			b.Fatal(err)
		}
	}
}
