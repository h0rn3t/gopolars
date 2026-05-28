package top30

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

var lightMode = flag.Bool("light", false, "run only 1K and 1M sizes")

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
	Object     string  `json:"object"`
	Iters      int     `json:"iters"`
	ElapsedSec float64 `json:"elapsed_sec"`
	Elements   int     `json:"elements"`
}

type summaryEntry struct {
	Object         string  `json:"object"`
	Op             string  `json:"op"`
	Size           string  `json:"size"`
	Elements       int     `json:"elements"`
	GoNsPerOp      int64   `json:"go_ns_per_op"`
	PythonSecPerOp float64 `json:"python_sec_per_op"`
	Ratio          float64 `json:"ratio"`
}

func sizesToRun() []int {
	if *lightMode {
		return []int{1_000, 1_000_000}
	}
	return benchSizes
}

func runPythonHarness(op, object, inputPath string, iters int) (harnessResult, error) {
	_, testFile, _, _ := runtime.Caller(0)
	benchDir := filepath.Dir(testFile)
	harnessPath := filepath.Join(benchDir, "harness.py")

	cmd := exec.Command("python3", harnessPath, "--op", op, "--object", object, "--input", inputPath, "--iters", fmt.Sprint(iters))
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

func hasPythonPolars() bool {
	if _, err := exec.LookPath("python3"); err != nil {
		return false
	}
	cmd := exec.Command("python3", "-c", "import polars")
	return cmd.Run() == nil
}

func makeDataFrame(n int) (polars.DataFrame, error) {
	ds := GenerateDataset(n, 42)
	g := make([]any, len(ds.G))
	v := make([]any, len(ds.V))
	ns := make([]any, len(ds.N))
	is := make([]any, len(ds.I))
	for idx := range ds.G {
		g[idx] = ds.G[idx]
		v[idx] = ds.V[idx]
		if ds.N[idx] == nil {
			ns[idx] = nil
		} else {
			ns[idx] = *ds.N[idx]
		}
		is[idx] = ds.I[idx]
	}
	return polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "g", Values: g},
			{Name: "v", Values: v},
			{Name: "n", Values: ns},
			{Name: "i", Values: is},
		},
	})
}

func writeTempArrow(b *testing.B, n int) string {
	ds := GenerateDataset(n, 42)
	tmpFile := filepath.Join(b.TempDir(), "data.arrow")
	if err := WriteArrowIPC(tmpFile, ds); err != nil {
		b.Fatalf("write arrow ipc: %v", err)
	}
	return tmpFile
}

