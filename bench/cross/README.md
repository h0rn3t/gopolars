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

Supported operations: `sum`, `min`, `max`, `mean`, `minmax`, `add`, `mul`.

## Output

- Go benchmark lines are emitted to stdout via `testing.B`.
- A JSON summary with paired results is written to `cross_summary.json` in this directory.
