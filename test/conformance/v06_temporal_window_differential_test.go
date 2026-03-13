package conformance

import (
	"os/exec"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestTemporalWindowDifferentialFixtureV06(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "ts", Values: []any{
				time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 10, 11, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
			}},
			{Name: "v", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "s", Values: []any{" a ", "bb", "ccc"}},
			{Name: "arr", Values: []any{[]any{"x", "y"}, []any{"z"}, []any{"w", "q"}}},
		},
	})
	_, err := df.Select(
		polars.Col("s").StrTrim().Alias("s_trim"),
		polars.Col("arr").ListGet(polars.Lit(int64(0))).Alias("arr0"),
		polars.Col("ts").DtHour().Alias("h"),
	)
	if err != nil {
		t.Fatalf("namespace fixture failed: %v", err)
	}
	out, err := df.GroupByDynamic(polars.DynamicGroupInput{
		By:      "ts",
		Every:   2 * time.Hour,
		Period:  2 * time.Hour,
		Closed:  "left",
		Label:   "left",
		AggExpr: polars.Sum(polars.Col("v")),
	})
	if err != nil {
		t.Fatalf("dynamic fixture failed: %v", err)
	}
	if out.Height() == 0 {
		t.Fatalf("unexpected empty fixture result")
	}
}
