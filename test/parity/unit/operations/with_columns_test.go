package operations

// Ported from py-polars/tests/unit/operations/test_with_columns.py (py-1.28.1, representative subset)

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func wcDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

// with_columns adds a derived column.
func TestWithColumnsAdd(t *testing.T) {
	t.Parallel()
	out, err := wcDF(t).WithColumns(polars.Col("a").Add(polars.Lit(int64(10))).Alias("b"))
	if err != nil {
		t.Fatalf("with_columns: %v", err)
	}
	if out.Width() != 2 {
		t.Fatalf("width: got %d, want 2", out.Width())
	}
	b, _ := out.GetColumn("b")
	for i, w := range []int64{11, 12, 13} {
		if v, _ := b.Value(i).(int64); v != w {
			t.Fatalf("b[%d]: got %v, want %d", i, b.Value(i), w)
		}
	}
}

// with_columns can replace an existing column.
func TestWithColumnsReplace(t *testing.T) {
	t.Parallel()
	out, err := wcDF(t).WithColumns(polars.Col("a").Mul(polars.Lit(int64(2))).Alias("a"))
	if err != nil {
		t.Fatalf("with_columns replace: %v", err)
	}
	if out.Width() != 1 {
		t.Fatalf("width: got %d, want 1", out.Width())
	}
	a, _ := out.GetColumn("a")
	for i, w := range []int64{2, 4, 6} {
		if v, _ := a.Value(i).(int64); v != w {
			t.Fatalf("a[%d]: got %v, want %d", i, a.Value(i), w)
		}
	}
}
