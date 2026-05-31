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

// BenchmarkCrossFilterSum benchmarks filter+sum on 1M float64 rows in Go
// (vectorized typed path) and compares with the Python Polars time reported
// by the harness.
func BenchmarkCrossFilterSum(b *testing.B) {
	var mu sync.Mutex
	var results []summaryEntry

	for _, n := range benchSizes {
		b.Run("filter_sum/size_"+humanize(n), func(b *testing.B) {
			data := makeFloat64Slice(n)

			// Write the dataset as an Arrow IPC file for the Python harness.
			tmpFile := filepath.Join(b.TempDir(), "data.arrow")
			if err := writeArrowIPC(tmpFile, map[string][]float64{"a": data}); err != nil {
				b.Fatalf("write arrow ipc: %v", err)
			}
			b.Cleanup(func() { _ = os.Remove(tmpFile) })

			// Build a gopolars DataFrame from the same data.
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

			// Python baseline (skipped when Python is not available).
			var pythonSecPerOp float64
			pyRes, pyErr := runPythonHarness("filter_sum", tmpFile, 5)
			if pyErr == nil {
				pythonSecPerOp = pyRes.ElapsedSec / float64(pyRes.Iters)
				b.ReportMetric(pythonSecPerOp, "python_sec/op")
			} else {
				b.Logf("python harness unavailable: %v", pyErr)
				b.ReportMetric(0, "python_sec/op")
			}

			b.SetBytes(int64(n * 8))
			b.ResetTimer()
			start := time.Now()
			for i := 0; i < b.N; i++ {
				filtered, err := df.Filter(polars.Col("a").Gt(polars.Lit(50.0)))
				if err != nil {
					b.Fatalf("filter: %v", err)
				}
				sv, ok := filtered.Series("a")
				if !ok {
					b.Fatal("series not found")
				}
				_ = sv.Sum()
			}
			b.StopTimer()
			goNsPerOp := time.Since(start).Nanoseconds() / int64(b.N)

			if pythonSecPerOp > 0 {
				speedup := pythonSecPerOp / (float64(goNsPerOp) / 1e9)
				b.ReportMetric(speedup, "go_vs_python_speedup")
			}

			mu.Lock()
			results = append(results, summaryEntry{
				Op:             "filter_sum",
				Size:           humanize(n),
				Elements:       n,
				GoNsPerOp:      goNsPerOp,
				PythonSecPerOp: pythonSecPerOp,
			})
			mu.Unlock()
		})
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
