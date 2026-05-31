# Cross-Language SIMD Benchmark

Compares Go gopolars SIMD operations against Python Polars on identical float64 datasets.

## Prerequisites

- Go 1.26+
- Python 3 with `polars` installed:
  ```bash
  pip install -r requirements-bench.txt
  ```

## Running

Run the full cross-language benchmark suite:

```bash
go test -bench=BenchmarkCross ./bench/cross/...
```

Run for a specific operation and size:

```bash
go test -bench=BenchmarkCross/sum/size_1K ./bench/cross/...
```

## Python Harness (standalone)

The Python harness can be executed independently:

```bash
python harness.py --op sum --input data.arrow --iters 10
```

Supported operations: `sum`, `min`, `max`, `mean`, `minmax`, `add`, `mul`, `filter_sum`.

## Output

- Go benchmark lines are emitted to stdout via `testing.B`.
- A JSON summary with paired results is written to `cross_summary.json` in this directory.
- The `filter_sum` pipeline benchmark writes a separate `filter_sum_summary.json`.

## filter+sum pipeline benchmark

`BenchmarkCrossFilterSum` runs a full filter+sum pipeline across two
selectivity profiles. The data is uniform in `[-50, 50)`, so:

- **empty** (`col("a") > 50`): ~0% of rows pass — isolates predicate evaluation
  and allocation cost (degenerate gather).
- **half** (`col("a") > 0`): ~50% pass — exercises realistic gather /
  materialization cost.

The Go path is the vectorized batch mask (parallel above ~32K rows) + typed
chunk gather → `Sum()` reading `[]float64` directly.

Run the Go-only benchmark (default — fast, deterministic, benchstat-clean):

```bash
go test ./bench/cross -bench=BenchmarkCrossFilterSum -benchmem -count=10
```

The Python Polars comparison is opt-in (spawning the harness per `-count`
iteration dominates wall time and adds noise to the Go metrics):

```bash
# Ensure Polars is installed in the active Python environment, then:
GOPOLARS_BENCH_PYTHON=1 go test ./bench/cross -bench=BenchmarkCrossFilterSum -benchmem -count=1 -benchtime=3s
```

If you use a project virtualenv (e.g. `.venv_polars_313`), prepend its `bin/`
to `PATH` so the Go test runner finds the right Python:

```bash
PATH="$(pwd)/.venv_polars_313/bin:$PATH" \
  GOPOLARS_BENCH_PYTHON=1 \
  go test ./bench/cross -bench=BenchmarkCrossFilterSum -benchmem -count=1 -benchtime=3s
```

Results are printed inline (ns/op, python_sec/op, go_vs_python_speedup) and
written to `filter_sum_summary.json`.

The harness threshold is parameterized (`harness.py --threshold`) so the Python
side matches each Go profile.

For a focused fused-vs-materializing comparison of `filter(...).sum()`, see
`BenchmarkFilterSumFused` in `pkg/frame`. Results vary by hardware; always
measure on your own machine.
