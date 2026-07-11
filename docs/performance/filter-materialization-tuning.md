# Filter / materialization at 1M+ rows: runtime tuning

`DataFrame.Filter` (and the other row-materializing ops: `drop_nulls`, `sort`,
`unique`) at the 1M reference scale is bounded by the Go runtime, not by
gopolars kernels. The gather kernels run at memory bandwidth, the predicate
eval is branchless and parallel, and the whole batch path executes as a single
fused worker wave (see `openspec/changes/filter-single-wave-gather/`). What
remains per call, measured on Apple M4 Pro (12 cores, Go 1.26):

| cost | ~share of 1.12 ms | tunable? |
|------|-------------------|----------|
| waking `GOMAXPROCS` parked OS threads for the wave | ~200 µs | no (runtime/OS) |
| GC cycles driven by the ~20 MB/op result allocation | ~300 µs | **yes — `GOGC` / `GOMEMLIMIT`** |
| predicate eval + gather kernels (bandwidth-bound) | ~450 µs | no (already at bandwidth) |

## GOGC / GOMEMLIMIT

Filter's output is unavoidable garbage from the collector's point of view:
each 1M-row call allocates ~20 MB of result columns. With the default
`GOGC=100` and a small live heap, that triggers a GC cycle every 1–3 calls,
and each cycle preempts the gather workers (write barriers, mark assists,
preemption signals). Raising `GOGC` — or setting a `GOMEMLIMIT` and
`GOGC=off` — trades heap headroom for materialization throughput:

| env | FilterHalf 1M | vs default |
|-----|---------------|------------|
| `GOGC=100` (default) | 1.12 ms | — |
| `GOGC=200` | 965 µs | −14% |
| `GOGC=400` | 880 µs | −21% |
| `GOGC=800` | 814 µs | −27% |

For batch/ETL workloads that hammer materializing ops, `GOGC=400`–`800` (or
`GOMEMLIMIT` sized to available RAM) is the single biggest lever available.

## Measured dead ends (do not re-attempt without new evidence)

- **Arrow offsets+bytes string storage** — rejected, made Filter +75% slower
  (interned `[]string` headers are cheaper to gather than string content);
  see `openspec/changes/arrow-string-storage/design.md`.
- **Bit-position decode before gather** (Arrow-style MLP trick) — slower than
  the `TrailingZeros64` chain on Apple Silicon; the OoO core already overlaps
  the loads.
- **Dictionary `uint32` codes for the low-cardinality string column** — the
  gather kernel is latency-bound on source reads, codes gather only ~9%
  faster than header gather; not worth a storage change.
- **Spin-wait instead of channel park at the wave's mid barrier** — +45%:
  spinners starve the allocating goroutine.
- **Fewer/more workers than `GOMAXPROCS`** — 12 is optimal on the reference
  machine; the op is bandwidth-bound, not over-sharded.
