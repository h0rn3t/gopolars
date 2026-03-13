# gopolars

`gopolars` is a high-performance Go DataFrame library inspired by Polars Python API.

## Current status

The project has completed MVP waves through v0.4 planning/apply and now covers a strong core for Go-native analytics pipelines.  
It is production-usable for many DataFrame workloads, but it is **not yet a full drop-in replacement** for Python Polars.

## Implemented capabilities

### DataFrame, Series and Expressions

- Eager and lazy execution over a columnar in-memory DataFrame engine
- Core DataFrame operations: `select`, `filter`, `with_columns`, `sort`, `limit`, `group_by`, `join`
- Extended DataFrame surface: `slice`, `unique`, `concat`, `fill_null`, `drop_nulls`
- Join modes: `inner`, `left`, `right`, `full`
- Public `Series` API with null-aware operations, vector arithmetic and comparisons
- Nested expression support for list/struct workflows:
  - `list_len`
  - `struct_field`
  - `explode`
  - `flatten` (struct flattening)

### SQL and Query Planning

- SQL parsing and planning for `SELECT` pipelines
- CTE support (`WITH ... AS (...)`)
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
- Explain and diagnostics output with stable schema for automation
- Operator-level execution report structure for telemetry integrations
- Unit/conformance tests, benchmarks, `go vet`, race tests, CI quality gates
- Compatibility governance artifacts:
  - versioning policy
  - migration notes
  - breaking-change evidence gate script

## Capability matrix

| Capability | Status |
| --- | --- |
| DataFrame eager API | ready |
| Lazy execution, scans, pushdown | ready |
| Series public API | ready |
| Nested transforms (explode/flatten + list/struct expr) | ready |
| SQL base + CTE + window expressions | ready |
| GroupBy and joins | ready |
| Streaming collect | ready |
| CSV/JSON/Parquet/IPC IO | ready |
| Arrow interoperability | ready |
| Cloud-style partitioned dataset scans | ready |
| Full Python Polars API parity | in progress |
| Full SQL parity with Python Polars SQLContext | in progress |
| Performance parity on all workloads | in progress |
| Ecosystem parity (all namespaces, plugins, advanced UDF patterns) | in progress |

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

- **Near term:** close high-impact parity gaps for SQL/window/nested workflows and harden cloud dataset behavior.
- **Mid term:** broaden API namespaces and improve planner/runtime performance consistency.
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