func BenchmarkTop30(b *testing.B) {
	if !hasPythonPolars() {
		b.Skip("python3 or polars not available; install with: pip install -r bench/top30/requirements-bench.txt")
	}

	var mu sync.Mutex
	var results []summaryEntry

	ctx := context.Background()
	sizes := sizesToRun()

	// DataFrame operations
	dataframeOps := []string{
		"filter", "select", "with_columns", "sort", "group_by",
		"join", "head", "tail", "unique", "fill_null", "drop_nulls",
	}
	for _, op := range dataframeOps {
		for _, n := range sizes {
			b.Run("DataFrame/"+op+"/size_"+humanize(n), func(b *testing.B) {
				runtime.GC()
				df, err := makeDataFrame(n)
				if err != nil {
					b.Fatalf("make dataframe: %v", err)
				}
				tmpFile := writeTempArrow(b, n)

				pyRes, err := runPythonHarness(op, "DataFrame", tmpFile, 10)
				if err != nil {
					b.Skipf("python harness: %v", err)
					return
				}
				pythonSecPerOp := pyRes.ElapsedSec / float64(pyRes.Iters)
				b.ReportMetric(pythonSecPerOp, "python_sec/op")

				b.ResetTimer()
				start := time.Now()
				for i := 0; i < b.N; i++ {
					_ = benchDataFrame(ctx, df, op)
				}
				b.StopTimer()
				goNsPerOp := time.Since(start).Nanoseconds() / int64(b.N)

				mu.Lock()
				results = append(results, summaryEntry{
					Object:         "DataFrame",
					Op:             op,
					Size:           humanize(n),
					Elements:       n,
					GoNsPerOp:      goNsPerOp,
					PythonSecPerOp: pythonSecPerOp,
					Ratio:          pythonSecPerOp / (float64(goNsPerOp) / 1e9),
				})
				mu.Unlock()
			})
		}
	}

	// Expr operations
	exprOps := []string{
		"cum_sum", "rank", "over", "fill_null", "fill_nan",
		"rolling_mean", "rolling_sum", "rolling_min", "rolling_max",
	}
	for _, op := range exprOps {
		for _, n := range sizes {
			b.Run("Expr/"+op+"/size_"+humanize(n), func(b *testing.B) {
				runtime.GC()
				df, err := makeDataFrame(n)
				if err != nil {
					b.Fatalf("make dataframe: %v", err)
				}
				tmpFile := writeTempArrow(b, n)

				pyRes, err := runPythonHarness(op, "Expr", tmpFile, 10)
				if err != nil {
					b.Skipf("python harness: %v", err)
					return
				}
				pythonSecPerOp := pyRes.ElapsedSec / float64(pyRes.Iters)
				b.ReportMetric(pythonSecPerOp, "python_sec/op")

				b.ResetTimer()
				start := time.Now()
				for i := 0; i < b.N; i++ {
					_ = benchExpr(ctx, df, op)
				}
				b.StopTimer()
				goNsPerOp := time.Since(start).Nanoseconds() / int64(b.N)

				mu.Lock()
				results = append(results, summaryEntry{
					Object:         "Expr",
					Op:             op,
					Size:           humanize(n),
					Elements:       n,
					GoNsPerOp:      goNsPerOp,
					PythonSecPerOp: pythonSecPerOp,
					Ratio:          pythonSecPerOp / (float64(goNsPerOp) / 1e9),
				})
				mu.Unlock()
			})
		}
	}

	// Series operations
	seriesOps := []string{
		"null_count", "drop_nans", "to_list", "is_null", "is_not_null", "fill_nan",
	}
	for _, op := range seriesOps {
		for _, n := range sizes {
			b.Run("Series/"+op+"/size_"+humanize(n), func(b *testing.B) {
				runtime.GC()
				df, err := makeDataFrame(n)
				if err != nil {
					b.Fatalf("make dataframe: %v", err)
				}
				s, err := df.GetColumn("v")
				if err != nil {
					b.Fatalf("get column: %v", err)
				}
				tmpFile := writeTempArrow(b, n)

				pyRes, err := runPythonHarness(op, "Series", tmpFile, 10)
				if err != nil {
					b.Skipf("python harness: %v", err)
					return
				}
				pythonSecPerOp := pyRes.ElapsedSec / float64(pyRes.Iters)
				b.ReportMetric(pythonSecPerOp, "python_sec/op")

				b.ResetTimer()
				start := time.Now()
				for i := 0; i < b.N; i++ {
					_ = benchSeries(s, op)
				}
				b.StopTimer()
				goNsPerOp := time.Since(start).Nanoseconds() / int64(b.N)

				mu.Lock()
				results = append(results, summaryEntry{
					Object:         "Series",
					Op:             op,
					Size:           humanize(n),
					Elements:       n,
					GoNsPerOp:      goNsPerOp,
					PythonSecPerOp: pythonSecPerOp,
					Ratio:          pythonSecPerOp / (float64(goNsPerOp) / 1e9),
				})
				mu.Unlock()
			})
		}
	}

	// LazyFrame operations
	lazyOps := []string{"collect", "sql", "inspect"}
	for _, op := range lazyOps {
		for _, n := range sizes {
			b.Run("LazyFrame/"+op+"/size_"+humanize(n), func(b *testing.B) {
				runtime.GC()
				df, err := makeDataFrame(n)
				if err != nil {
					b.Fatalf("make dataframe: %v", err)
				}
				tmpFile := writeTempArrow(b, n)

				pyRes, err := runPythonHarness(op, "LazyFrame", tmpFile, 10)
				if err != nil {
					b.Skipf("python harness: %v", err)
					return
				}
				pythonSecPerOp := pyRes.ElapsedSec / float64(pyRes.Iters)
				b.ReportMetric(pythonSecPerOp, "python_sec/op")

				b.ResetTimer()
				start := time.Now()
				for i := 0; i < b.N; i++ {
					_ = benchLazyFrame(ctx, df, op)
				}
				b.StopTimer()
				goNsPerOp := time.Since(start).Nanoseconds() / int64(b.N)

				mu.Lock()
				results = append(results, summaryEntry{
					Object:         "LazyFrame",
					Op:             op,
					Size:           humanize(n),
					Elements:       n,
					GoNsPerOp:      goNsPerOp,
					PythonSecPerOp: pythonSecPerOp,
					Ratio:          pythonSecPerOp / (float64(goNsPerOp) / 1e9),
				})
				mu.Unlock()
			})
		}
	}

	// SQLContext operations
	sqlOps := []string{"execute", "register", "tables"}
	for _, op := range sqlOps {
		for _, n := range sizes {
			b.Run("SQLContext/"+op+"/size_"+humanize(n), func(b *testing.B) {
				runtime.GC()
				df, err := makeDataFrame(n)
				if err != nil {
					b.Fatalf("make dataframe: %v", err)
				}
				tmpFile := writeTempArrow(b, n)

				pyRes, err := runPythonHarness(op, "SQLContext", tmpFile, 10)
				if err != nil {
					b.Skipf("python harness: %v", err)
					return
				}
				pythonSecPerOp := pyRes.ElapsedSec / float64(pyRes.Iters)
				b.ReportMetric(pythonSecPerOp, "python_sec/op")

				b.ResetTimer()
				start := time.Now()
				for i := 0; i < b.N; i++ {
					_ = benchSQLContext(ctx, df, op)
				}
				b.StopTimer()
				goNsPerOp := time.Since(start).Nanoseconds() / int64(b.N)

				mu.Lock()
				results = append(results, summaryEntry{
					Object:         "SQLContext",
					Op:             op,
					Size:           humanize(n),
					Elements:       n,
					GoNsPerOp:      goNsPerOp,
					PythonSecPerOp: pythonSecPerOp,
					Ratio:          pythonSecPerOp / (float64(goNsPerOp) / 1e9),
				})
				mu.Unlock()
			})
		}
	}

	if len(results) > 0 {
		writeReports(b, results)
	}
}

