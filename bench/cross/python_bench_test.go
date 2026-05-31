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
	"sync"
	"testing"
	"time"

	"github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/ipc"
	"github.com/apache/arrow/go/v18/arrow/memory"
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
	Op         string  `json:"op"`
	Iters      int     `json:"iters"`
	ElapsedSec float64 `json:"elapsed_sec"`
	Elements   int     `json:"elements"`
}

type summaryEntry struct {
	Op             string  `json:"op"`
	Size           string  `json:"size"`
	Elements       int     `json:"elements"`
	GoNsPerOp      int64   `json:"go_ns_per_op"`
	PythonSecPerOp float64 `json:"python_sec_per_op"`
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

	rec := array.NewRecord(schema, arrays, int64(rowCount))
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
				b.ReportMetric(pythonSecPerOp, "python_sec/op")

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
				b.ReportMetric(pythonSecPerOp, "python_sec/op")

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
		summaryPath := filepath.Join(benchDir, "cross_summary.json")
		f, err := os.Create(summaryPath)
		if err != nil {
			b.Fatalf("create summary file: %v", err)
		}
		defer func() { _ = f.Close() }()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			b.Fatalf("encode summary: %v", err)
		}
	}
}
