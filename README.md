# gopolars

[![CI](https://github.com/h0rn3t/gopolars/actions/workflows/ci.yml/badge.svg)](https://github.com/h0rn3t/gopolars/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/h0rn3t/gopolars/graph/badge.svg)](https://codecov.io/gh/h0rn3t/gopolars)
[![Release](https://img.shields.io/github/v/release/h0rn3t/gopolars?sort=semver)](https://github.com/h0rn3t/gopolars/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/h0rn3t/gopolars/pkg/polars.svg)](https://pkg.go.dev/github.com/h0rn3t/gopolars/pkg/polars)
[![Go 1.26+](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)](go.mod)

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
go get github.com/h0rn3t/gopolars@v0.3.0
```

Import the public API package:

```go
import "github.com/h0rn3t/gopolars/pkg/polars"
```

Requires **Go 1.26+**. Float64 min/max reductions use runtime-dispatched AVX2 on capable
`amd64` CPUs (one binary, no build tag); see [Performance / SIMD Acceleration](#performance--simd-acceleration).

## Current status

Latest release: **[v0.3.0](https://github.com/h0rn3t/gopolars/releases/tag/v0.3.0)**
([changelog vs v0.2.0](https://github.com/h0rn3t/gopolars/compare/v0.2.0...v0.3.0)).
The public API is versioned with SemVer; while `< v1.0.0` it may still evolve between minor
versions (see the versioning and migration notes under [`docs/`](docs/)).

The project has driven its internal parity waves up to the **v1.0 tracking matrix** ([`docs/parity/v1_0_coverage.json`](docs/parity/v1_0_coverage.json)) and now covers a broad core for Go-native analytics pipelines, including advanced joins, reshape operations, temporal windows, opt-in DuckDB SQL, and performance diagnostics.\
It is production-usable for many DataFrame workloads, but it is **not yet a full drop-in replacement** for Python Polars.

- ✅ Strong DataFrame/LazyFrame core for real analytics workloads
- ✅ Stable IO surface (CSV/JSON/Parquet/IPC + scans + pushdown)
- ✅ Opt-in SQL over in-memory frames via embedded DuckDB (`-tags duckdb,duckdb_arrow`)
- ✅ **75%** statement coverage for `./pkg/...` (unit + package tests; see [Testing](#testing))
- ✅ **668 / 673** tracked Python Polars methods implemented on the [full parity matrix](#python-polars-vs-gopolars-function-matrix) (5 rows intentionally out of scope: `DataFrame.__setitem__` + four Series non-goals — see matrix notes)

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
| Full Python Polars API parity (673-method tracked matrix)         | ✅ ~99.3% (668/673) |
| Performance parity on all workloads                               | 🚧 in progress |
| Ecosystem parity (all namespaces, plugins, advanced UDF patterns) | 🚧 in progress |

## Python Polars vs gopolars function matrix

**Full-matrix totals (673 tracked Python Polars methods on DataFrame, LazyFrame, Expr, Series):** **668 implemented**, **5** intentionally remaining (`DataFrame.__setitem__` plus four documented Series non-goals). Coverage **≈99.3%**.

### Coverage by object (full matrix)

| Object     | Implemented | Total in matrix |
| ---------- | ----------- | ---------------- |
| DataFrame  | 140         | 141              |
| LazyFrame  | 89          | 89               |
| Expr       | 217         | 217              |
| Series     | 222         | 226              |
| **Total**  | **668**     | **673**          |

### Remaining rows by priority (full matrix)

| Priority among not implemented | Count |
| -------------------------------- | ----- |
| high                             | 0     |
| medium                           | 0     |
| low                              | 5     |

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

Build (AVX2 used automatically on capable amd64 CPUs):

```bash
go build ./...
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
> **Methodology:** Go benchmarks run with `go test -benchmem -count=1 -benchtime=2s`; Python timings measured by the same harness with 10 repetitions per operation. Go time = min ns/op across calibration rounds. Python time = mean across 10 iterations.  
> **Dataset:** generated float64/string/int64 columns with ~5% null rate; sizes 1 K and 1 M rows (filter+sum also 10 K / 100 K / 10 M).  
> **"speedup":** `Go ×N` means gopolars is N× faster; `Py ×N` means Python Polars is N× faster.  
> **Measured for:** [v0.3.0](https://github.com/h0rn3t/gopolars/releases/tag/v0.3.0) (2026-07-11 local re-run).

Regenerate the tables from source:

```bash
# Run top-30 benchmark (requires Python + polars installed)
GOPOLARS_BENCH_PYTHON=1 go test ./bench/top30 -bench=BenchmarkTop30$ \
  -benchmem -count=1 -benchtime=2s -light -timeout=600s \
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
| `filter` | 1 K | 6.0 µs | 27.1 KB | 23 | 94.4 µs | **Go ×15.6** |
| `filter` | 1 M | 1.33 ms | 20.8 MB | 213 | 635.3 µs | Py ×2.1 |
| `select` | 1 K | 601 ns | 1.5 KB | 10 | 42.3 µs | **Go ×70.4** |
| `select` | 1 M | 475 ns | 1.5 KB | 10 | 48.2 µs | **Go ×101.4** |
| `with_columns` | 1 K | 584 ns | 1.5 KB | 10 | 9.0 µs | **Go ×15.5** |
| `with_columns` | 1 M | 470 ns | 1.5 KB | 10 | 8.8 µs | **Go ×18.7** |
| `sort` | 1 K | 23.4 µs | 66.6 KB | 19 | 173.4 µs | **Go ×7.4** |
| `sort` | 1 M | 17.42 ms | 62.0 MB | 179 | 14.03 ms | Py ×1.2 |
| `group_by` | 1 K | 14.9 µs | 18.6 KB | 41 | 683.1 µs | **Go ×46.0** |
| `group_by` | 1 M | 1.83 ms | 90.7 KB | 186 | 1.58 ms | Py ×1.2 |
| `join` | 1 K | 91.6 µs | 225.4 KB | 727 | 376.5 µs | **Go ×4.1** |
| `join` | 1 M | 6.84 ms | 101.4 MB | 1 164 | 6.30 ms | Py ×1.1 |
| `head` | 1 K | 464 ns | 1.6 KB | 10 | 633 ns | **Go ×1.4** |
| `head` | 1 M | 329 ns | 1.6 KB | 10 | 933 ns | **Go ×2.8** |
| `tail` | 1 K | 474 ns | 1.3 KB | 9 | 591 ns | **Go ×1.2** |
| `tail` | 1 M | 343 ns | 1.3 KB | 9 | 629 ns | **Go ×1.8** |
| `unique` | 1 K | 8.6 µs | 9.9 KB | 21 | 193.4 µs | **Go ×22.4** |
| `unique` | 1 M | 1.03 ms | 1.6 MB | 480 | 5.16 ms | **Go ×5.0** |
| `fill_null` | 1 K | 2.2 µs | 10.0 KB | 10 | 146.7 µs | **Go ×65.4** |
| `fill_null` | 1 M | 425.3 µs | 8.6 MB | 35 | 1.29 ms | **Go ×3.0** |
| `drop_nulls` | 1 K | 10.2 µs | 50.6 KB | 17 | 71.8 µs | **Go ×7.1** |
| `drop_nulls` | 1 M | 1.65 ms | 37.2 MB | 66 | 1.34 ms | Py ×1.2 |

### Expr operations

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `cum_sum` | 1 K | 2.6 µs | 10.7 KB | 14 | 13.2 µs | **Go ×5.1** |
| `cum_sum` | 1 M | 782.0 µs | 8.6 MB | 14 | 3.18 ms | **Go ×4.1** |
| `rank` | 1 K | 17.2 µs | 34.7 KB | 17 | 49.2 µs | **Go ×2.9** |
| `rank` | 1 M | 20.71 ms | 31.5 MB | 17 | 14.85 ms | Py ×1.4 |
| `over` (window) | 1 K | 16.0 µs | 28.2 KB | 26 | 289.6 µs | **Go ×18.1** |
| `over` (window) | 1 M | 18.21 ms | 24.8 MB | 26 | 10.86 ms | Py ×1.7 |
| `fill_null` | 1 K | 3.0 µs | 10.9 KB | 16 | 39.0 µs | **Go ×13.1** |
| `fill_null` | 1 M | 388.4 µs | 8.6 MB | 41 | 759.3 µs | **Go ×2.0** |
| `fill_nan` | 1 K | 1.2 µs | 1.9 KB | 15 | 79.4 µs | **Go ×66.4** |
| `fill_nan` | 1 M | 75.7 µs | 2.6 KB | 40 | 894.3 µs | **Go ×11.8** |
| `rolling_mean` | 1 K | 6.5 µs | 10.7 KB | 16 | 18.5 µs | **Go ×2.8** |
| `rolling_mean` | 1 M | 4.77 ms | 8.6 MB | 16 | 9.23 ms | **Go ×1.9** |
| `rolling_sum` | 1 K | 6.7 µs | 10.7 KB | 16 | 18.1 µs | **Go ×2.7** |
| `rolling_sum` | 1 M | 4.74 ms | 8.6 MB | 16 | 9.09 ms | **Go ×1.9** |
| `rolling_min` | 1 K | 4.8 µs | 11.5 KB | 17 | 15.5 µs | **Go ×3.2** |
| `rolling_min` | 1 M | 8.17 ms | 8.9 MB | 28 | 12.24 ms | **Go ×1.5** |
| `rolling_max` | 1 K | 5.1 µs | 11.5 KB | 17 | 15.3 µs | **Go ×3.0** |
| `rolling_max` | 1 M | 8.06 ms | 8.9 MB | 28 | 12.26 ms | **Go ×1.5** |

> Rolling window operations (`rolling_mean/sum/min/max`) use O(n) linear kernels — a running Neumaier-compensated accumulator for sum/mean and a monotonic-index deque for min/max — writing a single output buffer (≈9 MB/op at 1 M rows). gopolars outpaces Python Polars on all four at 1 M rows.

### Series operations

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `null_count` | 1 K | 2 ns | 0 B | 0 | 475 ns | **Go ×237.5** |
| `null_count` | 1 M | 2 ns | 0 B | 0 | 663 ns | **Go ×331.3** |
| `drop_nans` | 1 K | 410 ns | 300 B | 4 | 12.4 µs | **Go ×30.2** |
| `drop_nans` | 1 M | 85.5 µs | 988 B | 29 | 100.8 µs | **Go ×1.2** |
| `to_list` | 1 K | 9.5 µs | 23.8 KB | 1 001 | 9.8 µs | **Go ×1.0** |
| `to_list` | 1 M | 7.36 ms | 22.9 MB | 1 000 001 | 14.14 ms | **Go ×1.9** |
| `is_null` | 1 K | 395 ns | 2.2 KB | 4 | 11.5 µs | **Go ×29.2** |
| `is_null` | 1 M | 42.0 µs | 1.9 MB | 4 | 11.8 µs | Py ×3.6 |
| `is_not_null` | 1 K | 391 ns | 2.2 KB | 4 | 11.7 µs | **Go ×30.0** |
| `is_not_null` | 1 M | 53.9 µs | 1.9 MB | 4 | 13.8 µs | Py ×3.9 |
| `fill_nan` | 1 K | 396 ns | 300 B | 4 | 84.3 µs | **Go ×213.0** |
| `fill_nan` | 1 M | 74.3 µs | 988 B | 29 | 793.3 µs | **Go ×10.7** |

> `null_count` is O(1) (cached validity popcount) — **Go ×237–331** vs Python Polars.  
> `drop_nans` / `fill_nan` use parallel float64 kernels (~988 B/op at 1 M rows, down from tens of MB).  
> `is_null` / `is_not_null` at 1 M rows: Python still leads (~×3.6–3.9) via Rust SIMD bitpath; the gap is much smaller than before.

### LazyFrame

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `LazyFrame.collect` | 1 K | 83 ns | 192 B | 2 | 4.1 µs | **Go ×49.2** |
| `LazyFrame.collect` | 1 M | 70 ns | 192 B | 2 | 4.1 µs | **Go ×58.9** |
| `LazyFrame.inspect` | 1 K | 31 ns | 112 B | 1 | 1.1 µs | **Go ×37.0** |
| `LazyFrame.inspect` | 1 M | 21 ns | 112 B | 1 | 1.1 µs | **Go ×53.0** |

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
| **Large datasets (≥ 1 M rows)** | Python still leads on some compute-heavy ops (`sort`, `rank`, `unique`, `is_null`); gopolars leads on plan/dispatch, rolling windows, fill/drop kernels, and fused filter+sum |
| **group_by (small)** | **Go ×46** at 1 K rows — hash aggregation over typed slice with no boxing |
| **filter (small)** | **Go ×15.6** at 1 K rows — batch predicate mask via `evalbatch` + typed gather |
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
