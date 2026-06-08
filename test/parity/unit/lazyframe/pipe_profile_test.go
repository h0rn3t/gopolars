package lazyframe

// Ported from py-polars/tests/unit/lazyframe/test_lazyframe.py, test_serde.py and
// test_with_context.py (py-1.28.1, feasible subset): LazyFrame.pipe /
// pipe_with_schema / map_batches / profile / with_context.

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_with_context: another frame's columns become referenceable; here the
// context column c is used as a scalar via .first(). Result length follows the
// main frame; the null row propagates through the string concat.
func TestLazyWithContext(t *testing.T) {
	t.Parallel()
	dfA, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "b", DType: polars.String, Values: []any{"a", "c", nil}},
	}})
	if err != nil {
		t.Fatalf("dfA: %v", err)
	}
	dfB, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "c", Values: []any{"foo", "ham"}},
	}})
	if err != nil {
		t.Fatalf("dfB: %v", err)
	}
	out, err := dfA.Lazy().WithContext(dfB.Lazy()).
		Select(polars.Col("b").Add(polars.Col("c").First()).Alias("b")).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if out.Height() != 3 {
		t.Fatalf("height: got %d, want 3", out.Height())
	}
	b, _ := out.GetColumn("b")
	if v, _ := b.Value(0).(string); v != "afoo" {
		t.Fatalf("b[0]: got %v, want afoo", b.Value(0))
	}
	if v, _ := b.Value(1).(string); v != "cfoo" {
		t.Fatalf("b[1]: got %v, want cfoo", b.Value(1))
	}
	if b.Value(2) != nil {
		t.Fatalf("b[2]: got %v, want nil", b.Value(2))
	}
}

// test_with_context ShapeError: using a context column directly (full length)
// alongside a different-length main column is an error.
func TestLazyWithContextShapeError(t *testing.T) {
	t.Parallel()
	dfA, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("dfA: %v", err)
	}
	dfB, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "c", Values: []any{"foo", "ham"}},
	}})
	if err != nil {
		t.Fatalf("dfB: %v", err)
	}
	_, err = dfA.Lazy().WithContext(dfB.Lazy()).
		Select(polars.Col("a"), polars.Col("c")).
		Collect(context.Background())
	if err == nil {
		t.Fatal("expected a shape error combining a (len 3) with context c (len 2)")
	}
}

// LazyFrame.pipe threads the frame through a user function.
func TestLazyPipe(t *testing.T) {
	t.Parallel()
	out, err := lazyDF(t).Lazy().
		Pipe(func(l polars.LazyFrame) polars.LazyFrame {
			return l.Select(polars.Col("a"))
		}).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if cols := out.Columns(); len(cols) != 1 || cols[0] != "a" {
		t.Fatalf("columns: got %v, want [a]", out.Columns())
	}
}

// LazyFrame.pipe_with_schema also passes the (eager) schema to the function.
func TestLazyPipeWithSchema(t *testing.T) {
	t.Parallel()
	base := lazyDF(t).Lazy()
	var seen dtypes.Schema
	out, err := base.PipeWithSchema(func(l polars.LazyFrame, s dtypes.Schema) polars.LazyFrame {
		seen = s
		return l.Select(polars.Col("b"))
	}, base.Schema()).Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("schema passed: got %d fields, want 2", len(seen))
	}
	if cols := out.Columns(); len(cols) != 1 || cols[0] != "b" {
		t.Fatalf("columns: got %v, want [b]", out.Columns())
	}
}

// LazyFrame.map_batches applies a whole-DataFrame transform in the pipeline.
func TestLazyMapBatches(t *testing.T) {
	t.Parallel()
	out, err := lazyDF(t).Lazy().
		MapBatches(func(df polars.DataFrame) (polars.DataFrame, error) {
			return df.Head(2), nil
		}).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("height: got %d, want 2", out.Height())
	}
}

// LazyFrame.profile returns the materialized result plus a timing/plan report.
func TestLazyProfile(t *testing.T) {
	t.Parallel()
	result, report, err := lazyDF(t).Lazy().
		Filter(polars.Col("a").Gt(polars.Lit(int64(1)))).
		Profile(context.Background())
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if result.Height() != 3 {
		t.Fatalf("result height: got %d, want 3", result.Height())
	}
	if report == nil {
		t.Fatalf("profile report is nil")
	}
	if _, ok := report["operators"]; !ok {
		t.Fatalf("profile report missing 'operators' key: %v", report)
	}
}
