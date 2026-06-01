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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// filterSumSink keeps the benchmarked Sum result live so the compiler cannot
// eliminate the filter+sum pipeline as dead code.
var filterSumSink float64

// printSummaryTable prints a formatted ASCII table of filter+sum results to
// stdout. When Python measurements are absent (GOPOLARS_BENCH_PYTHON not set)
// the Python and winner columns are omitted.
func printSummaryTable(entries []summaryEntry) {
	if len(entries) == 0 {
		return
	}

	// Deduplicate by (op, size) — keep the last (warmest) value per key.
	type key struct{ op, size string }
	dedup := make(map[key]summaryEntry)
	var order []key
	for _, e := range entries {
		k := key{e.Op, e.Size}
		if _, exists := dedup[k]; !exists {
			order = append(order, k)
		}
		dedup[k] = e
	}

	hasPython := false
	for _, k := range order {
		if dedup[k].PythonSecPerOp > 0 {
			hasPython = true
			break
		}
	}

	// rpad / lpad count Unicode code points so µ in "µs" aligns correctly.
	rpad := func(s string, w int) string {
		n := len([]rune(s))
		if n >= w {
			return s
		}
		return s + strings.Repeat(" ", w-n)
	}
	lpad := func(s string, w int) string {
		n := len([]rune(s))
		if n >= w {
			return s
		}
		return strings.Repeat(" ", w-n) + s
	}

	fmtDur := func(ns int64) string {
		if ns <= 0 {
			return "n/a"
		}
		switch {
		case ns >= 1_000_000_000:
			return fmt.Sprintf("%.2f s", float64(ns)/1e9)
		case ns >= 1_000_000:
			return fmt.Sprintf("%.2f ms", float64(ns)/1e6)
		case ns >= 1_000:
			return fmt.Sprintf("%.2f µs", float64(ns)/1e3)
		default:
			return fmt.Sprintf("%d ns", ns)
		}
	}

	fmtBytes := func(bp int64) string {
		if bp <= 0 {
			return "n/a"
		}
		switch {
		case bp >= 1<<20:
			return fmt.Sprintf("%.1f MB", float64(bp)/(1<<20))
		case bp >= 1<<10:
			return fmt.Sprintf("%.1f KB", float64(bp)/(1<<10))
		default:
			return fmt.Sprintf("%d B", bp)
		}
	}
	fmtWinner := func(pySecPerOp float64, goNs int64) string {
		if pySecPerOp <= 0 || goNs <= 0 {
			return "n/a"
		}
		speedup := pySecPerOp / (float64(goNs) / 1e9)
		if speedup >= 1 {
			return fmt.Sprintf("Go  ×%.1f", speedup)
		}
		return fmt.Sprintf("Py  ×%.1f", 1/speedup)
	}

	// Group rows by op family (selectivity + engine), each ascending by size.
	sort.Slice(order, func(i, j int) bool {
		ei, ej := dedup[order[i]], dedup[order[j]]
		if ei.Op != ej.Op {
			return ei.Op < ej.Op
		}
		return ei.Elements < ej.Elements
	})

	const (
		wProfile = 22
		wTime    = 13
		wWinner  = 11
		wBytes   = 11
		wAllocs  = 10
	)

	// Column layout. With Python enabled the speed pair (Go / Python time +
	// winner) is followed by the memory pair (Go B/op / Python peak RSS) so each
	// metric sits next to its cross-language counterpart; allocs/op trails.
	divider := func() string {
		s := "+" + strings.Repeat("-", wProfile+2) + "+" + strings.Repeat("-", wTime+2)
		if hasPython {
			s += "+" + strings.Repeat("-", wTime+2) + "+" + strings.Repeat("-", wWinner+2)
		}
		s += "+" + strings.Repeat("-", wBytes+2)
		if hasPython {
			s += "+" + strings.Repeat("-", wBytes+2)
		}
		s += "+" + strings.Repeat("-", wAllocs+2)
		return s + "+"
	}

	row := func(profile, goStr, pyStr, winStr, goMem, pyMem, aStr string) {
		line := "| " + rpad(profile, wProfile) + " | " + lpad(goStr, wTime)
		if hasPython {
			line += " | " + lpad(pyStr, wTime) + " | " + lpad(winStr, wWinner)
		}
		line += " | " + lpad(goMem, wBytes)
		if hasPython {
			line += " | " + lpad(pyMem, wBytes)
		}
		line += " | " + lpad(aStr, wAllocs)
		fmt.Println(line + " |")
	}

	fmt.Println()
	fmt.Println(divider())
	if hasPython {
		row("profile / size", "Go time", "Py time", "winner", "Go B/op", "Py peak RSS", "allocs/op")
	} else {
		row("profile / size", "Go time", "", "", "Go B/op", "", "allocs/op")
	}
	fmt.Println(divider())

	lastOp := ""
	for _, k := range order {
		e := dedup[k]
		if lastOp != "" && lastOp != e.Op {
			fmt.Println(divider())
		}
		lastOp = e.Op

		profile := strings.TrimPrefix(e.Op, "filter_sum/") + " / " + e.Size
		pyNs := int64(e.PythonSecPerOp * 1e9)
		row(profile, fmtDur(e.GoNsPerOp), fmtDur(pyNs), fmtWinner(e.PythonSecPerOp, e.GoNsPerOp),
			fmtBytes(e.GoBytesPerOp), fmtBytes(e.PythonPeakRSSBytes), fmt.Sprintf("%d", e.GoAllocsPerOp))
	}

	fmt.Println(divider())
	if hasPython {
		fmt.Println("Go B/op = bytes allocated per op (Go heap); Py peak RSS = peak resident-set")
		fmt.Println("growth Polars drove for the op (process working set, coarser than B/op).")
	}
	fmt.Println()
}

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