func benchDataFrame(ctx context.Context, df polars.DataFrame, op string) error {
	var err error
	switch op {
	case "filter":
		_, err = df.Filter(polars.Col("v").Gt(polars.Lit(float64(0))))
	case "select":
		_, err = df.Select(polars.Col("g"), polars.Col("v"))
	case "with_columns":
		_, err = df.WithColumns(polars.Col("v").Alias("v2"))
	case "sort":
		_, err = df.Sort(polars.SortInput{By: []string{"v"}})
	case "group_by":
		gb := df.GroupBy("g")
		_, err = gb.Agg(polars.Col("v").Sum())
	case "join":
		_, err = df.Join(polars.JoinInput{Other: df, LeftOn: []string{"g"}, RightOn: []string{"g"}, How: polars.JoinTypeInner})
	case "head":
		_ = df.Head(100)
	case "tail":
		_ = df.Tail(100)
	case "unique":
		_, err = df.Unique("g")
	case "fill_null":
		_, err = df.FillNull(float64(0))
	case "drop_nulls":
		_ = df.DropNaNs("n")
	}
	return err
}

func benchExpr(_ context.Context, df polars.DataFrame, op string) error {
	var err error
	switch op {
	case "cum_sum":
		_, err = df.Select(polars.Col("v").CumSum())
	case "rank":
		_, err = df.Select(polars.Col("v").Rank())
	case "over":
		_, err = df.Select(polars.Col("v").CumSum().Over("g"))
	case "fill_null":
		_, err = df.Select(polars.Col("n").FillNull(polars.Lit(float64(0))))
	case "fill_nan":
		_, err = df.Select(polars.Col("n").FillNaN(polars.Lit(float64(0))))
	case "rolling_mean":
		_, err = df.Select(polars.Col("v").RollingMean(100))
	case "rolling_sum":
		_, err = df.Select(polars.Col("v").RollingSum(100))
	case "rolling_min":
		_, err = df.Select(polars.Col("v").RollingMin(100))
	case "rolling_max":
		_, err = df.Select(polars.Col("v").RollingMax(100))
	}
	return err
}

func benchSeries(s polars.Series, op string) error {
	var err error
	switch op {
	case "null_count":
		_ = s.NullCount()
	case "drop_nans":
		_ = s.DropNans()
	case "to_list":
		_ = s.ToList()
	case "is_null":
		_ = s.IsNull()
	case "is_not_null":
		_ = s.IsNotNull()
	case "fill_nan":
		_, err = s.FillNan(0.0)
	}
	return err
}

