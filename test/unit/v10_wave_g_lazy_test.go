package unit

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestV10WaveGLazySurface(t *testing.T) {
	t.Parallel()

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(1)}},
		},
	})
	if err != nil {
		t.Fatalf("new df failed: %v", err)
	}
	lf := df.Lazy()

	methods := []string{
		"CollectSchema", "Columns", "Width", "Schema", "Dtypes", "Describe",
		"JoinAsof", "MapBatches", "MergeSorted", "Pipe", "PipeWithSchema", "Rolling",
		"SelectSeq", "WithColumnsSeq", "WithContext", "Remove", "TopK", "Show", "Lazy",
		"SinkDelta", "SinkIceberg",
	}
	for _, name := range methods {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !reflect.ValueOf(lf).MethodByName(name).IsValid() {
				t.Fatalf("expected LazyFrame method %s to be exposed", name)
			}
		})
	}
}

func TestV10WaveGLazyMethods(t *testing.T) {
	t.Parallel()

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(1), float64(2), float64(3)}},
			{Name: "y", Values: []any{"a", "b", "c"}},
			{Name: "ts", Values: []any{
				time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
			}},
		},
	})
	if err != nil {
		t.Fatalf("new df failed: %v", err)
	}
	lf := df.Lazy()

	if got := callLazyScalar(t, lf, "Columns"); !reflect.DeepEqual(got, []string{"x", "y", "ts"}) {
		t.Fatalf("unexpected Columns: got %v", got)
	}
	if got := callLazyScalar(t, lf, "Width").(int); got != 3 {
		t.Fatalf("unexpected Width: got %d want 3", got)
	}
	if got := callLazyScalar(t, lf, "Dtypes"); !reflect.DeepEqual(got, []dtypes.DataType{polars.Float64, polars.String, polars.Datetime}) {
		t.Fatalf("unexpected Dtypes: got %v", got)
	}
	if got := callLazyScalar(t, lf, "CollectSchema"); !reflect.DeepEqual(got, callLazyScalar(t, lf, "Schema")) {
		t.Fatalf("expected CollectSchema and Schema to match")
	}

	describe := callLazyDataFrameResult(t, lf, "Describe")
	if describe.Width() == 0 || describe.Height() == 0 {
		t.Fatalf("unexpected Describe shape: %v", describe.Shape())
	}

	selectSeq := callLazyFrame(t, lf, "SelectSeq", polars.Col("x"))
	selected, err := selectSeq.Collect(context.Background())
	if err != nil {
		t.Fatalf("select_seq collect failed: %v", err)
	}
	if selected.Width() != 1 {
		t.Fatalf("unexpected SelectSeq width: got %d want 1", selected.Width())
	}

	withColsSeq := callLazyFrame(t, lf, "WithColumnsSeq", polars.Col("x").Alias("x_copy"))
	withColsDF, err := withColsSeq.Collect(context.Background())
	if err != nil {
		t.Fatalf("with_columns_seq collect failed: %v", err)
	}
	if _, err := withColsDF.GetColumn("x_copy"); err != nil {
		t.Fatalf("expected x_copy column: %v", err)
	}

	topK := callLazyFrame(t, lf, "TopK", 2, "x")
	topKDF, err := topK.Collect(context.Background())
	if err != nil {
		t.Fatalf("top_k collect failed: %v", err)
	}
	col, err := topKDF.GetColumn("x")
	if err != nil {
		t.Fatalf("get top_k column failed: %v", err)
	}
	if col.Value(0).(float64) != 3 || col.Value(1).(float64) != 2 {
		t.Fatalf("unexpected TopK values: %v, %v", col.Value(0), col.Value(1))
	}

	if show := callLazyScalar(t, lf, "Show", 2).(string); show == "" {
		t.Fatalf("expected non-empty Show")
	}

	lazyClone := callLazyFrame(t, lf, "Lazy")
	cloned, err := lazyClone.Collect(context.Background())
	if err != nil {
		t.Fatalf("lazy collect failed: %v", err)
	}
	if cloned.Height() != df.Height() {
		t.Fatalf("unexpected Lazy clone height: got %d want %d", cloned.Height(), df.Height())
	}

	removed := callLazyFrame(t, lf, "Remove", "y")
	removedDF, err := removed.Collect(context.Background())
	if err != nil {
		t.Fatalf("remove collect failed: %v", err)
	}
	if removedDF.Width() != 2 {
		t.Fatalf("unexpected Remove width: got %d want 2", removedDF.Width())
	}

	other, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(1), float64(2), float64(3)}},
			{Name: "z", Values: []any{"u", "v", "w"}},
		},
	})
	if err != nil {
		t.Fatalf("new other df failed: %v", err)
	}
	joined := callLazyFrame(t, lf, "JoinAsof", polars.JoinInput{
		Other:   other,
		LeftOn:  []string{"x"},
		RightOn: []string{"x"},
		How:     "asof",
	})
	joinedDF, err := joined.Collect(context.Background())
	if err != nil {
		t.Fatalf("join_asof collect failed: %v", err)
	}
	if joinedDF.Width() < df.Width() {
		t.Fatalf("unexpected JoinAsof width: got %d want >= %d", joinedDF.Width(), df.Width())
	}

	mapped := callLazyFrame(t, lf, "MapBatches", func(batch polars.DataFrame) (polars.DataFrame, error) {
		return batch.Select(polars.Col("x"))
	})
	mappedDF, err := mapped.Collect(context.Background())
	if err != nil {
		t.Fatalf("map_batches collect failed: %v", err)
	}
	if mappedDF.Width() != 1 {
		t.Fatalf("unexpected MapBatches width: got %d want 1", mappedDF.Width())
	}

	mergeOther, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "x", Values: []any{float64(4), float64(5), float64(6)}},
			{Name: "y", Values: []any{"d", "e", "f"}},
			{Name: "ts", Values: []any{
				time.Date(2026, 4, 1, 13, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 1, 14, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 1, 15, 0, 0, 0, time.UTC),
			}},
		},
	})
	if err != nil {
		t.Fatalf("new merge df failed: %v", err)
	}
	merged := callLazyFrame(t, lf, "MergeSorted", mergeOther.Lazy(), "x")
	mergedDF, err := merged.Collect(context.Background())
	if err != nil {
		t.Fatalf("merge_sorted collect failed: %v", err)
	}
	if mergedDF.Height() != df.Height()+mergeOther.Height() {
		t.Fatalf("unexpected MergeSorted height: got %d want %d", mergedDF.Height(), df.Height()+mergeOther.Height())
	}

	piped := callLazyFrame(t, lf, "Pipe", func(in polars.LazyFrame) polars.LazyFrame {
		return in.Select(polars.Col("x"))
	})
	pipedDF, err := piped.Collect(context.Background())
	if err != nil {
		t.Fatalf("pipe collect failed: %v", err)
	}
	if pipedDF.Width() != 1 {
		t.Fatalf("unexpected Pipe width: got %d want 1", pipedDF.Width())
	}

	pipeWithSchema := callLazyFrame(t, lf, "PipeWithSchema", func(in polars.LazyFrame, schema dtypes.Schema) polars.LazyFrame {
		if len(schema) == 0 {
			t.Fatalf("expected schema fields")
		}
		return in.Limit(1)
	}, df.Schema())
	pipeWithSchemaDF, err := pipeWithSchema.Collect(context.Background())
	if err != nil {
		t.Fatalf("pipe_with_schema collect failed: %v", err)
	}
	if pipeWithSchemaDF.Height() != 1 {
		t.Fatalf("unexpected PipeWithSchema height: got %d want 1", pipeWithSchemaDF.Height())
	}

	rolling := callLazyFrame(t, lf, "Rolling", polars.RollingMeanInput{
		By:     "ts",
		Value:  "x",
		Window: time.Hour,
		Output: "rx",
	})
	rollingDF, err := rolling.Collect(context.Background())
	if err != nil {
		t.Fatalf("rolling collect failed: %v", err)
	}
	if _, err := rollingDF.GetColumn("rx"); err != nil {
		t.Fatalf("expected rolling output column: %v", err)
	}

	withContext := callLazyFrame(t, lf, "WithContext", other.Lazy())
	withContextDF, err := withContext.Collect(context.Background())
	if err != nil {
		t.Fatalf("with_context collect failed: %v", err)
	}
	if withContextDF.Height() != df.Height() {
		t.Fatalf("unexpected WithContext height: got %d want %d", withContextDF.Height(), df.Height())
	}

	if err := callLazyError(t, lf, "SinkDelta", context.Background(), "/tmp/test-delta"); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected SinkDelta not supported error, got %v", err)
	}
	if err := callLazyError(t, lf, "SinkIceberg", context.Background(), "/tmp/test-iceberg"); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected SinkIceberg not supported error, got %v", err)
	}
}

