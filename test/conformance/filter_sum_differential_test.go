package conformance

import (
	"context"
	"encoding/json"
	"math"
	"os/exec"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// filterSumDiffN is above frame.parallelFilterThreshold so the lazy
// Filter().Sum() exercises the parallel fused masked-reduce path.
const filterSumDiffN = 50_000

// filterSumDiffValue is the deterministic dataset both engines reduce. Values
// are multiples of 0.5 in [-100, 100); the filtered sum stays well under 2^53,
// so it is exactly representable regardless of summation order — letting the Go
// fused reduction and Polars' SIMD reduction be compared without order-of-add
// drift.
func filterSumDiffValue(i int) float64 {
	return float64((i%400)-200) * 0.5
}

// TestDifferentialFilterSumAgainstPythonPolars validates the fused full-frame
// filter+sum (df.Lazy().Filter(col > 0).Sum()) against Python Polars on an
// identical, deterministically generated float64 column.
func TestDifferentialFilterSumAgainstPythonPolars(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	if err := exec.Command("python3", "-c", "import polars").Run(); err != nil {
		t.Skip("python polars is not installed")
	}

	got, err := runGoFilterSum()
	if err != nil {
		t.Fatalf("gopolars filter+sum failed: %v", err)
	}
	want, err := runPythonFilterSum()
	if err != nil {
		t.Fatalf("python polars filter+sum failed: %v", err)
	}
	if math.Abs(got-want) > 1e-6*(math.Abs(want)+1) {
		t.Fatalf("fused filter+sum diverges from Polars: got=%v want=%v", got, want)
	}
}

func runGoFilterSum() (float64, error) {
	vals := make([]any, filterSumDiffN)
	for i := range vals {
		vals[i] = filterSumDiffValue(i)
	}
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{{Name: "a", Values: vals}},
	})
	if err != nil {
		return 0, err
	}
	out, err := df.
		Lazy().
		Filter(polars.Col("a").Gt(polars.Lit(0.0))).
		Sum().
		Collect(context.Background())
	if err != nil {
		return 0, err
	}
	table, err := out.ToArrow(polars.ToArrowInput{})
	if err != nil {
		return 0, err
	}
	col, ok := table.Columns["a"]
	if !ok || len(col) == 0 {
		return 0, exec.ErrNotFound
	}
	return col[0].(float64), nil
}

func runPythonFilterSum() (float64, error) {
	script := `
import json
import polars as pl
vals = [((i % 400) - 200) * 0.5 for i in range(50000)]
res = pl.DataFrame({"a": vals}).lazy().filter(pl.col("a") > 0).sum().collect()
print(json.dumps(res["a"][0]))
`
	out, err := exec.Command("python3", "-c", script).Output()
	if err != nil {
		return 0, err
	}
	var sum float64
	if err := json.Unmarshal(out, &sum); err != nil {
		return 0, err
	}
	return sum, nil
}