// BenchmarkCrossFilterSum benchmarks a full filter+sum pipeline in Go across
// two execution paths, both selectivity profiles, and a range of sizes:
//
//   - eager: df.Filter(pred) materializes a filtered frame, then Series.Sum()
//     reduces it (two passes; this was the historical measurement).
//   - lazy:  df.Lazy().Filter(pred).Sum().Collect() triggers the fused
//     filter+reduce (frame.FilterAggregate / simd.MaskedReduceFloat64) — a
//     single masked pass with no materialized frame and no survivor-index slice.
//
// The Python Polars comparison is opt-in: set GOPOLARS_BENCH_PYTHON=1 to spawn
// the harness. The eager row is compared against Polars eager
// (df.filter(...)["a"].sum()); the lazy row against Polars lazy
// (lf.filter(...).select(col.sum()).collect()) and additionally against the
// Polars streaming engine (recorded as python_stream_sec/op). Spawning Python
// per iteration dominates wall time, so it is skipped by default.
//
// B/op and allocs/op are captured (via runtime.MemStats deltas around the timed
// loop) into filter_sum_summary.json so the eager allocation overhead and the
// fused path's near-zero allocation are both visible in the artifact.
func BenchmarkCrossFilterSum(b *testing.B) {
	ctx := context.Background()
	var mu sync.Mutex
	var results []summaryEntry
	pythonEnabled := os.Getenv("GOPOLARS_BENCH_PYTHON") != ""

	// engines pairs each gopolars execution path with the matching Polars
	// harness op so every row is an apples-to-apples comparison.
	engines := []struct {
		name string // display/summary label; also selects the Go path
		pyOp string // matching Python harness op
	}{
		{"eager", "filter_sum"},
		{"lazy", "filter_sum_lazy"},
		{"eager_direct", "filter_sum"},
	}

	for _, prof := range selectivityProfiles {
		for _, n := range benchSizes {
			data := makeFloat64Slice(n)

			// Build the gopolars DataFrame once; both paths read it (read-only).
			vals := make([]any, n)
			for i, v := range data {
				vals[i] = v
			}
			df, err := polars.NewDataFrame(polars.NewDataFrameInput{
				Columns: []frame.SeriesInput{{Name: "a", Values: vals}},
			})
			if err != nil {
				b.Fatalf("build df: %v", err)
			}
			pred := polars.Col("a").Gt(polars.Lit(prof.threshold))

			// Shared Arrow IPC input for the Python baselines (opt-in).
			var arrowFile string
			if pythonEnabled {
				arrowFile = filepath.Join(b.TempDir(), fmt.Sprintf("data_%s_%d.arrow", prof.name, n))
				if err := writeArrowIPC(arrowFile, map[string][]float64{"a": data}); err != nil {
					b.Fatalf("write arrow ipc: %v", err)
				}
				b.Cleanup(func() { _ = os.Remove(arrowFile) })
			}

			for _, eng := range engines {
				// Python baseline(s), measured once per (profile, size, engine).
				var pythonSecPerOp, pythonStreamSecPerOp float64
				var pythonPeakRSSBytes int64
				if pythonEnabled {
					if pyRes, pyErr := runPythonHarness(eng.pyOp, arrowFile, 5, prof.threshold); pyErr == nil {
						pythonSecPerOp = pyRes.ElapsedSec / float64(pyRes.Iters)
						pythonPeakRSSBytes = pyRes.PeakRSSBytes
					} else {
						b.Logf("python harness (%s) unavailable: %v", eng.pyOp, pyErr)
					}
					// For the lazy row, also capture Polars' streaming engine —
					// the closest analogue to gopolars' fused reduce.
					if eng.name == "lazy" {
						if pyRes, pyErr := runPythonHarness("filter_sum_stream", arrowFile, 5, prof.threshold); pyErr == nil {
							pythonStreamSecPerOp = pyRes.ElapsedSec / float64(pyRes.Iters)
						}
					}
				}

				opName := "filter_sum/" + prof.name + "/" + eng.name
				b.Run(opName+"/size_"+humanize(n), func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(n * 8))

					var ms0, ms1 runtime.MemStats
					b.ResetTimer()
					runtime.ReadMemStats(&ms0)
					start := time.Now()
					switch eng.name {
					case "eager":
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
					case "lazy":
						for i := 0; i < b.N; i++ {
							res, err := df.Lazy().Filter(pred).Sum().Collect(ctx)
							if err != nil {
								b.Fatalf("lazy collect: %v", err)
							}
							sv, ok := res.Series("a")
							if !ok {
								b.Fatal("series not found")
							}
							filterSumSink = sv.Sum()
						}
					case "eager_direct":
						// Eager fused fast path: a single masked pass with no
						// materialized frame, no survivor-index slice, and no plan
						// construction (cf. eager materialize + lazy plan overhead).
						for i := 0; i < b.N; i++ {
							res, err := df.FilterAggregateDirect(pred, "sum", []string{"a"})
							if err != nil {
								b.Fatalf("eager-direct: %v", err)
							}
							filterSumSink = res["a"]
						}
					}
					elapsed := time.Since(start)
					runtime.ReadMemStats(&ms1)
					b.StopTimer()

					goNsPerOp := elapsed.Nanoseconds() / int64(b.N)
					goBytesPerOp := int64((ms1.TotalAlloc - ms0.TotalAlloc) / uint64(b.N))
					goAllocsPerOp := int64((ms1.Mallocs - ms0.Mallocs) / uint64(b.N))

					// Report the Python baseline metrics here, AFTER ResetTimer —
					// ResetTimer calls clear(b.extra), so any ReportMetric issued
					// before it (as these once were) is silently dropped from the
					// benchmark line. The values are constant for the run, so
					// reporting them post-loop is exact.
					if pythonSecPerOp > 0 {
						b.ReportMetric(pythonSecPerOp, "python_sec/op")
						b.ReportMetric(pythonSecPerOp/(float64(goNsPerOp)/1e9), "go_vs_python_speedup")
					}
					if pythonStreamSecPerOp > 0 {
						b.ReportMetric(pythonStreamSecPerOp, "python_stream_sec/op")
					}
					if pythonPeakRSSBytes > 0 {
						b.ReportMetric(float64(pythonPeakRSSBytes), "python_peak_rss_B")
					}

					entry := summaryEntry{
						Op:                   opName,
						Size:                 humanize(n),
						Elements:             n,
						GoNsPerOp:            goNsPerOp,
						GoBytesPerOp:         goBytesPerOp,
						GoAllocsPerOp:        goAllocsPerOp,
						PythonSecPerOp:       pythonSecPerOp,
						PythonStreamSecPerOp: pythonStreamSecPerOp,
						PythonPeakRSSBytes:   pythonPeakRSSBytes,
					}
					if eng.name == "eager_direct" {
						entry.GoEagerDirectNsPerOp = goNsPerOp
						entry.GoEagerDirectBytesPerOp = goBytesPerOp
						entry.GoEagerDirectAllocsPerOp = goAllocsPerOp
					}
					mu.Lock()
					results = append(results, entry)
					mu.Unlock()
				})
			}
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

	printSummaryTable(results)
}
