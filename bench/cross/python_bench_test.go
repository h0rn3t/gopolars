package cross

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/h0rn3t/gopolars/pkg/simd"
)

var benchSizes = []int{1_000, 10_000, 100_000, 1_000_000, 10_000_000}

func humanize(n int) string {
	switch n {
	case 1_000:
		return "1K"
	case 10_000:
		return "10K"
	case 100_000:
		return "100K"
	case 1_000_000:
		return "1M"
	case 10_000_000:
		return "10M"
	}
	return "unknown"
}

type harnessResult struct {
	Op           string  `json:"op"`
	Iters        int     `json:"iters"`
	ElapsedSec   float64 `json:"elapsed_sec"`
	Elements     int     `json:"elements"`
	PeakRSSBytes int64   `json:"peak_rss_bytes"`
}

type summaryEntry struct {
	Op                   string  `json:"op"`
	Size                 string  `json:"size"`
	Elements             int     `json:"elements"`
	GoNsPerOp            int64   `json:"go_ns_per_op"`
	GoBytesPerOp         int64   `json:"go_bytes_per_op,omitempty"`
	GoAllocsPerOp        int64   `json:"go_allocs_per_op,omitempty"`
	PythonSecPerOp       float64 `json:"python_sec_per_op"`
	PythonStreamSecPerOp float64 `json:"python_stream_sec_per_op,omitempty"`
	// PythonPeakRSSBytes is the peak resident-set growth Polars drove for one
	// op (high-water working set, from the harness's ru_maxrss delta), the
	// memory counterpart to PythonSecPerOp. It is a process-level peak, not an
	// allocation count, so it is coarser than Go's B/op and should be read as an
	// order-of-magnitude footprint rather than an exact figure.
	PythonPeakRSSBytes int64 `json:"python_peak_rss_bytes,omitempty"`
	// Eager-direct fused path (DataFrame.FilterAggregateDirect). Populated only
	// on the "eager_direct" engine rows; omitted elsewhere. Captured separately
	// so the artifact exposes the eager fused path's time/allocation profile next
	// to the eager-materialize and lazy-fused rows.
	GoEagerDirectNsPerOp     int64 `json:"go_eager_direct_ns_per_op,omitempty"`
	GoEagerDirectBytesPerOp  int64 `json:"go_eager_direct_bytes_per_op,omitempty"`
	GoEagerDirectAllocsPerOp int64 `json:"go_eager_direct_allocs_per_op,omitempty"`
}

var benchRNG = rand.New(rand.NewSource(42))

func makeFloat64Slice(n int) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = benchRNG.Float64()*100 - 50
	}
	return s
}

func writeArrowIPC(path string, cols map[string][]float64) error {
	if len(cols) == 0 {
		return fmt.Errorf("no columns provided")
	}
	var rowCount int
	for _, v := range cols {
		rowCount = len(v)
		break
	}

	alloc := memory.NewGoAllocator()
	fields := make([]arrow.Field, 0, len(cols))
	arrays := make([]arrow.Array, 0, len(cols))

	for name, data := range cols {
		builder := array.NewFloat64Builder(alloc)
		builder.AppendValues(data, nil)
		arr := builder.NewFloat64Array()
		builder.Release()
		defer arr.Release()
		fields = append(fields, arrow.Field{Name: name, Type: arrow.PrimitiveTypes.Float64})
		arrays = append(arrays, arr)
	}

	schema := arrow.NewSchema(fields, nil)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	writer, err := ipc.NewFileWriter(f, ipc.WithSchema(schema))
	if err != nil {
		return err
	}
	defer func() { _ = writer.Close() }()

	rec := array.NewRecordBatch(schema, arrays, int64(rowCount))
	defer rec.Release()

	return writer.Write(rec)
}

func pythonBin() string {
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			_ = path
			return name
		}
	}
	return "python3"
}

