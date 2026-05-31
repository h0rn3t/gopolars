# gopolars

[![CI](https://github.com/h0rn3t/gopolars/actions/workflows/ci.yml/badge.svg)](https://github.com/h0rn3t/gopolars/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/h0rn3t/gopolars/graph/badge.svg)](https://codecov.io/gh/h0rn3t/gopolars)

`gopolars` is a high-performance Go DataFrame library inspired by Polars Python API.

## About

| | |
| --- | --- |
| **API docs** | [UK](https://h0rn3t.github.io/gopolars/) · [EN](https://h0rn3t.github.io/gopolars/en.html) — `pkg/polars` reference (Go syntax highlighting) |
| **Module** | `github.com/h0rn3t/gopolars/pkg/polars` |
| **Godoc** | [pkg.go.dev/.../pkg/polars](https://pkg.go.dev/github.com/h0rn3t/gopolars/pkg/polars) |

Set the repository **Website** (GitHub → Settings → About) to the API docs URL above so it appears in the sidebar.

## Current status

The project has completed parity waves through **v0.6** and now covers a broad core for Go-native analytics pipelines, including advanced joins, reshape operations, temporal windows, and performance diagnostics.\
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
- CTE support (`WITH ... AS (...)`)
- Subqueries in `FROM`
- Set operations: `UNION`, `INTERSECT`, `EXCEPT`
- Window expression support with `PARTITION BY` and `ORDER BY`
- Logical optimization passes including pushdown and adaptive planning rules

### IO and Interoperability

- CSV, JSON/NDJSON, Parquet, IPC read/write support
- Source-level lazy scan for CSV/JSON/Parquet/IPC
- Projection and predicate pushdown on scan pipelines
- Partition-aware Parquet dataset scan (multi-file layout)
- Partition pruning by predicate for dataset scans
- Arrow import/export bridge
- Object store URI mapping profile (`s3://`, `gcs://`, `az://`) via environment-configured roots

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
| GroupBy, temporal windows and joins                               | ✅ ready        |
| Streaming collect                                                 | ✅ ready        |
| CSV/JSON/Parquet/IPC IO                                           | ✅ ready        |
| Arrow interoperability                                            | ✅ ready        |
| Cloud-style partitioned dataset scans                             | ✅ ready        |
| Explain/telemetry schema v2 and perf markers                      | ✅ ready        |
| Full Python Polars API parity (680-method tracked matrix)         | ✅ ~99.3% (675/680) |
| Full SQL parity with Python Polars SQLContext                     | 🚧 in progress |
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
| `Filter` | `evalbatch.Plan` produces a `[]bool` mask; `chunk.Column.Filter` gathers with a single typed copy — no `expr.Eval` per row |
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
- `CompareGTFloat64`, `CompareEQInt64` — filter mask generation
- `AndMask` — boolean mask combination
- `CompressIndices` — compress bool mask to `[]int` for gather

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
| `group_by` sum/mean on 1M rows (100 groups) | ~4–6× (direct slice read, no per-row eval) |
| `sort` on 1M Int64/Float64 rows | ~1.5–2× (typed comparator, no boxing) |
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
