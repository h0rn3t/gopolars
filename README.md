# gopolars

`gopolars` is a high-performance Go DataFrame library inspired by Polars Python API.

## Current status

The project has completed parity waves through **v0.6** and now covers a broad core for Go-native analytics pipelines, including advanced joins, reshape operations, temporal windows, and performance diagnostics.  
It is production-usable for many DataFrame workloads, but it is **not yet a full drop-in replacement** for Python Polars.

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
- Unit/conformance tests, benchmarks, `go vet`, race tests, CI quality gates
- Compatibility governance artifacts:
  - versioning policy
  - migration notes
  - breaking-change evidence gate script
  - performance budget and regression evidence scripts

## Capability matrix

| Capability | Status |
| --- | --- |
| DataFrame eager API | ready |
| Lazy execution, scans, pushdown | ready |
| Series public API | ready |
| Nested transforms (explode/flatten + list/struct expr) | ready |
| SQL base + CTE + window expressions | ready |
| GroupBy, temporal windows and joins | ready |
| Streaming collect | ready |
| CSV/JSON/Parquet/IPC IO | ready |
| Arrow interoperability | ready |
| Cloud-style partitioned dataset scans | ready |
| Explain/telemetry schema v2 and perf markers | ready |
| Full Python Polars API parity | in progress |
| Full SQL parity with Python Polars SQLContext | in progress |
| Performance parity on all workloads | in progress |
| Ecosystem parity (all namespaces, plugins, advanced UDF patterns) | in progress |

## Python Polars vs gopolars function matrix

Status values are explicit: `реализовано` or `не реализовано`.

| Python Polars function | gopolars equivalent | Статус |
| --- | --- | --- |
| `DataFrame.select` | `DataFrame.Select` | реализовано |
| `DataFrame.filter` | `DataFrame.Filter` | реализовано |
| `DataFrame.with_columns` | `DataFrame.WithColumns` | реализовано |
| `DataFrame.group_by(...).agg(...)` | `DataFrame.GroupBy(...).Agg(...)` | реализовано |
| `DataFrame.group_by_dynamic` | `DataFrame.GroupByDynamic` | реализовано |
| `DataFrame.sort` | `DataFrame.Sort` | реализовано |
| `DataFrame.join` (`inner/left/right/full`) | `DataFrame.Join` + `JoinType*` | реализовано |
| `DataFrame.join` (`semi/anti/cross/asof`) | `DataFrame.Join` + `JoinTypeSemi/Anti/Cross/Asof` | реализовано |
| `DataFrame.melt` | `DataFrame.Melt` | реализовано |
| `DataFrame.pivot` | `DataFrame.Pivot` | реализовано |
| `DataFrame.explode` | `DataFrame.Explode` | реализовано |
| `DataFrame.unique` | `DataFrame.Unique` | реализовано |
| `DataFrame.fill_null` | `DataFrame.FillNull` | реализовано |
| `DataFrame.drop_nulls` | `DataFrame.DropNulls` | реализовано |
| `DataFrame.drop` | `DataFrame.Drop` | реализовано |
| `DataFrame.rename` | `DataFrame.Rename` | реализовано |
| `DataFrame.slice/head/tail` | `DataFrame.Slice/Head/Tail` | реализовано |
| `DataFrame.sample` | — | не реализовано |
| `DataFrame.partition_by` | — | не реализовано |
| `DataFrame.upsample` | — | не реализовано |
| `DataFrame.map_rows` / row UDF | — | не реализовано |
| `LazyFrame.collect` | `LazyFrame.Collect` | реализовано |
| `LazyFrame.collect(streaming=True)` | `LazyFrame.CollectStreaming` | реализовано |
| `LazyFrame.sink_csv/parquet/ipc` | `LazyFrame.SinkCSV/SinkParquet/SinkIPC` | реализовано |
| `LazyFrame.group_by_dynamic` | `LazyFrame.GroupByDynamic` | реализовано |
| `LazyFrame.rolling_mean` pattern | `LazyFrame.RollingMean` | реализовано |
| `Expr.str.to_lowercase` | `Expr.StrLower` | реализовано |
| `Expr.str.to_uppercase` | `Expr.StrUpper` | реализовано |
| `Expr.str.replace_all` | `Expr.StrReplace` | реализовано |
| `Expr.str.strip_chars` | `Expr.StrTrim` | реализовано |
| `Expr.str.starts_with` | `Expr.StartsWith` | реализовано |
| `Expr.dt.year/month/day/hour/weekday` | `Expr.DtYear/DtMonth/DtDay/DtHour/DtWeekday` | реализовано |
| `Expr.list.len` | `Expr.ListLen` | реализовано |
| `Expr.list.contains` | `Expr.ListContains` | реализовано |
| `Expr.list.get` | `Expr.ListGet` | реализовано |
| `Expr.struct.field` | `Expr.StructField` | реализовано |
| `SQLContext.execute` (`SELECT`) | `polars.SQLFromDataFrame` | реализовано |
| SQL `WITH` (CTE) | parser/binder/planner support | реализовано |
| SQL subquery in `FROM` | parser/binder/planner support | реализовано |
| SQL `UNION/INTERSECT/EXCEPT` | parser/planner/engine set-op nodes | реализовано |
| Multi-table SQL catalog (`register many frames`) | — | не реализовано |
| SQL `INSERT/UPDATE/DELETE` | — | не реализовано |

### Полная матрица методов Python Polars

- Полный список методов из Python Polars (stable) с проверкой статуса в gopolars: [docs/parity/python_polars_full_matrix.md](docs/parity/python_polars_full_matrix.md)
- Отдельный shortlist для ближайшей волны: [docs/parity/v0_7_top30_functions.md](docs/parity/v0_7_top30_functions.md)
- Методика: автоматическая сверка публичного API Python Polars с публичным API gopolars по объектам `DataFrame`, `LazyFrame`, `Expr`, `Series`, `SQLContext`.
- В полной матрице добавлена колонка `Приоритет` для всех нереализованных методов (`high` / `medium` / `low`).
- В конце файла добавлен раздел `Top-30 функций для v0.7 (по приоритету реализации)`.

| Объект | Реализовано | Всего в Python Polars |
| --- | --- | --- |
| DataFrame | 61 | 141 |
| LazyFrame | 25 | 89 |
| Expr | 27 | 217 |
| Series | 13 | 226 |
| SQLContext | 4 | 7 |
| Итого | 130 | 680 |

| Приоритет среди нереализованных | Количество |
| --- | --- |
| high | 9 |
| medium | 27 |
| low | 514 |

## What is still needed to replace Python Polars

To position `gopolars` as a practical replacement for Python Polars in most teams, the following areas still need expansion:

1. **Broader API parity**
   - Full namespace coverage across string/datetime/list/struct/window semantics
   - Deeper expression parity for edge-case behavior and error contracts

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

- **Near term:** expand SQL/catalog surface, close remaining namespace long-tail, and harden performance budgets on larger datasets.
- **Mid term:** improve planner/runtime adaptability for mixed temporal + join + reshape workloads.
- **Final parity push:** close long-tail semantic differences and publish repeatable parity evidence for release readiness.

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
