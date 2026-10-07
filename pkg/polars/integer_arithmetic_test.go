package polars

import (
	"math"
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

func TestInt64ArithmeticIsExactAndWraps(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		op   string
		a, b int64
		want int64
	}{
		{"above 2^53", "add", 1<<53 + 1, 0, 1<<53 + 1},
		{"max minus one", "sub", math.MaxInt64, 1, math.MaxInt64 - 1},
		{"max plus one wraps", "add", math.MaxInt64, 1, math.MinInt64},
		{"min minus one wraps", "sub", math.MinInt64, 1, math.MaxInt64},
		{"square wraps", "mul", 3037000500, 3037000500, -9223372036709301616},
		{"min times minus one wraps", "mul", math.MinInt64, -1, math.MinInt64},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// The second row checks that a null operand still yields null.
			a := mustSeries(t, "a", dtypes.Int64, []any{tc.a, nil})
			b := mustSeries(t, "b", dtypes.Int64, []any{tc.b, int64(1)})
			want := []any{tc.want, nil}

			var got Series
			var err error
			var e Expr
			switch tc.op {
			case "add":
				got, err = a.Add(b)
				e = Col("a").Add(Col("b"))
			case "sub":
				got, err = a.Sub(b)
				e = Col("a").Sub(Col("b"))
			case "mul":
				got, err = a.Mul(b)
				e = Col("a").Mul(Col("b"))
			}
			if err != nil {
				t.Fatalf("Series %s(%d, %d) error = %v", tc.op, tc.a, tc.b, err)
			}
			if got.DataType() != dtypes.Int64 || !slices.Equal(got.ToList(), want) {
				t.Errorf("Series %s(%d, %d) = %s %v, want %s %v", tc.op, tc.a, tc.b, got.DataType(), got.ToList(), dtypes.Int64, want)
			}

			d := mscFrame(t,
				frame.SeriesInput{Name: "a", DType: dtypes.Int64, Values: []any{tc.a, nil}},
				frame.SeriesInput{Name: "b", DType: dtypes.Int64, Values: []any{tc.b, int64(1)}},
			)
			out, err := d.Select(e.Alias("r"))
			if err != nil {
				t.Fatalf("Select(a %s b) error = %v", tc.op, err)
			}
			if got := frameColumnValues(t, out, "r"); !slices.Equal(got, want) {
				t.Errorf("Select(a %s b) with a=%d, b=%d = %v, want %v", tc.op, tc.a, tc.b, got, want)
			}
		})
	}
}

func TestInt64SumsAreExact(t *testing.T) {
	t.Parallel()

	const want = int64(9007199254740995)
	d := mscFrame(t,
		mscCol("k", "x", "x"),
		mscCol("c", "p", "p"),
		mscCol("v", int64(9007199254740993), int64(2)),
	)

	summed, err := d.Lazy().Select(Col("v")).Sum().Collect(t.Context())
	if err != nil {
		t.Fatalf("Lazy().Sum().Collect() error = %v", err)
	}
	if got := frameColumnValues(t, summed, "v"); !slices.Equal(got, []any{want}) {
		t.Errorf("Lazy().Sum() v = %v, want [%d]", got, want)
	}

	pivoted, err := d.Pivot(PivotInput{Index: "k", Columns: "c", Values: "v", Agg: "sum"})
	if err != nil {
		t.Fatalf("Pivot(sum) error = %v", err)
	}
	if got := frameColumnValues(t, pivoted, "p"); !slices.Equal(got, []any{want}) {
		t.Errorf("Pivot(sum) p = %v, want [%d]", got, want)
	}
}
