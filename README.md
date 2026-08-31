# gopolars

[![CI](https://github.com/h0rn3t/gopolars/actions/workflows/ci.yml/badge.svg)](https://github.com/h0rn3t/gopolars/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/h0rn3t/gopolars/graph/badge.svg)](https://codecov.io/gh/h0rn3t/gopolars)
[![Release](https://img.shields.io/github/v/release/h0rn3t/gopolars?sort=semver)](https://github.com/h0rn3t/gopolars/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/h0rn3t/gopolars/pkg/polars.svg)](https://pkg.go.dev/github.com/h0rn3t/gopolars/pkg/polars)
[![Go 1.27+](https://img.shields.io/badge/Go-1.27%2B-00ADD8?logo=go)](go.mod)

`gopolars` is a high-performance Go DataFrame library inspired by Polars Python API.

## About

| | |
| --- | --- |
| **API docs** | [UK](https://h0rn3t.github.io/gopolars/) · [EN](https://h0rn3t.github.io/gopolars/en.html) — `pkg/polars` reference (Go syntax highlighting) |
| **Module** | `github.com/h0rn3t/gopolars/pkg/polars` |
| **Godoc** | [pkg.go.dev/.../pkg/polars](https://pkg.go.dev/github.com/h0rn3t/gopolars/pkg/polars) |

Set the repository **Website** (GitHub → Settings → About) to the API docs URL above so it appears in the sidebar.

## Installation

```bash
go get github.com/h0rn3t/gopolars@latest
```

Or pin the latest release:

```bash
go get github.com/h0rn3t/gopolars@v0.4.1
```

Import the public API package:

```go
import "github.com/h0rn3t/gopolars/pkg/polars"
```

Requires **Go 1.27+**. Float64 min/max reductions use runtime-dispatched AVX2 on capable
`amd64` CPUs (one binary, no build tag). Building with `GOEXPERIMENT=simd` additionally enables
portable vector kernels (NEON on `arm64`, AVX/AVX2/AVX512 on `amd64`) for the reductions and the
fused filter-reduce path; see [Performance / SIMD Acceleration](#performance--simd-acceleration).

## Current status

Latest release: **[v0.4.1](https://github.com/h0rn3t/gopolars/releases/tag/v0.4.1)**
([changelog vs v0.4.0](https://github.com/h0rn3t/gopolars/compare/v0.4.0...v0.4.1)).
The public API is versioned with SemVer; while `< v1.0.0` it may still evolve between minor
versions — see the [versioning policy](docs/versioning_policy.md). `v0.4.1` is a patch release
and changes no public API; the [v0.4.0 migration notes](docs/v0_4_migration.md) still cover the
one **breaking** change in the `v0.4` line (`DataFrame.Clone` now shares column buffers).

The project has driven its internal parity waves up to the **v1.0 tracking matrix** ([`docs/parity/v1_0_coverage.json`](docs/parity/v1_0_coverage.json)) and now covers a broad core for Go-native analytics pipelines, including advanced joins, reshape operations, temporal windows, opt-in DuckDB SQL, and performance diagnostics.\
It is production-usable for many DataFrame workloads, but it is **not yet a full drop-in replacement** for Python Polars.

- ✅ Strong DataFrame/LazyFrame core for real analytics workloads
- ✅ Stable IO surface (CSV/JSON/Parquet/IPC + scans + pushdown)
- ✅ Opt-in SQL over in-memory frames via embedded DuckDB (`-tags duckdb,duckdb_arrow`)
- ✅ **75%** statement coverage for `./pkg/...` (unit + package tests; see [Testing](#testing))
- ✅ **659 / 670** public Python Polars methods implemented, measured against **Polars 1.41.2** ([full parity matrix](#python-polars-vs-gopolars-function-matrix)) — 11 named gaps, listed below

### What's new in v0.4.1

Patch release — performance and internals only, no public API change.

- **Null-free columns no longer allocate a validity mask.** Constructing a column with no nulls
  (`NewDataFrame` from `[]any`, and every typed constructor) skipped straight past an invariant the
  engine already relied on: `Gather`/`Slice` produced nil-validity columns and every reader guards
  for nil, yet construction still materialized an n-byte all-false slice. It now stays nil until a
  null exists, which also makes the column's first `NullCount()` free instead of a full scan
- **`Series.IsNull` / `Series.IsNotNull` are 30–67% faster** at 1 M rows and allocate half as much
  (1968 KiB → 984 KiB). The validity negation behind `is_not_null` moved onto the sharded kernel
  path, so a column that actually carries nulls went 307.8 µs → 101.2 µs

Measured through the public `NewDataFrame` path with `benchstat` (n=8, p=0.000) on the hardware
listed under [Benchmark](#benchmark-gopolars-vs-python-polars). The parity tables below are still
the `v0.4.0` measurement run and do not yet reflect these numbers.

### What's new in v0.4.0

- **BREAKING — `DataFrame.Clone` shares column buffers** instead of deep-copying, matching Python
  Polars' `clone` (3.02 ms / 44 MB → 0.20 µs at 1 M rows). Reads, null positions and structural
  independence are unchanged; see the [migration notes](docs/v0_4_migration.md)
- **Metadata ops are now O(columns), not O(rows)** — `Row(i)` no longer materializes the whole
  frame to return one row (153 ms → 0.19 µs at 1 M), and `Rename`/`Drop` share the untouched
  columns (**Go ×20** / **×201** vs Polars)
- **Parallel window functions** — `over` builds partition ids across all cores
  (18.7 ms → 4.89 ms at 1 M, **Go ×2.1**); `rank` moved onto the parallel argsort
  (21.2 ms → 14.1 ms, now at parity)
- **CSV writer ~7× faster** (223.6 ms → 30.6 ms at 1 M) — cells append into a reused byte buffer
  instead of allocating a string each, and row ranges format concurrently; output is byte-identical
- **Sorting fix** — the radix argsort placed `-0.0` before `0.0`, breaking stability for keys
  containing both; the sequential and parallel paths are both corrected
- **Parity budgets ratcheted** — 25 workload floors raised and 4 brought into scope after a fresh
  cross-language run

### What's new in v0.3.0

- **DuckDB-backed SQL** — `DataFrame.SQL` / `LazyFrame.SQL` / `SQLContext` run through an embedded DuckDB engine (opt-in CGO build tags); the previous hand-rolled `pkg/sql` path is retired
- **Faster null and slice kernels** — parallel float64 fill/drop, parallel column slicing, optimized `DropNulls`
- **Faster CSV / Parquet writers** and expanded nested Arrow support
- **Broader package and SQL parity tests** around the DuckDB-backed execution path

## Implemented capabilities

### DataFrame, Series and Expressions

- Eager and lazy execution over a columnar in-memory DataFrame engine
- Core DataFrame operations: `select`, `filter`, `with_columns`, `sort`, `limit`, `group_by`, `join`
- Extended DataFrame surface: `slice`, `head`, `tail`, `unique`, `concat`, `fill_null`/`fill_nan`, `drop_nulls`/`drop_nans`, `drop`, `rename`, `gather_every`, `top_k`/`bottom_k`, `interpolate`, `transpose`, horizontal reducers (`sum`/`min`/`max`/`mean_horizontal`), and `describe`
- Eager fused fast path: `FilterAggregateDirect` computes `filter().sum/min/max/mean/count` in a single masked pass without building a lazy plan or materializing a filtered DataFrame
- Join modes: `inner`, `left`, `right`, `full`, `semi`, `anti`, `cross`, `asof` (plus predicate `join_where`)
- Temporal analytics: `group_by_dynamic`, time-based `rolling`, and the full rolling-aggregate family (`rolling_mean`/`sum`/`min`/`max`/`median`/`std`/`var`/`quantile`, their `*_by` variants, and `ewm_mean`)
- Reshape support: `melt`/`unpivot`, `pivot`, `unstack`, `explode`, `unnest`, struct `flatten`, `transpose`
- Rich public `Series` API: null-aware vector arithmetic and comparisons, aggregations, cumulative ops (`cum_sum`/`max`/`min`/`prod`/`count`), rolling and EWM windows, ranking/sorting, binning (`cut`/`qcut`/`hist`), bitwise ops, plus `str`/`dt`/`arr`/`struct`/`cat`/`bin` namespaces
- Expression API (217 tracked `Expr` methods) with namespaces for string/datetime/list/struct workflows:
  - string (`str`): `str_lower`, `str_upper`, `str_len`/`str_char_len`, `str_trim`/`str_ltrim`/`str_rtrim`, `str_replace`/`str_replace_all`, `str_substr`, `str_left`/`str_right`, `str_pad_start`/`str_pad_end`, `str_split_part`, `str_reverse`, `str_to_title`, `str_concat`/`str_concat_ws`, `starts_with`/`ends_with`, `contains`/`str_like`
  - datetime (`dt`): `dt_year`, `dt_month`, `dt_day`, `dt_hour`, `dt_minute`, `dt_second`, `dt_weekday`, `dt_ordinal_day`
  - list (`arr`): `list_len`, `list_get`, `list_contains`
  - struct: `struct_field`, `struct` packing
  - window / analytics: `over`, `rank`, `cum_*`, `rolling_*`, `ewm_mean`, `diff`, `shift`, `pct_change`, `interpolate`, `fill_null`/`fill_nan`
  - reshape: `explode`, `flatten` (struct flattening)

### IO and Interoperability

- CSV, JSON/NDJSON, Parquet, IPC read/write support
- `write_database` / `read_database` over external SQL databases via an ADBC (Arrow Database Connectivity) engine
- Source-level lazy scan for CSV/JSON/Parquet/IPC
- Projection and predicate pushdown on scan pipelines
- Partition-aware Parquet dataset scan (multi-file layout)
- Partition pruning by predicate for dataset scans
- Arrow import/export bridge (Apache `arrow-go/v18`)
- Object store URI mapping profile (`s3://`, `gcs://`, `az://`) via environment-configured roots

#### Database IO (ADBC)

`DataFrame.WriteDatabase` and `polars.ReadDatabase` move data to/from external SQL databases as Arrow
record batches via [ADBC](https://arrow.apache.org/adbc/) — the driver creates the table from the
Arrow schema and bulk-loads it, so gopolars writes no SQL DDL, dialect, or placeholder code.

```go
// Bring your own open *adbc.Connection (e.g. SQLite, PostgreSQL, FlightSQL, Snowflake).
n, err := df.WriteDatabase(polars.WriteDatabaseInput{
    TableName:     "analytics.public.events", // catalog.schema.table (quoting honored)
    IfTableExists: polars.IfTableExistsReplace, // fail (default) | append | replace
    Conn:          conn,                         // an adbc.Connection
    BatchSize:     10000,                        // streamed in batches
}) // n = rows affected (-1 if the driver doesn't report it)

out, err := polars.ReadDatabase(ctx, polars.ReadDatabaseInput{
    Query: "SELECT id, name FROM events ORDER BY id",
    Conn:  conn,
})
```

- **Connection model**: pass an already-open `adbc.Connection` via `Conn`, or a `DriverName` +
  `DriverOptions` to open one through the ADBC driver manager.
- **Trade-off**: the ADBC + `arrow-go/v18` graph is a direct dependency, and the SQLite/PostgreSQL
  drivers require **CGO** (FlightSQL/Snowflake are pure Go). A `CGO_ENABLED=0` build still compiles;
  driver-name connections then return a clear "CGO required" error, while a caller-provided
  `adbc.Connection` keeps working.
- **Type fidelity**: Int64/Float64/Boolean/String/Datetime map natively; Decimal/Categorical degrade
  to text and List/Struct are rejected on write (documented gaps). Round-trip fidelity also depends on
  the backend's type system — e.g. SQLite's weak typing returns Boolean as integer 1/0 and Datetime as
  ISO text, while a strongly-typed engine preserves them.
- **Running the integration tests**: `pip install adbc-driver-sqlite`, then `go test ./pkg/io/database/`
  — the suite auto-discovers the bundled SQLite driver (CGO) and runs a real write→read round-trip;
  without it those tests skip.

### SQL over in-memory frames (opt-in, DuckDB)

Run SQL queries directly against in-memory DataFrames via an **embedded DuckDB
engine**. This is **opt-in behind build tags** — the default build stays pure-Go
(`CGO_ENABLED=0`):

```bash
go build -tags "duckdb duckdb_arrow" ./...   # links a bundled DuckDB static lib (CGO; ~+50 MB)
```

```go
// single frame, addressable as `self`
lf, _ := df.SQL(ctx, "SELECT g, sum(v) AS s FROM self GROUP BY g")

// multi-table joins via a context
ctx := polars.NewSQLContext()
_ = ctx.Register("orders", orders)
_ = ctx.Register("users", users)
out, _ := ctx.Execute(c, "SELECT u.name, sum(o.amount) FROM orders o JOIN users u ON o.uid = u.id GROUP BY u.name")
```

- Without the `duckdb duckdb_arrow` tags, the SQL methods return a clear
  "build with -tags duckdb" error and the binary stays pure-Go.
- The dialect is **DuckDB's**, not polars' native `polars-sql`. Compatibility with
  the py-polars SQL suite (1.28.1) is measured in `test/parity/unit/sql/`:
  **140 MATCH / 33 GAP / 0 FAIL** across 22 ported files; divergences (e.g. integer
  `/` is true division, no Date/Struct/List result dtypes) are documented inline.
- Frames cross the boundary as Arrow (reusing `pkg/io/arrow`); SQL is an eager
  step (`collect → DuckDB → DataFrame`), not fused into the lazy plan.

### Streaming, Diagnostics and Quality

- Streaming collect with bounded-memory path and deterministic fallback
- Explain and diagnostics output with stable schema for automation (`schema_version: v2`)
- Operator-level execution report structure for telemetry integrations (duration, memory, temporal operator markers)
- Unit/conformance tests (**75%** `pkg/` statement coverage), benchmarks, `go vet`, race tests, CI quality gates
- Compatibility governance artifacts:
  - versioning policy
  - migration notes
  - breaking-change evidence gate script
  - performance budget and regression evidence scripts

## Capability matrix

| Capability                                                        | Status         |
| ----------------------------------------------------------------- | -------------- |
| DataFrame eager API                                               | ✅ ready        |
| Lazy execution, scans, pushdown                                   | ✅ ready        |
| Series public API                                                 | ✅ ready        |
| Nested transforms (explode/flatten + list/struct expr)            | ✅ ready        |
| GroupBy, temporal windows and joins                               | ✅ ready        |
| Streaming collect                                                 | ✅ ready        |
| CSV/JSON/Parquet/IPC IO                                           | ✅ ready        |
| Arrow interoperability                                            | ✅ ready        |
| Cloud-style partitioned dataset scans                             | ✅ ready        |
| Explain/telemetry schema v2 and perf markers                      | ✅ ready        |
| SQL over in-memory frames (opt-in DuckDB)                         | ✅ ready        |
| Full Python Polars API parity (670 methods, Polars 1.41.2)        | ✅ ~98.4% (659/670) |
| Performance parity on all workloads                               | 🚧 in progress |
| Ecosystem parity (all namespaces, plugins, advanced UDF patterns) | 🚧 in progress |

## Python Polars vs gopolars function matrix

**Full-matrix totals (670 public Python Polars methods on `DataFrame`, `LazyFrame`, `Expr`, `Series`, measured against Polars 1.41.2):** **659 implemented**, **11** remaining. Coverage **≈98.4%**.

The matrix is generated, not hand-maintained — see [`POLARS_PARITY_TABLE.md`](POLARS_PARITY_TABLE.md), produced by [`gen_parity_table.py`](gen_parity_table.py). Each Python class is checked against the corresponding Go type only (`DataFrame`/`LazyFrame`/`Series` against their interfaces in `pkg/polars/types.go`, `Expr` against methods with an `Expr` receiver in `pkg/expr`); a method is never credited to one class because another type happens to expose the same name. The matrix tracks method *presence*, not signature or semantics — those are covered by `test/parity` and `test/conformance`.

### Coverage by object (full matrix)

| Object     | Implemented | Total in matrix |
| ---------- | ----------- | ---------------- |
| DataFrame  | 136         | 137              |
| LazyFrame  | 88          | 91               |
| Expr       | 216         | 219              |
| Series     | 219         | 223              |
| **Total**  | **659**     | **670**          |

### Remaining rows (full matrix)

| Object    | Not implemented                         |
| --------- | --------------------------------------- |
| DataFrame | `gather`                                |
| LazyFrame | `execute`, `fetch`, `gather`            |
| Expr      | `is_empty`, `len`, `register_plugin`    |
| Series    | `cumulative_eval`, `ext`, `plot`, `sql` |

To regenerate the matrix against the Polars release you care about, run the generator with an interpreter that has `polars` installed — the version it measured is stamped into the table header:

```bash
python3 gen_parity_table.py
```

## What is still needed to replace Python Polars

To position `gopolars` as a practical replacement for Python Polars in most teams, the following areas still need expansion:

1. **Broader API parity**
   - Semantic edge cases and error contracts vs Python Polars (the tracked matrix is largely covered; remaining gaps are documented in the matrix)
   - Deeper namespace behavior where Python has richer edge-case semantics
2. **Performance and scale hardening**
   - More optimization rules and workload-adaptive planning
   - Larger benchmark corpus and stricter regression budgets
3. **Cloud and lakehouse robustness**
   - Expanded object-store behavior and dataset semantics at scale
   - More integration coverage for partitioned and heterogeneous datasets
4. **Compatibility and migration experience**
   - Continued stabilization of deprecation/migration workflows
   - Clear release evidence for every potentially breaking alignment change

## Roadmap focus

- **Near term:** align remaining semantic edge cases on the covered API surface, and harden performance budgets on larger datasets.
- **Mid term:** improve planner/runtime adaptability for mixed temporal + join + reshape workloads.
- **Final parity push:** close long-tail semantic differences and publish repeatable parity evidence for release readiness.

## Internal storage architecture

Each column is backed by a **typed `chunk.Column`** (dense typed slice + validity
mask) rather than a boxed `[]any`. This eliminates per-element heap allocations
on all hot paths.

| dtype | backing type |
| ----- | ------------ |
| `Int64` | `[]int64` |
| `Float64` | `[]float64` |
| `Boolean` | `[]bool` |
| `String` / `Categorical` / `Enum` | `[]string` |
| `Datetime` | `[]time.Time` |
| nested / other | `[]any` (slow boxed path) |

### Vectorized hot paths

| Operation | Typed fast path |
| --------- | --------------- |
| `Filter` | `evalbatch.Plan` produces a predicate mask; `chunk.Column.Filter` gathers with a single typed copy — no `expr.Eval` per row |
| `FilterAggregateDirect` (eager `filter().sum/min/max/mean/count`) | fused single pass over a predicate `Bitmap` via `simd.MaskedReduceFloat64` — no lazy plan, no surviving-index slice, no materialized filtered column |
| `WithColumns` (arithmetic/compare/logic) | batch kernel over typed backing — no `[]any` intermediate |
| `GroupBy` sum/mean/min/max on numeric columns | reads `[]float64` / `[]int64` directly, skipping `expr.Eval` per row |
| `Sort` on Int64/Float64 | pre-built typed comparator reads the backing slice — no `Value(i)` boxing in the hot path |
| `Join` key building on Int64/String | reads typed buffer; avoids `fmt.Sprintf` of `any` |
| `ToTable` / `ToArrowRecord` | reads typed backing; does not call `s.Value(i)` for primitive dtypes |
| Concat (lazy scan materialize) | `chunk.ConcatColumns` copies typed slices with `copy()` — no boxing |
| Series aggregations (`Sum`/`Min`/`Max`/`Mean`/etc.) | reads `[]float64` or `[]int64` directly |

Row-wise `expr.Eval` is preserved as a fallback for unsupported operations
(reverse access, dynamic JSON, unknown ops) and emits a debug log when triggered.

### Rollback

Set `GOPOLARS_TYPED_STORAGE=0` to force the row-wise path everywhere. This flag
is available for diagnosing regressions and will be removed after the typed path
matures.

## Performance / SIMD Acceleration

`gopolars` uses runtime-dispatched numeric kernels in `pkg/simd`. **No build tag
is required** — one binary selects AVX2 or scalar at runtime.

On **amd64**, `MinFloat64` / `MaxFloat64` / `MinMaxFloat64` dispatch to
hand-written AVX2 assembly when `cpu.X86.HasAVX2` is true, and fall back to a
scalar multi-accumulator path on pre-AVX2 CPUs. `SumFloat64`,
`DotProductFloat64`, `AddSlicesFloat64`, and `MulSlicesFloat64` stay scalar
everywhere — the Go compiler already auto-vectorizes those loops (measured
hand-written AVX2 add/mul was slower on EPYC).

Other column kernels (always available, all platforms):

- `CompareGTFloat64`, `CompareEQInt64` — `[]bool` filter mask generation
- `AndMask` — boolean mask combination
- `CompressIndices` — compress a predicate `Bitmap` to `[]int` for gather

### Packed predicate bitmaps

Filter predicates are carried as a packed `Bitmap` (`[]uint64`, one bit per row)
instead of a `[]bool` byte mask. This cuts mask bandwidth 8× and lets the
reduce/compress kernels work a word at a time with `math/bits`
(`OnesCount64`, `TrailingZeros64`) — a single instruction each on amd64 and
arm64.

- `Bitmap` data structure with `BitmapNew`, `BitmapSet`, `BitmapGet`,
  `BitmapPopcount`
- `CompareGTFloat64Bitmap`, `CompareEQInt64Bitmap` — bitmap counterparts of the
  comparison kernels
- `BitmapAnd` — word-at-a-time logical AND of two predicate bitmaps
- `MaskedReduceFloat64` — fused filter + reduce: sums/min/max/count over the
  surviving, non-null rows of a bitmap in a single pass, with no surviving-index
  slice or materialized filtered column

### Portable vector kernels (`GOEXPERIMENT=simd`, opt-in)

Building with `GOEXPERIMENT=simd` compiles an additional set of kernels written
against the Go 1.27 stdlib `simd` package. Those types are vector-length-agnostic
and backed by NEON on `arm64`, AVX/AVX2/AVX512 on `amd64`, so **`arm64` gets
hardware SIMD for the first time** — it ships no hand-written assembly. On
`amd64` the existing AVX2 assembly keeps priority and the portable path serves as
the pre-AVX2 fallback.

Measured on Apple M4 Pro (`arm64`/NEON, 2 `float64` lanes — x86 vectors are 4–8
lanes wide and should do better), 1M rows, `GOEXPERIMENT=simd` vs the default
build:

| Kernel | default | `GOEXPERIMENT=simd` | Delta |
| --- | --- | --- | --- |
| `MinMaxWhereFloat64` | 4178.3 µs | 435.6 µs | **−90%** |
| `MaxFloat64` | 277.8 µs | 133.8 µs | **−52%** |
| `SumWhereFloat64` (filter+sum) | 547.5 µs | 267.8 µs | **−51%** |
| `MinFloat64` | 272.0 µs | 151.0 µs | **−44%** |
| `MinMaxFloat64` | 324.7 µs | 239.6 µs | **−26%** |

`SumFloat64`, `AddSlicesFloat64`, `MulSlicesFloat64` and the bitmap comparison
kernels are deliberately left scalar — measured at or below parity with the
vector form. See `pkg/simd/doc.go` for the numbers behind each exclusion.

Correctness never depends on the experiment: every kernel keeps its scalar body
as the fallback, and the equivalence tests pin the vector results (bit-identical
min/max/count, sums within reduction-order tolerance) to it.

Reproduce the table above — `benchstat` runs via `go run`, so it adds nothing to
`go.mod`:

```bash
make bench-simd                              # A/B the whole bench/micro sweep
make bench-simd BENCH='SumWhere|MinMax'      # narrow the selection
make bench-simd BENCH_COUNT=10 BENCH_TIME=1s # tighten the confidence interval
```

Build (AVX2 used automatically on capable amd64 CPUs):

```bash
go build ./...                      # default: AVX2 at runtime on amd64
GOEXPERIMENT=simd go build ./...    # + portable vector kernels (arm64 + amd64)
```

On non-AMD64 architectures (e.g., ARM64) or pre-AVX2 amd64, the library uses
scalar multi-accumulator loops with **bit-identical min/max results** (pinned by
an equivalence test). Correctness never depends on AVX2 being present.

### Expected performance profile

| Workload | Expected speedup vs old `[]any` path |
| -------- | ------------------------------------ |
| `filter+sum` on 1M Float64 rows | ~3–8× (vectorized mask + typed sum) |
| `group_by` sum/mean on 1M rows (100 groups) | ~6–9× (parallel shard tables + typed running reduce, scales with cores) |
| `sort` on 1M Int64/Float64 rows | ~2× argsort (parallel-merge radix; multi-key via radix leading key + comparator tie-break) |
| Arrow IPC round-trip (1M rows) | ~2–3× (no intermediate `[]any`) |

Run the built-in benchmarks to measure on your hardware:

```bash
# Filter + sum on 1M rows
go test ./bench/micro -bench=BenchmarkFilterSum -benchmem

# Full cross-engine comparison vs Python Polars (requires Python + polars installed)
go test ./bench/cross -bench=BenchmarkCross -benchmem
```

Micro-benchmark results are documented in
[`bench/micro/simd_results.md`](bench/micro/simd_results.md).

## Benchmark: gopolars vs Python Polars

> **Hardware:** Apple M4 Pro · Go 1.26.5 · Python Polars 1.41.2 · macOS arm64  
> **Methodology:** Go benchmarks run with `go test -benchmem` at the default `-benchtime=1s`; Python timings measured by the same harness with 10 repetitions per operation. Go time = min ns/op across calibration rounds. Python time = mean across 10 iterations. Sub-microsecond operations need at least `-benchtime=1s`: at `10x`/`100x` they are dominated by noise.  
> **Dataset:** generated float64/string/int64 columns with ~5% null rate; sizes 1 K and 1 M rows (filter+sum also 10 K / 100 K / 10 M).  
> **"speedup":** `Go ×N` means gopolars is N× faster; `Py ×N` means Python Polars is N× faster.  
> **Measured for:** [v0.4.0](https://github.com/h0rn3t/gopolars/releases/tag/v0.4.0) (2026-08-04 local re-run).

Regenerate the tables from source:

```bash
# Run top-30 benchmark (requires Python + polars installed)
# One run regenerates top30_summary.{json,csv,md} and benchmem.txt together, so the
# timings, the memory figures and the ratcheted parity budgets all come from the
# same measurement. Note `-light` goes after the package path (it is a test flag).
go test -run '^$' -bench 'BenchmarkTop30$' \
  -benchmem -timeout=120m ./bench/top30/ -light \
  2>&1 | tee bench/top30/benchmem.txt

# Run filter+sum cross-engine comparison vs Python Polars
./run-bench.sh --python

# Regenerate tables
python3 bench/gen_comparison_table.py --benchmem bench/top30/benchmem.txt \
  --output bench/comparison_table.md
```

### DataFrame operations

> `unique` compares equivalent semantics on both engines: full-frame keep-first in
> maintain-order on the key subset (Go `df.Unique("g")`; Python
> `df.unique(subset=["g"], keep="first", maintain_order=True)`) — not a narrowed
> `select("g").unique()`. The `join` row times the dimension build (`unique`) plus
> the join, so its speedup improved alongside `unique`.

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `filter` | 1 K | 5.8 µs | 23.1 KB | 22 | 109.8 µs | **Go ×18.9** |
| `filter` | 1 M | 1.13 ms | 19.8 MB | 116 | 615.8 µs | Py ×1.8 |
| `select` | 1 K | 653 ns | 1.5 KB | 10 | 56.4 µs | **Go ×86.4** |
| `select` | 1 M | 467 ns | 1.5 KB | 10 | 46.4 µs | **Go ×99.3** |
| `with_columns` | 1 K | 568 ns | 1.5 KB | 10 | 8.5 µs | **Go ×14.9** |
| `with_columns` | 1 M | 449 ns | 1.5 KB | 10 | 9.1 µs | **Go ×20.2** |
| `sort` | 1 K | 24.2 µs | 66.6 KB | 19 | 201.3 µs | **Go ×8.3** |
| `sort` | 1 M | 16.09 ms | 62.0 MB | 153 | 11.98 ms | Py ×1.3 |
| `group_by` | 1 K | 14.9 µs | 18.6 KB | 41 | 706.5 µs | **Go ×47.5** |
| `group_by` | 1 M | 1.48 ms | 90.7 KB | 186 | 1.63 ms | **Go ×1.1** |
| `join` | 1 K | 90.0 µs | 217.4 KB | 727 | 393.2 µs | **Go ×4.4** |
| `join` | 1 M | 6.93 ms | 95.2 MB | 1590 | 5.94 ms | Py ×1.2 |
| `head` | 1 K | 466 ns | 1.6 KB | 10 | 621 ns | **Go ×1.3** |
| `head` | 1 M | 412 ns | 1.6 KB | 10 | 1.0 µs | **Go ×2.5** |
| `tail` | 1 K | 466 ns | 1.6 KB | 10 | 612 ns | **Go ×1.3** |
| `tail` | 1 M | 327 ns | 1.6 KB | 10 | 658 ns | **Go ×2.0** |
| `unique` | 1 K | 8.9 µs | 1.9 KB | 21 | 149.6 µs | **Go ×16.8** |
| `unique` | 1 M | 1.08 ms | 70.6 KB | 147 | 4.85 ms | **Go ×4.5** |
| `fill_null` | 1 K | 2.5 µs | 10.0 KB | 10 | 116.0 µs | **Go ×46.1** |
| `fill_null` | 1 M | 478.9 µs | 8.6 MB | 35 | 905.6 µs | **Go ×1.9** |
| `drop_nulls` | 1 K | 10.5 µs | 42.7 KB | 17 | 65.8 µs | **Go ×6.2** |
| `drop_nulls` | 1 M | 1.63 ms | 35.4 MB | 64 | 1.31 ms | Py ×1.2 |
| `clone` | 1 K | 238 ns | 672 B | 5 | 333 ns | **Go ×1.4** |
| `clone` | 1 M | 189 ns | 672 B | 5 | 329 ns | **Go ×1.7** |
| `drop` | 1 K | 319 ns | 800 B | 8 | 43.3 µs | **Go ×135.7** |
| `drop` | 1 M | 342 ns | 800 B | 8 | 68.8 µs | **Go ×201.1** |
| `rename` | 1 K | 410 ns | 1.1 KB | 9 | 6.7 µs | **Go ×16.3** |
| `rename` | 1 M | 329 ns | 1.1 KB | 9 | 6.7 µs | **Go ×20.4** |
| `row` | 1 K | 191 ns | 440 B | 7 | 279 ns | **Go ×1.5** |
| `row` | 1 M | 163 ns | 440 B | 7 | 292 ns | **Go ×1.8** |

### Expr operations

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `cum_sum` | 1 K | 2.6 µs | 10.7 KB | 14 | 13.1 µs | **Go ×5.1** |
| `cum_sum` | 1 M | 779.3 µs | 8.6 MB | 14 | 2.94 ms | **Go ×3.8** |
| `rank` | 1 K | 18.4 µs | 34.7 KB | 17 | 67.2 µs | **Go ×3.7** |
| `rank` | 1 M | 14.08 ms | 31.5 MB | 47 | 13.71 ms | Py ×1.0 |
| `over` | 1 K | 15.4 µs | 28.2 KB | 28 | 326.6 µs | **Go ×21.2** |
| `over` | 1 M | 4.89 ms | 25.2 MB | 137 | 10.16 ms | **Go ×2.1** |
| `fill_null` | 1 K | 3.3 µs | 10.9 KB | 16 | 52.4 µs | **Go ×15.9** |
| `fill_null` | 1 M | 402.3 µs | 8.6 MB | 41 | 675.0 µs | **Go ×1.7** |
| `fill_nan` | 1 K | 1.2 µs | 1.9 KB | 15 | 73.1 µs | **Go ×60.7** |
| `fill_nan` | 1 M | 75.5 µs | 2.6 KB | 40 | 871.1 µs | **Go ×11.5** |
| `rolling_mean` | 1 K | 6.4 µs | 10.7 KB | 16 | 18.3 µs | **Go ×2.8** |
| `rolling_mean` | 1 M | 4.51 ms | 8.6 MB | 16 | 9.22 ms | **Go ×2.0** |
| `rolling_sum` | 1 K | 6.6 µs | 10.7 KB | 16 | 20.0 µs | **Go ×3.0** |
| `rolling_sum` | 1 M | 4.39 ms | 8.6 MB | 16 | 9.05 ms | **Go ×2.1** |
| `rolling_min` | 1 K | 4.9 µs | 11.5 KB | 17 | 15.2 µs | **Go ×3.1** |
| `rolling_min` | 1 M | 8.11 ms | 8.9 MB | 28 | 12.05 ms | **Go ×1.5** |
| `rolling_max` | 1 K | 5.3 µs | 11.5 KB | 17 | 15.3 µs | **Go ×2.9** |
| `rolling_max` | 1 M | 8.12 ms | 8.9 MB | 28 | 12.15 ms | **Go ×1.5** |

> Rolling window operations (`rolling_mean/sum/min/max`) use O(n) linear kernels — a running Neumaier-compensated accumulator for sum/mean and a monotonic-index deque for min/max — writing a single output buffer (≈9 MB/op at 1 M rows). gopolars outpaces Python Polars on all four at 1 M rows.

### Series operations

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `null_count` | 1 K | 2 ns | 0 B | 0 | 504 ns | **Go ×252.1** |
| `null_count` | 1 M | 2 ns | 0 B | 0 | 683 ns | **Go ×341.6** |
| `drop_nans` | 1 K | 396 ns | 300 B | 4 | 13.2 µs | **Go ×33.4** |
| `drop_nans` | 1 M | 73.7 µs | 988 B | 29 | 99.7 µs | **Go ×1.4** |
| `to_list` | 1 K | 9.7 µs | 23.8 KB | 1001 | 10.4 µs | **Go ×1.1** |
| `to_list` | 1 M | 7.42 ms | 22.9 MB | 1000001 | 13.26 ms | **Go ×1.8** |
| `is_null` | 1 K | 415 ns | 2.2 KB | 4 | 11.8 µs | **Go ×28.5** |
| `is_null` | 1 M | 45.1 µs | 1.9 MB | 4 | 11.8 µs | Py ×3.8 |
| `is_not_null` | 1 K | 409 ns | 2.2 KB | 4 | 11.5 µs | **Go ×28.1** |
| `is_not_null` | 1 M | 68.7 µs | 1.9 MB | 4 | 13.6 µs | Py ×5.0 |
| `fill_nan` | 1 K | 393 ns | 300 B | 4 | 95.9 µs | **Go ×244.0** |
| `fill_nan` | 1 M | 73.9 µs | 988 B | 29 | 852.7 µs | **Go ×11.5** |

> `null_count` is O(1) (cached validity popcount) — **Go ×237–331** vs Python Polars.  
> `drop_nans` / `fill_nan` use parallel float64 kernels (~988 B/op at 1 M rows, down from tens of MB).  
> `is_null` / `is_not_null` at 1 M rows: Python still leads (~×3.6–3.9) via Rust SIMD bitpath; the gap is much smaller than before.

### LazyFrame

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `LazyFrame.collect` | 1 K | 86 ns | 192 B | 2 | 4.0 µs | **Go ×47.0** |
| `LazyFrame.collect` | 1 M | 69 ns | 192 B | 2 | 4.8 µs | **Go ×70.1** |
| `LazyFrame.inspect` | 1 K | 31 ns | 112 B | 1 | 1.2 µs | **Go ×38.3** |
| `LazyFrame.inspect` | 1 M | 20 ns | 112 B | 1 | 1.2 µs | **Go ×57.9** |

> `LazyFrame.collect`/`inspect` on a no-op plan are near-zero-cost (pointer return + 192/112 B fixed metadata); Python Polars pays a Rust→Python dispatch + GIL overhead on every call regardless of plan complexity.

### Filter + Sum pipeline — memory and performance

> Three execution paths: **eager** (filter → materialize → sum), **lazy fused** (single masked scan, filter+reduce plan), **eager-direct** (`FilterAggregateDirect`: fused single pass, no plan, no materialized frame).  
> **Go B/op** = heap bytes per operation. **Py peak RSS** = process resident-set growth for the operation.

#### 0% selectivity — no rows pass (predicate `col("a") > 50`)

| engine | size | Go time | Go B/op | Go allocs | Py time | speedup | Py peak RSS |
|--------|------|---------|---------|-----------|---------|---------|-------------|
| eager | 1 K | 1.5 µs | 1.5 KB | 13 | 230.5 µs | **Go ×151.1** | 8.2 MB |
| lazy fused | 1 K | 2.3 µs | 5.6 KB | 31 | 88.8 µs | **Go ×38.5** | 8.8 MB |
| eager-direct | 1 K | 1.1 µs | 720 B | 7 | 124.6 µs | **Go ×112.7** | 8.3 MB |
| eager | 10 K | 9.8 µs | 2.6 KB | 13 | 63.0 µs | **Go ×6.4** | 8.2 MB |
| lazy fused | 10 K | 9.3 µs | 5.6 KB | 31 | 82.1 µs | **Go ×8.8** | 8.9 MB |
| eager-direct | 10 K | 8.0 µs | 720 B | 7 | 88.8 µs | **Go ×11.1** | 8.3 MB |
| eager | 100 K | 48.7 µs | 32.4 KB | 64 | 79.2 µs | **Go ×1.6** | 8.9 MB |
| lazy fused | 100 K | 38.4 µs | 8.2 KB | 58 | 71.1 µs | **Go ×1.9** | 9.6 MB |
| eager-direct | 100 K | 35.3 µs | 3.3 KB | 34 | 96.5 µs | **Go ×2.7** | 9.0 MB |
| eager | 1 M | 218.8 µs | 261.2 KB | 64 | 287.2 µs | **Go ×1.3** | 16.0 MB |
| lazy fused | 1 M | 173.4 µs | 8.2 KB | 58 | 212.4 µs | **Go ×1.2** | 16.6 MB |
| eager-direct | 1 M | 172.1 µs | 3.3 KB | 34 | 203.6 µs | **Go ×1.2** | 16.0 MB |
| eager | 10 M | 1.60 ms | 2.4 MB | 64 | 1.25 ms | Py ×1.3 | 85.7 MB |
| lazy fused | 10 M | 1.36 ms | 8.2 KB | 58 | 1.25 ms | Py ×1.1 | 86.3 MB |
| eager-direct | 10 M | 1.30 ms | 3.3 KB | 34 | 1.28 ms | Py ×1.0 | 85.8 MB |

#### 50% selectivity — half rows pass (predicate `col("a") > 0`)

| engine | size | Go time | Go B/op | Go allocs | Py time | speedup | Py peak RSS |
|--------|------|---------|---------|-----------|---------|---------|-------------|
| eager | 1 K | 3.0 µs | 15.7 KB | 16 | 112.2 µs | **Go ×37.3** | 8.3 MB |
| lazy fused | 1 K | 2.6 µs | 5.6 KB | 32 | 97.7 µs | **Go ×37.7** | 8.9 MB |
| eager-direct | 1 K | 1.1 µs | 720 B | 7 | 84.4 µs | **Go ×74.3** | 8.3 MB |
| eager | 10 K | 30.0 µs | 122.6 KB | 16 | 73.9 µs | **Go ×2.5** | 8.4 MB |
| lazy fused | 10 K | 9.5 µs | 5.6 KB | 32 | 101.2 µs | **Go ×10.6** | 9.0 MB |
| eager-direct | 10 K | 8.1 µs | 720 B | 7 | 90.7 µs | **Go ×11.2** | 8.4 MB |
| eager | 100 K | 225.4 µs | 1.2 MB | 67 | 99.8 µs | Py ×2.3 | 9.4 MB |
| lazy fused | 100 K | 40.5 µs | 8.2 KB | 59 | 107.8 µs | **Go ×2.7** | 10.0 MB |
| eager-direct | 100 K | 36.2 µs | 3.3 KB | 34 | 92.1 µs | **Go ×2.5** | 9.4 MB |
| eager | 1 M | 1.33 ms | 11.7 MB | 67 | 399.2 µs | Py ×3.3 | 19.9 MB |
| lazy fused | 1 M | 173.5 µs | 8.2 KB | 59 | 518.9 µs | **Go ×3.0** | 21.5 MB |
| eager-direct | 1 M | 172.1 µs | 3.3 KB | 34 | 394.4 µs | **Go ×2.3** | 19.9 MB |
| eager | 10 M | 11.77 ms | 116.9 MB | 67 | 7.04 ms | Py ×1.7 | 124.1 MB |
| lazy fused | 10 M | 1.34 ms | 8.2 KB | 59 | 6.57 ms | **Go ×4.9** | 125.7 MB |
| eager-direct | 10 M | 1.30 ms | 3.3 KB | 34 | 6.12 ms | **Go ×4.7** | 124.1 MB |

> The **eager path allocates proportionally to the number of rows that survive the filter** (it materializes the surviving rows into a new column). The **lazy/eager-direct paths stay near-constant** (≈3–8 KB at 1 M–10 M rows) because they perform a single masked pass with no materialized intermediate.

### Key takeaways

| area | observation |
|------|-------------|
| **Small datasets (≤ 10 K rows)** | gopolars matches or beats Python Polars on most operations — Python's Rust→Python overhead dominates at small sizes |
| **Large datasets (≥ 1 M rows)** | Python still leads on `sort` (×1.3), `filter` (×1.8) and the validity masks (`is_null`/`is_not_null`); gopolars leads on plan/dispatch, metadata ops, window functions (`over`), rolling windows, `unique`, fill/drop kernels, and fused filter+sum. `rank` and `group_by` are at parity |
| **Metadata ops — O(columns)** | `drop` **Go ×201**, `rename` **Go ×20**, `clone` **Go ×1.7**, `row` **Go ×1.8** at 1 M rows — these share column buffers instead of copying, so their cost does not scale with row count |
| **Window functions** | `over` **Go ×2.1** at 1 M rows — partition ids are built across all cores |
| **group_by (small)** | **Go ×47** at 1 K rows — hash aggregation over typed slice with no boxing |
| **filter (small)** | **Go ×18.9** at 1 K rows — batch predicate mask via `evalbatch` + typed gather |
| **filter+sum fused** | `eager-direct` up to **×112** faster than Python at 1 K (0% selectivity); **×4.7** at 10 M with 50% selectivity |
| **Memory — eager path** | Allocates proportionally to surviving rows: ~11.7 MB/op at 1 M rows with 50% selectivity |
| **Memory — fused paths** | Lazy and eager-direct stay near-constant (≈3.3–8.2 KB at 1 M–10 M rows) regardless of selectivity |
| **Rolling windows** | O(n) linear kernels; gopolars outpaces Python Polars on `rolling_mean/sum/min/max` at 1 M rows (~8.6–8.9 MB/op) |
| **Series null/NaN kernels** | `null_count` is O(1); `drop_nans`/`fill_nan` parallel float64 paths drop allocation from tens of MB to ~1 KB |
| **Metadata ops** | `LazyFrame.collect` / `inspect` are **Go ×37–59×** — near-zero allocation (112–192 B) per call |
| **`select` / `with_columns`** | **Go ×15–101×** — cheap column projection / alias vs Python dispatch overhead |

## Testing

Statement coverage is measured over library code in `./pkg/...` using both package-local tests and `test/unit`:

```bash
go test ./pkg/... ./test/unit/... -coverpkg=./pkg/... -skip V06Performance
```

Current coverage: **75%** (as of the latest local run with the command above).

Codecov tracks `./pkg/...` on CI; see [`docs/codecov.md`](docs/codecov.md) for upload details.

## Quick start

```go
io := polars.NewIO()
df, _ := io.ReadCSV(polars.ReadCSVInput{
    Path: "data.csv",
    HasHeader: true,
    Separator: ',',
})

out, _ := df.
    Lazy().
    Filter(polars.Col("value").Gt(polars.Lit(int64(10)))).
    GroupBy("city").
    Agg(polars.Sum(polars.Col("value"))).
    Collect(context.Background())

_ = out
```

## Examples

- [Parquet scan](examples/parquet_scan/main.go)
- [Lazy pushdown](examples/lazy_pushdown/main.go)
- [Join variants](examples/join_variants/main.go)
- [Streaming collect](examples/streaming_collect/main.go)

## API documentation (local)

Bilingual static reference (Ukrainian / English) with Go syntax highlighting in examples:

```bash
# from repo root — open in browser
open docs/index.html   # macOS
# or: python3 -m http.server 8765 --directory docs
```

Published on push to `main` when `docs/**` changes — see [`.github/workflows/pages.yml`](.github/workflows/pages.yml).
