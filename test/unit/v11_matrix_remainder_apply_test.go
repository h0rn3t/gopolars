package unit

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestPyInteropArrayLikeMatchesToNumpy(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2)}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := polars.PyArrayDataFrame(df), df.ToNumpy(); len(got) != len(want) {
		t.Fatalf("len mismatch")
	}
}

func TestDataFrameSubSelectColumns(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1)}},
			{Name: "b", Values: []any{"x"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := df.SubSelectColumns("b")
	if err != nil {
		t.Fatal(err)
	}
	if sub.Width() != 1 {
		t.Fatalf("width %d", sub.Width())
	}
}

func TestLazyFrameMatchToSchemaAndSubSelect(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1)}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	lf := df.Lazy().SubSelectColumns("a")
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if out.Width() != 1 {
		t.Fatalf("width %d", out.Width())
	}
	schema := dtypes.Schema{{Name: "b", Type: dtypes.Float64}}
	lf2 := df.Lazy().MatchToSchema(schema)
	collected, err := lf2.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if collected.Width() != 2 {
		t.Fatalf("expected added column, width=%d", collected.Width())
	}
}

func TestSeriesStrLower(t *testing.T) {
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: dtypes.String, Values: []any{"Ab", nil}})
	if err != nil {
		t.Fatal(err)
	}
	low, err := s.Str().Lower()
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := low.Value(0).(string); v != "ab" {
		t.Fatalf("got %v", low.Value(0))
	}
}

func TestSeriesStrWrongDType(t *testing.T) {
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "n", DType: dtypes.Int64, Values: []any{int64(1)}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Str().Lower()
	if err == nil {
		t.Fatal("expected dtype error")
	}
}

func TestExprReverseAndRollingVarSelect(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "v", Values: []any{1.0, 2.0, 10.0, 11.0}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rev, err := df.Select(polars.Col("v").Reverse().Alias("rv"))
	if err != nil {
		t.Fatal(err)
	}
	v0, err := rev.Item(0, "rv")
	if err != nil {
		t.Fatal(err)
	}
	v3, err := rev.Item(3, "rv")
	if err != nil {
		t.Fatal(err)
	}
	if v0 != 11.0 || v3 != 1.0 {
		t.Fatalf("reverse column %+v", rev.ToDicts())
	}
	rv, err := df.Select(polars.Col("v").RollingVar(2).Alias("var2"))
	if err != nil {
		t.Fatal(err)
	}
	cell, err := rv.Item(1, "var2")
	if err != nil {
		t.Fatal(err)
	}
	if cell == nil {
		t.Fatal("expected variance at row 1")
	}
}

func TestFilterReversePredicate(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "v", Values: []any{1.0, 2.0, 3.0}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Рядок 0: reverse(col)[0] = col[2] = 3.0 > 2.5
	f, err := df.Filter(polars.Col("v").Reverse().Gt(polars.Lit(2.5)))
	if err != nil {
		t.Fatal(err)
	}
	if f.Height() != 1 {
		t.Fatalf("height %d", f.Height())
	}
}

func TestErrPySetItemNotSupported(t *testing.T) {
	if polars.ErrPySetItemNotSupported == nil {
		t.Fatal("nil error")
	}
}