func callLazyFrame(t *testing.T, lf polars.LazyFrame, method string, args ...any) polars.LazyFrame {
	t.Helper()

	results := callLazyMethod(t, lf, method, args...)
	if len(results) != 1 {
		t.Fatalf("expected %s to return one value, got %d", method, len(results))
	}
	out, ok := results[0].Interface().(polars.LazyFrame)
	if !ok {
		t.Fatalf("expected %s to return polars.LazyFrame, got %T", method, results[0].Interface())
	}
	return out
}

func callLazyDataFrameResult(t *testing.T, lf polars.LazyFrame, method string, args ...any) polars.DataFrame {
	t.Helper()

	results := callLazyMethod(t, lf, method, args...)
	if len(results) != 2 {
		t.Fatalf("expected %s to return (polars.DataFrame, error), got %d values", method, len(results))
	}
	if err, _ := results[1].Interface().(error); err != nil {
		t.Fatalf("%s returned error: %v", method, err)
	}
	out, ok := results[0].Interface().(polars.DataFrame)
	if !ok {
		t.Fatalf("expected %s to return polars.DataFrame, got %T", method, results[0].Interface())
	}
	return out
}

func callLazyScalar(t *testing.T, lf polars.LazyFrame, method string, args ...any) any {
	t.Helper()

	results := callLazyMethod(t, lf, method, args...)
	if len(results) != 1 {
		t.Fatalf("expected %s to return one value, got %d", method, len(results))
	}
	return results[0].Interface()
}

func callLazyError(t *testing.T, lf polars.LazyFrame, method string, args ...any) error {
	t.Helper()

	results := callLazyMethod(t, lf, method, args...)
	if len(results) != 1 {
		t.Fatalf("expected %s to return one value, got %d", method, len(results))
	}
	err, _ := results[0].Interface().(error)
	return err
}

func callLazyMethod(t *testing.T, lf polars.LazyFrame, method string, args ...any) []reflect.Value {
	t.Helper()

	target := reflect.ValueOf(lf)
	fn := target.MethodByName(method)
	if !fn.IsValid() {
		t.Fatalf("expected LazyFrame method %s to be exposed", method)
	}
	callArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		callArgs[i] = reflect.ValueOf(arg)
	}
	return fn.Call(callArgs)
}
