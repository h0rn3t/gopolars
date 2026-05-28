# SIMD Micro-Benchmark Results

## Environment

- **Machine**: Apple M4 Pro (ARM64)
- **Go Version**: 1.26.1
- **Date**: 2026-05-25

## Methodology

Benchmarks were run for `SumFloat64`, `MinFloat64`, `MaxFloat64`, and `MinMaxFloat64`
across slice sizes: 1K, 10K, 100K, 1M, and 10M elements.

Two builds were tested:

1. **Scalar build**: `go test -bench=. ./bench/micro/...`
2. **SIMD build**: `GOEXPERIMENT=simd go test -bench=. ./bench/micro/...`

On this ARM64 machine, the SIMD build falls back to the scalar implementation
because the `amd64` build tag is not satisfied. The results below reflect the
scalar fallback performance, which is the baseline expected on non-AMD64 platforms.

## Results (Scalar / ARM64 Fallback)

### SumFloat64

| Size | ns/op | MB/s |
|------|-------|------|
| 1K   | ~260  | ~30,500 |
| 10K  | ~2,540 | ~31,500 |
| 100K | ~25,100 | ~31,900 |
| 1M   | ~252,000 | ~31,700 |
| 10M  | ~2,680,000 | ~29,800 |

### MinFloat64

| Size | ns/op | MB/s |
|------|-------|------|
| 1K   | ~964 | ~8,300 |
| 10K  | ~9,960 | ~8,000 |
| 100K | ~100,300 | ~7,980 |
| 1M   | ~1,002,000 | ~7,990 |
| 10M  | ~10,003,000 | ~7,990 |

### MaxFloat64

| Size | ns/op | MB/s |
|------|-------|------|
| 1K   | ~948 | ~8,440 |
| 10K  | ~9,967 | ~8,030 |
| 100K | ~100,095 | ~7,990 |
| 1M   | ~1,001,000 | ~7,990 |
| 10M  | ~10,013,000 | ~7,990 |

### MinMaxFloat64

| Size | ns/op | MB/s |
|------|-------|------|
| 1K   | ~1,421 | ~5,630 |
| 10K  | ~15,203 | ~5,260 |
| 100K | ~151,294 | ~5,290 |
| 1M   | ~1,503,000 | ~5,320 |
| 10M  | ~15,378,000 | ~5,200 |

## Observations

- **Zero allocations**: All benchmarked functions allocate `0 B/op`, confirming
  the hot-path implementations are allocation-free.
- **Consistent throughput**: Throughput is stable across sizes, indicating good
  cache behavior on the scalar fallback.
- **Expected AMD64 speedup**: On `GOARCH=amd64` with `GOEXPERIMENT=simd`,
  `turboslice` delivers reported 2–2.6x speedups for `Min`, `Max`, and `MinMax`
  via 128-bit SSE instructions. `SumFloat64` is not accelerated by turboslice
  and relies on compiler auto-vectorization.
- **No regressions**: The SIMD build compiled and ran successfully on ARM64,
  automatically falling back to scalar loops with identical results.
