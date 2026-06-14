# Cross-Language Top30 Benchmark

Compares Go gopolars against Python Polars for the 30 most commonly used DataFrame, Expr, Series, LazyFrame, and SQLContext operations.

## Prerequisites

- Go 1.26+
- Python 3 with `polars` and `pyarrow` installed:
  ```bash
  pip install -r requirements-bench.txt
  ```

## Running

Run the full benchmark suite (all sizes: 1K, 10K, 100K, 1M, 10M):

```bash
go test -bench=BenchmarkTop30 ./bench/top30/...
```

Run in light mode (1K and 1M only) for fast CI feedback:

```bash
go test -bench=BenchmarkTop30 ./bench/top30/... -light
```

Run a specific operation:

```bash
go test -bench=BenchmarkTop30/DataFrame/filter/size_1K ./bench/top30/...
```

## Python Harness (standalone)

The Python harness can be executed independently:

```bash
python harness.py --op filter --input data.arrow --iters 100
```

Supported objects and operations:
- **DataFrame**: filter, select, with_columns, sort, group_by, join, head, tail, unique, fill_null, drop_nulls
- **Expr**: cum_sum, rank, over, fill_null, fill_nan, rolling_mean, rolling_sum, rolling_min, rolling_max
- **Series**: null_count, drop_nans, to_list, is_null, is_not_null, fill_nan
- **LazyFrame**: collect, sql, inspect
- **SQLContext**: execute, register, tables

## Output

- Go benchmark lines are emitted to stdout via `testing.B`.
- `top30_summary.json` — JSON summary with paired Go/Python results.
- `top30_summary.csv` — CSV report for spreadsheet import.
- `top30_summary.md` — Markdown table for human reading.

## Updating Baselines

After a successful full run, copy the JSON summary into `baselines/`:

```bash
cp top30_summary.json baselines/top30_baseline.json
```

Then verify the performance budget gate passes:

```bash
./scripts/check_top30_performance_budgets.sh
```

## Parity gate

The **parity gate** enforces "performance parity on all workloads": every
workload tracked here at the **1M-row reference scale** carries a parity budget
versus Python Polars, and the gate fails if any in-scope workload falls below
its budget or regresses below its committed baseline.

- **Ratio convention:** `ratio = python_time / go_time`. A ratio `≥ 1.0` means
  gopolars is at least as fast as Python Polars; higher is faster.
- **Budgets** live in [`docs/performance/parity_budgets.json`](../../docs/performance/parity_budgets.json):
  each workload has an `r_min` (minimum acceptable ratio). The default target is
  `r_min = 1.0`; a workload whose `r_min` is below `1.0` must carry a
  `justification` (e.g. ops where Polars' multi-threaded chunked engine is ahead
  of the single-threaded gopolars path at 1M). Harness artifacts can be marked
  `out_of_scope`.
- **Baseline** lives in [`baselines/parity_baseline.json`](baselines/parity_baseline.json):
  the committed measured ratios. The gate flags any workload that drops below
  its baseline by more than the budget `tolerance`, even if the absolute `r_min`
  is still met.

The gate is the Go test `TestParityBudgets`, which reads the committed
`top30_summary.json` (no Python needed), so it runs on every push as part of
`go test ./...`. The nightly `Parity (nightly refresh)` workflow re-runs the
cross-language benchmark against Python Polars and re-checks the gate on fresh
numbers.

```bash
# Run the gate locally (Go-only, against the committed summary):
go test -run TestParityBudgets ./bench/top30/...
```

**Ratcheting budgets:** after an optimization, re-run `./run-bench.sh --python`
(or the full top30 suite) to regenerate `top30_summary.json`, inspect the diff,
then raise each improved workload's `r_min` toward `1.0` and refresh
`baselines/parity_baseline.json`.
