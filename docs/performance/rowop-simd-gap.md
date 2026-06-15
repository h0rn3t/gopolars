# Row-op (filter / drop_nulls) gap assessment — non-SIMD feasibility

Change: `profile-tune-join-and-rowop-headroom` §3. Reference machine: Apple M4 Pro
(arm64), Go 1.26, Polars 1.41.2, 1,000,000-row `bench/top30` dataset
(`g` String 5-distinct, `v` Float64, `n` Float64 ~10% null, `i` Int64).

## Question

Can `filter` (Py ×4.6) and `drop_nulls` (Py ×3.7) at 1M reach Polars parity
**without SIMD**? (SIMD compaction is non-portable on arm64 per the prior change's
design.)

## Measurements (after §2)

| op | Go 1M | Polars 1M | gap |
|----|-------|-----------|-----|
| `filter` (v>0, ~50%) | 2.71 ms | 0.50 ms | Py ×5.4 |
| `drop_nulls` (`n`, ~10% null) | 4.98 ms | 1.30 ms | Py ×3.8 |

CPU profile of `BenchmarkFilterHalf` + `BenchmarkDropNullsSparse`
(`go test -bench -cpuprofile`, `go tool pprof -top`):

| symbol | share | what |
|--------|-------|------|
| `runtime.pthread_cond_wait` / `_signal` | ~44% | GC assist + goroutine scheduling |
| `runtime.scanObject` / `madvise` / `gcWriteBarrier` / `findObject` | ~33% | GC marking + sweeping the materialized output |
| `chunk.gatherSlice[string,int]` | ~9% | gathering the String column `g` |
| `frame.keepFromDropped` | ~7.5% | building the keep `[]int` from the drop mask |
| `simd.CompareGTFloat64Bitmap` | **~3.3%** | the predicate itself |

## Attribution

The predicate is **already SIMD** (`CompareGTFloat64Bitmap`) and costs ~3% — it is
not the gap. The dominant cost is **output materialization and the GC pressure it
creates**, concentrated in the String column:

- gopolars stores String columns as `[]string` — one pointer (header) per row.
  Gathering 500K–900K survivors allocates a new `[]string` full of pointers, which
  the GC must then scan (`scanObject`) and write-barrier (`gcWriteBarrier`). That
  GC work, plus the assist/scheduling it triggers (`pthread_cond_*`, `madvise`),
  is the bulk of the time.
- Polars (Arrow) stores strings as a contiguous `offsets[]` + `bytes[]` buffer with
  no per-row pointers. Its filter is a bulk `memcpy` of the kept byte ranges and
  the result has **no pointers to scan** — so it pays neither the per-row gather
  nor the GC-scan cost.

## Portable levers evaluated

- **Bitmap-driven contiguous-run gather** (bulk `copy()` of consecutive kept rows):
  helps only long-run, fixed-width gathers. `filter` at 50% selectivity has a
  scattered keep-set (runs ≈ 1–2 rows) so it gains nothing; `drop_nulls` sparse has
  long runs but its dominant cost is the **String** gather + GC, which a run-copy
  of `[]string` does not reduce (it still allocates and scans the same pointers).
  Expected gain: small, on the fixed-width columns only.
- **Bounds-check elimination / layout tweaks** in `gatherSlice`: micro-level; the
  gather loop is not the bottleneck (the GC scan of its output is).
- **arm64 NEON compaction spike**: would accelerate fixed-width compaction, but the
  gap is String-materialization + GC, which NEON does not address. Not pursued.

## Decision: NO-GO (for a gather/predicate change)

The `filter`/`drop_nulls` gap is **not** predicate- or gather-algorithm-bound; it
is **String-column materialization + GC-bound**, rooted in the `[]string` column
representation. No portable gather/predicate optimization closes it:

- The predicate is already SIMD (~3%).
- A contiguous-run gather does not reduce the String column's allocate-and-scan cost
  and does nothing for the scattered `filter` keep-set.
- NEON would accelerate only the ~non-dominant fixed-width portion.

The real lever is an **Arrow-style String layout** (`offsets[]` + `bytes[]`, no
per-row pointers), which removes both the per-row gather and the GC scan — a
column-storage architecture change, explicitly out of scope here. Recorded as the
residual; a future `arrow-string-storage` change is the path to parity.

No behavioral code change is made for §3 (results unchanged). The run-gather
prototype was evaluated against this profile and declined: the profile predicts a
small gain confined to fixed-width columns while the dominant String/GC cost is
untouched, so it is not worth the added gather complexity.

## Reproduce

```
go test ./bench/top30/ -run '^$' -bench 'BenchmarkFilterHalf$|BenchmarkDropNullsSparse$' \
  -benchtime=50x -cpuprofile=/tmp/row.prof
go tool pprof -top /tmp/row.prof
# Polars reference:
python3 bench/top30/harness.py --op filter     --object DataFrame --input <1M.arrow> --iters 10
python3 bench/top30/harness.py --op drop_nulls  --object DataFrame --input <1M.arrow> --iters 10
```
