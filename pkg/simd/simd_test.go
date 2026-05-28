package simd

import (
	"math"
	"reflect"
	"testing"
)

func TestSumFloat64(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5}, 5},
		{"multi", []float64{1, 2, 3, 4}, 10},
		{"negative", []float64{-1, 1}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SumFloat64(c.in)
			if got != c.want {
				t.Fatalf("SumFloat64(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestMinFloat64(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5}, 5},
		{"multi", []float64{3, 1, 2}, 1},
		{"negative", []float64{-5, -10, -3}, -10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MinFloat64(c.in)
			if got != c.want {
				t.Fatalf("MinFloat64(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestMaxFloat64(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5}, 5},
		{"multi", []float64{3, 1, 2}, 3},
		{"negative", []float64{-5, -10, -3}, -3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MaxFloat64(c.in)
			if got != c.want {
				t.Fatalf("MaxFloat64(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestMinMaxFloat64(t *testing.T) {
	cases := []struct {
		name    string
		in      []float64
		wantMin float64
		wantMax float64
	}{
		{"empty", []float64{}, 0, 0},
		{"single", []float64{5}, 5, 5},
		{"multi", []float64{3, 1, 2}, 1, 3},
		{"negative", []float64{-5, -10, -3}, -10, -3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotMin, gotMax := MinMaxFloat64(c.in)
			if gotMin != c.wantMin || gotMax != c.wantMax {
				t.Fatalf("MinMaxFloat64(%v) = (%v, %v), want (%v, %v)", c.in, gotMin, gotMax, c.wantMin, c.wantMax)
			}
		})
	}
}

func TestCountFloat64(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want int64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5}, 1},
		{"multi", []float64{1, 2, 3}, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CountFloat64(c.in)
			if got != c.want {
				t.Fatalf("CountFloat64(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestAddSlicesFloat64(t *testing.T) {
	cases := []struct {
		name string
		a    []float64
		b    []float64
		want []float64
	}{
		{"empty_a", []float64{}, []float64{1, 2}, []float64{}},
		{"empty_b", []float64{1, 2}, []float64{}, []float64{}},
		{"equal", []float64{1, 2, 3}, []float64{4, 5, 6}, []float64{5, 7, 9}},
		{"mismatch", []float64{1, 2, 3}, []float64{4, 5}, []float64{5, 7}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AddSlicesFloat64(c.a, c.b)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("AddSlicesFloat64(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestMulSlicesFloat64(t *testing.T) {
	cases := []struct {
		name string
		a    []float64
		b    []float64
		want []float64
	}{
		{"empty_a", []float64{}, []float64{1, 2}, []float64{}},
		{"empty_b", []float64{1, 2}, []float64{}, []float64{}},
		{"equal", []float64{1, 2, 3}, []float64{4, 5, 6}, []float64{4, 10, 18}},
		{"mismatch", []float64{1, 2, 3}, []float64{4, 5}, []float64{4, 10}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MulSlicesFloat64(c.a, c.b)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("MulSlicesFloat64(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestDotProductFloat64(t *testing.T) {
	cases := []struct {
		name string
		a    []float64
		b    []float64
		want float64
	}{
		{"empty_a", []float64{}, []float64{1, 2}, 0},
		{"empty_b", []float64{1, 2}, []float64{}, 0},
		{"equal", []float64{1, 2, 3}, []float64{4, 5, 6}, 32},
		{"mismatch", []float64{1, 2, 3}, []float64{4, 5}, 14},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DotProductFloat64(c.a, c.b)
			if math.Abs(got-c.want) > 1e-12 {
				t.Fatalf("DotProductFloat64(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}