func runPythonHarness(op, inputPath string, iters int, threshold float64) (harnessResult, error) {
	_, testFile, _, _ := runtime.Caller(0)
	benchDir := filepath.Dir(testFile)
	harnessPath := filepath.Join(benchDir, "harness.py")

	cmd := exec.Command(pythonBin(), harnessPath, "--op", op, "--input", inputPath,
		"--iters", fmt.Sprint(iters), "--threshold", fmt.Sprint(threshold))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return harnessResult{}, fmt.Errorf("python harness failed: %w (stderr: %s)", err, stderr.String())
	}

	var res harnessResult
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		return harnessResult{}, fmt.Errorf("parse harness output: %w (stdout: %s)", err, stdout.String())
	}
	return res, nil
}

func BenchmarkCross(b *testing.B) {
	var mu sync.Mutex
	var results []summaryEntry

	singleOps := []string{"sum", "min", "max", "mean", "minmax"}
	doubleOps := []string{"add", "mul"}

	for _, op := range singleOps {
		for _, n := range benchSizes {
			b.Run(op+"/size_"+humanize(n), func(b *testing.B) {
				data := makeFloat64Slice(n)
				tmpFile := filepath.Join(b.TempDir(), "data.arrow")
				if err := writeArrowIPC(tmpFile, map[string][]float64{"a": data}); err != nil {
					b.Fatalf("write arrow ipc: %v", err)
				}
				b.Cleanup(func() {
					_ = os.Remove(tmpFile)
				})

				pyRes, err := runPythonHarness(op, tmpFile, 10, 50.0)
				if err != nil {
					b.Fatalf("python harness: %v", err)
				}
				pythonSecPerOp := pyRes.ElapsedSec / float64(pyRes.Iters)

				b.SetBytes(int64(n * 8))
				b.ResetTimer()
				start := time.Now()
				switch op {
				case "sum":
					for i := 0; i < b.N; i++ {
						simd.SumFloat64(data)
					}
				case "min":
					for i := 0; i < b.N; i++ {
						simd.MinFloat64(data)
					}
				case "max":
					for i := 0; i < b.N; i++ {
						simd.MaxFloat64(data)
					}
				case "mean":
					for i := 0; i < b.N; i++ {
						_ = simd.SumFloat64(data) / float64(len(data))
					}
				case "minmax":
					for i := 0; i < b.N; i++ {
						simd.MinMaxFloat64(data)
					}
				}
				b.StopTimer()
				goNsPerOp := time.Since(start).Nanoseconds() / int64(b.N)

				// Report Python metrics AFTER ResetTimer (which clears b.extra) so
				// they are not silently dropped from the benchmark line — the same
				// gotcha handled in BenchmarkCrossFilterSum.
				b.ReportMetric(pythonSecPerOp, "python_sec/op")
				b.ReportMetric(pythonSecPerOp/(float64(goNsPerOp)/1e9), "go_vs_python_speedup")

				mu.Lock()
				results = append(results, summaryEntry{
					Op:             op,
					Size:           humanize(n),
					Elements:       n,
					GoNsPerOp:      goNsPerOp,
					PythonSecPerOp: pythonSecPerOp,
				})
				mu.Unlock()
			})
		}
	}

	for _, op := range doubleOps {
		for _, n := range benchSizes {
			b.Run(op+"/size_"+humanize(n), func(b *testing.B) {
				a := makeFloat64Slice(n)
				c := makeFloat64Slice(n)
				tmpFile := filepath.Join(b.TempDir(), "data.arrow")
				if err := writeArrowIPC(tmpFile, map[string][]float64{"a": a, "b": c}); err != nil {
					b.Fatalf("write arrow ipc: %v", err)
				}
				b.Cleanup(func() {
					_ = os.Remove(tmpFile)
				})

				pyRes, err := runPythonHarness(op, tmpFile, 10, 50.0)
				if err != nil {
					b.Fatalf("python harness: %v", err)
				}
				pythonSecPerOp := pyRes.ElapsedSec / float64(pyRes.Iters)

				b.SetBytes(int64(n * 8 * 2))
				b.ResetTimer()
				start := time.Now()
				switch op {
				case "add":
					for i := 0; i < b.N; i++ {
						simd.AddSlicesFloat64(a, c)
					}
				case "mul":
					for i := 0; i < b.N; i++ {
						simd.MulSlicesFloat64(a, c)
					}
				}
				b.StopTimer()
				goNsPerOp := time.Since(start).Nanoseconds() / int64(b.N)

				// See the singleOps note: report after ResetTimer so the metrics
				// survive on the benchmark line.
				b.ReportMetric(pythonSecPerOp, "python_sec/op")
				b.ReportMetric(pythonSecPerOp/(float64(goNsPerOp)/1e9), "go_vs_python_speedup")

				mu.Lock()
				results = append(results, summaryEntry{
					Op:             op,
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
		if err := writeSummaryJSON(filepath.Join(benchDir, "cross_summary.json"), results); err != nil {
			b.Fatalf("write summary json: %v", err)
		}
		if err := writeSummaryMarkdown(filepath.Join(benchDir, "cross_summary.md"), results, "Cross-language SIMD primitives — gopolars vs Python Polars"); err != nil {
			b.Fatalf("write summary markdown: %v", err)
		}
		b.Logf("summary written to %s (json+md)", filepath.Join(benchDir, "cross_summary.json"))
	}
}

// writeSummaryJSON writes the run results to a JSON file with stable formatting.
func writeSummaryJSON(path string, results []summaryEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

// writeSummaryMarkdown renders results as a GitHub-flavored Markdown table. The
// output is the same data the JSON holds, but it lands as a human-readable
// artifact in PRs and CI runs (in addition to stdout from printSummaryTable).
//
// Layout per row: profile/op, size, Go time, Polars time, winner, Go B/op,
// Polars peak RSS, Go allocs/op. The two memory columns measure different
// things and are not directly comparable unit-for-unit; the table header
// disclaims this once so the markdown stays compact.
func writeSummaryMarkdown(path string, results []summaryEntry, title string) error {
	if len(results) == 0 {
		return nil
	}

	// Deduplicate by (op, size) — keep the warmest (last) sample, same as
	// printSummaryTable does for the ASCII output.
	type key struct{ op, size string }
	dedup := make(map[key]summaryEntry)
	var order []key
	for _, e := range results {
		k := key{e.Op, e.Size}
		if _, exists := dedup[k]; !exists {
			order = append(order, k)
		}
		dedup[k] = e
	}
	sort.Slice(order, func(i, j int) bool {
		ei, ej := dedup[order[i]], dedup[order[j]]
		if ei.Op != ej.Op {
			return ei.Op < ej.Op
		}
		return ei.Elements < ej.Elements
	})

	hasPython := false
	for _, k := range order {
		if dedup[k].PythonSecPerOp > 0 {
			hasPython = true
			break
		}
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
			return fmt.Sprintf("**Go ×%.1f**", speedup)
		}
		return fmt.Sprintf("Py ×%.1f", 1/speedup)
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "Generated by `go test ./bench/cross`; do not edit by hand.\n\n")
	if hasPython {
		fmt.Fprintf(&b, "Go B/op = bytes allocated per op (Go heap); Polars peak RSS = ")
		fmt.Fprintf(&b, "peak resident-set growth Polars drove for the op (process working set, coarser than B/op).\n\n")
	}

	header := "| op | size | Go time | Polars time | winner | Go B/op | Polars peak RSS | allocs/op |"
	divider := "|---|---|---|---|---|---|---|---|"
	fmt.Fprintln(&b, header)
	fmt.Fprintln(&b, divider)
	for _, k := range order {
		e := dedup[k]
		pyNs := int64(e.PythonSecPerOp * 1e9)
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s | %s | %d |\n",
			e.Op, e.Size,
			fmtDur(e.GoNsPerOp), fmtDur(pyNs),
			fmtWinner(e.PythonSecPerOp, e.GoNsPerOp),
			fmtBytes(e.GoBytesPerOp), fmtBytes(e.PythonPeakRSSBytes),
			e.GoAllocsPerOp,
		)
	}

	return os.WriteFile(path, b.Bytes(), 0o644)
}
