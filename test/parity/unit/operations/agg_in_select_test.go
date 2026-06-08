package operations

// Ported from py-polars aggregation-in-context semantics (test_aggregations.py /
// test_select.py, py-1.28.1): aggregation expressions are allowed in select and
// with_columns, reducing to a scalar that broadcasts (with_columns) or yields a
// single row (a pure-aggregation select).

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func aggSelectDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// A nested aggregation broadcasts: f - f.mean() centres the column.
func TestAggNestedBroadcast(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "f", DType: polars.Float64, Values: []any{1.0, 2.0, 3.0, 4.0}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out, err := df.WithColumns(
		polars.Col("f").Sub(polars.Mean(polars.Col("f"))).Alias("centered"),
	)
	if err != nil {
		t.Fatalf("with_columns: %v", err)
	}
	c, _ := out.GetColumn("centered")
	// mean(f)=2.5; centered = [-1.5,-0.5,0.5,1.5]
	for i, w := range []float64{-1.5, -0.5, 0.5, 1.5} {
		if v := toFloatAny(c.Value(i)); v != w {
			t.Fatalf("centered[%d]: got %v, want %v", i, c.Value(i), w)
		}
	}
}

// pl.col(x).first() is a scalar usable inside a row-wise expression.
func TestAggFirstNested(t *testing.T) {
	t.Parallel()
	out, err := aggSelectDF(t).Select(
		polars.Col("a").Add(polars.Col("a").First()).Alias("r"),
	)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if out.Height() != 4 {
		t.Fatalf("height: got %d, want 4", out.Height())
	}
	r, _ := out.GetColumn("r")
	// a + a.first() = a + 1
	for i, w := range []int64{2, 3, 4, 5} {
		if v, _ := r.Value(i).(int64); v != w {
			t.Fatalf("r[%d]: got %v, want %d", i, r.Value(i), w)
		}
	}
}

// A select of only aggregations reduces to a single row.
func TestAggPureSelectSingleRow(t *testing.T) {
	t.Parallel()
	out, err := aggSelectDF(t).Select(
		polars.Sum(polars.Col("a")).Alias("s"),
		polars.Max(polars.Col("a")).Alias("mx"),
	)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if out.Height() != 1 {
		t.Fatalf("height: got %d, want 1", out.Height())
	}
	s, _ := out.GetColumn("s")
	mx, _ := out.GetColumn("mx")
	if v := toFloatAny(s.Value(0)); v != 10 {
		t.Fatalf("sum: got %v, want 10", s.Value(0))
	}
	if v := toFloatAny(mx.Value(0)); v != 4 {
		t.Fatalf("max: got %v, want 4", mx.Value(0))
	}
}
