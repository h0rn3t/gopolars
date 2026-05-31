package conformance

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDifferentialGroupByAgainstPythonPolars(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	check := exec.Command("python3", "-c", "import polars")
	if err := check.Run(); err != nil {
		t.Skip("python polars is not installed")
	}

	got, err := runGoPolarsPipeline()
	if err != nil {
		t.Fatalf("gopolars pipeline failed: %v", err)
	}
	want, err := runPythonPolarsPipeline()
	if err != nil {
		t.Fatalf("python polars pipeline failed: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("row count mismatch: got=%d want=%d", len(got), len(want))
	}
	for i := range got {
		if got[i].City != want[i].City || got[i].Total != want[i].Total {
			t.Fatalf("row %d mismatch: got=%+v want=%+v", i, got[i], want[i])
		}
	}
}

type cityTotal struct {
	City  string
	Total int64
}

func runGoPolarsPipeline() ([]cityTotal, error) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "city", Values: []any{"kyiv", "lviv", "kyiv", "odesa"}},
			{Name: "value", Values: []any{int64(10), int64(20), int64(30), int64(5)}},
		},
	})
	if err != nil {
		return nil, err
	}
	out, err := df.
		Lazy().
		Filter(polars.Col("value").Gt(polars.Lit(int64(10)))).
		GroupBy("city").
		Agg(polars.Sum(polars.Col("value")).Alias("total")).
		Sort(polars.SortInput{By: []string{"city"}}).
		Collect(context.Background())
	if err != nil {
		return nil, err
	}
	table, err := out.ToArrow(polars.ToArrowInput{})
	if err != nil {
		return nil, err
	}
	cities, ok := table.Columns["city"]
	if !ok {
		return nil, exec.ErrNotFound
	}
	totals, ok := table.Columns["total"]
	if !ok {
		return nil, exec.ErrNotFound
	}
	rows := make([]cityTotal, 0, len(cities))
	for i := range cities {
		rows = append(rows, cityTotal{
			City:  cities[i].(string),
			Total: totals[i].(int64),
		})
	}
	return rows, nil
}

func runPythonPolarsPipeline() ([]cityTotal, error) {
	script := `
import json
import polars as pl
df = pl.DataFrame({"city": ["kyiv", "lviv", "kyiv", "odesa"], "value": [10, 20, 30, 5]})
res = (
    df.lazy()
      .filter(pl.col("value") > 10)
      .group_by("city")
      .agg(pl.col("value").sum().alias("total"))
      .sort("city")
      .collect()
)
print(json.dumps(res.to_dicts(), separators=(",", ":")))
`
	cmd := exec.Command("python3", "-c", script)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, err
	}
	result := make([]cityTotal, 0, len(rows))
	for _, row := range rows {
		city, _ := row["city"].(string)
		total := int64(row["total"].(float64))
		result = append(result, cityTotal{City: city, Total: total})
	}
	return result, nil
}
