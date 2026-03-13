package unit

import (
	"context"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestNamespaceParityV06(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "s", Values: []any{"  abc ", "xyz"}},
			{Name: "ts", Values: []any{
				time.Date(2026, 3, 10, 15, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 11, 1, 30, 0, 0, time.UTC),
			}},
			{Name: "arr", Values: []any{[]any{"a", "b"}, []any{"x"}}},
		},
	})
	out, err := df.Select(
		polars.Col("s").StrTrim().Alias("trimmed"),
		polars.Col("s").StrTrim().StartsWith(polars.Lit("ab")).Alias("starts_ab"),
		polars.Col("ts").DtHour().Alias("hour"),
		polars.Col("ts").DtWeekday().Alias("weekday"),
		polars.Col("arr").ListGet(polars.Lit(int64(1))).Alias("arr_1"),
	)
	if err != nil {
		t.Fatalf("namespace select failed: %v", err)
	}
	trimmed, _ := out.Series("trimmed")
	if trimmed.Value(0) != "abc" {
		t.Fatalf("unexpected trimmed value")
	}
	starts, _ := out.Series("starts_ab")
	if starts.Value(0) != true {
		t.Fatalf("unexpected starts_with result")
	}
}

func TestTemporalWindowParityV06(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "ts", Values: []any{
				time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 10, 11, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 10, 13, 0, 0, 0, time.UTC),
			}},
			{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		},
	})
	dyn, err := df.GroupByDynamic(polars.DynamicGroupInput{
		By:      "ts",
		Every:   2 * time.Hour,
		Period:  2 * time.Hour,
		Closed:  "left",
		Label:   "left",
		AggExpr: polars.Sum(polars.Col("v")),
	})
	if err != nil {
		t.Fatalf("group_by_dynamic failed: %v", err)
	}
	if dyn.Height() == 0 {
		t.Fatalf("dynamic output is empty")
	}
	eagerRolling, err := df.RollingMean(polars.RollingMeanInput{
		By:      "ts",
		Value:   "v",
		Window:  2 * time.Hour,
		MinRows: 1,
		Output:  "roll",
		Closed:  "both",
	})
	if err != nil {
		t.Fatalf("eager rolling failed: %v", err)
	}
	lazyRolling, err := df.Lazy().RollingMean(polars.RollingMeanInput{
		By:      "ts",
		Value:   "v",
		Window:  2 * time.Hour,
		MinRows: 1,
		Output:  "roll",
		Closed:  "both",
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("lazy rolling failed: %v", err)
	}
	if eagerRolling.Height() != lazyRolling.Height() {
		t.Fatalf("rolling eager/lazy mismatch")
	}
}

func TestExplainDiagnosticsV06Markers(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "ts", Values: []any{
				time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 10, 11, 0, 0, 0, time.UTC),
			}},
			{Name: "v", Values: []any{int64(1), int64(2)}},
		},
	})
	diag := df.Lazy().
		RollingMean(polars.RollingMeanInput{By: "ts", Value: "v", Window: time.Hour, MinRows: 1, Output: "roll"}).
		ExplainDiagnostics(true)
	if diag["schema_version"] != "v2" {
		t.Fatalf("unexpected diagnostics schema")
	}
	if _, ok := diag["temporal_window_operations"]; !ok {
		t.Fatalf("missing temporal diagnostics marker")
	}
	if _, ok := diag["performance_markers"]; !ok {
		t.Fatalf("missing performance markers")
	}
}
