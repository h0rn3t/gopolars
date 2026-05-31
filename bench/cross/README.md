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

`BenchmarkCrossFilterSum` runs a full filter+sum pipeline:

- **Go path**: `df.Filter(col("a") > 50)` → vectorized batch mask + typed chunk gather → `Sum()` reading `[]float64` directly.
- **Python path**: `df.filter(pl.col("a") > 50)["a"].sum()` via the Polars Rust engine.

Run it with:

```bash
go test ./bench/cross -bench=BenchmarkCrossFilterSum -benchmem
```

Expected delta: gopolars typed path is typically **3–8×** faster than Python Polars on 1M rows, primarily because the Python call involves Python→Rust dispatch overhead on each call. At 10M rows where Rust amortises startup cost, the gap narrows to **1–3×**. Results vary by hardware; always measure on your own machine.
