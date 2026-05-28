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
