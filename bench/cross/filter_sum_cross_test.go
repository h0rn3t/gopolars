package cross

// BenchmarkCrossFilterSum measures a full filter+sum pipeline in gopolars vs
// Python Polars on identical float64 datasets, documenting the delta.
//
// Results are appended to cross_summary.json alongside the simd-primitive
// benchmarks from BenchmarkCross.
//
// Run:
//
//	go test ./bench/cross -bench=BenchmarkCrossFilterSum -benchmem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// filterSumSink keeps the benchmarked Sum result live so the compiler cannot
// eliminate the filter+sum pipeline as dead code.
var filterSumSink float64

// selectivityProfiles exercises the filter+sum pipeline at two selectivities.
// makeFloat64Slice draws uniformly from [-50, 50), so:
//   - "empty": col("a") > 50 selects ~0% of rows (degenerate gather, isolates
//     predicate-evaluation and allocation cost).
//   - "half":  col("a") >  0 selects ~50% of rows (realistic gather and
//     materialization cost).
var selectivityProfiles = []struct {
	name      string
	threshold float64
}{
	{"empty", 50.0},
	{"half", 0.0},
}

// BenchmarkCrossFilterSum benchmarks a full filter+sum pipeline in Go (the
// vectorized typed path) across both selectivity profiles and a range of sizes.
//
// The Python Polars comparison is opt-in: set GOPOLARS_BENCH_PYTHON=1 to spawn
// the harness. It is skipped by default because spawning Python per -count
// iteration dominates wall time and adds noise to the benchstat-tracked Go
// metrics (ns/op, B/op, allocs/op).
func BenchmarkCrossFilterSum(b *testing.B) {
	var mu sync.Mutex
	var results []summaryEntry
	pythonEnabled := os.Getenv("GOPOLARS_BENCH_PYTHON") != ""

	for _, prof := range selectivityProfiles {
		for _, n := range benchSizes {
			b.Run("filter_sum/"+prof.name+"/size_"+humanize(n), func(b *testing.B) {
				data := makeFloat64Slice(n)

				// Build a gopolars DataFrame from the data.
				vals := make([]any, n)
				for i, v := range data {
					vals[i] = v
				}
				df, err := polars.NewDataFrame(polars.NewDataFrameInput{
					Columns: []frame.SeriesInput{
						{Name: "a", Values: vals},
					},
				})
				if err != nil {
					b.Fatalf("build df: %v", err)
				}

				// Python baseline (opt-in via GOPOLARS_BENCH_PYTHON).
				var pythonSecPerOp float64
				if pythonEnabled {
					tmpFile := filepath.Join(b.TempDir(), "data.arrow")
					if err := writeArrowIPC(tmpFile, map[string][]float64{"a": data}); err != nil {
						b.Fatalf("write arrow ipc: %v", err)
					}
					b.Cleanup(func() { _ = os.Remove(tmpFile) })
					pyRes, pyErr := runPythonHarness("filter_sum", tmpFile, 5, prof.threshold)
					if pyErr == nil {
						pythonSecPerOp = pyRes.ElapsedSec / float64(pyRes.Iters)
						b.ReportMetric(pythonSecPerOp, "python_sec/op")
					} else {
						b.Logf("python harness unavailable: %v", pyErr)
					}
				}

				pred := polars.Col("a").Gt(polars.Lit(prof.threshold))
				b.SetBytes(int64(n * 8))
				b.ResetTimer()
				start := time.Now()
				for i := 0; i < b.N; i++ {
					filtered, err := df.Filter(pred)
					if err != nil {
						b.Fatalf("filter: %v", err)
					}
					sv, ok := filtered.Series("a")
					if !ok {
						b.Fatal("series not found")
					}
					filterSumSink = sv.Sum()
				}
				b.StopTimer()
				goNsPerOp := time.Since(start).Nanoseconds() / int64(b.N)

				if pythonSecPerOp > 0 {
					speedup := pythonSecPerOp / (float64(goNsPerOp) / 1e9)
					b.ReportMetric(speedup, "go_vs_python_speedup")
				}

				mu.Lock()
				results = append(results, summaryEntry{
					Op:             "filter_sum/" + prof.name,
					Size:           humanize(n),
					Elements:       n,
					GoNsPerOp:      goNsPerOp,
					PythonSecPerOp: pythonSecPerOp,
				})
				mu.Unlock()
			})
		}
	}

	if len(results) > 0 {
		_, testFile, _, _ := runtime.Caller(0)
		benchDir := filepath.Dir(testFile)
		summaryPath := filepath.Join(benchDir, "filter_sum_summary.json")
		f, err := os.Create(summaryPath)
		if err != nil {
			b.Fatalf("create summary: %v", err)
		}
		defer func() { _ = f.Close() }()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			b.Fatalf("encode summary: %v", err)
		}
		b.Logf("filter+sum delta written to %s", summaryPath)
	}
}
