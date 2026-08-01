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

## Which parallel regime runs (columns vs `GOMAXPROCS`)

The batch path has two parallel regimes, and which one runs depends on the
column count relative to `GOMAXPROCS`:

- `len(columns) <= GOMAXPROCS` → the fused wave: workers split **rows**, each
  worker gathers its row range across every column.
- `len(columns) > GOMAXPROCS` → `takeColumnsBitmap`: one **column** per worker,
  round-robin.

The round-robin regime balances only when columns clearly outnumber workers.
At `columns == GOMAXPROCS` every worker gets exactly one column, so wall-time
becomes the most expensive column (a `String` gather costs ~2.4x an `Int64` one
— 16-byte headers plus GC write barriers) while the rest idle. Measured on the
4-column bench frame at 1M rows, `GOMAXPROCS=4` (median of 6 × 1 s):

| | one column per worker | fused wave |
|---|---|---|
| `BenchmarkFilterHalf` | 2.22 ms | **1.87 ms** |
| `BenchmarkDropNullsSparse` | 4.22 ms | **2.75 ms** |

Hence the boundary is `<=`, not `<`. It does not extend further: at
`GOMAXPROCS=4`, filter over 6 columns is 2.51 ms fused vs 3.29 ms round-robin
(fused wins), over 8 columns 3.43 vs 3.66 ms (noise), and over 12 columns
5.50 vs 4.79 ms — **fused loses**, because each worker then holds `columns`
input plus `columns` output streams and prefetch collapses. At
`columns == GOMAXPROCS == 12` the two regimes tie for filter (3.9 vs 3.7 ms,
within run-to-run spread) and the fused wave wins for drop_nulls
(4.97 vs 6.34 ms), so the `<=` boundary is safe on a 12-core machine too.

This matters for CI: the GitHub runner has 4 vCPUs and the bench frame has 4
columns, so before this boundary fix the runner exercised a different regime
than the 12-core machine the budgets were seeded on.

### Measuring: use env `GOMAXPROCS`, not `-cpu`

`go test -cpu=N` does **not** apply to the first `-count` iteration in a
process: that run still uses the previous `GOMAXPROCS`, which silently measures
the wrong regime. Control parallelism with the environment variable
(`GOMAXPROCS=4 go test -bench=...`) and use `allocs/op` as the regime check —
on the 4-column frame filter reports 114 allocs/op with 12 shards versus 58–60
with 4.

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