func benchLazyFrame(ctx context.Context, df polars.DataFrame, op string) error {
	var err error
	lf := df.Lazy()
	switch op {
	case "collect":
		_, err = lf.Collect(ctx)
	case "sql":
		var sqlLF polars.LazyFrame
		sqlLF, err = lf.SQL(ctx, "SELECT g, v FROM self WHERE v > 0", "self")
		if err == nil {
			_, err = sqlLF.Collect(ctx)
		}
	case "inspect":
		_ = lf.Inspect()
	}
	return err
}

func benchSQLContext(ctx context.Context, df polars.DataFrame, op string) error {
	var err error
	sqlCtx := polars.NewSQLContext()
	switch op {
	case "execute":
		if err = sqlCtx.Register("t", df); err != nil {
			return err
		}
		var lf polars.LazyFrame
		lf, err = sqlCtx.Execute(ctx, "SELECT * FROM t")
		if err == nil {
			_, err = lf.Collect(ctx)
		}
	case "register":
		err = sqlCtx.Register("t", df)
	case "tables":
		if err = sqlCtx.Register("t", df); err != nil {
			return err
		}
		_ = sqlCtx.Tables()
	}
	return err
}

func writeReports(b *testing.B, results []summaryEntry) {
	_, testFile, _, _ := runtime.Caller(0)
	benchDir := filepath.Dir(testFile)

	writeStr := func(f *os.File, s string) {
		if _, err := f.WriteString(s); err != nil {
			b.Fatalf("write string: %v", err)
		}
	}

	// JSON
	jsonPath := filepath.Join(benchDir, "top30_summary.json")
	f, err := os.Create(jsonPath)
	if err != nil {
		b.Fatalf("create json summary: %v", err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		b.Fatalf("encode json summary: %v", err)
	}
	if err := f.Close(); err != nil {
		b.Fatalf("close json summary: %v", err)
	}

	// CSV
	csvPath := filepath.Join(benchDir, "top30_summary.csv")
	cf, err := os.Create(csvPath)
	if err != nil {
		b.Fatalf("create csv summary: %v", err)
	}
	cw := csv.NewWriter(cf)
	if err := cw.Write([]string{"object", "operation", "size", "go_ns_per_op", "python_ms_per_op", "ratio", "status"}); err != nil {
		b.Fatalf("write csv header: %v", err)
	}
	for _, r := range results {
		status := "ok"
		if r.Ratio > 2.0 {
			status = "slow"
		}
		if err := cw.Write([]string{
			r.Object,
			r.Op,
			r.Size,
			fmt.Sprintf("%d", r.GoNsPerOp),
			fmt.Sprintf("%.6f", r.PythonSecPerOp*1000),
			fmt.Sprintf("%.4f", r.Ratio),
			status,
		}); err != nil {
			b.Fatalf("write csv row: %v", err)
		}
	}
	cw.Flush()
	if err := cf.Close(); err != nil {
		b.Fatalf("close csv summary: %v", err)
	}

	// Markdown
	mdPath := filepath.Join(benchDir, "top30_summary.md")
	mf, err := os.Create(mdPath)
	if err != nil {
		b.Fatalf("create markdown summary: %v", err)
	}
	defer func() {
		if err := mf.Close(); err != nil {
			b.Fatalf("close markdown summary: %v", err)
		}
	}()
	writeStr(mf, "# Top30 Benchmark Results\n\n")
	objects := []string{"DataFrame", "Expr", "Series", "LazyFrame", "SQLContext"}
	for _, obj := range objects {
		writeStr(mf, fmt.Sprintf("## %s\n\n", obj))
		writeStr(mf, "| operation | size | go_ns/op | python_ms/op | ratio |\n")
		writeStr(mf, "|-----------|------|----------|--------------|-------|\n")
		for _, r := range results {
			if r.Object == obj {
				writeStr(mf, fmt.Sprintf("| %s | %s | %d | %.6f | %.4f |\n", r.Op, r.Size, r.GoNsPerOp, r.PythonSecPerOp*1000, r.Ratio))
			}
		}
		writeStr(mf, "\n")
	}
}

func BenchmarkTop30Dummy(_ *testing.B) {
	// placeholder to satisfy go test when no benchmarks are selected
}

func init() {
	_ = strings.Repeat("", 0)
}
