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
go get github.com/h0rn3t/gopolars@v0.1.0
```

Import the public API package:

```go
import "github.com/h0rn3t/gopolars/pkg/polars"
```

Requires **Go 1.26+**. SIMD acceleration is optional and opt-in
(`GOEXPERIMENT=simd` on `amd64`); see [Performance / SIMD Acceleration](#performance--simd-acceleration).

## Current status

First tagged release: **[v0.1.0](https://github.com/h0rn3t/gopolars/releases/tag/v0.1.0)**. The public
API is versioned with SemVer; while `< v1.0.0` it may still evolve between minor versions (see the
versioning and migration notes under [`docs/`](docs/)).

The project has completed internal parity waves through **v0.6** and now covers a broad core for Go-native analytics pipelines, including advanced joins, reshape operations, temporal windows, and performance diagnostics.\
It is production-usable for many DataFrame workloads, but it is **not yet a full drop-in replacement** for Python Polars.

- ✅ Strong DataFrame/LazyFrame core for real analytics workloads
- ✅ Stable IO surface (CSV/JSON/Parquet/IPC + scans + pushdown)
- ✅ **75%** statement coverage for `./pkg/...` (unit + package tests; see [Testing](#testing))
- ✅ **675 / 680** tracked Python Polars methods implemented on the [full parity matrix](docs/parity/python_polars_full_matrix.md) (5 rows intentionally out of scope: `DataFrame.__setitem__` + four Series non-goals — see matrix notes)

## Implemented capabilities

### DataFrame, Series and Expressions

- Eager and lazy execution over a columnar in-memory DataFrame engine
- Core DataFrame operations: `select`, `filter`, `with_columns`, `sort`, `limit`, `group_by`, `join`
- Extended DataFrame surface: `slice`, `head`, `tail`, `unique`, `concat`, `fill_null`, `drop_nulls`, `drop`, `rename`
- Eager fused fast path: `FilterAggregateDirect` computes `filter().sum/min/max/mean/count` in a single masked pass without building a lazy plan or materializing a filtered DataFrame
- Join modes: `inner`, `left`, `right`, `full`, `semi`, `anti`, `cross`, `asof`
- Temporal analytics: `group_by_dynamic`, `rolling_mean`
- Reshape support: `melt`, `pivot`
- Public `Series` API with null-aware operations, vector arithmetic and comparisons
- Expression namespaces for string/datetime/list/struct workflows:
  - `list_len`
  - `list_contains`
  - `list_get`
  - `struct_field`
  - `str_lower`, `str_upper`, `str_len`, `str_replace`, `str_trim`, `starts_with`, `contains`
  - `dt_year`, `dt_month`, `dt_day`, `dt_hour`, `dt_weekday`
  - `explode`
  - `flatten` (struct flattening)

### SQL and Query Planning

- SQL parsing and planning for `SELECT` pipelines
- Session DDL via `SQLContext`: `CREATE TABLE <name> AS <select>`, `DROP TABLE [IF EXISTS]`, `TRUNCATE TABLE`, `SHOW TABLES`, `EXPLAIN <select>`
- Table functions in `FROM`/`JOIN`: `read_csv('path')`, `read_parquet('path')`, `read_json('path')`, `read_ipc('path')` (with aliases, usable in CTEs and subqueries)
- Joins: `INNER`, `LEFT`, `RIGHT`, `FULL [OUTER]`, `CROSS` (and comma cross joins), with table aliases and qualified columns
- Boolean predicate logic (`AND`/`OR`/`NOT`) with correct precedence in `WHERE`/`HAVING`/`ON`
- `CASE WHEN … THEN … ELSE … END`, `IS [NOT] NULL`, `IN`/`BETWEEN`/`LIKE`, `CAST(x AS t)` / `x::t`
- Scalar functions:
  - string: `UPPER`, `LOWER`, `LENGTH`/`CHAR_LENGTH`, `OCTET_LENGTH`, `BIT_LENGTH`, `SUBSTR`, `LEFT`, `RIGHT`, `TRIM`/`LTRIM`/`RTRIM`, `CONCAT`, `CONCAT_WS`, `REPLACE`, `REVERSE`, `INITCAP`, `LPAD`/`RPAD`, `SPLIT_PART`, `STRPOS`/`POSITION`, `STARTS_WITH`/`ENDS_WITH`, `REGEXP_LIKE`
  - math: `ABS`, `ROUND`, `CEIL`, `FLOOR`, `POWER`, `SQRT`, `CBRT`, `MOD`, `EXP`, `LN`, `LOG` (1/2-arg), `LOG2`, `LOG10`, `LOG1P`, `SIGN`, `PI()`, `DEGREES`/`RADIANS`, `SIN`/`COS`/`TAN`/`COT`, `ASIN`/`ACOS`/`ATAN`/`ATAN2`
  - temporal: `YEAR`/`MONTH`/`DAY`/`HOUR`/`MINUTE`/`SECOND`, `DATE_PART('part', d)` / `EXTRACT(part FROM d)`, `DAYOFWEEK`, `DAYOFYEAR`/`ORDINAL_DAY`
  - conditional: `COALESCE`, `NULLIF`, `IFNULL`, `IF(cond, a, b)`, `GREATEST`/`LEAST`
- Aggregates: `SUM`/`MIN`/`MAX`/`AVG`/`COUNT`/`COUNT(DISTINCT col)`/`N_UNIQUE`/`STDDEV`/`VARIANCE`/`MEDIAN`/`FIRST`/`LAST`, with `GROUP BY`/`HAVING`, `ORDER BY`, `SELECT DISTINCT`, `LIMIT`/`OFFSET`
- Window functions: `SUM`/`MEAN`/`MIN`/`MAX`/`COUNT`, `ROW_NUMBER`, `RANK`/`DENSE_RANK`, `LAG`/`LEAD` (offset + default), `FIRST_VALUE`/`LAST_VALUE` with `PARTITION BY` and `ORDER BY`
- CTE support (`WITH ... AS (...)`)
- Subqueries in `FROM`
- Set operations: `UNION`, `INTERSECT`, `EXCEPT`
- Logical optimization passes including pushdown and adaptive planning rules
- Out of scope (clear error): DML (`INSERT`/`UPDATE`/`DELETE`), `ALTER TABLE`, correlated subqueries — matching Polars SQL

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
| SQL base + CTE + window expressions                               | ✅ ready        |
| SQL joins, boolean/CASE/IN/BETWEEN/LIKE, scalar fns, DISTINCT/OFFSET, CAST | ✅ ready |
| GroupBy, temporal windows and joins                               | ✅ ready        |
| Streaming collect                                                 | ✅ ready        |
| CSV/JSON/Parquet/IPC IO                                           | ✅ ready        |
| Arrow interoperability                                            | ✅ ready        |
| Cloud-style partitioned dataset scans                             | ✅ ready        |
| Explain/telemetry schema v2 and perf markers                      | ✅ ready        |
| Full Python Polars API parity (680-method tracked matrix)         | ✅ ~99.3% (675/680) |
| Full SQL surface: DDL, SQL table functions, complete fn catalog   | 🚧 in progress |
| Performance parity on all workloads                               | 🚧 in progress |
| Ecosystem parity (all namespaces, plugins, advanced UDF patterns) | 🚧 in progress |

## Python Polars vs gopolars function matrix

**Full-matrix totals (680 tracked Python Polars methods on DataFrame, LazyFrame, Expr, Series, SQLContext):** **675 implemented**, **5** intentionally remaining (`DataFrame.__setitem__` plus four documented Series non-goals). Coverage **≈99.3%**.

### Coverage by object (full matrix)

| Object     | Implemented | Total in matrix |
| ---------- | ----------- | ---------------- |
| DataFrame  | 140         | 141              |
| LazyFrame  | 89          | 89               |
| Expr       | 217         | 217              |
| Series     | 222         | 226              |
| SQLContext | 7           | 7                |
| **Total**  | **675**     | **680**          |

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
2. **Deeper SQL parity**
   - Wider SQL surface and compatibility with advanced analytical query patterns
   - Stronger parity guarantees across planner and execution semantics
3. **Performance and scale hardening**
   - More optimization rules and workload-adaptive planning
   - Larger benchmark corpus and stricter regression budgets
4. **Cloud and lakehouse robustness**
   - Expanded object-store behavior and dataset semantics at scale
   - More integration coverage for partitioned and heterogeneous datasets
5. **Compatibility and migration experience**
   - Continued stabilization of deprecation/migration workflows
   - Clear release evidence for every potentially breaking alignment change

## Roadmap focus

- **Near term:** expand SQL/catalog surface, align remaining semantic edge cases on the covered API surface, and harden performance budgets on larger datasets.
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

`gopolars` can optionally use SIMD-accelerated numeric kernels on AMD64 when
built with **Go 1.26+** and the experimental `simd` flag.

Supported SIMD-accelerated operations (in `pkg/simd`):

- `SumFloat64`, `MinFloat64`, `MaxFloat64`, `MinMaxFloat64` on `[]float64`
- Element-wise `AddSlicesFloat64`, `MulSlicesFloat64`
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

**SIMD requires**: Go 1.26+, `amd64` target, `GOEXPERIMENT=simd`.

Build with SIMD acceleration:

```bash
GOEXPERIMENT=simd go build ./...
```

Build without (fully functional scalar fallback on all platforms):

```bash
go build ./...
```

On non-AMD64 architectures (e.g., ARM64) or without `GOEXPERIMENT=simd`, the
library automatically falls back to scalar loops with **identical results**.
Correctness never depends on SIMD being enabled — the CI pipeline runs without
`GOEXPERIMENT=simd` by default.

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

> **Hardware:** Apple M4 Pro · Go 1.26 · Python Polars 1.41.2 · macOS arm64  
> **Methodology:** Go benchmarks run with `go test -benchmem -count=1 -benchtime=2s`; Python timings measured by the same harness with 10 repetitions per operation. Go time = min ns/op across calibration rounds. Python time = mean across 10 iterations.  
> **Dataset:** generated float64/string/int64 columns with ~5% null rate; sizes 1 K and 1 M rows.  
> **"speedup":** `Go ×N` means gopolars is N× faster; `Py ×N` means Python Polars is N× faster.

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

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `filter` | 1 K | 9.3 µs | 28.3 KB | 25 | 98.1 µs | **Go ×10.5** |
| `filter` | 1 M | 5.81 ms | 25.1 MB | 112 | 564 µs | Py ×10.3 |
| `select` | 1 K | 279 ns | 864 B | 7 | 43.9 µs | **Go ×157** |
| `select` | 1 M | 221 ns | 864 B | 7 | 35.9 µs | **Go ×163** |
| `with_columns` | 1 K | 403 ns | 1.1 KB | 8 | 8.5 µs | **Go ×21** |
| `with_columns` | 1 M | 416 ns | 1.1 KB | 8 | 8.0 µs | **Go ×25.5** |
| `sort` | 1 K | 29.2 µs | 71.2 KB | 22 | 170 µs | **Go ×5.8** |
| `sort` | 1 M | 35.9 ms | 68.1 MB | 78 | 12.7 ms | Py ×2.8 |
| `group_by` | 1 K | 14.6 µs | 18.9 KB | 42 | 691 µs | **Go ×47** |
| `group_by` | 1 M | 2.08 ms | 92.8 KB | 187 | 1.50 ms | Py ×1.4 |
| `join` | 1 K | 148 µs | 311 KB | 1 385 | 349 µs | **Go ×2.4** |
| `join` | 1 M | 47.4 ms | 122 MB | 2 120 | 6.40 ms | Py ×7.4 |
| `head` | 1 K | 2.1 µs | 7.2 KB | 19 | 620 ns | Py ×3.4 |
| `head` | 1 M | 1.6 µs | 7.2 KB | 19 | 612 ns | Py ×2.5 |
| `tail` | 1 K | 2.2 µs | 7.4 KB | 21 | 604 ns | Py ×3.7 |
| `tail` | 1 M | 1.6 µs | 7.4 KB | 21 | 629 ns | Py ×2.5 |
| `unique` | 1 K | 9.6 µs | 9.9 KB | 24 | 161 µs | **Go ×16.8** |
| `unique` | 1 M | 16.4 ms | 7.6 MB | 24 | 4.92 ms | Py ×3.3 |
| `fill_null` | 1 K | 1.9 µs | 9.9 KB | 9 | 133 µs | **Go ×70** |
| `fill_null` | 1 M | 1.16 ms | 8.6 MB | 9 | 871 µs | Py ×1.3 |
| `drop_nulls` | 1 K | 48.1 µs | 53.6 KB | 20 | 114 µs | **Go ×2.4** |
| `drop_nulls` | 1 M | 43.9 ms | 49.6 MB | 20 | 1.36 ms | Py ×32.4 |

### Expr operations

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `cum_sum` | 1 K | 1.9 µs | 10.0 KB | 10 | 13.1 µs | **Go ×6.8** |
| `cum_sum` | 1 M | 768 µs | 8.6 MB | 10 | 2.87 ms | **Go ×3.7** |
| `rank` | 1 K | 17.3 µs | 34.8 KB | 13 | 56.4 µs | **Go ×3.3** |
| `rank` | 1 M | 22.7 ms | 33.0 MB | 13 | 16.2 ms | Py ×1.4 |
| `over` (window) | 1 K | 13.4 µs | 27.5 KB | 22 | 224 µs | **Go ×16.7** |
| `over` (window) | 1 M | 18.1 ms | 24.8 MB | 22 | 9.10 ms | Py ×2.0 |
| `fill_null` | 1 K | 2.1 µs | 10.1 KB | 11 | 53.3 µs | **Go ×25** |
| `fill_null` | 1 M | 1.14 ms | 8.6 MB | 11 | 659 µs | Py ×1.7 |
| `fill_nan` | 1 K | 2.2 µs | 10.1 KB | 11 | 99.5 µs | **Go ×46** |
| `fill_nan` | 1 M | 1.22 ms | 8.6 MB | 11 | 798 µs | Py ×1.5 |
| `rolling_mean` | 1 K | 5.9 µs | 10.0 KB | 12 | 17.7 µs | **Go ×3.0** |
| `rolling_mean` | 1 M | 4.97 ms | 8.6 MB | 12 | 8.89 ms | **Go ×1.8** |
| `rolling_sum` | 1 K | 5.8 µs | 10.0 KB | 12 | 19.0 µs | **Go ×3.3** |
| `rolling_sum` | 1 M | 5.13 ms | 8.6 MB | 12 | 8.80 ms | **Go ×1.7** |
| `rolling_min` | 1 K | 4.3 µs | 10.8 KB | 13 | 15.9 µs | **Go ×3.7** |
| `rolling_min` | 1 M | 8.05 ms | 8.9 MB | 24 | 11.9 ms | **Go ×1.5** |
| `rolling_max` | 1 K | 4.5 µs | 10.8 KB | 13 | 14.8 µs | **Go ×3.3** |
| `rolling_max` | 1 M | 8.04 ms | 8.9 MB | 24 | 12.0 ms | **Go ×1.5** |

> Rolling window operations (`rolling_mean/sum/min/max`) use O(n) linear kernels — a running Neumaier-compensated accumulator for sum/mean and a monotonic-index deque for min/max — writing a single output buffer (≈9 MB/op at 1 M rows, down from ~895 MB). gopolars now outpaces Python Polars on all four at 1 M rows.

### Series operations

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `null_count` | 1 K | 511 ns | 0 B | 0 | 449 ns | Py ×1.1 |
| `null_count` | 1 M | 509 µs | 0 B | 0 | 662 ns | Py ×768 |
| `drop_nans` | 1 K | 12.3 µs | 33.0 KB | 1 005 | 12.3 µs | **Go ×1.0** |
| `drop_nans` | 1 M | 8.74 ms | 31.5 MB | 1 000 005 | 99.5 µs | Py ×84 |
| `to_list` | 1 K | 9.7 µs | 23.8 KB | 1 001 | 10.4 µs | **Go ×1.1** |
| `to_list` | 1 M | 7.65 ms | 22.9 MB | 1 000 001 | 14.3 ms | **Go ×1.9** |
| `is_null` | 1 K | 3.5 µs | 18.2 KB | 5 | 12.2 µs | **Go ×3.5** |
| `is_null` | 1 M | 1.70 ms | 17.2 MB | 5 | 10.5 µs | Py ×162 |
| `is_not_null` | 1 K | 3.7 µs | 18.2 KB | 5 | 10.3 µs | **Go ×2.8** |
| `is_not_null` | 1 M | 1.91 ms | 17.2 MB | 5 | 11.7 µs | Py ×163 |
| `fill_nan` | 1 K | 12.2 µs | 33.0 KB | 1 005 | 77.3 µs | **Go ×6.4** |
| `fill_nan` | 1 M | 8.50 ms | 31.5 MB | 1 000 005 | 945 µs | Py ×8.9 |

> `null_count` at 1 M rows: Python Polars uses an O(1) cached counter; gopolars scans the validity mask on every call — a single-line fix tracked as a near-term optimization.  
> `is_null`/`is_not_null` at 1 M rows: Python delegates to a Rust SIMD bitcount; gopolars copies and inverts the validity mask — also tracked.

### LazyFrame & SQLContext

| operation | size | Go time | Go B/op | allocs/op | Py time | speedup |
|-----------|------|---------|---------|-----------|---------|---------|
| `LazyFrame.collect` | 1 K | 124 ns | 304 B | 4 | 3.3 µs | **Go ×27** |
| `LazyFrame.collect` | 1 M | 101 ns | 304 B | 4 | 4.2 µs | **Go ×42** |
| `LazyFrame.sql` | 1 K | 31.2 µs | 50.2 KB | 82 | 11.2 µs | Py ×2.8 |
| `LazyFrame.sql` | 1 M | 7.93 ms | 37.5 MB | 169 | 11.5 µs | Py ×675 |
| `LazyFrame.inspect` | 1 K | 28 ns | 96 B | 1 | 1.1 µs | **Go ×40** |
| `LazyFrame.inspect` | 1 M | 20 ns | 96 B | 1 | 24.9 µs | **Go ×1 245** |
| `SQLContext.execute` | 1 K | 1.7 µs | 3.7 KB | 31 | 6.2 µs | **Go ×3.6** |
| `SQLContext.execute` | 1 M | 1.4 µs | 3.7 KB | 31 | 6.7 µs | **Go ×4.8** |
| `SQLContext.register` | 1 K | 184 ns | 800 B | 3 | 2.2 µs | **Go ×12.9** |
| `SQLContext.register` | 1 M | 160 ns | 800 B | 3 | 2.9 µs | **Go ×17.9** |
| `SQLContext.tables` | 1 K | 251 ns | 816 B | 4 | 1.9 µs | **Go ×7.6** |
| `SQLContext.tables` | 1 M | 274 ns | 816 B | 4 | 2.1 µs | **Go ×7.6** |

> `LazyFrame.collect`/`inspect` on a no-op plan are near-zero-cost (pointer return + 304/96 B fixed metadata); Python Polars pays a Rust→Python dispatch + GIL overhead on every call regardless of plan complexity.

### Filter + Sum pipeline — memory and performance

> Three execution paths: **eager** (filter → materialize → sum), **lazy fused** (single masked scan, filter+reduce plan), **eager-direct** (`FilterAggregateDirect`: fused single pass, no plan, no materialized frame).  
> **Go B/op** = heap bytes per operation. **Py peak RSS** = process resident-set growth for the operation.

#### 0% selectivity — no rows pass (predicate `col("a") > 50`)

| engine | size | Go time | Go B/op | Go allocs | Py time | speedup | Py peak RSS |
|--------|------|---------|---------|-----------|---------|---------|-------------|
| eager | 1 K | 86.6 µs | 1.2 KB | 11 | 87.5 µs | **Go ×1.0** | 8.2 MB |
| lazy fused | 1 K | 201 µs | 6.8 KB | 42 | 95.3 µs | Py ×2.1 | 8.8 MB |
| eager-direct | 1 K | 7.2 µs | 816 B | 8 | 76.9 µs | **Go ×10.7** | 8.4 MB |
| eager | 10 K | 38.0 µs | 2.3 KB | 11 | 70.6 µs | **Go ×1.9** | 8.2 MB |
| lazy fused | 10 K | 42.0 µs | 7.3 KB | 40 | 93.4 µs | **Go ×2.2** | 9.0 MB |
| eager-direct | 10 K | 42.0 µs | 1.9 KB | 8 | 67.2 µs | **Go ×1.6** | 8.3 MB |
| eager | 100 K | 264 µs | 37.1 KB | 73 | 119 µs | Py ×2.2 | 8.9 MB |
| lazy fused | 100 K | 131 µs | 29.9 KB | 115 | 112 µs | Py ×1.2 | 9.6 MB |
| eager-direct | 100 K | 94.4 µs | 22.4 KB | 77 | 101 µs | **Go ×1.1** | 9.2 MB |
| eager | 1 M | 509 µs | 267 KB | 73 | 218 µs | Py ×2.3 | 16.0 MB |
| lazy fused | 1 M | 339 µs | 143 KB | 113 | 213 µs | Py ×1.6 | 16.6 MB |
| eager-direct | 1 M | 245 µs | 134 KB | 73 | 237 µs | **Go ×1.0** | 16.0 MB |
| eager | 10 M | 3.17 ms | 2.4 MB | 72 | 1.29 ms | Py ×2.5 | 85.8 MB |
| lazy fused | 10 M | 1.87 ms | 1.2 MB | 117 | 1.31 ms | Py ×1.4 | 86.3 MB |
| eager-direct | 10 M | 1.61 ms | 1.2 MB | 80 | 1.33 ms | Py ×1.2 | 85.7 MB |

#### 50% selectivity — half rows pass (predicate `col("a") > 0`)

| engine | size | Go time | Go B/op | Go allocs | Py time | speedup | Py peak RSS |
|--------|------|---------|---------|-----------|---------|---------|-------------|
| eager | 1 K | 26.4 µs | 16.0 KB | 15 | 92.9 µs | **Go ×3.5** | 8.3 MB |
| lazy fused | 1 K | 39.7 µs | 6.3 KB | 42 | 94.4 µs | **Go ×2.4** | 9.0 MB |
| eager-direct | 1 K | 13.4 µs | 816 B | 8 | 71.2 µs | **Go ×5.3** | 8.3 MB |
| eager | 10 K | 88.4 µs | 127.6 KB | 15 | 71.4 µs | Py ×1.2 | 8.4 MB |
| lazy fused | 10 K | 86.0 µs | 7.3 KB | 41 | 102 µs | **Go ×1.2** | 9.0 MB |
| eager-direct | 10 K | 72.3 µs | 1.9 KB | 8 | 75.1 µs | **Go ×1.0** | 8.5 MB |
| eager | 100 K | 471 µs | 1.2 MB | 74 | 101 µs | Py ×4.6 | 9.5 MB |
| lazy fused | 100 K | 193 µs | 24.5 KB | 103 | 84.1 µs | Py ×2.3 | 10.1 MB |
| eager-direct | 100 K | 179 µs | 22.9 KB | 78 | 89.0 µs | Py ×2.0 | 9.5 MB |
| eager | 1 M | 3.75 ms | 12.2 MB | 77 | 391 µs | Py ×9.6 | 19.9 MB |
| lazy fused | 1 M | 913 µs | 141 KB | 110 | 602 µs | Py ×1.5 | 25.4 MB |
| eager-direct | 1 M | 662 µs | 133 KB | 70 | 410 µs | Py ×1.6 | 19.9 MB |
| eager | 10 M | 32.3 ms | 121.6 MB | 77 | 8.18 ms | Py ×3.9 | 124.1 MB |
| lazy fused | 10 M | 6.09 ms | 1.2 MB | 114 | 6.78 ms | **Go ×1.1** | 125.7 MB |
| eager-direct | 10 M | 7.94 ms | 1.2 MB | 72 | 9.08 ms | **Go ×1.1** | 124.1 MB |

> The **eager path allocates proportionally to the number of rows that survive the filter** (it materializes the surviving rows into a new column). The **lazy/eager-direct paths maintain near-constant allocation relative to dataset size** because they perform a single masked pass with no materialized intermediate.

### Key takeaways

| area | observation |
|------|-------------|
| **Small datasets (≤ 10 K rows)** | gopolars matches or beats Python Polars on most operations — Python's Rust→Python overhead dominates at small sizes |
| **Large datasets (≥ 1 M rows)** | Python Polars is 5–500× faster on compute-heavy operations (`sort`, `rank`, `rolling_*`, `is_null` at scale) that use Rust SIMD/parallelism internally; gopolars is faster on plan-and-dispatch operations (`collect`, `inspect`, `SQLContext.*`) |
| **group_by (small)** | **Go ×12.7** at 1 K rows — hash aggregation over typed slice with no boxing |
| **filter (small)** | **Go ×8.9** at 1 K rows — batch predicate mask via `evalbatch` + typed gather, no per-row boxing |
| **filter+sum fused** | `eager-direct` up to **×10.7** faster than Python Polars at 1 K rows; competes at 10 M rows |
| **Memory — eager path** | Allocates proportionally to surviving rows: 12 MB/op at 1 M rows with 50% selectivity |
| **Memory — fused paths** | Lazy and eager-direct stay near-constant regardless of selectivity (bitmap + single-pass reduce, ~133–141 KB at 1 M rows) |
| **Rolling windows** | gopolars uses O(n·w) scalar loops allocating ~895 MB at 1 M rows with window=100; Python Polars uses Rust SIMD O(n) — 10–42× gap; SIMD sliding-window kernels are on the roadmap |
| **Metadata ops** | `LazyFrame.collect` (no-op plan), `inspect`, `SQLContext.register/tables` are **Go ×8–1 245×** — near-zero allocation (96–816 B) per call regardless of DataFrame size |
| **`select` / `with_columns` at 1 M rows** | Python Polars is 53–440× faster — it performs a zero-copy column re-reference; gopolars currently re-evaluates the expression plan and copies column metadata |

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
