package polars

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

func TestQuantileNeverPanics(t *testing.T) {
	t.Parallel()

	nan := math.NaN()
	cases := []struct {
		q float64
		// want is the linear-interpolation quantile of [1 2 3 4]; wantFrame is
		// the nearest-rank value DataFrame.Quantile returns.
		want, wantFrame float64
	}{
		{nan, nan, nan},
		{math.Inf(1), 4, 4},
		{math.Inf(-1), 1, 1},
		{-0.5, 1, 1},
		{1.5, 4, 4},
		{0, 1, 1},
		{0.5, 2.5, 3},
		{1, 4, 4},
	}
	for _, tc := range cases {
		t.Run(strconv.FormatFloat(tc.q, 'g', -1, 64), func(t *testing.T) {
			t.Parallel()

			s := mustSeries(t, "v", dtypes.Int64, []any{int64(1), int64(2), int64(3), int64(4)})
			if got := s.Quantile(tc.q); !sameFloatOrNaN(got, tc.want) {
				t.Errorf("Series[1 2 3 4].Quantile(%v) = %v, want %v", tc.q, got, tc.want)
			}

			// A NaN probability has no quantile, so the full window is null.
			var last any = tc.want
			if math.IsNaN(tc.want) {
				last = nil
			}
			wantRolling := []any{nil, nil, nil, last}
			if got := s.RollingQuantile(4, tc.q).ToList(); !slices.Equal(got, wantRolling) {
				t.Errorf("Series[1 2 3 4].RollingQuantile(4, %v) = %v, want %v", tc.q, got, wantRolling)
			}
			by := mustSeries(t, "by", dtypes.Int64, []any{int64(1), int64(2), int64(3), int64(4)})
			rolled, err := s.RollingQuantileBy(by, 4, tc.q)
			if err != nil {
				t.Fatalf("RollingQuantileBy(4, %v) error = %v", tc.q, err)
			}
			if got := rolled.ToList(); !slices.Equal(got, wantRolling) {
				t.Errorf("Series[1 2 3 4].RollingQuantileBy(4, %v) = %v, want %v", tc.q, got, wantRolling)
			}

			d := mscFrame(t, mscCol("v", int64(1), int64(2), int64(3), int64(4)))
			if got := d.Quantile(tc.q)["v"]; !sameFloatOrNaN(got, tc.wantFrame) {
				t.Errorf("DataFrame{v: [1 2 3 4]}.Quantile(%v)[v] = %v, want %v", tc.q, got, tc.wantFrame)
			}
		})
	}
}

func TestLazyQuantileRejectsInvalidProbability(t *testing.T) {
	t.Parallel()

	for _, q := range []float64{math.NaN(), 1.5, -0.5, math.Inf(1), math.Inf(-1)} {
		t.Run(strconv.FormatFloat(q, 'g', -1, 64), func(t *testing.T) {
			t.Parallel()

			d := mscFrame(t, mscCol("v", 1.0, 2.0))
			out, err := d.Lazy().Quantile(q).Collect(t.Context())
			if err == nil || !strings.Contains(err.Error(), "quantile must be between 0.0 and 1.0") {
				t.Errorf("Lazy().Quantile(%v).Collect() = %v, %v; want the quantile range error", q, out, err)
			}
		})
	}
}

func TestValueReplacingMethodsKeepEveryRow(t *testing.T) {
	t.Parallel()

	ints := func(t *testing.T) Series {
		return mustSeries(t, "n", dtypes.Int64, []any{int64(1), int64(2), int64(3)})
	}
	lists := func(t *testing.T, values ...any) Series {
		return mustSeries(t, "n", dtypes.List, values)
	}
	cases := []struct {
		name      string
		call      func(t *testing.T) Series
		wantDType dtypes.DataType
		want      []any
	}{
		{
			name:      "Replace int with int",
			call:      func(t *testing.T) Series { return ints(t).Replace(int64(2), int64(5)) },
			wantDType: dtypes.Int64,
			want:      []any{int64(1), int64(5), int64(3)},
		},
		{
			name:      "Replace int with float",
			call:      func(t *testing.T) Series { return ints(t).Replace(int64(2), 2.5) },
			wantDType: dtypes.Float64,
			want:      []any{1.0, 2.5, 3.0},
		},
		{
			name:      "Replace int with string",
			call:      func(t *testing.T) Series { return ints(t).Replace(int64(2), "x") },
			wantDType: dtypes.String,
			want:      []any{"1", "x", "3"},
		},
		{
			name: "Replace float with int",
			call: func(t *testing.T) Series {
				return mustSeries(t, "n", dtypes.Float64, []any{1.5, 2.5}).Replace(1.5, int64(2))
			},
			wantDType: dtypes.Float64,
			want:      []any{2.0, 2.5},
		},
		{
			name: "ReplaceStrict int with string",
			call: func(t *testing.T) Series {
				out, err := ints(t).ReplaceStrict(int64(2), "x")
				if err != nil {
					t.Fatalf("ReplaceStrict(2, x) error = %v", err)
				}
				return out
			},
			wantDType: dtypes.String,
			want:      []any{"1", "x", "3"},
		},
		{
			name:      "ExtendConstant Go int",
			call:      func(t *testing.T) Series { return ints(t).ExtendConstant(7, 2) },
			wantDType: dtypes.Int64,
			want:      []any{int64(1), int64(2), int64(3), int64(7), int64(7)},
		},
		{
			name:      "ExtendConstant string",
			call:      func(t *testing.T) Series { return ints(t).ExtendConstant("a", 1) },
			wantDType: dtypes.String,
			want:      []any{"1", "2", "3", "a"},
		},
		{
			name:      "ExtendConstant null",
			call:      func(t *testing.T) Series { return ints(t).ExtendConstant(nil, 1) },
			wantDType: dtypes.Int64,
			want:      []any{int64(1), int64(2), int64(3), nil},
		},
		{
			name:      "Explode mixed lists",
			call:      func(t *testing.T) Series { return lists(t, []any{int64(1)}, []any{"a"}).Explode() },
			wantDType: dtypes.String,
			want:      []any{"1", "a"},
		},
		{
			name:      "Explode ints and floats",
			call:      func(t *testing.T) Series { return lists(t, []any{int64(1)}, []any{2.5}).Explode() },
			wantDType: dtypes.Float64,
			want:      []any{1.0, 2.5},
		},
		{
			name:      "Flatten mixed lists",
			call:      func(t *testing.T) Series { return lists(t, []any{int64(1), nil}, []any{"a"}).Flatten() },
			wantDType: dtypes.String,
			want:      []any{"1", nil, "a"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.call(t)
			if got.Name() != "n" || got.DataType() != tc.wantDType || !slices.Equal(got.ToList(), tc.want) {
				t.Errorf("%s = %q %s %v, want %q %s %v", tc.name, got.Name(), got.DataType(), got.ToList(), "n", tc.wantDType, tc.want)
			}
		})
	}
}

func TestReplaceStrictReportsMissingValue(t *testing.T) {
	t.Parallel()

	s := mustSeries(t, "n", dtypes.Int64, []any{int64(1), int64(2)})
	if out, err := s.ReplaceStrict(int64(9), int64(0)); err == nil {
		t.Errorf("ReplaceStrict(9, 0) = %v, nil; want an error", out.ToList())
	}
}

// sameFloatOrNaN reports whether a and b are equal or both NaN.
func sameFloatOrNaN(a, b float64) bool {
	return a == b || math.IsNaN(a) && math.IsNaN(b)
}
