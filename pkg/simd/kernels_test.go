package simd

import (
	"reflect"
	"testing"
)

func TestCompareGTFloat64(t *testing.T) {
	cases := []struct {
		name      string
		in        []float64
		threshold float64
		want      []bool
	}{
		{"empty", []float64{}, 1, []bool{}},
		{"mixed", []float64{1, 2, 3, 4}, 2, []bool{false, false, true, true}},
		{"all_false", []float64{1, 1, 1}, 5, []bool{false, false, false}},
		{"negatives", []float64{-3, -1, 0}, -2, []bool{false, true, true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CompareGTFloat64(c.in, c.threshold)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("CompareGTFloat64(%v, %v) = %v, want %v", c.in, c.threshold, got, c.want)
			}
		})
	}
}

func TestCompareEQInt64(t *testing.T) {
	cases := []struct {
		name   string
		in     []int64
		target int64
		want   []bool
	}{
		{"empty", []int64{}, 1, []bool{}},
		{"mixed", []int64{1, 2, 2, 3}, 2, []bool{false, true, true, false}},
		{"none", []int64{1, 3, 5}, 2, []bool{false, false, false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CompareEQInt64(c.in, c.target)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("CompareEQInt64(%v, %v) = %v, want %v", c.in, c.target, got, c.want)
			}
		})
	}
}

func TestAndMask(t *testing.T) {
	cases := []struct {
		name string
		a    []bool
		b    []bool
		want []bool
	}{
		{"empty", []bool{}, []bool{}, []bool{}},
		{"equal", []bool{true, true, false}, []bool{true, false, false}, []bool{true, false, false}},
		{"mismatch_len", []bool{true, true, true}, []bool{true, false}, []bool{true, false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AndMask(c.a, c.b)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("AndMask(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestCompressIndices(t *testing.T) {
	cases := []struct {
		name string
		mask []bool
		want []int
	}{
		{"empty", []bool{}, []int{}},
		{"simple", []bool{true, false, true}, []int{0, 2}},
		{"none", []bool{false, false}, []int{}},
		{"all", []bool{true, true}, []int{0, 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CompressIndices(c.mask)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("CompressIndices(%v) = %v, want %v", c.mask, got, c.want)
			}
		})
	}
}
